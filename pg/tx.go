package pgimpl

import (
	"context"
	"database/sql"
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
type Tx interface {
	Finalize(opErr error) error
}

// New begins a new transaction on the given pool. The returned *Tx
// should be finalized with Finalize to commit or roll back.
func NewTx(ctx context.Context, db *sql.DB) (Tx, error) {
	tx, err := db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return nil, err
	}
	return txWrapper{tx}, nil
}

var _ Tx = txWrapper{}

type txWrapper struct {
	*sql.Tx
}

// Finalize will commit the transaction if opErr is nil; otherwise it rolls back.
// It returns any error encountered during commit or rollback.
func (t txWrapper) Finalize(opErr error) error {
	if opErr != nil {
		_ = t.Rollback()
		return opErr
	}
	return t.Commit()
}

type txOpts struct {
	Tx           Tx
	AutoFinalize bool
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

func Opts(ctx context.Context, db *sql.DB, opts any) (txOpts, error) {
	options, _ := opts.([]TxOpt)

	o := txOpts{}

	// Apply any user-supplied TX first
	for i := range options {
		options[i](&o)
	}

	// If no Tx supplied, begin a new one
	if o.Tx == nil {
		tx, err := NewTx(ctx, db)
		if err != nil {
			return o, err
		}
		o.Tx = tx
		o.AutoFinalize = true
	}

	return o, nil
}
