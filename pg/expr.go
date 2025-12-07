package pgimpl

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/Masterminds/squirrel"
	"github.com/pedramktb/go-gimpl"
	"github.com/pedramktb/go-tagerr"
)

// FromExpr converts an gimpl.Expr logical/quantified expression tree into a squirrel.Sqlizer fragment
// and a slice of args that is compatible with postgres.
//
// Conventions implemented:
//   - A CondExpr whose Field has no dots (e.g. "status") and is NOT nested beneath a quantifier
//     is treated as a direct column comparison: status = ?
//   - A CondExpr whose Field contains dots OR that appears beneath a quantifier context is treated
//     as a JSONB path comparison. The first segment before the first dot (or the quantifier's array
//     element alias) is the JSONB root. Remaining segments become a jsonb path using the '#>' operator.
//   - Quantifiers (ANY / ALL) over a jsonb array path are translated into EXISTS / NOT EXISTS with
//     jsonb_array_elements(). Nested quantifiers are supported (aliases are generated q0, q1, ...).
//   - IN / NIN for JSON values rely on the jsonb containment operator '<@'
//     Column IN / NIN use '= ANY (?)' / '!= ALL (?)' patterns.
func FromExpr(e gimpl.Expr) (squirrel.Sqlizer, error) {
	if e.Expr == nil {
		return nil, nil
	}
	if e.Sample == nil {
		return nil, tagerr.ErrInternal.Wrap(gimpl.ErrInvalidExpr.Wrap(errors.New("Expr.Sample must be set")))
	}
	sample, ok := e.Sample.(Entity)
	if !ok {
		return nil, tagerr.ErrInternal.Wrap(gimpl.ErrInvalidExpr.Wrap(errors.New("Expr.Sample must implement pgimpl.Entity interface")))
	}
	query, args, err := (&builder{}).build(e.Expr, buildContext{sample: sample, sqlMode: true, sqlFirstLevel: true})
	if err != nil {
		return nil, err
	}

	return squirrel.Expr(query, args...), nil
}

type builder struct {
	quantDepth int
	args       []any
}

type buildContext struct {
	sample        any
	sqlMode       bool
	sqlFirstLevel bool
	fieldPath     []string
	quantAlias    string
}

// build recursively builds SQL and returns predicate + args slice (shared slice on builder).
func (b *builder) build(expr any, ctx buildContext) (string, []any, error) {
	switch v := expr.(type) {
	case gimpl.LogExpr:
		left, _, err := b.build(v.Left.Expr, ctx)
		if err != nil {
			return "", nil, err
		}
		right, _, err := b.build(v.Right.Expr, ctx)
		if err != nil {
			return "", nil, err
		}
		op := strings.ToUpper(string(v.Op))
		return "(" + left + " " + op + " " + right + ")", b.args, nil
	case gimpl.CondExpr:
		q, arg, err := b.buildCond(v, ctx)
		if err != nil {
			return "", nil, err
		}
		if arg != nil {
			b.args = append(b.args, arg)
		}
		return q, b.args, nil
	case gimpl.QuantExpr:
		q, err := b.buildQuant(v, ctx)
		if err != nil {
			return "", nil, err
		}
		return q, b.args, nil
	default:
		return "", nil, tagerr.ErrInternal.Wrap(gimpl.ErrInvalidExpr.Wrap(fmt.Errorf("unknown expr type %T", expr)))
	}
}

// buildQuant decides between SQL array quantification and JSON array quantification based on accumulated path context.
func (b *builder) buildQuant(qe gimpl.QuantExpr, ctx buildContext) (string, error) {
	sample, subPath, err := buildQuantPath(ctx.sample, qe.Field)
	if err != nil {
		return "", err
	}
	fullPath := append(ctx.fieldPath, subPath...)
	if ctx.sqlMode {
		if ctx.sqlFirstLevel {
			if len(fullPath) == 0 {
				return "", tagerr.ErrInternal.Wrap(gimpl.ErrInvalidExpr.Wrap(errors.New("empty sql array field path")))
			}
			if len(fullPath) > 1 {
				return b.buildJSONQuant(qe, ctx, sample, fullPath)
			}
			return b.buildColumnQuant(qe, ctx, sample, fullPath[0])
		}

		if len(subPath) == 0 {
			return b.buildColumnQuant(qe, ctx, sample, fullPath[0])
		}
		return b.buildJSONQuant(qe, ctx, sample, fullPath)
	}

	return b.buildJSONQuant(qe, ctx, sample, fullPath)
}

