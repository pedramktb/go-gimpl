package pgimpl

import (
	"context"
	"database/sql"

	"github.com/Masterminds/squirrel"
	"github.com/pedramktb/go-gimpl"
)

type updater[U UpdateEntity] struct {
	db    *sql.DB
	table string
}

// Updater returns a gimpl.Update function for the specified entity type and table name
// U must be a pointer struct that implements the UpdateEntity interface
func Updater[U UpdateEntity](db *sql.DB, table string) gimpl.Update[U] {
	return (&updater[U]{db, table}).Update
}

func (u *updater[U]) Update(ctx context.Context, item U, txOpts ...gimpl.TxOpt) (err error) {
	tOpts, err := TxOpts(ctx, u.db, txOpts)
	if err != nil {
		return err
	}

	if tOpts.AutoFinalize {
		defer func() { err = tOpts.Tx.Finalize(err) }()
	}

	builder := squirrel.Update(u.table).PlaceholderFormat(squirrel.Dollar)
	for i := range item.IdentifyColumns() {
		builder = builder.Where(squirrel.Eq{item.IdentifyColumns()[i]: item.IdentifyColumnPtrs()[i]})
	}
	for i := range item.UpdateColumns() {
		if item.UpdateColumnPtrs()[i] != nil {
			builder = builder.Set(item.UpdateColumns()[i], item.UpdateColumnPtrs()[i])
		}
	}
	query, args, err := builder.ToSql()
	if err != nil {
		return gimpl.ErrDatastoreUnhandled.Wrap(err).WithStack()
	}
	if _, err := tOpts.Tx.ExecContext(ctx, query, args...); err != nil {
		return gimpl.ErrDatastoreUnhandled.Wrap(err).WithStack()
	}
	return nil
}
