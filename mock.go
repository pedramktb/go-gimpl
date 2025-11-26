package gimpl

import (
	"context"

	"github.com/stretchr/testify/mock"
)

var _ Finder[Entity] = &MockFinder[Entity]{}

type MockFinder[E Entity] struct{ mock.Mock }

func (m *MockFinder[E]) FindOne(ctx context.Context, filter Expr, txOpts ...TxOpt) (E, error) {
	args := m.Called(ctx, filter, txOpts)
	return args.Get(0).(E), args.Error(1)
}
func (m *MockFinder[E]) Find(ctx context.Context, filter Expr, paginateOpts []PaginateOpt, txOpts ...TxOpt) (Paginated[E], error) {
	args := m.Called(ctx, filter, paginateOpts, txOpts)
	return args.Get(0).(Paginated[E]), args.Error(1)
}

var _ Creator[CreateEntity] = &MockCreator[CreateEntity]{}

type MockCreator[C CreateEntity] struct{ mock.Mock }

func (m *MockCreator[C]) Create(ctx context.Context, items []C, txOpts ...TxOpt) error {
	args := m.Called(ctx, items, txOpts)
	return args.Error(0)
}

var _ Updater[UpdateEntity] = &MockUpdater[UpdateEntity]{}

type MockUpdater[U UpdateEntity] struct{ mock.Mock }

func (m *MockUpdater[U]) Update(ctx context.Context, items []U, txOpts ...TxOpt) error {
	args := m.Called(ctx, items, txOpts)
	return args.Error(0)
}

var _ Saver[SaveEntity] = &MockSaver[SaveEntity]{}

type MockSaver[S SaveEntity] struct{ mock.Mock }

func (m *MockSaver[S]) Save(ctx context.Context, items []S, txOpts ...TxOpt) error {
	args := m.Called(ctx, items, txOpts)
	return args.Error(0)
}

var _ Remover[Entity] = &MockRemover[Entity]{}

type MockRemover[E Entity] struct{ mock.Mock }

func (m *MockRemover[E]) RemoveOne(ctx context.Context, filter Expr, txOpts ...TxOpt) error {
	args := m.Called(ctx, filter, txOpts)
	return args.Error(0)
}
func (m *MockRemover[E]) Remove(ctx context.Context, filter Expr, txOpts ...TxOpt) error {
	args := m.Called(ctx, filter, txOpts)
	return args.Error(0)
}
