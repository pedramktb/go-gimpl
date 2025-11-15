package pgimpl

import (
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/pedramktb/go-gimpl"
)

// ApplySortsToQuery returns the given query with sorting and cursors applied. It also
// returns a reverse query for fetching the previous first result required for the prev cursor.
func ApplySortsToQuery(s gimpl.Sorts, query squirrel.SelectBuilder) (_ squirrel.SelectBuilder, reverse *squirrel.SelectBuilder) {
	rev := query
	if cursorCond := cursorQuery(s); cursorCond != nil {
		query = query.Where(cursorCond)
	}
	if len(s.Sorts) > 0 && s.Sorts[0].CursorPart != nil {
		rs := s.Reverse()
		if reverseCond := cursorQuery(rs); reverseCond != nil {
			rev = rev.Where(reverseCond)
		}
		r := orderQuery(rs, rev)
		reverse = &r
	}
	return orderQuery(s, query), reverse
}

func cursorQuery(s gimpl.Sorts) squirrel.Sqlizer {
	var cursorCond squirrel.Sqlizer
	for i := range s.Sorts {
		// No cursor at all
		if s.Sorts[i].CursorPart == nil {
			break
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
			if preCond == nil {
				preCond = squirrel.Eq{s.Sorts[j].Field: cursorPart}
				continue
			}
			preCond = squirrel.And{preCond, squirrel.Eq{s.Sorts[j].Field: cursorPart}}
		}

		var correspCond squirrel.Sqlizer
		cursorPart := normalizeNil(s.Sorts[i].CursorPart)
		if s.Sorts[i].Direction == gimpl.SortAsc {
			if cursorPart != nil {
				correspCond = squirrel.Or{squirrel.Gt{s.Sorts[i].Field: cursorPart}, squirrel.Eq{s.Sorts[i].Field: nil}}
			} else {
				continue
			}
		} else {
			if cursorPart != nil {
				correspCond = squirrel.Lt{s.Sorts[i].Field: cursorPart}
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

	return cursorCond
}

func orderQuery(s gimpl.Sorts, query squirrel.SelectBuilder) squirrel.SelectBuilder {
	for i := range s.Sorts {
		query = query.OrderBy(fmt.Sprintf("%s %s", s.Sorts[i].Field, s.Sorts[i].Direction))
	}
	return query
}
