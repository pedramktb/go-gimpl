package pgimpl

import (
	"context"
	"database/sql"

	"github.com/Masterminds/squirrel"
	"github.com/pedramktb/go-gimpl"
)

type getter[E Entity] struct {
	db    *sql.DB
	table string
}

// Getter returns a gimpl.Get function for the specified entity type and table name
// E must be a pointer struct that implements the Entity interface
func Getter[E Entity](db *sql.DB, table string) gimpl.Get[E] {
	return (&getter[E]{db, table}).Get
}

func (g *getter[E]) Get(ctx context.Context, locateOpts []gimpl.LocateOpt, paginateOpts []gimpl.PaginateOpt, txOpts ...gimpl.TxOpt) (_ gimpl.Paginated[E], err error) {
	locOpts := gimpl.LocateOpts(locateOpts...)
	pagOpts := gimpl.PaginateOpts(paginateOpts...)
	tOpts, err := TxOpts(ctx, g.db, txOpts)
	if err != nil {
		return gimpl.Paginated[E]{}, err
	}
	if tOpts.AutoFinalize {
		defer func() { err = tOpts.Tx.Finalize(err) }()
	}

	builder := squirrel.Select((*new(E)).GetColumns()...).PlaceholderFormat(squirrel.Dollar).From(g.table)
	filter, err := FromExpr(locOpts.Filter)
	if err != nil {
		return gimpl.Paginated[E]{}, gimpl.ErrDatastoreUnhandled.Wrap(err).WithStack()
	}
	builder = builder.Where(filter)

	// Create a count query based on the base query
	countQuery, countArgs, err := squirrel.Select("COUNT(*)").PlaceholderFormat(squirrel.Dollar).FromSelect(builder, "addr_count").ToSql()
	if err != nil {
		return gimpl.Paginated[E]{}, gimpl.ErrDatastoreUnhandled.Wrap(err).WithStack()
	}
	var total uint64
	if err := tOpts.Tx.QueryRowContext(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return gimpl.Paginated[E]{}, gimpl.ErrDatastoreUnhandled.Wrap(err).WithStack()
	}

	// Apply sorts and pagination
	var prevCursorBuilder *squirrel.SelectBuilder
	builder, prevCursorBuilder = FromSorts(pagOpts.Sorts, builder)
	query, args, err := builder.Limit(pagOpts.Limit + 1).ToSql()
	if err != nil {
		return gimpl.Paginated[E]{}, gimpl.ErrDatastoreUnhandled.Wrap(err).WithStack()
	}
	results, err := g.get(ctx, tOpts, query, args...)
	if err != nil {
		return gimpl.Paginated[E]{}, gimpl.ErrDatastoreUnhandled.Wrap(err).WithStack()
	}

	var prevResults []E
	if prevCursorBuilder != nil {
		prevCursorQuery, prevCursorArgs, err := prevCursorBuilder.Limit(pagOpts.Limit + 1).ToSql()
		if err != nil {
			return gimpl.Paginated[E]{}, gimpl.ErrDatastoreUnhandled.Wrap(err).WithStack()
		}
		prevResults, err = g.get(ctx, tOpts, prevCursorQuery, prevCursorArgs...)
		if err != nil {
			return gimpl.Paginated[E]{}, gimpl.ErrDatastoreUnhandled.Wrap(err).WithStack()
		}
	}

	var lastItem, prevLastItem Entity
	if uint64(len(results)) > pagOpts.Limit {
		lastItem = results[pagOpts.Limit-1]
		results = results[:pagOpts.Limit]
	}
	if uint64(len(prevResults)) > pagOpts.Limit {
		prevLastItem = prevResults[pagOpts.Limit-1]
	}
	next, prev := pagOpts.Sorts.Cursors(lastItem, prevLastItem)

	return gimpl.Paginated[E]{
		Items: results,
		Meta: gimpl.PaginationMeta{
			Total: total,
			Next:  next,
			Prev:  prev,
		},
	}, nil
}

func (g *getter[E]) get(ctx context.Context, tOpts txOpts, query string, args ...any) ([]E, error) {
	rows, err := tOpts.Tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, gimpl.ErrDatastoreUnhandled.Wrap(err).WithStack()
	}
	defer rows.Close()

	var results []E
	for rows.Next() {
		e := (*new(E)).New().(E)
		err := rows.Scan(e.GetColumnPtrs()...)
		if err != nil {
			return nil, gimpl.ErrDatastoreUnhandled.Wrap(err).WithStack()
		}
		results = append(results, e)
	}
	if err := rows.Err(); err != nil {
		return nil, gimpl.ErrDatastoreUnhandled.Wrap(err).WithStack()
	}

	return results, nil
}
