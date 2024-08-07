package db

import (
	"context"
	"strconv"

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

	query := `
SELECT
  users.id,
  users.name,
  users.email,
  api_keys.key,
  api_keys.valid
FROM
  users
  left join api_keys on users.id = api_keys.user_id
  and api_keys.valid = true
WHERE
  users.id = $1
`

	err := us.db.QueryRow(ctx, query, id).Scan(&user.Id, &user.Name, &user.Email, &key, &validKey)
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

	deleteBucketsQuery := `
select
  buckets.id
from
  buckets
  join endpoints on endpoints.user_id = $1
where
  endpoints.user_id = $1
group by buckets.id;
`

	rows, err := us.db.Query(ctx, deleteBucketsQuery, id)
	if err != nil {
		return err
	}
	var buckets []string
	for rows.Next() {
		var b int
		err = rows.Scan(&b)
		if err != nil {
			return err
		}
		s := strconv.Itoa(b)
		buckets = append(buckets, s)
	}

	for _, b := range buckets {
		_, err = tx.Exec(ctx, "DELETE FROM buckets WHERE id = $1", b)
		if err != nil {
			return err
		}
	}

	deleteEndpointsQuery := `
select
  endpoints.id
from
  endpoints
where
  endpoints.user_id = $1;
`
	rows, err = us.db.Query(ctx, deleteEndpointsQuery, id)
	if err != nil {
		return err
	}

	var endpoints []int
	for rows.Next() {
		var e int
		err = rows.Scan(&e)
		if err != nil {
			return err
		}
		endpoints = append(endpoints, e)
	}

	for _, e := range endpoints {
		_, err = tx.Exec(ctx, "DELETE FROM endpoints WHERE id = $1", e)
		if err != nil {
			return err
		}
	}

	deleteApiKeysQuery := `
select
  api_keys.id
from
  api_keys
where
  api_keys.user_id = $1;
`

	rows, err = us.db.Query(ctx, deleteApiKeysQuery, id)
	if err != nil {
		return err
	}

	var apiKeys []int
	for rows.Next() {
		var k int
		err = rows.Scan(&k)
		if err != nil {
			return err
		}
		apiKeys = append(apiKeys, k)
	}

	for _, k := range apiKeys {
		_, err = tx.Exec(ctx, "DELETE FROM api_keys WHERE id = $1", k)
		if err != nil {
			return err
		}
	}

	deleteUsersQuery := `
select
  users.id
from
  users
where
  users.id = $1;
`

	rows, err = us.db.Query(ctx, deleteUsersQuery, id)
	if err != nil {
		return err
	}

	var users []string
	for rows.Next() {
		var u string
		err = rows.Scan(&u)
		if err != nil {
			return err
		}
		users = append(users, u)
	}

	for _, u := range users {
		_, err = tx.Exec(ctx, "DELETE FROM users WHERE id = $1", u)
		if err != nil {
			return err
		}
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
