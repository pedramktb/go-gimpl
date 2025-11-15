package gimpl

import "context"

// Get is a datasource function type that can be used for retrieval queries
type Get[E Entity] func(ctx context.Context, locateOpts []LocateOpt, paginateOpts ...PaginateOpt) (Paginated[E], error)

// Create is a datasource function type that can be used for creation queries
type Create[C CreateEntity] func(ctx context.Context, items []C, txOpts ...TxOpt) error

// Replace is a datasource function type that can be used for replace queries
type Replace[E Entity] func(ctx context.Context, items []E, txOpts ...TxOpt) error

// Update is a datasource function type that can be used for update queries
type Update[U UpdateEntity] func(ctx context.Context, items []U, txOpts ...TxOpt) error

// Delete is a datasource function type that can be used for deletion queries
type Delete[E Entity] func(ctx context.Context, locateOpts []LocateOpt, txOpts ...TxOpt) error
