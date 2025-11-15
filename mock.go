package gimpl

import (
	"context"

	"github.com/stretchr/testify/mock"
)

// Compile time checks
type e struct{}

func (e) Sortable(string) bool   { return false }
func (e) Filterable(string) bool { return false }
func (e) Pointer(string) any     { return nil }

type u struct{}

func (u) Pointer(string) any { return nil }

var _ Create[e] = (&Mock[e, u]{}).Create
var _ Get[e] = (&Mock[e, u]{}).Get
var _ Update[u] = (&Mock[e, u]{}).Update
var _ Delete[e] = (&Mock[e, u]{}).Delete

// A mock that mocks all generic Mock interfaces for Entity E and UpdateEntity U
// To be used in tests
type Mock[E Entity, U UpdateEntity] struct{ mock.Mock }

func (m *Mock[E, U]) Create(ctx context.Context, items []E, txOpts ...TxOpt) error {
	args := m.Called(ctx, items, txOpts)
	return args.Error(0)
}
func (m *Mock[E, U]) Replace(ctx context.Context, items []E, txOpts ...TxOpt) error {
	args := m.Called(ctx, items, txOpts)
	return args.Error(0)
}
func (m *Mock[E, U]) Get(ctx context.Context, locateOpts []LocateOpt, paginateOpts ...PaginateOpt) (Paginated[E], error) {
	args := m.Called(ctx, locateOpts, paginateOpts)
	return args.Get(0).(Paginated[E]), args.Error(1)
}
func (m *Mock[E, U]) Update(ctx context.Context, items []U, txOpts ...TxOpt) error {
	args := m.Called(ctx, items, txOpts)
	return args.Error(0)
}
func (m *Mock[E, U]) Delete(ctx context.Context, locateOpts []LocateOpt, txOpts ...TxOpt) error {
	args := m.Called(ctx, locateOpts, txOpts)
	return args.Error(0)
}
