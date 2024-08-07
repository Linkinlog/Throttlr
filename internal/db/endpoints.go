package db

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/linkinlog/throttlr/internal/models"
)

func NewEndpointStore(db *pgx.Conn) *EndpointStore {
	return &EndpointStore{db: db}
}

type EndpointStore struct{ db *pgx.Conn }

func (es *EndpointStore) AllForKey(ctx context.Context, apiKeyId int) ([]*models.Endpoint, error) {
	rows, err := es.db.Query(ctx, "SELECT original_url, throttlr_url, key FROM endpoints JOIN api_keys on api_keys.id = $1 where api_keys.valid = true", apiKeyId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var endpoints []*models.Endpoint
	for rows.Next() {
		e := &models.Endpoint{}
		err := rows.Scan(&e.OriginalUrl, &e.ThrottlrPath, &e.ApiKey)
		if err != nil {
			return nil, err
		}
		endpoints = append(endpoints, e)
	}

	return endpoints, nil
}

func (es *EndpointStore) Store(ctx context.Context, e *models.Endpoint) (int, error) {
	tx, err := es.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return 0, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	keyStore := NewKeyStore(es.db)
	key, err := keyStore.IdFromKey(e.ApiKey)
	if err != nil {
		return 0, err
	}

	// not a huge fan of any of this but it works for now
	for {
		var exists bool
		err := es.db.QueryRow(ctx, "SELECT EXISTS(SELECT throttlr_url FROM endpoints WHERE throttlr_url = $1 and endpoints.api_key_id = $2)", e.OriginalUrl, key).Scan(&exists)
		if err != nil {
			return 0, err
		}
		if exists {
			e.ThrottlrPath = models.GeneratePath()
		} else {
			break
		}
	}

	var id int
	err = tx.QueryRow(ctx, "INSERT INTO endpoints (api_key_id, original_url, throttlr_url) VALUES ($1, $2, $3) Returning id", key, e.OriginalUrl, e.ThrottlrPath).Scan(&id)
	if err != nil {
		return 0, err
	}

	return id, tx.Commit(ctx)
}

func (es *EndpointStore) ExistsByOriginal(ctx context.Context, endpoint *models.Endpoint, apiKeyId int) (bool, error) {
	var exists bool
	err := es.db.QueryRow(ctx, "SELECT EXISTS(SELECT id FROM endpoints WHERE original_url = $1 and api_key_id = $2)", endpoint.OriginalUrl, apiKeyId).Scan(&exists)
	return exists, err
}

func (es *EndpointStore) ExistsByThrottlr(ctx context.Context, endpoint *models.Endpoint, apiKeyId int) (bool, error) {
	var exists bool
	err := es.db.QueryRow(ctx, "SELECT EXISTS(SELECT id FROM endpoints WHERE throttlr_url = $1 and api_key_id = $2)", endpoint.ThrottlrPath, apiKeyId).Scan(&exists)
	return exists, err
}

func (es *EndpointStore) Fill(ctx context.Context, endpoint *models.Endpoint, apiKeyId int) error {
	err := es.db.QueryRow(ctx, "SELECT api_key_id, original_url FROM endpoints WHERE throttlr_url = $1 and api_key_id = $2", endpoint.ThrottlrPath, apiKeyId).Scan(&endpoint.ApiKey, &endpoint.OriginalUrl)
	return err
}
