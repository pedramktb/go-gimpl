package pgimpl

import (
	"context"
	"database/sql"
	"errors"

	"github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pedramktb/go-gimpl"
	"github.com/pedramktb/go-tagerr"
)

type deleter[E Entity] struct {
	db    *sql.DB
	table string
}

// Deleter returns a gimpl.Delete function for the specified entity type and table name
// E must implement the Entity interface
func Deleter[E Entity](db *sql.DB, table string) gimpl.Delete[E] {
	return (&deleter[E]{db, table}).Delete
}

func (d *deleter[E]) Delete(ctx context.Context, locateOpts []gimpl.LocateOpt, txOpts ...gimpl.TxOpt) (err error) {
	locOpts := gimpl.LocateOpts(locateOpts...)
	tOpts, err := TxOpts(ctx, d.db, txOpts)
	if err != nil {
		return err
	}
	if tOpts.AutoFinalize {
		defer func() { err = tOpts.Tx.Finalize(err) }()
	}

	builder := squirrel.Delete(d.table).PlaceholderFormat(squirrel.Dollar)
	filter, err := FromExpr(locOpts.Filter)
	if err != nil {
		return gimpl.ErrDatastoreUnhandled.Wrap(err).WithStack()
	}
	if filter == nil {
		// Prevent full table delete
		return gimpl.ErrDatastoreUnhandled.Wrap(gimpl.ErrInvalidExpr.Wrap(errors.New("delete operation requires a filter expression")))
	}
	builder = builder.Where(filter)
	query, args, err := builder.ToSql()
	if tag, err := tOpts.Tx.ExecContext(ctx, query, args...); err != nil {
		// Check for foreign‐key violation (in use)
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			return gimpl.ErrInUse
		}
		return gimpl.ErrDatastoreUnhandled.Wrap(err).WithStack()
	} else {
		if n, err := tag.RowsAffected(); err != nil {
			return gimpl.ErrDatastoreUnhandled.Wrap(err).WithStack()
		} else if n == 0 {
			return tagerr.ErrNotFound
		}
	}
	return nil
}
