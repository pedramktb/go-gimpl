package pgimpl_test

import (
	"encoding/json"
	"testing"

	"github.com/Masterminds/squirrel"
	"github.com/pedramktb/go-gimpl"
	pgimpl "github.com/pedramktb/go-gimpl/pg"
	"github.com/stretchr/testify/require"
)

var _ pgimpl.Entity = ExprTestEntity{}

type ExprTestEntity struct{}

func (e ExprTestEntity) FilterPtr(field string) any      { return nil }
func (e ExprTestEntity) SortPtr(field string) any        { return nil }
func (e ExprTestEntity) Column(field string) string      { return "pg_" + field }
func (e ExprTestEntity) Columns() []string               { return nil }
func (e ExprTestEntity) NewWithColumnPtrs() (any, []any) { return &e, nil }

// helper to get SQL + args from FromExpr
func toSQL(t *testing.T, e gimpl.Expr) (string, []any) {
	t.Helper()
	sqlizer, err := pgimpl.FromExpr(e)
	require.NoError(t, err)
	q, args, err := sqlizer.ToSql()
	require.NoError(t, err)
	return q, args
}

func TestFromExpr_SimpleColumn(t *testing.T) {
	e := gimpl.Expr{Sample: ExprTestEntity{}, Expr: gimpl.CondExpr{Field: "status", Op: gimpl.CondOpEQ, Val: "active"}}
	q, args := toSQL(t, e)
	require.Equal(t, "pg_status = ?", q)
	require.Equal(t, []any{"active"}, args)
}

func TestFromExpr_MissingSample(t *testing.T) {
	e := gimpl.Expr{Expr: gimpl.CondExpr{Field: "status", Op: gimpl.CondOpEQ, Val: "active"}}
	_, err := pgimpl.FromExpr(e)
	require.Error(t, err)
	require.ErrorIs(t, err, gimpl.ErrInvalidExpr)
}

func TestFromExpr_JSONPathSimple(t *testing.T) {
	e := gimpl.Expr{Sample: ExprTestEntity{}, Expr: gimpl.CondExpr{Field: "meta.version", Op: gimpl.CondOpEQ, Val: "1"}}
	q, args := toSQL(t, e)
	marshaled, _ := json.Marshal("1")
	require.Equal(t, "pg_meta #> '{version}' = ?::jsonb", q)
	require.Equal(t, []any{marshaled}, args)
}

func TestFromExpr_LogicalAnd(t *testing.T) {
	left := gimpl.Expr{Sample: ExprTestEntity{}, Expr: gimpl.CondExpr{Field: "status", Op: gimpl.CondOpEQ, Val: "active"}}
	right := gimpl.Expr{Sample: ExprTestEntity{}, Expr: gimpl.CondExpr{Field: "meta.version", Op: gimpl.CondOpEQ, Val: "1"}}
	e := gimpl.Expr{Sample: ExprTestEntity{}, Expr: gimpl.LogExpr{Left: left, Op: gimpl.LogOpAnd, Right: right}}
	q, args := toSQL(t, e)
	marshaled, _ := json.Marshal("1")
	require.Equal(t, "(pg_status = ? AND pg_meta #> '{version}' = ?::jsonb)", q)
	require.Equal(t, []any{"active", marshaled}, args)
}

func TestFromExpr_RegexColumn(t *testing.T) {
	e := gimpl.Expr{Sample: ExprTestEntity{}, Expr: gimpl.CondExpr{Field: "status", Op: gimpl.CondOpRE, Val: "^act"}}
	q, args := toSQL(t, e)
	require.Equal(t, "pg_status ~ ?", q)
	require.Equal(t, []any{"^act"}, args)
}

func TestFromExpr_RegexJSON(t *testing.T) {
	e := gimpl.Expr{Sample: ExprTestEntity{}, Expr: gimpl.CondExpr{Field: "meta.name", Op: gimpl.CondOpRE, Val: "^foo"}}
	q, args := toSQL(t, e)
	require.Equal(t, "(pg_meta #> '{name}') #>> '{}' ~ ?", q)
	require.Equal(t, []any{"^foo"}, args)
}

func TestFromExpr_IN_Column(t *testing.T) {
	vals := []string{"a", "b"}
	e := gimpl.Expr{Sample: ExprTestEntity{}, Expr: gimpl.CondExpr{Field: "status", Op: gimpl.CondOpIN, Val: vals}}
	q, args := toSQL(t, e)
	require.Equal(t, "pg_status = ANY (?)", q)
	require.Equal(t, []any{vals}, args)
}

func TestFromExpr_NIN_Column(t *testing.T) {
	vals := []string{"x", "y"}
	e := gimpl.Expr{Sample: ExprTestEntity{}, Expr: gimpl.CondExpr{Field: "status", Op: gimpl.CondOpNIN, Val: vals}}
	q, args := toSQL(t, e)
	require.Equal(t, "pg_status <> ALL (?)", q)
	require.Equal(t, []any{vals}, args)
}

