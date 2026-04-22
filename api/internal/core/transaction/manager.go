package transaction

import (
	"context"
	"database/sql"
	"errors"
)

type Manager interface {
	RunInTx(ctx context.Context, fn func(tx *sql.Tx) error) error
}

type manager struct {
	db *sql.DB
}

func NewManager(db *sql.DB) Manager {
	return &manager{db: db}
}

func (m *manager) RunInTx(ctx context.Context, fn func(tx *sql.Tx) error) error {
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	fnErr := fn(tx)
	if fnErr == nil {
		if commitErr := tx.Commit(); commitErr != nil {
			return commitErr
		}
		return nil
	}

	if rbErr := tx.Rollback(); rbErr != nil {
		return errors.Join(fnErr, rbErr)
	}

	return fnErr
}
