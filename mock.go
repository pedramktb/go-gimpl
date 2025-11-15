package gimpl

import (
	"context"

	"github.com/stretchr/testify/mock"
)

// Compile time checks
var _ Entity = e{}
var _ UpdateEntity = u{}
var _ CreateEntity = c{}

type e struct{}

func (e) GetPtrs() []any       { return nil }
func (e) SortPtr(string) any   { return nil }
func (e) FilterPtr(string) any { return nil }

type c struct{}

func (c) CreatePtrs() []any { return nil }

type r struct{}

func (r) ReplacePtrs() []any { return nil }

type u struct{}

func (u) UpdatePtrs() []any { return nil }

var _ Get[e] = (&Mock[e, c, u, r]{}).Get
var _ Create[c] = (&Mock[e, c, u, r]{}).Create
var _ Replace[e] = (&Mock[e, c, u, r]{}).Replace
var _ Update[u] = (&Mock[e, c, u, r]{}).Update
var _ Delete[e] = (&Mock[e, c, u, r]{}).Delete

// A mock that mocks all generic Mock interfaces for Entity E and UpdateEntity U
// To be used in tests
type Mock[E Entity, C CreateEntity, U UpdateEntity, R ReplaceEntity] struct{ mock.Mock }

func (m *Mock[E, C, U, R]) Get(ctx context.Context, locateOpts []LocateOpt, paginateOpts ...PaginateOpt) (Paginated[E], error) {
	args := m.Called(ctx, locateOpts, paginateOpts)
	return args.Get(0).(Paginated[E]), args.Error(1)
}
func (m *Mock[E, C, U, R]) Create(ctx context.Context, items []C, txOpts ...TxOpt) error {
	args := m.Called(ctx, items, txOpts)
	return args.Error(0)
}
func (m *Mock[E, C, U, R]) Replace(ctx context.Context, items []E, txOpts ...TxOpt) error {
	args := m.Called(ctx, items, txOpts)
	return args.Error(0)
}
func (m *Mock[E, C, U, R]) Update(ctx context.Context, items []U, txOpts ...TxOpt) error {
	args := m.Called(ctx, items, txOpts)
	return args.Error(0)
}
func (m *Mock[E, C, U, R]) Delete(ctx context.Context, locateOpts []LocateOpt, txOpts ...TxOpt) error {
	args := m.Called(ctx, locateOpts, txOpts)
	return args.Error(0)
}
