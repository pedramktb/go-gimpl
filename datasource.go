package gimpl

import (
	"context"
	"errors"

	"github.com/pedramktb/go-tagerr"
)

var (
	ErrDatastoreUnhandled = tagerr.ErrInternal.Wrap(&tagerr.Err{
		Err: errors.New("datastore unhandled error"),
		Tag: "unhandled_datastore_error",
	})
	ErrInUse = tagerr.ErrFailedPreCond.Wrap(&tagerr.Err{
		Err: errors.New("entity in use and is referenced by other entities"),
		Tag: "entity_in_use",
	})
)

// Finder is a datasource type that can be used for retrieval queries
type Finder[E Entity] interface {
	FindOne(ctx context.Context, filter Expr, txOpts ...TxOpt) (E, error)
	Find(ctx context.Context, filter Expr, paginateOpts []PaginateOpt, txOpts ...TxOpt) (Paginated[E], error)
}

// Creator is a datasource type that can be used for creation queries
type Creator[C CreateEntity] interface {
	Create(ctx context.Context, items []C, txOpts ...TxOpt) error
}

// Updater is a datasource type that can be used for update queries
type Updater[U UpdateEntity] interface {
	Update(ctx context.Context, items []U, txOpts ...TxOpt) error
}

// Saver is a datasource type that can be used for create-or-update queries
type Saver[S SaveEntity] interface {
	Save(ctx context.Context, items []S, txOpts ...TxOpt) error
}

// Remover is a datasource type that can be used for deletion queries
type Remover[E Entity] interface {
	RemoveOne(ctx context.Context, filter Expr, txOpts ...TxOpt) error
	Remove(ctx context.Context, filter Expr, txOpts ...TxOpt) error
}
