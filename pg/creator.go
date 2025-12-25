package pgimpl

import (
	"context"
	"database/sql"

	"github.com/Masterminds/squirrel"
	"github.com/pedramktb/go-gimpl"
)

type creator[C CreateEntity] struct {
	db    *sql.DB
	table string
}

// Creator returns a gimpl.Creator for the specified entity type and table name
// C must implement the CreateEntity interface
func Creator[C CreateEntity](db *sql.DB, table string) gimpl.Creator[C] {
	return &creator[C]{db, table}
}

func (c *creator[C]) Create(ctx context.Context, items []C, txOpts ...gimpl.TxOpt) (err error) {
	if len(items) == 0 {
		return nil
	}

	tOpts, err := TxOpts(ctx, c.db, txOpts)
	if err != nil {
		return err
	}
	if tOpts.AutoFinalize {
		defer func() { err = tOpts.Tx.Finalize(err) }()
	}

	builder := squirrel.Insert(c.table).PlaceholderFormat(squirrel.Dollar).Columns((*new(C)).CreatePgColumns()...)
	for i := range items {
		builder = builder.Values(items[i].CreatePgColumnVals()...)
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
