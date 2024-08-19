package db

import (
	"context"
	"log/slog"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

func NewKeyStore(db *pgxpool.Pool) *KeyStore {
	s := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	return &KeyStore{db: db, s: s}
}

type KeyStore struct {
	db *pgxpool.Pool
	s  *slog.Logger
}

func (ks *KeyStore) Exists(key string, ctx context.Context) (bool, int) {
	var exists bool
	var id int
	err := ks.db.QueryRow(ctx, "SELECT EXISTS(SELECT id FROM api_keys WHERE key = $1)", key).Scan(&exists)
	if err != nil {
		ks.s.Error("failed to check if api key exists", "err", err)
		return false, 0
	}
	if exists {
		err = ks.db.QueryRow(ctx, "SELECT id FROM api_keys WHERE key = $1", key).Scan(&id)
		if err != nil {
			ks.s.Error("failed to get api key id", "err", err)
			return false, 0
		}
	}
	return exists, id
}

func (ks *KeyStore) Valid(id int, ctx context.Context) bool {
	var valid bool
	err := ks.db.QueryRow(ctx, "SELECT valid FROM api_keys WHERE id = $1", id).Scan(&valid)
	if err != nil {
		ks.s.Error("failed to check if api key is valid", "err", err)
	}
	return valid
}

func (ks *KeyStore) IdFromKey(key string, ctx context.Context) (string, error) {
	var id string
	err := ks.db.QueryRow(ctx, "SELECT id FROM api_keys WHERE key = $1", key).Scan(&id)
	return id, err
}

func (ks *KeyStore) UserIdFromKey(key string, ctx context.Context) (string, error) {
	var id string
	err := ks.db.QueryRow(ctx, "SELECT user_id FROM api_keys WHERE key = $1 and valid = true", key).Scan(&id)
	return id, err
}
