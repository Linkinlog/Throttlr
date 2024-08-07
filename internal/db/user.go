package db

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/linkinlog/throttlr/internal/models"
)

func NewUserStore(db *pgx.Conn) *UserStore {
	return &UserStore{db: db}
}

type UserStore struct {
	db *pgx.Conn
}

func (us *UserStore) Store(ctx context.Context, u models.User) error {
	tx, err := us.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	_, err = tx.Exec(ctx, "INSERT INTO users (id, name, email) VALUES ($1, $2, $3)", u.Id, u.Name, u.Email)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, "INSERT INTO api_keys (user_id, key, valid) VALUES ($1, $2, $3)", u.Id, u.ApiKey, true)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (us *UserStore) ById(ctx context.Context, id string) (*models.User, error) {
	var user models.User
	var validKey *bool
	var key *string

	err := us.db.QueryRow(ctx, "SELECT users.id, users.name, users.email, api_keys.key, api_keys.valid FROM users left join api_keys on users.id = api_keys.user_id and api_keys.valid = true WHERE users.id = $1", id).Scan(&user.Id, &user.Name, &user.Email, &key, &validKey)
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
	tx, err := us.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	_, err = tx.Exec(ctx, "DELETE FROM users WHERE id = $1", id)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, "DELETE FROM api_keys WHERE user_id = $1", id)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (us *UserStore) RegenerateApiKey(ctx context.Context, u models.User) (uuid.UUID, error) {
	tx, err := us.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return uuid.UUID{}, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	_, err = tx.Exec(ctx, "UPDATE api_keys SET valid = false WHERE key = $1", u.ApiKey.String())
	if err != nil {
		return uuid.UUID{}, err
	}

	key := uuid.New()
	_, err = tx.Exec(ctx, "INSERT INTO api_keys (user_id, key, valid) VALUES ($1, $2, $3)", u.Id, key.String(), true)
	if err != nil {
		return uuid.UUID{}, err
	}

	return key, tx.Commit(ctx)
}
