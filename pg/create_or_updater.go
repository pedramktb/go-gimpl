package pgimpl

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/Masterminds/squirrel"
	"github.com/pedramktb/go-gimpl"
)

type createOrUpdater[CU CreateOrUpdateEntity] struct {
	db    *sql.DB
	table string
}

// CreateOrUpdater returns a gimpl.CreateOrUpdate function for the specified entity type and table name
// CR must be a pointer struct that implements the CreateOrUpdateEntity interface
func CreateOrUpdater[CU CreateOrUpdateEntity](db *sql.DB, table string) gimpl.CreateOrUpdate[CU] {
	return (&createOrUpdater[CU]{db, table}).CreateOrUpdate
}

func (cu *createOrUpdater[CU]) CreateOrUpdate(ctx context.Context, items []CU, txOpts ...gimpl.TxOpt) (err error) {
	if len(items) == 0 {
		return nil
	}

	sample := (*new(CU))

	tOpts, err := TxOpts(ctx, cu.db, txOpts)
	if err != nil {
		return err
	}
	if tOpts.AutoFinalize {
		defer func() { err = tOpts.Tx.Finalize(err) }()
	}

	builder := squirrel.
		Insert(cu.table).
		PlaceholderFormat(squirrel.Dollar).
		Columns(sample.CreateColumns()...)

	for i := range items {
		builder = builder.Values(items[i].CreateColumnVals()...)
	}

	builder = builder.Suffix("ON CONFLICT (" + strings.Join(sample.IdentifyColumns(), ",") +
		") DO UPDATE SET " + strings.Join(func() []string {
		setClauses := make([]string, len(sample.UpdateColumns()))
		for i, c := range sample.UpdateColumns() {
			setClauses[i] = fmt.Sprintf("%s=EXCLUDED.%s", c, c)
		}
		return setClauses
	}(), ",") + ";")

	query, args, err := builder.ToSql()
	if err != nil {
		return gimpl.ErrDatastoreUnhandled.Wrap(err).WithStack()
	}
	if _, err := tOpts.Tx.ExecContext(ctx, query, args...); err != nil {
		return gimpl.ErrDatastoreUnhandled.Wrap(err).WithStack()
	}
	return nil
}
