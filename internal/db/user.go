package db

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
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

	_, err = tx.ExecContext(ctx, "INSERT INTO api_keys (user_id, key, valid) VALUES (?, ?, ?)", u.Id, u.ApiKey, true)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (us *UserStore) ById(ctx context.Context, id string) (*models.User, error) {
	var user models.User
	var validKey *bool
	var key *string

	err := us.db.QueryRowContext(ctx, "SELECT users.id, users.name, users.email, api_keys.key, api_keys.valid FROM users left join api_keys on users.id = api_keys.user_id and api_keys.valid = true WHERE users.id = ?", id).Scan(&user.Id, &user.Name, &user.Email, &key, &validKey)
	if err != nil {
		return &models.User{}, err
	}

	// shouldnt ever be invalid, but just in case
	if validKey != nil && *validKey && key != nil {
		user.ApiKey = uuid.MustParse(*key)
	}

	return &models.User{
		Id:     user.Id,
		Name:   user.Name,
		Email:  user.Email,
		ApiKey: user.ApiKey,
	}, nil
}

func (us *UserStore) Delete(ctx context.Context, id string) error {
	tx, err := us.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	_, err = tx.ExecContext(ctx, "DELETE FROM users WHERE id = ?", id)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, "DELETE FROM api_keys WHERE user_id = ?", id)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (us *UserStore) RegenerateApiKey(ctx context.Context, u models.User) (uuid.UUID, error) {
	tx, err := us.db.BeginTx(ctx, nil)
	if err != nil {
		return uuid.UUID{}, err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	_, err = tx.ExecContext(ctx, "UPDATE api_keys SET valid = false WHERE key = ?", u.ApiKey.String())
	if err != nil {
		return uuid.UUID{}, err
	}

	key := uuid.New()
	_, err = tx.ExecContext(ctx, "INSERT INTO api_keys (user_id, key, valid) VALUES (?, ?, ?)", u.Id, key.String(), true)
	if err != nil {
		return uuid.UUID{}, err
	}

	return key, tx.Commit()
}
