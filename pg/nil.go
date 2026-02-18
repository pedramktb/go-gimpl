package pgimpl

import "github.com/pedramktb/go-gimpl"

var _ Entity = Nil[Entity]{}

type Nil[E Entity] struct {
	gimpl.Nil[E]
}

func (o Nil[E]) NewWithPgColumnPtrs() (any, []any) {
	return &Nil[E]{}, nil
}

func (o Nil[E]) PgPath(path string) []string {
	return o.Val.PgPath(path)
}

func (o Nil[E]) PgColumns() []string {
	return nil
}
