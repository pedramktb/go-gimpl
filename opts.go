package gimpl

type locateOpts struct {
	filters Expr
	search  string
}

type LocateOpt func(*locateOpts)

func WithFilter(filters Expr) func(*locateOpts) {
	return func(o *locateOpts) {
		o.filters = filters
	}
}

func (o *locateOpts) Filter() Expr {
	return o.filters
}

func WithSearch(search string) func(*locateOpts) {
	return func(o *locateOpts) {
		o.search = search
	}
}

func (o *locateOpts) Search() string {
	return o.search
}

type paginateOpts struct {
	limit PaginationLimit
	sorts Sorts
}

type PaginateOpt func(*paginateOpts)

func WithLimit(limit PaginationLimit) func(*paginateOpts) {
	return func(o *paginateOpts) {
		o.limit = limit
	}
}

func (o *paginateOpts) Limit() PaginationLimit {
	return o.limit
}

func WithSorts[E Entity](sorts Sorts) func(*paginateOpts) {
	return func(o *paginateOpts) {
		o.sorts = sorts
	}
}

func (o *paginateOpts) GetSorts() Sorts {
	return o.sorts
}

type txOpts struct {
	txOpts []any
}

type TxOpt func(*txOpts)

func WithTxOpts(opts ...any) func(*txOpts) {
	return func(o *txOpts) {
		o.txOpts = opts
	}
}

func (o *txOpts) TxOpts() []any {
	return o.txOpts
}
