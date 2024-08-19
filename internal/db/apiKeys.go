package db

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

func NewKeyStore(db *pgxpool.Pool) *KeyStore {
	return &KeyStore{db: db}
}

type KeyStore struct {
	db *pgxpool.Pool
}

func (ks *KeyStore) Exists(key string) (bool, int) {
	ctx := context.Background()
	var exists bool
	var id int
	_ = ks.db.QueryRow(ctx, "SELECT EXISTS(SELECT id FROM api_keys WHERE key = $1)", key).Scan(&exists)
	if exists {
		_ = ks.db.QueryRow(ctx, "SELECT id FROM api_keys WHERE key = $1", key).Scan(&id)
	}
	return exists, id
}

func (ks *KeyStore) Valid(id int) bool {
	ctx := context.Background()
	var valid bool
	_ = ks.db.QueryRow(ctx, "SELECT valid FROM api_keys WHERE id = $1", id).Scan(&valid)
	return valid
}

func (ks *KeyStore) IdFromKey(key string) (string, error) {
	ctx := context.Background()
	var id string
	err := ks.db.QueryRow(ctx, "SELECT id FROM api_keys WHERE key = $1", key).Scan(&id)
	return id, err
}

func (ks *KeyStore) UserIdFromKey(key string) (string, error) {
	ctx := context.Background()
	var id string
	err := ks.db.QueryRow(ctx, "SELECT user_id FROM api_keys WHERE key = $1 and valid = true", key).Scan(&id)
	return id, err
}
