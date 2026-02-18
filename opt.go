package gimpl

import "github.com/pedramktb/go-typx"

var _ Entity = Opt[Entity]{}

type Opt[E Entity] struct {
	typx.Opt[E]
}

func (o Opt[E]) FilterPtr(path string) any {
	return o.Val.FilterPtr(path)
}

func (o Opt[E]) SortPtr(path string) any {
	return o.Val.SortPtr(path)
}
