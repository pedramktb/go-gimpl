package pgimpl

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/Masterminds/squirrel"
	"github.com/pedramktb/go-gimpl"
	"github.com/pedramktb/go-tagerr"
)

type updater[U UpdateEntity] struct {
	db    *sql.DB
	table string
}

// Updater returns a gimpl.Update function for the specified entity type and table name
// U must implement the UpdateEntity interface
func Updater[U UpdateEntity](db *sql.DB, table string) gimpl.Updater[U] {
	return &updater[U]{db, table}
}

func (u *updater[U]) Update(ctx context.Context, items []U, txOpts ...gimpl.TxOpt) (err error) {
	if len(items) == 0 {
		return nil
	}

	sample := (*new(U))

	tOpts, err := TxOpts(ctx, u.db, txOpts)
	if err != nil {
		return err
	}
	if tOpts.AutoFinalize {
		defer func() { err = tOpts.Tx.Finalize(err) }()
	}

	// Check if all items exist
	checkBuilder := squirrel.Select("COUNT(*)").PlaceholderFormat(squirrel.Dollar).From(u.table)
	var orConditions []squirrel.Sqlizer
	for i := range items {
		andConditions := squirrel.And{}
		for j := range sample.IdentifyColumns() {
			andConditions = append(andConditions, squirrel.Eq{sample.IdentifyColumns()[j]: items[i].IdentifyColumnVals()[j]})
		}
		orConditions = append(orConditions, andConditions)
	}
	checkBuilder = checkBuilder.Where(squirrel.Or(orConditions))
	query, args, err := checkBuilder.ToSql()
	if err != nil {
		return gimpl.ErrDatastoreUnhandled.Wrap(err).WithStack()
	}
	var count int
	if err := tOpts.Tx.QueryRowContext(ctx, query, args...).Scan(&count); err != nil {
		return gimpl.ErrDatastoreUnhandled.Wrap(err).WithStack()
	}
	if count < len(items) {
		return tagerr.ErrNotFound.Wrap(fmt.Errorf("not all entities to update were found"))
	}

	// Creation is not going to happen, but needed for query validation
	builder := squirrel.Insert(u.table).PlaceholderFormat(squirrel.Dollar).
		Columns(append(sample.IdentifyColumns(), sample.UpdateColumns()...)...)
	for i := range items {
		builder = builder.Values(append(items[i].IdentifyColumnVals(), items[i].UpdateColumnVals()...)...)
	}
	builder = builder.Suffix("ON CONFLICT (" + strings.Join(sample.IdentifyColumns(), ",") +
		") DO UPDATE SET " + strings.Join(func() []string {
		setClauses := make([]string, len(sample.UpdateColumns()))
		for i, c := range sample.UpdateColumns() {
			setClauses[i] = fmt.Sprintf("%s=EXCLUDED.%s", c, c)
		}
		return setClauses
	}(), ",") + ";")
	query, args, err = builder.ToSql()
	if err != nil {
		return gimpl.ErrDatastoreUnhandled.Wrap(err).WithStack()
	}
	if _, err := tOpts.Tx.ExecContext(ctx, query, args...); err != nil {
		return gimpl.ErrDatastoreUnhandled.Wrap(err).WithStack()
	}

	return nil
}
