package pgimpl_test

import (
	"testing"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/pedramktb/go-gimpl"
	pgimpl "github.com/pedramktb/go-gimpl/pg"
	"github.com/stretchr/testify/require"
)

var _ pgimpl.Entity = PaginationTestEntity{}

type PaginationTestEntity struct {
	ID     uuid.UUID
	Name   string
	Score  int
	Active bool
}

func (e PaginationTestEntity) FilterPtr(field string) any { return nil }

func (e PaginationTestEntity) SortPtr(field string) any {
	switch field {
	case "id":
		return &e.ID
	case "name":
		return &e.Name
	case "score":
		return &e.Score
	case "active":
		return &e.Active
	default:
		return nil
	}
}

func (e PaginationTestEntity) Column(field string) string {
	switch field {
	case "id":
		return "pg_id"
	case "name":
		return "pg_name"
	case "score":
		return "pg_score"
	case "active":
		return "pg_active"
	default:
		return ""
	}
}

func (e PaginationTestEntity) Columns() []string {
	return []string{"pg_id", "pg_name", "pg_score", "pg_active"}
}

func (e PaginationTestEntity) NewWithColumnPtrs() (pgimpl.Entity, []any) {
	return e, []any{&e.ID, &e.Name, &e.Score, &e.Active}
}

// helper to get SQL + args from FromSorts
func sortsToSQL(t *testing.T, s gimpl.Sorts, base squirrel.SelectBuilder) (string, []any, *squirrel.SelectBuilder) {
	t.Helper()
	query, reverse, err := pgimpl.FromSorts(s, base)
	require.NoError(t, err)
	sql, args, err := query.ToSql()
	require.NoError(t, err)
	return sql, args, reverse
}

func TestFromSorts_Empty(t *testing.T) {
	base := squirrel.Select("*").From("test")
	s := gimpl.Sorts{Sample: PaginationTestEntity{}, Sorts: []gimpl.Sort{}}

	query, reverse, err := pgimpl.FromSorts(s, base)
	require.NoError(t, err)
	require.Nil(t, reverse)

	sql, args, err := query.ToSql()
	require.NoError(t, err)
	require.Equal(t, "SELECT * FROM test", sql)
	require.Empty(t, args)
}

func TestFromSorts_NoSample(t *testing.T) {
	base := squirrel.Select("*").From("test")
	s := gimpl.Sorts{
		Sorts: []gimpl.Sort{{Field: "name", Direction: gimpl.SortAsc}},
	}

	_, _, err := pgimpl.FromSorts(s, base)
	require.Error(t, err)
	require.ErrorIs(t, err, gimpl.ErrInvalidSorting)
}

type invalidEntity struct{}

func (invalidEntity) FilterPtr(field string) any               { return nil }
func (invalidEntity) SortPtr(field string) any                 { return nil }
func (invalidEntity) Column(field string) string               { return "" }
func (invalidEntity) Columns() []string                        { return nil }
func (invalidEntity) NewWithColumnPtrs() (gimpl.Entity, []any) { return invalidEntity{}, nil }

func TestFromSorts_InvalidSample(t *testing.T) {
	base := squirrel.Select("*").From("test")
	s := gimpl.Sorts{
		Sample: invalidEntity{}, // Not a pgimpl.Entity
		Sorts:  []gimpl.Sort{{Field: "name", Direction: gimpl.SortAsc}},
	}

	_, _, err := pgimpl.FromSorts(s, base)
	require.Error(t, err)
	require.ErrorIs(t, err, gimpl.ErrInvalidSorting)
}

func TestFromSorts_SingleFieldAsc(t *testing.T) {
	base := squirrel.Select("*").From("test")
	s := gimpl.Sorts{
		Sample: PaginationTestEntity{},
		Sorts:  []gimpl.Sort{{Field: "name", Direction: gimpl.SortAsc}},
	}

	sql, args, reverse := sortsToSQL(t, s, base)
	require.Equal(t, "SELECT * FROM test ORDER BY pg_name asc", sql)
	require.Empty(t, args)
	require.Nil(t, reverse)
}

func TestFromSorts_SingleFieldDesc(t *testing.T) {
	base := squirrel.Select("*").From("test")
	s := gimpl.Sorts{
		Sample: PaginationTestEntity{},
		Sorts:  []gimpl.Sort{{Field: "score", Direction: gimpl.SortDesc}},
	}

	sql, args, reverse := sortsToSQL(t, s, base)
	require.Equal(t, "SELECT * FROM test ORDER BY pg_score desc", sql)
	require.Empty(t, args)
	require.Nil(t, reverse)
}

