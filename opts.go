package gimpl

type locateOpts struct {
	Filter Expr
	Search string
}

func LocateOpts(opts ...LocateOpt) *locateOpts {
	o := &locateOpts{}
	for _, opt := range opts {
		opt(o)
	}
	return o
}

type LocateOpt func(*locateOpts)

func WithFilter(filter Expr) func(*locateOpts) {
	return func(o *locateOpts) {
		o.Filter = filter
	}
}

func WithSearch(search string) func(*locateOpts) {
	return func(o *locateOpts) {
		o.Search = search
	}
}

type paginateOpts struct {
	Limit uint64
	Sorts Sorts
}

func PaginateOpts(opts ...PaginateOpt) *paginateOpts {
	o := &paginateOpts{}
	for _, opt := range opts {
		opt(o)
	}
	return o
}

type PaginateOpt func(*paginateOpts)

func WithLimit(limit uint64) func(*paginateOpts) {
	return func(o *paginateOpts) {
		o.Limit = limit
	}
}

func WithSorts[E Entity](sorts Sorts) func(*paginateOpts) {
	return func(o *paginateOpts) {
		o.Sorts = sorts
	}
}

type txOpts struct {
	TxOpts []any
}

func TxOpts(opts ...TxOpt) *txOpts {
	o := &txOpts{}
	for _, opt := range opts {
		opt(o)
	}
	return o
}

type TxOpt func(*txOpts)

func WithTxOpts(opts ...any) func(*txOpts) {
	return func(o *txOpts) {
		o.TxOpts = opts
	}
}
