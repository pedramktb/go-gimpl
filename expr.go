package gimpl

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/pedramktb/go-tagerr"
)

var (
	ErrInvalidExpr = tagerr.ErrInvalidReq.Wrap(&tagerr.Err{
		Err: errors.New("invalid expression"),
		Tag: "invalid_expr",
	})
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

// QuantExpr is a quantifier expression that applies a nested expression relative to each element in an array path.
type QuantExpr struct {
	Path string
	Op   QuantOp
	Expr Expr
}

// CondExpr is a Condition expression (field comparison).
type CondExpr struct {
	Path string
	Op   CondOp
	Val  any
}

// UnmarshalJSON unmarshal for top-level Expr. Delegates to decodeExpr with root entity context.
func (e *Expr) UnmarshalJSON(src []byte) error {
	if e.Sample == nil {
		return tagerr.ErrInternal.Wrap(ErrInvalidExpr.Wrap(errors.New("Expr.Sample must be set before unmarshalling")))
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
			Path string          `json:"path"`
			Op   CondOp          `json:"op"`
			Val  json.RawMessage `json:"val"`
		}
		if err := json.Unmarshal(src, &tmp); err != nil {
			return nil, err
		}
		val, err := resolveVal(root, tmp.Op, tmp.Path)
		if err != nil {
			return nil, fmt.Errorf("resolving condition path %q: %w", tmp.Path, err)
		}
		if err := json.Unmarshal(tmp.Val, val); err != nil {
			return nil, fmt.Errorf("condition value for path %q: %w", tmp.Path, err)
		}
		return CondExpr{Path: tmp.Path, Op: tmp.Op, Val: reflect.ValueOf(val).Elem().Interface()}, nil
	}
	if QuantOp(probe.Op).isValid() {
		var tmp struct {
			Path string          `json:"path"`
			Op   QuantOp         `json:"op"`
			Expr json.RawMessage `json:"expr"`
		}
		if err := json.Unmarshal(src, &tmp); err != nil {
			return nil, err
		}
		elem, err := resolveArrayElem(root, tmp.Path)
		if err != nil {
			return nil, fmt.Errorf("quantifier path %q: %w", tmp.Path, err)
		}
		nested, err := decodeExpr(tmp.Expr, elem)
		if err != nil {
			return nil, err
		}
		return QuantExpr{Path: tmp.Path, Op: tmp.Op, Expr: Expr{Expr: nested}}, nil
	}
	return nil, fmt.Errorf("invalid operator %q", probe.Op)
}

func resolveVal(root any, op CondOp, path string) (any, error) {
	if path != "" {
		for field := range strings.SplitSeq(path, ".") {
			ent, ok := root.(Entity)
			if !ok {
				return nil, fmt.Errorf("field %q in path %q is not an entity", field, path)
			}
			root = ent.FilterPtr(field)
			if root == nil {
				return nil, fmt.Errorf("field %q in path %q not found", field, path)
			}
		}
	}

	// For the IN and NIN operators, we need to unmarshal into *[]TYPE instead of *TYPE
	if op == CondOpIN || op == CondOpNIN {
		root = reflect.New(reflect.SliceOf(reflect.TypeOf(root).Elem())).Interface()
	}

	return root, nil
}

func resolveArrayElem(root any, path string) (any, error) {
	last := path
	for field := range strings.SplitSeq(path, ".") {
		ent, ok := root.(Entity)
		if !ok {
			return nil, fmt.Errorf("field %q in path %q is not an entity", field, path)
		}
		root = ent.FilterPtr(field)
		if root == nil {
			return nil, fmt.Errorf("field %q in path %q not found", field, path)
		}
		last = field
	}

	t := reflect.TypeOf(root)
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	if t.Kind() != reflect.Slice && t.Kind() != reflect.Array {
		return nil, fmt.Errorf("field %q in path %q is not an array or slice", last, path)
	}

	return reflect.New(t.Elem()).Interface(), nil
}
