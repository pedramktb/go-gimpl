package pgimpl

import (
	"context"
	"database/sql"

	"github.com/Masterminds/squirrel"
	"github.com/pedramktb/go-gimpl"
)

type replacer[R ReplaceEntity] struct {
	db    *sql.DB
	table string
}

// Replacer returns a gimpl.Replace function for the specified entity type and table name
// R must be a pointer struct that implements the ReplaceEntity interface
func Replacer[R ReplaceEntity](db *sql.DB, table string) gimpl.Replace[R] {
	return (&replacer[R]{db, table}).Replace
}

func (r *replacer[R]) Replace(ctx context.Context, items []R, txOpts ...gimpl.TxOpt) (err error) {
	tOpts, err := TxOpts(ctx, r.db, txOpts)
	if err != nil {
		return err
	}

	if tOpts.AutoFinalize {
		defer func() { err = tOpts.Tx.Finalize(err) }()
	}

	builder := squirrel.Update(r.table)
	// TODO: Batch replace
	query, args, err := builder.ToSql()
	if err != nil {
		return gimpl.ErrDatastoreUnhandled.Wrap(err).WithStack()
	}
	if _, err := tOpts.Tx.ExecContext(ctx, query, args...); err != nil {
		return gimpl.ErrDatastoreUnhandled.Wrap(err).WithStack()
	}
	return nil
}
