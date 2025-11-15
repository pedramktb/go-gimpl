package gimpl

import (
	"bytes"
	"encoding/base64"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
)

const (
	PaginationDefaultLimit PaginationLimit = 10
)

type PaginationLimit uint8

type PaginationMeta struct {
	Total uint64
	Next  Cursor
	Prev  Cursor
}

type Paginated[E Entity] struct {
	Items []E
	Meta  PaginationMeta
}

// Cursor is the list of values in the same order of sorts
type Cursor []any

type SortDirection string

const (
	SortAsc  SortDirection = "asc"
	SortDesc SortDirection = "desc"
)

func (d SortDirection) isValid() bool {
	return d == SortAsc || d == SortDesc
}

// Cursor is here because its meaningless without Sort.
type Sort struct {
	Field      string
	Direction  SortDirection
	CursorPart any
}

type Sorts struct {
	Sample Entity
	Sorts  []Sort
}

func (s Sorts) Reverse() Sorts {
	rs := make([]Sort, len(s.Sorts))
	copy(rs, s.Sorts)
	for i := range rs {
		switch rs[i].Direction {
		case SortAsc:
			rs[i].Direction = SortDesc
		case SortDesc:
			rs[i].Direction = SortAsc
		}
	}
	return Sorts{
		Sample: s.Sample,
		Sorts:  rs,
	}
}

// Cursor returns the cursor for a given item.
// The item is not included in the result but every result comes after it.
func (s Sorts) Cursor(cursorItem Entity) Cursor {
	if cursorItem == nil {
		return nil
	}
	cursor := make(Cursor, len(s.Sorts))
	for i := range s.Sorts {
		cursor[i] = reflect.ValueOf(cursorItem.Pointer(s.Sorts[i].Field)).Elem().Interface()
	}
	return cursor
}

// Cursors returns the next cursor from the last item in the results and
// the prev cursor from the first item in the previous results.
func (s Sorts) Cursors(lastItem, prevLastItem Entity) (next, prev Cursor) {
	return s.Cursor(lastItem), s.Cursor(prevLastItem)
}

func (s *Sorts) FromStr(sorts []string, cursor string) error {
	// Add UUID as the last sorter to allow cursor pagination
	sorts = append(sorts, "id:asc")

	var cur []string
	if cursor != "" {
		// Split the cursor into parts
		curBytes, err := base64.StdEncoding.DecodeString(cursor)
		if err != nil {
			return ErrInvalidCursor.Wrap(err)
		}

		cur, err = csv.NewReader(bytes.NewReader(bytes.ReplaceAll(curBytes, []byte{'"'}, []byte{'"', '"', '"'}))).Read()
		if err != nil {
			return ErrInvalidCursor.Wrap(err)
		}

		// Either null cursor or the same number of parts as sorters
		if len(cur) != len(sorts) {
			return ErrInvalidCursor.Wrap(ErrMismatchInSortAndCursor)
		}
	}

	s.Sorts = make([]Sort, len(sorts))
	for i := range sorts {
		split := strings.SplitN(sorts[i], ":", 2)
		if !SortDirection(split[1]).isValid() {
			return errors.New("invalid sort direction")
		}
		s.Sorts[i] = Sort{
			Field:     split[0],
			Direction: SortDirection(split[1]),
		}

		// Create cursor with the same type as the field
		cursorPart := s.Sample.Pointer(split[0])
		if cursorPart == nil {
			return fmt.Errorf("field %s not found", split[0])
		}
		if len(cur) != 0 {
			// If there is a cursor part, unmarshal it
			if err := json.Unmarshal([]byte(cur[i]), cursorPart); err != nil {
				return ErrInvalidCursor.Wrap(err)
			}
			s.Sorts[i].CursorPart = reflect.ValueOf(cursorPart).Elem().Interface()
		}
	}

	return nil
}
