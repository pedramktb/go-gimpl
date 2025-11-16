package gimpl

import "context"

// Get is a datasource function type that can be used for retrieval queries
type Get[E Entity] func(ctx context.Context, locateOpts []LocateOpt, paginateOpts []PaginateOpt, txOpts ...TxOpt) (Paginated[E], error)

// Create is a datasource function type that can be used for creation queries
type Create[C CreateEntity] func(ctx context.Context, items []C, txOpts ...TxOpt) error

// Replace is a datasource function type that can be used for replacement queries
type Replace[R ReplaceEntity] func(ctx context.Context, items []R, txOpts ...TxOpt) error

// CreateOrReplace is a datasource function type that can be used for create‐or‐replace queries
type CreateOrReplace[CR CreateOrReplaceEntity] func(ctx context.Context, items []CR, txOpts ...TxOpt) error

// Update is a datasource function type that can be used for update queries
type Update[U UpdateEntity] func(ctx context.Context, item U, txOpts ...TxOpt) error

// Delete is a datasource function type that can be used for deletion queries
type Delete[E Entity] func(ctx context.Context, locateOpts []LocateOpt, txOpts ...TxOpt) error
