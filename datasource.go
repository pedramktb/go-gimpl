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

// Get is a datasource function type that can be used for retrieval queries
type Get[E Entity] func(ctx context.Context, locateOpts []LocateOpt, paginateOpts []PaginateOpt, txOpts ...TxOpt) (Paginated[E], error)

// Create is a datasource function type that can be used for creation queries
type Create[C CreateEntity] func(ctx context.Context, items []C, txOpts ...TxOpt) error

// Update is a datasource function type that can be used for update queries
type Update[U UpdateEntity] func(ctx context.Context, items []U, txOpts ...TxOpt) error

// CreateOrUpdate is a datasource function type that can be used for create‐or‐update queries
type CreateOrUpdate[CU CreateOrUpdateEntity] func(ctx context.Context, items []CU, txOpts ...TxOpt) error

// Delete is a datasource function type that can be used for deletion queries
type Delete[E Entity] func(ctx context.Context, locateOpts []LocateOpt, txOpts ...TxOpt) error
