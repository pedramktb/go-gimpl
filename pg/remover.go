package pgimpl

import (
	"context"
	"database/sql"
	"errors"

	"github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/lib/pq"
	"github.com/pedramktb/go-gimpl"
	"github.com/pedramktb/go-tagerr"
)

type remover[E Entity] struct {
	db    *sql.DB
	table string
}

// Remover returns a gimpl.Remover for the specified entity type and table name
// E must implement the Entity interface
func Remover[E Entity](db *sql.DB, table string) gimpl.Remover[E] {
	return &remover[E]{db, table}
}

func (d *remover[E]) RemoveOne(ctx context.Context, filter gimpl.Expr, txOpts ...gimpl.TxOpt) (err error) {
	tOpts, err := TxOpts(ctx, d.db, txOpts)
	if err != nil {
		return err
	}
	if tOpts.AutoFinalize {
		defer func() { err = tOpts.Tx.Finalize(err) }()
	}

	builder := squirrel.Delete(d.table).PlaceholderFormat(squirrel.Dollar)
	sqlFilter, err := FromExpr(filter)
	if err != nil {
		return gimpl.ErrDatastoreUnhandled.Wrap(err).WithStack()
	}
	if sqlFilter == nil {
		// Prevent full table delete
		return gimpl.ErrDatastoreUnhandled.Wrap(gimpl.ErrInvalidExpr.Wrap(errors.New("delete operation requires a filter expression")))
	}
	query, args, err := builder.Where(sqlFilter).ToSql()
	if err != nil {
		return gimpl.ErrDatastoreUnhandled.Wrap(err).WithStack()
	}
	if tag, err := tOpts.Tx.ExecContext(ctx, query, args...); err != nil {
		var (
			pgxErr *pgconn.PgError
			pqErr  *pq.Error
		)
		switch {
		case errors.As(err, &pgxErr):
			if pgxErr.Code == "23503" {
				return gimpl.ErrInUse
			}
		case errors.As(err, &pqErr):
			if pqErr.Code == "23503" {
				return gimpl.ErrInUse
			}
		}
		return gimpl.ErrDatastoreUnhandled.Wrap(err).WithStack()
	} else {
		if n, err := tag.RowsAffected(); err != nil {
			return gimpl.ErrDatastoreUnhandled.Wrap(err).WithStack()
		} else if n == 0 {
			return tagerr.ErrNotFound
		} else if n > 1 {
			return gimpl.ErrDatastoreUnhandled.Wrap(errors.New("RemoveOne affected multiple rows"))
		}
	}
	return nil
}

func (d *remover[E]) Remove(ctx context.Context, filter gimpl.Expr, txOpts ...gimpl.TxOpt) (err error) {
	tOpts, err := TxOpts(ctx, d.db, txOpts)
	if err != nil {
		return err
	}
	if tOpts.AutoFinalize {
		defer func() { err = tOpts.Tx.Finalize(err) }()
	}

	builder := squirrel.Delete(d.table).PlaceholderFormat(squirrel.Dollar)
	sqlFilter, err := FromExpr(filter)
	if err != nil {
		return gimpl.ErrDatastoreUnhandled.Wrap(err).WithStack()
	}
	if sqlFilter == nil {
		// Prevent full table delete
		return gimpl.ErrDatastoreUnhandled.Wrap(gimpl.ErrInvalidExpr.Wrap(errors.New("delete operation requires a filter expression")))
	}
	query, args, err := builder.Where(sqlFilter).ToSql()
	if err != nil {
		return gimpl.ErrDatastoreUnhandled.Wrap(err).WithStack()
	}
	if _, err := tOpts.Tx.ExecContext(ctx, query, args...); err != nil {
		var (
			pgxErr *pgconn.PgError
			pqErr  *pq.Error
		)
		switch {
		case errors.As(err, &pgxErr):
			if pgxErr.Code == "23503" {
				return gimpl.ErrInUse
			}
		case errors.As(err, &pqErr):
			if pqErr.Code == "23503" {
				return gimpl.ErrInUse
			}
		}
		return gimpl.ErrDatastoreUnhandled.Wrap(err).WithStack()
	}

	return nil
}
