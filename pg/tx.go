package pgimpl

import (
	"context"
	"database/sql"
	"errors"

	"github.com/pedramktb/go-gimpl"
)

type CreateTx func(context.Context) (Tx, error)

func NewTxCreator(db *sql.DB) CreateTx {
	return (&txCreator{db: db}).tx
}

type txCreator struct{ db *sql.DB }

func (c *txCreator) tx(ctx context.Context) (Tx, error) {
	return NewTx(ctx, c.db)
}

// Tx wraps a sql.Tx to provide helper methods for committing or rolling back.
type Tx struct {
	*sql.Tx
}

// New begins a new transaction on the given pool. The returned *Tx
// should be finalized with Finalize to commit or roll back.
func NewTx(ctx context.Context, db *sql.DB) (Tx, error) {
	tx, err := db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return Tx{}, gimpl.ErrDatastoreUnhandled.Wrap(err).WithStack()
	}
	return Tx{tx}, nil
}

// Finalize will commit the transaction if err is nil; otherwise it rolls back.
// It returns any error encountered during commit or rollback.
func (t Tx) Finalize(err error) error {
	if err != nil {
		if err2 := t.Rollback(); err2 != nil {
			return errors.Join(err, gimpl.ErrDatastoreUnhandled.Wrap(err2).WithStack())
		}
		return err
	}
	if err := t.Commit(); err != nil {
		return gimpl.ErrDatastoreUnhandled.Wrap(err).WithStack()
	}
	return nil
}

type txOpts struct {
	Tx           Tx
	AutoFinalize bool
}

func TxOpts(ctx context.Context, db *sql.DB, opts any) (txOpts, error) {
	options, _ := opts.([]TxOpt)

	o := txOpts{}

	// Apply any user-supplied TX first
	for i := range options {
		options[i](&o)
	}

	// If no Tx supplied, begin a new one
	if o.Tx.Tx == nil {
		tx, err := NewTx(ctx, db)
		if err != nil {
			return o, err
		}
		o.Tx = tx
		o.AutoFinalize = true
	}

	return o, nil
}

type TxOpt func(*txOpts)

// WithTx instructs the repository to use the provided transaction instead of creating a new one.
// Any error in creating or setting audit parameters must be handled before passing the Tx here.
func WithTx(tx Tx) TxOpt {
	return func(o *txOpts) {
		o.Tx = tx
	}
}

// WithTxAutoFinalize instructs the repository to finalize the transaction after the operation completes.
func WithTxAutoFinalize() TxOpt {
	return func(o *txOpts) {
		o.AutoFinalize = true
	}
}
