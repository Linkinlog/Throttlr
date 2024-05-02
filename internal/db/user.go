package db

import (
	"context"
	"database/sql"

	"github.com/linkinlog/throttlr/internal/models"
	_ "modernc.org/sqlite"
)

func NewUserStore(db *sql.DB) *UserStore {
	return &UserStore{db: db}
}

type UserStore struct {
	db *sql.DB
}

func (us *UserStore) Store(ctx context.Context, u models.User) error {
	tx, err := us.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	_, err = tx.ExecContext(ctx, "INSERT INTO users (id, name, email) VALUES (?, ?, ?)", u.Id, u.Name, u.Email)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (us *UserStore) ById(ctx context.Context, id string) (models.User, error) {
	var u models.User
	err := us.db.QueryRowContext(ctx, "SELECT id, email FROM users WHERE id = ?", id).Scan(&u.Id, &u.Email)
	if err != nil {
		return models.User{}, err
	}

	return u, nil
}
