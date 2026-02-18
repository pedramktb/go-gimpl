package gimpl

import "github.com/pedramktb/go-typx"

var _ Entity = Dyn{}

type Dyn struct {
	typx.Dyn
}

func (d Dyn) FilterPtr(path string) any {
	return &Dyn{}
}

func (d Dyn) SortPtr(path string) any {
	return nil
}
