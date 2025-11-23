package gimpl

import (
	"context"

	"github.com/stretchr/testify/mock"
)

var _ Get[Entity] = (&MockGetter[Entity]{}).Get

type MockGetter[E Entity] struct{ mock.Mock }

func (m *MockGetter[E]) Get(ctx context.Context, locateOpts []LocateOpt, paginateOpts []PaginateOpt, txOpts ...TxOpt) (Paginated[E], error) {
	args := m.Called(ctx, locateOpts, paginateOpts)
	return args.Get(0).(Paginated[E]), args.Error(1)
}

var _ Create[CreateEntity] = (&MockCreator[CreateEntity]{}).Create

type MockCreator[C CreateEntity] struct{ mock.Mock }

func (m *MockCreator[C]) Create(ctx context.Context, items []C, txOpts ...TxOpt) error {
	args := m.Called(ctx, items, txOpts)
	return args.Error(0)
}

var _ Update[UpdateEntity] = (&MockUpdater[UpdateEntity]{}).Update

type MockUpdater[U UpdateEntity] struct{ mock.Mock }

func (m *MockUpdater[U]) Update(ctx context.Context, items []U, txOpts ...TxOpt) error {
	args := m.Called(ctx, items, txOpts)
	return args.Error(0)
}

var _ CreateOrUpdate[CreateOrUpdateEntity] = (&MockCreateOrUpdater[CreateOrUpdateEntity]{}).CreateOrUpdate

type MockCreateOrUpdater[CU CreateOrUpdateEntity] struct{ mock.Mock }

func (m *MockCreateOrUpdater[CU]) CreateOrUpdate(ctx context.Context, items []CU, txOpts ...TxOpt) error {
	args := m.Called(ctx, items, txOpts)
	return args.Error(0)
}

var _ Delete[Entity] = (&MockDeleter[Entity]{}).Delete

type MockDeleter[E Entity] struct{ mock.Mock }

func (m *MockDeleter[E]) Delete(ctx context.Context, locateOpts []LocateOpt, txOpts ...TxOpt) error {
	args := m.Called(ctx, locateOpts, txOpts)
	return args.Error(0)
}
