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

func (us *UserStore) ById(ctx context.Context, id string) (*models.User, error) {
	user := struct {
		models.User
		deletedAt *string
	}{}

	err := us.db.QueryRowContext(ctx, "SELECT id, name, email, deleted_at FROM users WHERE id = ?", id).Scan(&user.Id, &user.Name, &user.Email, &user.deletedAt)
	if err != nil {
		return &models.User{}, err
	}

	if user.deletedAt != nil {
		return &models.User{}, sql.ErrNoRows
	}

	return &models.User{
		Id:    user.Id,
		Name:  user.Name,
		Email: user.Email,
	}, nil
}
