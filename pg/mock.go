package pgimpl

import (
	"context"

	"github.com/stretchr/testify/mock"
)

// TxMock is a mock implementation of Tx and TxCreator for testing purposes.
type TxMock struct{ mock.Mock }

var _ CreateTx = (&TxMock{}).Create
var _ Tx = &TxMock{}

func (t *TxMock) Create(ctx context.Context) (Tx, error) {
	args := t.Called(ctx)
	err := args.Error(0)
	if err != nil {
		return nil, err
	}
	return &TxMock{}, nil
}

func (t *TxMock) Finalize(opErr error) error {
	args := t.Called(opErr)
	if opErr != nil {
		return opErr
	}
	return args.Error(0)
}
