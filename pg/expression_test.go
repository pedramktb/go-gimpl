package pgimpl_test

import (
	"encoding/json"
	"testing"

	"github.com/Masterminds/squirrel"
	"github.com/pedramktb/go-gimpl"
	pgimpl "github.com/pedramktb/go-gimpl/pg"
	"github.com/stretchr/testify/require"
)

// helper to get SQL + args from ExprToQuery
func toSQL(t *testing.T, e gimpl.Expr) (string, []any) {
	t.Helper()
	sqlizer, err := pgimpl.ExprToQuery(e)
	require.NoError(t, err)
	q, args, err := sqlizer.ToSql()
	require.NoError(t, err)
	return q, args
}

func TestExprToQuery_SimpleColumn(t *testing.T) {
	e := gimpl.Expr{Expr: gimpl.CondExpr{Field: "status", Op: gimpl.CondOpEQ, Val: "active"}}
	q, args := toSQL(t, e)
	require.Equal(t, "status = ?", q)
	require.Equal(t, []any{"active"}, args)
}

func TestExprToQuery_JSONPathSimple(t *testing.T) {
	e := gimpl.Expr{Expr: gimpl.CondExpr{Field: "meta.version", Op: gimpl.CondOpEQ, Val: "1"}}
	q, args := toSQL(t, e)
	marshaled, _ := json.Marshal("1")
	require.Equal(t, "meta #> '{version}' = ?::jsonb", q)
	require.Equal(t, []any{marshaled}, args)
}

func TestExprToQuery_LogicalAnd(t *testing.T) {
	left := gimpl.Expr{Expr: gimpl.CondExpr{Field: "status", Op: gimpl.CondOpEQ, Val: "active"}}
	right := gimpl.Expr{Expr: gimpl.CondExpr{Field: "meta.version", Op: gimpl.CondOpEQ, Val: "1"}}
	e := gimpl.Expr{Expr: gimpl.LogExpr{Left: left, Op: gimpl.LogOpAnd, Right: right}}
	q, args := toSQL(t, e)
	marshaled, _ := json.Marshal("1")
	require.Equal(t, "(status = ? AND meta #> '{version}' = ?::jsonb)", q)
	require.Equal(t, []any{"active", marshaled}, args)
}

func TestExprToQuery_RegexColumn(t *testing.T) {
	e := gimpl.Expr{Expr: gimpl.CondExpr{Field: "status", Op: gimpl.CondOpRE, Val: "^act"}}
	q, args := toSQL(t, e)
	require.Equal(t, "status ~ ?", q)
	require.Equal(t, []any{"^act"}, args)
}

func TestExprToQuery_RegexJSON(t *testing.T) {
	e := gimpl.Expr{Expr: gimpl.CondExpr{Field: "meta.name", Op: gimpl.CondOpRE, Val: "^foo"}}
	q, args := toSQL(t, e)
	require.Equal(t, "(meta #> '{name}') #>> '{}' ~ ?", q)
	require.Equal(t, []any{"^foo"}, args)
}

func TestExprToQuery_IN_Column(t *testing.T) {
	vals := []string{"a", "b"}
	e := gimpl.Expr{Expr: gimpl.CondExpr{Field: "status", Op: gimpl.CondOpIN, Val: vals}}
	q, args := toSQL(t, e)
	require.Equal(t, "status = ANY (?)", q)
	require.Equal(t, []any{vals}, args)
}

func TestExprToQuery_NIN_Column(t *testing.T) {
	vals := []string{"x", "y"}
	e := gimpl.Expr{Expr: gimpl.CondExpr{Field: "status", Op: gimpl.CondOpNIN, Val: vals}}
	q, args := toSQL(t, e)
	require.Equal(t, "status <> ALL (?)", q)
	require.Equal(t, []any{vals}, args)
}

func TestExprToQuery_IN_JSON(t *testing.T) {
	vals := []int{1, 2}
	e := gimpl.Expr{Expr: gimpl.CondExpr{Field: "meta.ids", Op: gimpl.CondOpIN, Val: vals}}
	q, args := toSQL(t, e)
	marshaled, _ := json.Marshal(vals)
	require.Equal(t, "meta #> '{ids}' <@ ?::jsonb", q)
	require.Equal(t, []any{marshaled}, args)
}

