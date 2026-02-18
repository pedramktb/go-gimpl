package pgimpl

import (
	"strings"

	"github.com/pedramktb/go-gimpl"
)

var _ Entity = Dyn{}

type Dyn struct {
	gimpl.Dyn
}

func (d Dyn) NewWithPgColumnPtrs() (any, []any) {
	return &Dyn{}, nil
}

func (d Dyn) PgPath(path string) []string {
	if path == "" {
		return nil
	}
	return strings.Split(path, ".")
}

func (d Dyn) PgColumns() []string {
	return nil
}
