package pgimpl

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/Masterminds/squirrel"
	"github.com/pedramktb/go-gimpl"
)

type createOrReplacer[CR CreateOrReplaceEntity] struct {
	db    *sql.DB
	table string
}

// CreateOrReplacer returns a gimpl.CreateOrReplace function for the specified entity type and table name
// CR must be a pointer struct that implements the CreateOrReplaceEntity interface
func CreateOrReplacer[CR CreateOrReplaceEntity](db *sql.DB, table string) gimpl.CreateOrReplace[CR] {
	return (&createOrReplacer[CR]{db, table}).CreateOrReplace
}

func (r *createOrReplacer[CR]) CreateOrReplace(ctx context.Context, items []CR, txOpts ...gimpl.TxOpt) (err error) {
	sample := (*new(CR))

	tOpts, err := TxOpts(ctx, r.db, txOpts)
	if err != nil {
		return err
	}

	if tOpts.AutoFinalize {
		defer func() { err = tOpts.Tx.Finalize(err) }()
	}

	builder := squirrel.
		Insert(r.table).
		PlaceholderFormat(squirrel.Dollar).
		Columns(append(sample.IdentifyColumns(), sample.ReplaceColumns()...)...)

	for i := range items {
		builder = builder.Values(append(items[i].IdentifyColumnPtrs(), items[i].ReplaceColumnPtrs()...)...)
	}

	builder = builder.Suffix("ON CONFLICT (" + strings.Join(sample.IdentifyColumns(), ",") +
		") DO UPDATE SET " + strings.Join(func() []string {
		setClauses := make([]string, len(sample.ReplaceColumns()))
		for i, c := range sample.ReplaceColumns() {
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