func TestFromSorts_MultipleFields(t *testing.T) {
	base := squirrel.Select("*").From("test")
	s := gimpl.Sorts{
		Sample: PaginationTestEntity{},
		Sorts: []gimpl.Sort{
			{Field: "score", Direction: gimpl.SortDesc},
			{Field: "name", Direction: gimpl.SortAsc},
		},
	}

	sql, args, reverse := sortsToSQL(t, s, base)
	require.Equal(t, "SELECT * FROM test ORDER BY pg_score desc, pg_name asc", sql)
	require.Empty(t, args)
	require.Nil(t, reverse)
}

func TestFromSorts_InvalidField(t *testing.T) {
	base := squirrel.Select("*").From("test")
	s := gimpl.Sorts{
		Sample: PaginationTestEntity{},
		Sorts:  []gimpl.Sort{{Field: "nonexistent", Direction: gimpl.SortAsc}},
	}

	_, _, err := pgimpl.FromSorts(s, base)
	require.Error(t, err)
	require.ErrorIs(t, err, gimpl.ErrInvalidSorting)
}

func TestFromSorts_WithCursorAsc(t *testing.T) {
	base := squirrel.Select("*").From("test")
	s := gimpl.Sorts{
		Sample: PaginationTestEntity{},
		Sorts: []gimpl.Sort{
			{Field: "name", Direction: gimpl.SortAsc, CursorPart: "alice"},
		},
	}

	sql, args, reverse := sortsToSQL(t, s, base)
	require.Contains(t, sql, "WHERE")
	require.Contains(t, sql, "pg_name > ?")
	require.Contains(t, sql, "ORDER BY pg_name asc")
	require.Equal(t, []any{"alice"}, args)
	require.NotNil(t, reverse)
}

func TestFromSorts_WithCursorDesc(t *testing.T) {
	base := squirrel.Select("*").From("test")
	s := gimpl.Sorts{
		Sample: PaginationTestEntity{},
		Sorts: []gimpl.Sort{
			{Field: "score", Direction: gimpl.SortDesc, CursorPart: 100},
		},
	}

	sql, args, reverse := sortsToSQL(t, s, base)
	require.Contains(t, sql, "WHERE")
	require.Contains(t, sql, "pg_score < ?")
	require.Contains(t, sql, "ORDER BY pg_score desc")
	require.Equal(t, []any{100}, args)
	require.NotNil(t, reverse)
}

func TestFromSorts_WithNilCursorAsc(t *testing.T) {
	base := squirrel.Select("*").From("test")
	var nilName *string
	s := gimpl.Sorts{
		Sample: PaginationTestEntity{},
		Sorts: []gimpl.Sort{
			{Field: "name", Direction: gimpl.SortAsc, CursorPart: nilName},
		},
	}

	sql, args, reverse := sortsToSQL(t, s, base)
	// Nil cursor with asc should be skipped (no WHERE clause)
	require.NotContains(t, sql, "WHERE")
	require.Contains(t, sql, "ORDER BY pg_name asc")
	require.Empty(t, args)
	require.NotNil(t, reverse)
}

func TestFromSorts_WithNilCursorDesc(t *testing.T) {
	base := squirrel.Select("*").From("test")
	var nilScore *int
	s := gimpl.Sorts{
		Sample: PaginationTestEntity{},
		Sorts: []gimpl.Sort{
			{Field: "score", Direction: gimpl.SortDesc, CursorPart: nilScore},
		},
	}

	sql, args, reverse := sortsToSQL(t, s, base)
	// Nil cursor with desc should be skipped (no WHERE clause added for this part)
	require.NotContains(t, sql, "WHERE")
	require.Contains(t, sql, "ORDER BY pg_score desc")
	require.Empty(t, args)
	require.NotNil(t, reverse)
}

func TestFromSorts_MultiFieldWithCursor(t *testing.T) {
	base := squirrel.Select("*").From("test")
	s := gimpl.Sorts{
		Sample: PaginationTestEntity{},
		Sorts: []gimpl.Sort{
			{Field: "score", Direction: gimpl.SortDesc, CursorPart: 100},
			{Field: "name", Direction: gimpl.SortAsc, CursorPart: "alice"},
		},
	}

	sql, args, reverse := sortsToSQL(t, s, base)
	require.Contains(t, sql, "WHERE")
	// Should have cursor conditions for both fields
	require.Contains(t, sql, "pg_score")
	require.Contains(t, sql, "pg_name")
	require.Contains(t, sql, "ORDER BY pg_score desc, pg_name asc")
	require.Equal(t, []any{100, 100, "alice"}, args)
	require.NotNil(t, reverse)
}

