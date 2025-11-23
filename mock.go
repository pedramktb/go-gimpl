package gimpl

import (
	"context"

	"github.com/stretchr/testify/mock"
)

// A mock that mocks all generic Mock interfaces for Entity E, CreateEntity C, UpdateEntity U, and ReplaceEntity R
// To be used in tests
type Mock[E Entity, C CreateEntity, U UpdateEntity, CU CreateOrUpdateEntity] struct{ mock.Mock }

func (m *Mock[E, C, U, CU]) Get(ctx context.Context, locateOpts []LocateOpt, paginateOpts []PaginateOpt, txOpts ...TxOpt) (Paginated[E], error) {
	args := m.Called(ctx, locateOpts, paginateOpts)
	return args.Get(0).(Paginated[E]), args.Error(1)
}
func (m *Mock[E, C, U, CU]) Create(ctx context.Context, items []C, txOpts ...TxOpt) error {
	args := m.Called(ctx, items, txOpts)
	return args.Error(0)
}
func (m *Mock[E, C, U, CU]) Update(ctx context.Context, items []U, txOpts ...TxOpt) error {
	args := m.Called(ctx, items, txOpts)
	return args.Error(0)
}
func (m *Mock[E, C, U, CU]) CreateOrUpdate(ctx context.Context, items []CU, txOpts ...TxOpt) error {
	args := m.Called(ctx, items, txOpts)
	return args.Error(0)
}
func (m *Mock[E, C, U, CU]) Delete(ctx context.Context, locateOpts []LocateOpt, txOpts ...TxOpt) error {
	args := m.Called(ctx, locateOpts, txOpts)
	return args.Error(0)
}

type e struct{}

func (e) SortPtr(string) any   { return nil }
func (e) FilterPtr(string) any { return nil }

type c struct{}

func (c) CreatePtrs() []any { return nil }

type r struct{}

func (r) ReplacePtrs() []any { return nil }

type u struct{}

func (u) UpdatePtrs() []any { return nil }

// Ensure Mock implements all required interfaces
var _ Get[e] = (&Mock[e, c, u, r]{}).Get
var _ Create[c] = (&Mock[e, c, u, r]{}).Create
var _ Update[u] = (&Mock[e, c, u, r]{}).Update
var _ CreateOrUpdate[r] = (&Mock[e, c, u, r]{}).CreateOrUpdate
var _ Delete[e] = (&Mock[e, c, u, r]{}).Delete