func buildQuantPath(sample any, field string) (any, []string, error) {
	if field == "" {
		return sample, nil, nil
	}
	fields := strings.Split(field, ".")
	path := make([]string, 0, len(fields))
	for i := range fields {
		entitySample, ok := sample.(Entity)
		if !ok {
			return nil, nil, tagerr.ErrInternal.Wrap(gimpl.ErrInvalidExpr.Wrap(fmt.Errorf("parent of field %q is not an entity", fields[i])))
		}
		sample = entitySample.FilterPtr(fields[i])
		if sample == nil {
			return nil, nil, tagerr.ErrInternal.Wrap(gimpl.ErrInvalidExpr.Wrap(fmt.Errorf("field %q in path %q was not found or is not filterable", fields[i], field)))
		}
		part := entitySample.Column(fields[i])
		if part == "" {
			return nil, nil, tagerr.ErrInternal.Wrap(gimpl.ErrInvalidExpr.Wrap(fmt.Errorf("field %q in path %q has no associated column", fields[i], field)))
		}
		path = append(path, part)
	}
	t := reflect.TypeOf(sample)
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	if t.Kind() != reflect.Slice && t.Kind() != reflect.Array {
		return nil, nil, fmt.Errorf("field %q in path %q is not an array or slice", fields[len(fields)-1], field)
	}

	return reflect.New(t.Elem()).Interface(), path, nil

}

// buildColumnQuant emits EXISTS / NOT EXISTS checks that iterate native Postgres arrays with unnest().
func (b *builder) buildColumnQuant(qe gimpl.QuantExpr, ctx buildContext, sample any, fullPath string) (string, error) {
	alias := fmt.Sprintf("q%d", b.quantDepth)
	b.quantDepth++

	var arrayExpr string
	if ctx.quantAlias != "" {
		arrayExpr = ctx.quantAlias
	} else {
		arrayExpr = fullPath
	}
	if arrayExpr == "" {
		return "", tagerr.ErrInternal.Wrap(gimpl.ErrInvalidExpr.Wrap(errors.New("empty sql array field path")))
	}

	childCtx := buildContext{
		sample:        sample,
		sqlMode:       true,
		sqlFirstLevel: false,
		fieldPath:     []string{fullPath},
		quantAlias:    alias + ".elem",
	}

	nestedSQL, _, err := b.build(qe.Expr.Expr, childCtx)
	if err != nil {
		return "", err
	}

	switch qe.Op {
	case gimpl.QuantOpAny:
		return fmt.Sprintf("EXISTS (SELECT 1 FROM unnest(%s) AS %s(elem) WHERE %s)", arrayExpr, alias, nestedSQL), nil
	case gimpl.QuantOpAll:
		return fmt.Sprintf("NOT EXISTS (SELECT 1 FROM unnest(%s) AS %s(elem) WHERE NOT (%s))", arrayExpr, alias, nestedSQL), nil
	default:
		return "", tagerr.ErrInternal.Wrap(gimpl.ErrInvalidExpr.Wrap(fmt.Errorf("unknown quantifier %s", qe.Op)))
	}
}

func (b *builder) buildJSONQuant(qe gimpl.QuantExpr, ctx buildContext, sample any, fullPath []string) (string, error) {
	alias := fmt.Sprintf("q%d", b.quantDepth)
	b.quantDepth++

	arrayExpr, err := jsonExpr(ctx, fullPath)
	if err != nil {
		return "", err
	}

	childCtx := buildContext{
		sample:        sample,
		sqlMode:       false,
		sqlFirstLevel: false,
		fieldPath:     fullPath,
		quantAlias:    alias + ".elem",
	}

	nestedSQL, _, err := b.build(qe.Expr.Expr, childCtx)
	if err != nil {
		return "", err
	}

	safeArrayExpr := fmt.Sprintf("CASE WHEN jsonb_typeof(%s) = 'array' THEN %s ELSE '[]'::jsonb END", arrayExpr, arrayExpr)
	switch qe.Op {
	case gimpl.QuantOpAny:
		return fmt.Sprintf("EXISTS (SELECT 1 FROM jsonb_array_elements(%s) AS %s(elem) WHERE %s)", safeArrayExpr, alias, nestedSQL), nil
	case gimpl.QuantOpAll:
		return fmt.Sprintf("NOT EXISTS (SELECT 1 FROM jsonb_array_elements(%s) AS %s(elem) WHERE NOT (%s))", safeArrayExpr, alias, nestedSQL), nil
	default:
		return "", tagerr.ErrInternal.Wrap(gimpl.ErrInvalidExpr.Wrap(fmt.Errorf("unknown quantifier %s", qe.Op)))
	}
}

// buildCond builds a condition expression.
func (b *builder) buildCond(c gimpl.CondExpr, ctx buildContext) (string, any, error) {
	subPath, err := buildCondPath(ctx.sample, c.Field)
	if err != nil {
		return "", nil, err
	}
	fullPath := append(ctx.fieldPath, subPath...)
	if ctx.sqlMode {
		if ctx.quantAlias == "" {
			if len(fullPath) == 0 {
				return "", nil, tagerr.ErrInternal.Wrap(gimpl.ErrInvalidExpr.Wrap(errors.New("empty condition field path")))
			}
			if len(fullPath) > 1 {
				path, err := jsonExpr(ctx, fullPath)
				if err != nil {
					return "", nil, err
				}
				return buildJSONCond(path, c.Op, c.Val)
			}
			return buildColumnCond(fullPath[0], c.Op, c.Val)
			// combinePath joins the accumulated parent fieldPath with the current relative field name.
		}

		if !slices.Equal(fullPath, ctx.fieldPath) {
			path, err := jsonExpr(ctx, fullPath)
			if err != nil {
				return "", nil, err
			}
			return buildJSONCond(path, c.Op, c.Val)
		}

		return buildColumnCond(ctx.quantAlias, c.Op, c.Val)
	}

	path, err := jsonExpr(ctx, fullPath)
	if err != nil {
		return "", nil, err
	}
	return buildJSONCond(path, c.Op, c.Val)
}

