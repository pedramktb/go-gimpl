package gimpl

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"
)

// LogOp for logical operators
type LogOp string

const (
	LogOpAnd LogOp = "and" // logical and
	LogOpOr  LogOp = "or"  // logical or
)

func (o LogOp) isValid() bool { return slices.Contains([]LogOp{LogOpAnd, LogOpOr}, o) }

// QuantOp is quantifier operators (first-class array quantifiers)
type QuantOp string

const (
	QuantOpAny QuantOp = "any" // there exists an element in array
	QuantOpAll QuantOp = "all" // for all elements in array
)

// CondOp for conditional operators
type CondOp string

const (
	CondOpRE  CondOp = "re"  // regular expression (string)
	CondOpNIN CondOp = "nin" // not in list
	CondOpIN  CondOp = "in"  // in list
	CondOpEQ  CondOp = "eq"  // equality
	CondOpNE  CondOp = "ne"  // not equal
	CondOpGT  CondOp = "gt"  // greater than
	CondOpLT  CondOp = "lt"  // less than
	CondOpGTE CondOp = "gte" // greater than or equal to
	CondOpLTE CondOp = "lte" // less than or equal to
)

func (o CondOp) isValid() bool {
	return slices.Contains([]CondOp{
		CondOpRE, CondOpIN, CondOpNIN, CondOpEQ, CondOpNE,
		CondOpGT, CondOpLT, CondOpGTE, CondOpLTE,
	}, o)
}

func (q QuantOp) isValid() bool { return slices.Contains([]QuantOp{QuantOpAny, QuantOpAll}, q) }

// Expr is the root expression wrapper
type Expr struct {
	Sample Entity // Only used for unmarshalling, must be set before unmarshalling
	Expr   any
}

// LogExpr is a logical expression combining two sub-expressions.
type LogExpr struct {
	Left  Expr
	Op    LogOp
	Right Expr
}

// QuantExpr is a quantifier expression that applies a nested expression relative to each element in an array field.
type QuantExpr struct {
	Field string
	Op    QuantOp
	Expr  Expr
}

// CondExpr is a Condition expression (field comparison).
type CondExpr struct {
	Field string
	Op    CondOp
	Val   any
}

// UnmarshalJSON unmarshal for top-level Expr. Delegates to decodeExpr with root entity context.
func (e *Expr) UnmarshalJSON(src []byte) error {
	if e.Sample == nil {
		return ErrInvalidExpr.Wrap(errors.New("Expr.Sample must be set before unmarshalling"))
	}
	expr, err := decodeExpr(src, e.Sample)
	if err != nil {
		return ErrInvalidExpr.Wrap(err)
	}
	e.Expr = expr
	return nil
}

// decodeExpr recursively decodes any expression variant using the provided root entity context (for field type inference).
func decodeExpr(src json.RawMessage, root any) (any, error) {
	var probe struct {
		Op string `json:"op"`
	}
	if err := json.Unmarshal(src, &probe); err != nil {
		return nil, err
	}
	if LogOp(probe.Op).isValid() {
		var tmp struct {
			Left  json.RawMessage `json:"left"`
			Op    LogOp           `json:"op"`
			Right json.RawMessage `json:"right"`
		}
		if err := json.Unmarshal(src, &tmp); err != nil {
			return nil, err
		}
		left, err := decodeExpr(tmp.Left, root)
		if err != nil {
			return nil, err
		}
		right, err := decodeExpr(tmp.Right, root)
		if err != nil {
			return nil, err
		}
		return LogExpr{Left: Expr{Expr: left}, Op: tmp.Op, Right: Expr{Expr: right}}, nil
	}
	if CondOp(probe.Op).isValid() {
		var tmp struct {
			Field string          `json:"field"`
			Op    CondOp          `json:"op"`
			Val   json.RawMessage `json:"val"`
		}
		if err := json.Unmarshal(src, &tmp); err != nil {
			return nil, err
		}
		val, err := resolveVal(root, tmp.Op, tmp.Field)
		if err != nil {
			return nil, fmt.Errorf("resolving condition field %s: %w", tmp.Field, err)
		}
		if err := json.Unmarshal(tmp.Val, val); err != nil {
			return nil, fmt.Errorf("condition value for field %s: %w", tmp.Field, err)
		}
		return CondExpr{Field: tmp.Field, Op: tmp.Op, Val: reflect.ValueOf(val).Elem().Interface()}, nil
	}
	if QuantOp(probe.Op).isValid() {
		var tmp struct {
			Field string          `json:"field"`
			Op    QuantOp         `json:"op"`
			Expr  json.RawMessage `json:"expr"`
		}
		if err := json.Unmarshal(src, &tmp); err != nil {
			return nil, err
		}
		elem, err := resolveArrayElem(root, tmp.Field)
		if err != nil {
			return nil, fmt.Errorf("quantifier field %s: %w", tmp.Field, err)
		}
		nested, err := decodeExpr(tmp.Expr, elem)
		if err != nil {
			return nil, err
		}
		return QuantExpr{Field: tmp.Field, Op: tmp.Op, Expr: Expr{Expr: nested}}, nil
	}
	return nil, fmt.Errorf("invalid operator %s", probe.Op)
}

func resolveVal(root any, op CondOp, field string) (any, error) {
	if field != "" {
		segments := strings.SplitSeq(field, ".")
		for seg := range segments {
			ent, ok := root.(Entity)
			if !ok {
				return nil, fmt.Errorf("field %s in %s is not an entity", seg, field)
			}
			root = ent.FilterPtr(seg)
			if root == nil {
				return nil, fmt.Errorf("field %s not found", field)
			}
		}
	}

	// For the IN and NIN operators, we need to unmarshal into *[]TYPE instead of *TYPE
	if op == CondOpIN || op == CondOpNIN {
		root = reflect.New(reflect.SliceOf(reflect.TypeOf(root).Elem())).Interface()
	}

	return root, nil
}

func resolveArrayElem(root any, field string) (any, error) {
	segments := strings.SplitSeq(field, ".")
	for seg := range segments {
		ent, ok := root.(Entity)
		if !ok {
			return nil, fmt.Errorf("field %s in %s is not an entity", seg, field)
		}
		root = ent.FilterPtr(seg)
		if root == nil {
			return nil, fmt.Errorf("field %s not found", field)
		}
	}

	t := reflect.TypeOf(root)
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	if t.Kind() != reflect.Slice && t.Kind() != reflect.Array {
		return nil, fmt.Errorf("field %s is not an array or slice", field)
	}

	return reflect.New(t.Elem()).Interface(), nil
}
