package pgimpl

import "github.com/pedramktb/go-gimpl"

var _ Entity = Opt[Entity]{}

type Opt[E Entity] struct {
	gimpl.Opt[E]
}

func (o Opt[E]) NewWithPgColumnPtrs() (any, []any) {
	return &Opt[E]{}, nil
}

func (o Opt[E]) PgPath(path string) []string {
	return o.Val.PgPath(path)
}

func (o Opt[E]) PgColumns() []string {
	return nil
}