// jsonExpr resolves the jsonb expression to access targetPath relative to the current context.
// It keeps track of whether we are operating on a root column or a nested array element alias.
func jsonExpr(ctx buildContext, targetPath []string) (string, error) {
	if ctx.quantAlias == "" {
		// Root lookup: first segment references the jsonb column, remainder forms the path.
		if len(targetPath) == 0 {
			return "", tagerr.ErrInternal.Wrap(gimpl.ErrInvalidExpr.Wrap(errors.New("empty json expression path")))
		}
		if len(targetPath) == 1 {
			return targetPath[0], nil
		}
		return targetPath[0] + " #> '{" + strings.Join(targetPath[1:], ",") + "}'", nil
	}

	if len(targetPath) < len(ctx.fieldPath) {
		return "", tagerr.ErrInternal.Wrap(gimpl.ErrInvalidExpr.Wrap(errors.New("invalid json expression path")))
	}
	// When inside a quantifier, strip the already traversed prefix and build a path relative to the element alias.
	relSegs := targetPath[len(ctx.fieldPath):]
	if len(relSegs) == 0 {
		return ctx.quantAlias, nil
	}
	return ctx.quantAlias + " #> '{" + strings.Join(relSegs, ",") + "}'", nil
}

func buildCondPath(sample any, field string) ([]string, error) {
	if field == "" {
		return nil, nil
	}
	fields := strings.Split(field, ".")
	path := make([]string, 0, len(fields))
	for i := range fields {
		entitySample, ok := sample.(Entity)
		if !ok {
			return nil, tagerr.ErrInternal.Wrap(gimpl.ErrInvalidExpr.Wrap(fmt.Errorf("parent of field %q is not an entity", fields[i])))
		}
		sample = entitySample.FilterPtr(fields[i])
		part := entitySample.Column(fields[i])
		if part == "" {
			return nil, tagerr.ErrInternal.Wrap(gimpl.ErrInvalidExpr.Wrap(fmt.Errorf("field %q in path %q has no associated column", fields[i], field)))
		}
		path = append(path, part)
	}
	return path, nil
}

// buildJSONCond Compares JSON values using jsonb operators for the condition expression.
func buildJSONCond(path string, op gimpl.CondOp, val any) (query string, arg any, err error) {
	if op == gimpl.CondOpRE {
		return "(" + path + ") #>> '{}' ~ ?", val, nil
	}
	jsonText, err := json.Marshal(val)
	if err != nil {
		return "", nil, err
	}
	switch op {
	case gimpl.CondOpEQ:
		return path + " = ?::jsonb", jsonText, nil
	case gimpl.CondOpNE:
		return path + " <> ?::jsonb", jsonText, nil
	case gimpl.CondOpGT:
		return path + " > ?::jsonb", jsonText, nil
	case gimpl.CondOpLT:
		return path + " < ?::jsonb", jsonText, nil
	case gimpl.CondOpGTE:
		return path + " >= ?::jsonb", jsonText, nil
	case gimpl.CondOpLTE:
		return path + " <= ?::jsonb", jsonText, nil
	case gimpl.CondOpIN:
		return path + " <@ ?::jsonb", jsonText, nil
	case gimpl.CondOpNIN:
		return "NOT (" + path + " <@ ?::jsonb)", jsonText, nil
	default:
		return "", nil, tagerr.ErrInternal.Wrap(gimpl.ErrInvalidExpr.Wrap(errors.New("unknown condition operator")))
	}
}

// buildColumnCond compares non-JSON column values using standard SQL operators for condition expressions.
func buildColumnCond(col string, op gimpl.CondOp, val any) (string, any, error) {
	val = normalizeNil(val)
	switch op {
	case gimpl.CondOpRE:
		return col + " ~ ?", val, nil
	case gimpl.CondOpEQ:
		if val == nil {
			return col + " IS NULL", nil, nil
		}
		return col + " = ?", val, nil
	case gimpl.CondOpNE:
		if val == nil {
			return col + " IS NOT NULL", nil, nil
		}
		return col + " <> ?", val, nil
	case gimpl.CondOpGT:
		return col + " > ?", val, nil
	case gimpl.CondOpLT:
		return col + " < ?", val, nil
	case gimpl.CondOpGTE:
		return col + " >= ?", val, nil
	case gimpl.CondOpLTE:
		return col + " <= ?", val, nil
	case gimpl.CondOpIN:
		return col + " = ANY (?)", val, nil
	case gimpl.CondOpNIN:
		return col + " <> ALL (?)", val, nil
	default:
		return "", nil, tagerr.ErrInternal.Wrap(gimpl.ErrInvalidExpr.Wrap(errors.New("unknown condition operator")))
	}
}
