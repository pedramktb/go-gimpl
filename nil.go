package gimpl

import "github.com/pedramktb/go-typx"

var _ Entity = Nil[Entity]{}

type Nil[E Entity] struct {
	typx.Nil[E]
}

func (n Nil[E]) FilterPtr(field string) any {
	return n.Val.FilterPtr(field)
}

func (n Nil[E]) SortPtr(field string) any {
	return n.Val.SortPtr(field)
}
