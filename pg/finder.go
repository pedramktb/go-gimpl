package pgimpl

import (
	"context"
	"database/sql"
	"errors"

	"github.com/Masterminds/squirrel"
	"github.com/pedramktb/go-gimpl"
	"github.com/pedramktb/go-tagerr"
)

type finder[E Entity] struct {
	db    *sql.DB
	table string
}

// Finder returns a gimpl.Finder for the specified entity type and table name
// E must implement the Entity interface
func Finder[E Entity](db *sql.DB, table string) gimpl.Finder[E] {
	return &finder[E]{db, table}
}

func (f *finder[E]) FindOne(ctx context.Context, filter gimpl.Expr, txOpts ...gimpl.TxOpt) (E, error) {
	sample := (*new(E))
	tOpts, err := TxOpts(ctx, f.db, txOpts)
	if err != nil {
		return sample, err
	}
	if tOpts.AutoFinalize {
		defer func() { err = tOpts.Tx.Finalize(err) }()
	}

	builder := squirrel.Select(sample.PgColumns()...).PlaceholderFormat(squirrel.Dollar).From(f.table)
	sqlFilter, err := FromExpr(filter)
	if err != nil {
		return sample, err
	}
	query, args, err := builder.Where(sqlFilter).ToSql()
	if err != nil {
		return sample, gimpl.ErrDatastoreUnhandled.Wrap(err).WithStack()
	}
	e, ptrs := sample.NewWithPgColumnPtrs()
	if err = tOpts.Tx.QueryRowContext(ctx, query, args...).Scan(ptrs...); errors.Is(err, sql.ErrNoRows) {
		return sample, tagerr.ErrNotFound
	} else if err != nil {
		return sample, gimpl.ErrDatastoreUnhandled.Wrap(err).WithStack()
	}
	return *(e.(*E)), nil
}

func (f *finder[E]) Find(ctx context.Context, filter gimpl.Expr, paginateOpts []gimpl.PaginateOpt, txOpts ...gimpl.TxOpt) (_ gimpl.Paginated[E], err error) {
	pagOpts := gimpl.PaginateOpts(paginateOpts...)
	tOpts, err := TxOpts(ctx, f.db, txOpts)
	if err != nil {
		return gimpl.Paginated[E]{}, err
	}
	if tOpts.AutoFinalize {
		defer func() { err = tOpts.Tx.Finalize(err) }()
	}

	builder := squirrel.Select((*new(E)).PgColumns()...).PlaceholderFormat(squirrel.Dollar).From(f.table)
	sqlFilter, err := FromExpr(filter)
	if err != nil {
		return gimpl.Paginated[E]{}, err
	}
	builder = builder.Where(sqlFilter)

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
	builder, prevCursorBuilder, err = FromSorts(pagOpts.Sorts, builder)
	if err != nil {
		return gimpl.Paginated[E]{}, err
	}
	query, args, err := builder.Limit(pagOpts.Limit + 1).ToSql()
	if err != nil {
		return gimpl.Paginated[E]{}, gimpl.ErrDatastoreUnhandled.Wrap(err).WithStack()
	}
	results, err := f.fetch(ctx, tOpts, query, args...)
	if err != nil {
		return gimpl.Paginated[E]{}, gimpl.ErrDatastoreUnhandled.Wrap(err).WithStack()
	}

	var prevResults []E
	if prevCursorBuilder != nil {
		prevCursorQuery, prevCursorArgs, err := prevCursorBuilder.Limit(pagOpts.Limit + 1).ToSql()
		if err != nil {
			return gimpl.Paginated[E]{}, gimpl.ErrDatastoreUnhandled.Wrap(err).WithStack()
		}
		prevResults, err = f.fetch(ctx, tOpts, prevCursorQuery, prevCursorArgs...)
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

func (f *finder[E]) fetch(ctx context.Context, tOpts txOpts, query string, args ...any) ([]E, error) {
	rows, err := tOpts.Tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, gimpl.ErrDatastoreUnhandled.Wrap(err).WithStack()
	}
	defer rows.Close()

	var results []E
	for rows.Next() {
		e, ptrs := (*new(E)).NewWithPgColumnPtrs()
		err := rows.Scan(ptrs...)
		if err != nil {
			return nil, gimpl.ErrDatastoreUnhandled.Wrap(err).WithStack()
		}
		results = append(results, *(e.(*E)))
	}
	if err := rows.Err(); err != nil {
		return nil, gimpl.ErrDatastoreUnhandled.Wrap(err).WithStack()
	}

	return results, nil
}