func TestExprToQuery_QuantAny(t *testing.T) {
	// ANY items where element.id = 1
	inner := gimpl.Expr{Expr: gimpl.CondExpr{Field: "id", Op: gimpl.CondOpEQ, Val: 1}}
	e := gimpl.Expr{Expr: gimpl.QuantExpr{Field: "items", Op: gimpl.QuantOpAny, Expr: inner}}
	q, args := toSQL(t, e)
	marshaledOne, _ := json.Marshal(1)
	expected := "EXISTS (SELECT 1 FROM unnest(items) AS q0(elem) WHERE q0.elem #> '{id}' = ?::jsonb)"
	require.Equal(t, expected, q)
	require.Equal(t, []any{marshaledOne}, args)
}

func TestExprToQuery_QuantAll(t *testing.T) {
	inner := gimpl.Expr{Expr: gimpl.CondExpr{Field: "id", Op: gimpl.CondOpGT, Val: 10}}
	e := gimpl.Expr{Expr: gimpl.QuantExpr{Field: "items", Op: gimpl.QuantOpAll, Expr: inner}}
	q, args := toSQL(t, e)
	marshaledTen, _ := json.Marshal(10)
	expected := "NOT EXISTS (SELECT 1 FROM unnest(items) AS q0(elem) WHERE NOT (q0.elem #> '{id}' > ?::jsonb))"
	require.Equal(t, expected, q)
	require.Equal(t, []any{marshaledTen}, args)
}

func TestExprToQuery_NestedQuantifiers(t *testing.T) {
	// ANY items where ALL sub.id eq 5
	deepest := gimpl.Expr{Expr: gimpl.CondExpr{Field: "id", Op: gimpl.CondOpEQ, Val: 5}}
	allSub := gimpl.Expr{Expr: gimpl.QuantExpr{Field: "sub", Op: gimpl.QuantOpAll, Expr: deepest}}
	anyItems := gimpl.Expr{Expr: gimpl.QuantExpr{Field: "items", Op: gimpl.QuantOpAny, Expr: allSub}}
	q, args := toSQL(t, anyItems)
	marshaledFive, _ := json.Marshal(5)
	// Validate structural pieces instead of full exact string to reduce brittleness
	require.Len(t, args, 1)
	require.Equal(t, marshaledFive, args[0])
	require.Contains(t, q, "EXISTS (SELECT 1 FROM unnest(items) AS q0(elem) WHERE ")
	require.Contains(t, q, "NOT EXISTS (SELECT 1 FROM jsonb_array_elements(CASE WHEN jsonb_typeof(q0.elem #> '{sub}') = 'array' THEN q0.elem #> '{sub}' ELSE '[]'::jsonb END) AS q1(elem) WHERE NOT (q1.elem #> '{id}' = ?::jsonb))")
}

func TestExprToQuery_InvalidOperator(t *testing.T) {
	e := gimpl.Expr{Expr: gimpl.CondExpr{Field: "status", Op: gimpl.CondOp("bogus"), Val: 1}}
	_, err := pgimpl.ExprToQuery(e)
	require.Error(t, err)
	require.ErrorIs(t, err, gimpl.ErrInvalidExpr)
}

func TestApplyExprToQuery_NoExpr(t *testing.T) {
	base := squirrel.Select("id").From("table")
	// filters.Expr nil triggers passthrough
	filters := gimpl.Expr{} // zero value; Expr field is nil
	out, err := pgimpl.ApplyExprToQuery(base, filters)
	require.NoError(t, err)
	sqlStr, args, err := out.ToSql()
	require.NoError(t, err)
	require.Equal(t, "SELECT id FROM table", sqlStr)
	require.Empty(t, args)
}

func TestApplyExprToQuery_WithExpr(t *testing.T) {
	base := squirrel.Select("id").From("table")
	filters := gimpl.Expr{Expr: gimpl.CondExpr{Field: "status", Op: gimpl.CondOpEQ, Val: "active"}}
	out, err := pgimpl.ApplyExprToQuery(base, filters)
	require.NoError(t, err)
	sqlStr, args, err := out.ToSql()
	require.NoError(t, err)
	require.Equal(t, "SELECT id FROM table WHERE status = ?", sqlStr)
	require.Equal(t, []any{"active"}, args)
}