func TestFromSorts_MultiFieldPartialCursor(t *testing.T) {
	base := squirrel.Select("*").From("test")
	s := gimpl.Sorts{
		Sample: PaginationTestEntity{},
		Sorts: []gimpl.Sort{
			{Field: "score", Direction: gimpl.SortDesc, CursorPart: 100},
			{Field: "name", Direction: gimpl.SortAsc}, // No cursor for second field
		},
	}

	sql, args, reverse := sortsToSQL(t, s, base)
	require.Contains(t, sql, "WHERE")
	require.Contains(t, sql, "pg_score < ?")
	require.Contains(t, sql, "ORDER BY pg_score desc, pg_name asc")
	require.Equal(t, []any{100}, args)
	// Reverse query is created since first sort has a cursor part
	require.NotNil(t, reverse)
}

func TestFromSorts_ReverseQuery(t *testing.T) {
	base := squirrel.Select("*").From("test")
	s := gimpl.Sorts{
		Sample: PaginationTestEntity{},
		Sorts: []gimpl.Sort{
			{Field: "name", Direction: gimpl.SortAsc, CursorPart: "bob"},
		},
	}

	query, reverse, err := pgimpl.FromSorts(s, base)
	require.NoError(t, err)
	require.NotNil(t, reverse)

	// Check main query
	sql, args, err := query.ToSql()
	require.NoError(t, err)
	require.Contains(t, sql, "pg_name > ?")
	require.Contains(t, sql, "ORDER BY pg_name asc")
	require.Equal(t, []any{"bob"}, args)

	// Check reverse query - should have opposite direction and condition
	revSQL, revArgs, err := reverse.ToSql()
	require.NoError(t, err)
	require.Contains(t, revSQL, "pg_name < ?")
	require.Contains(t, revSQL, "ORDER BY pg_name desc")
	require.Equal(t, []any{"bob"}, revArgs)
}

func TestFromSorts_ComplexCursor(t *testing.T) {
	base := squirrel.Select("*").From("test")
	testID := uuid.New()
	s := gimpl.Sorts{
		Sample: PaginationTestEntity{},
		Sorts: []gimpl.Sort{
			{Field: "score", Direction: gimpl.SortDesc, CursorPart: 95},
			{Field: "name", Direction: gimpl.SortAsc, CursorPart: "charlie"},
			{Field: "id", Direction: gimpl.SortAsc, CursorPart: testID},
		},
	}

	sql, args, reverse := sortsToSQL(t, s, base)
	require.Contains(t, sql, "WHERE")
	require.Contains(t, sql, "ORDER BY pg_score desc, pg_name asc, pg_id asc")
	// Should have multiple cursor conditions
	require.Len(t, args, 6) // score (1) + score,name (2) + score,name,id (3)
	require.NotNil(t, reverse)
}

func TestFromSorts_InvalidCursorField(t *testing.T) {
	base := squirrel.Select("*").From("test")
	s := gimpl.Sorts{
		Sample: PaginationTestEntity{},
		Sorts: []gimpl.Sort{
			{Field: "nonexistent", Direction: gimpl.SortAsc, CursorPart: "value"},
		},
	}

	_, _, err := pgimpl.FromSorts(s, base)
	require.Error(t, err)
	require.ErrorIs(t, err, gimpl.ErrInvalidCursor)
}

func TestFromSorts_CursorWithNilHandling(t *testing.T) {
	base := squirrel.Select("*").From("test")
	s := gimpl.Sorts{
		Sample: PaginationTestEntity{},
		Sorts: []gimpl.Sort{
			{Field: "name", Direction: gimpl.SortAsc, CursorPart: "test"},
		},
	}

	sql, args, reverse := sortsToSQL(t, s, base)
	// For ASC with non-nil cursor, should handle both > cursor and IS NULL
	require.Contains(t, sql, "pg_name > ?")
	require.Contains(t, sql, "pg_name IS NULL")
	require.Equal(t, []any{"test"}, args)
	require.NotNil(t, reverse)
}
