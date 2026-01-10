package pgimpl

import (
	"errors"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/pedramktb/go-gimpl"
	"github.com/pedramktb/go-tagerr"
)

// FromSorts returns the given query with sorting and cursors applied. It also
// returns a reverse query for fetching the previous first result required for the prev cursor.
func FromSorts(s gimpl.Sorts, query squirrel.SelectBuilder) (_ squirrel.SelectBuilder, reverse *squirrel.SelectBuilder, err error) {
	if len(s.Sorts) == 0 {
		return query, nil, nil
	}
	if s.Sample == nil {
		return squirrel.SelectBuilder{}, nil, tagerr.ErrInternal.Wrap(gimpl.ErrInvalidSorting.Wrap(errors.New("Sorts.Sample must be set")))
	}
	sample, ok := s.Sample.(Entity)
	if !ok {
		return squirrel.SelectBuilder{}, nil, tagerr.ErrInternal.Wrap(gimpl.ErrInvalidSorting.Wrap(errors.New("Sorts.Sample must implement pgimpl.Entity interface")))
	}
	rev := query
	if cursorCond, err := cursorQuery(s, sample.PgColumn); err != nil {
		return squirrel.SelectBuilder{}, nil, err
	} else if cursorCond != nil {
		query = query.Where(cursorCond)
	}
	if len(s.Sorts) > 0 && s.Sorts[0].CursorPart != nil {
		rs := s.Reverse()
		if reverseCond, err := cursorQuery(rs, sample.PgColumn); err != nil {
			return squirrel.SelectBuilder{}, nil, err
		} else if reverseCond != nil {
			rev = rev.Where(reverseCond)
		}
		r, err := orderQuery(rs, rev, sample.PgColumn)
		if err != nil {
			return squirrel.SelectBuilder{}, nil, err
		}
		reverse = &r
	}
	query, err = orderQuery(s, query, sample.PgColumn)
	if err != nil {
		return squirrel.SelectBuilder{}, nil, err
	}
	return query, reverse, nil
}

func cursorQuery(s gimpl.Sorts, fieldColumn func(string) string) (squirrel.Sqlizer, error) {
	var cursorCond squirrel.Sqlizer
	for i := range s.Sorts {
		// No cursor at all
		if s.Sorts[i].CursorPart == nil {
			break
		}

		column := fieldColumn(s.Sorts[i].Field)
		if column == "" {
			return nil, tagerr.ErrInternal.Wrap(gimpl.ErrInvalidCursor.Wrap(fmt.Errorf("field %q has no corresponding column", s.Sorts[i].Field)))
		}

		// In i'th sort/cursor part we need previous one as equal to the cursor
		var preCond squirrel.Sqlizer
		for j := range i {
			cursorPart := normalizeNil(s.Sorts[j].CursorPart)
			if cursorPart == nil && s.Sorts[j].Direction == gimpl.SortDesc {
				// If the cursor part is nil and the direction is descending, it means we want
				// all non-nil values for this field, so we skip adding any condition for this part
				continue
			}
			column := fieldColumn(s.Sorts[j].Field)
			if column == "" {
				return nil, tagerr.ErrInternal.Wrap(gimpl.ErrInvalidCursor.Wrap(fmt.Errorf("field %q has no corresponding column", s.Sorts[j].Field)))
			}
			if preCond == nil {
				preCond = squirrel.Eq{column: cursorPart}
				continue
			}
			preCond = squirrel.And{preCond, squirrel.Eq{column: cursorPart}}
		}

		var correspCond squirrel.Sqlizer
		cursorPart := normalizeNil(s.Sorts[i].CursorPart)
		if s.Sorts[i].Direction == gimpl.SortAsc {
			if cursorPart != nil {
				correspCond = squirrel.Or{squirrel.Gt{column: cursorPart}, squirrel.Eq{column: nil}}
			} else {
				continue
			}
		} else {
			if cursorPart != nil {
				correspCond = squirrel.Lt{column: cursorPart}
			} else {
				continue
			}
		}

		if preCond != nil {
			correspCond = squirrel.And{preCond, correspCond}
		}

		// If its the first condition, set it as the cursorCond else add it to the cursorCond
		if cursorCond == nil {
			cursorCond = correspCond
			continue
		}
		cursorCond = squirrel.Or{cursorCond, correspCond}
	}

	return cursorCond, nil
}

func orderQuery(s gimpl.Sorts, query squirrel.SelectBuilder, fieldColumn func(string) string) (squirrel.SelectBuilder, error) {
	for i := range s.Sorts {
		column := fieldColumn(s.Sorts[i].Field)
		if column == "" {
			return squirrel.SelectBuilder{}, tagerr.ErrInternal.Wrap(gimpl.ErrInvalidSorting.Wrap(fmt.Errorf("field %q has no corresponding column", s.Sorts[i].Field)))
		}
		query = query.OrderBy(fmt.Sprintf("%s %s", column, s.Sorts[i].Direction))
	}
	return query, nil
}
