package db

import (
	"context"
	"database/sql"

	"github.com/linkinlog/throttlr/internal/models"
)

func NewEndpointStore(db *sql.DB) *EndpointStore {
	return &EndpointStore{db: db}
}

type EndpointStore struct{ db *sql.DB }

func (es *EndpointStore) Store(ctx context.Context, e *models.Endpoint) (int, error) {
	tx, err := es.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	keyStore := NewKeyStore(es.db)
	key, err := keyStore.IdFromKey(e.ApiKey)
	if err != nil {
		return 0, err
	}

	var id int
	err = tx.QueryRowContext(ctx, "INSERT INTO endpoints (api_key_id, original_url, throttlr_url) VALUES (?, ?, ?) Returning id", key, e.OriginalUrl, e.ThrottlrPath).Scan(&id)
	if err != nil {
		return 0, err
	}

	return id, tx.Commit()
}

func (es *EndpointStore) Exists(ctx context.Context, endpoint *models.Endpoint) (bool, error) {
	var exists bool
	err := es.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT id FROM endpoints WHERE original_url = ?)", endpoint.OriginalUrl).Scan(&exists)
	return exists, err
}