func TestFromExpr_IN_JSON(t *testing.T) {
	vals := []int{1, 2}
	e := gimpl.Expr{Sample: ExprTestEntity{}, Expr: gimpl.CondExpr{Field: "meta.ids", Op: gimpl.CondOpIN, Val: vals}}
	q, args := toSQL(t, e)
	marshaled, _ := json.Marshal(vals)
	require.Equal(t, "pg_meta #> '{ids}' <@ ?::jsonb", q)
	require.Equal(t, []any{marshaled}, args)
}

func TestFromExpr_QuantAny(t *testing.T) {
	// ANY items where element.id = 1
	inner := gimpl.Expr{Sample: ExprTestEntity{}, Expr: gimpl.CondExpr{Field: "id", Op: gimpl.CondOpEQ, Val: 1}}
	e := gimpl.Expr{Sample: ExprTestEntity{}, Expr: gimpl.QuantExpr{Field: "items", Op: gimpl.QuantOpAny, Expr: inner}}
	q, args := toSQL(t, e)
	marshaledOne, _ := json.Marshal(1)
	expected := "EXISTS (SELECT 1 FROM unnest(pg_items) AS q0(elem) WHERE q0.elem #> '{pg_id}' = ?::jsonb)"
	require.Equal(t, expected, q)
	require.Equal(t, []any{marshaledOne}, args)
}

func TestFromExpr_QuantAll(t *testing.T) {
	inner := gimpl.Expr{Sample: ExprTestEntity{}, Expr: gimpl.CondExpr{Field: "id", Op: gimpl.CondOpGT, Val: 10}}
	e := gimpl.Expr{Sample: ExprTestEntity{}, Expr: gimpl.QuantExpr{Field: "items", Op: gimpl.QuantOpAll, Expr: inner}}
	q, args := toSQL(t, e)
	marshaledTen, _ := json.Marshal(10)
	expected := "NOT EXISTS (SELECT 1 FROM unnest(pg_items) AS q0(elem) WHERE NOT (q0.elem #> '{pg_id}' > ?::jsonb))"
	require.Equal(t, expected, q)
	require.Equal(t, []any{marshaledTen}, args)
}

func TestFromExpr_NestedQuantifiers(t *testing.T) {
	// ANY items where ALL sub.id eq 5
	deepest := gimpl.Expr{Sample: ExprTestEntity{}, Expr: gimpl.CondExpr{Field: "id", Op: gimpl.CondOpEQ, Val: 5}}
	allSub := gimpl.Expr{Sample: ExprTestEntity{}, Expr: gimpl.QuantExpr{Field: "sub", Op: gimpl.QuantOpAll, Expr: deepest}}
	anyItems := gimpl.Expr{Sample: ExprTestEntity{}, Expr: gimpl.QuantExpr{Field: "items", Op: gimpl.QuantOpAny, Expr: allSub}}
	q, args := toSQL(t, anyItems)
	marshaledFive, _ := json.Marshal(5)
	// Validate structural pieces instead of full exact string to reduce brittleness
	require.Len(t, args, 1)
	require.Equal(t, marshaledFive, args[0])
	require.Contains(t, q, "EXISTS (SELECT 1 FROM unnest(pg_items) AS q0(elem) WHERE ")
	require.Contains(t, q, "NOT EXISTS (SELECT 1 FROM jsonb_array_elements(CASE WHEN jsonb_typeof(q0.elem #> '{pg_sub}') = 'array' THEN q0.elem #> '{pg_sub}' ELSE '[]'::jsonb END) AS q1(elem) WHERE NOT (q1.elem #> '{pg_id}' = ?::jsonb))")
}

func TestFromExpr_InvalidOperator(t *testing.T) {
	e := gimpl.Expr{Sample: ExprTestEntity{}, Expr: gimpl.CondExpr{Field: "name", Op: gimpl.CondOp("bogus"), Val: 1}}
	_, err := pgimpl.FromExpr(e)
	require.Error(t, err)
	require.ErrorIs(t, err, gimpl.ErrInvalidExpr)
}

func TestFromExpr_NoExpr(t *testing.T) {
	base := squirrel.Select("pg_id").From("table")
	// filters.Expr nil triggers passthrough
	filters := gimpl.Expr{} // zero value; Expr field is nil
	filter, err := pgimpl.FromExpr(filters)
	out := base.Where(filter)
	require.NoError(t, err)
	sqlStr, args, err := out.ToSql()
	require.NoError(t, err)
	require.Equal(t, "SELECT pg_id FROM table", sqlStr)
	require.Empty(t, args)
}

func TestApplyFromExpr_WithExpr(t *testing.T) {
	base := squirrel.Select("pg_id").From("table")
	filters := gimpl.Expr{Sample: ExprTestEntity{}, Expr: gimpl.CondExpr{Field: "name", Op: gimpl.CondOpEQ, Val: "active"}}
	filter, err := pgimpl.FromExpr(filters)
	out := base.Where(filter)
	require.NoError(t, err)
	sqlStr, args, err := out.ToSql()
	require.NoError(t, err)
	require.Equal(t, "SELECT pg_id FROM table WHERE pg_name = ?", sqlStr)
	require.Equal(t, []any{"active"}, args)
}
