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

func (es *EndpointStore) AllForKey(ctx context.Context, apiKeyId int) ([]*models.Endpoint, error) {
	rows, err := es.db.QueryContext(ctx, "SELECT original_url, throttlr_url FROM endpoints JOIN api_keys on api_keys.id = ? where api_keys.valid = true", apiKeyId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var endpoints []*models.Endpoint
	for rows.Next() {
		e := &models.Endpoint{}
		err := rows.Scan(&e.OriginalUrl, &e.ThrottlrPath)
		if err != nil {
			return nil, err
		}
		endpoints = append(endpoints, e)
	}

	return endpoints, nil
}

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

	// not a huge fan of any of this but it works for now
	for {
		var exists bool
		err := es.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT throttlr_url FROM endpoints WHERE throttlr_url = ?)", e.OriginalUrl, e.ApiKey).Scan(&exists)
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
	err = tx.QueryRowContext(ctx, "INSERT INTO endpoints (api_key_id, original_url, throttlr_url) VALUES (?, ?, ?) Returning id", key, e.OriginalUrl, e.ThrottlrPath).Scan(&id)
	if err != nil {
		return 0, err
	}

	return id, tx.Commit()
}

func (es *EndpointStore) ExistsByOriginal(ctx context.Context, endpoint *models.Endpoint, apiKeyId int) (bool, error) {
	var exists bool
	err := es.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT id FROM endpoints WHERE original_url = ? and api_key_id = ?)", endpoint.OriginalUrl, apiKeyId).Scan(&exists)
	return exists, err
}

func (es *EndpointStore) ExistsByThrottlr(ctx context.Context, endpoint *models.Endpoint, apiKeyId int) (bool, error) {
	var exists bool
	err := es.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT id FROM endpoints WHERE throttlr_url = ? and api_key_id = ?)", endpoint.ThrottlrPath, apiKeyId).Scan(&exists)
	return exists, err
}

func (es *EndpointStore) Fill(ctx context.Context, endpoint *models.Endpoint, apiKeyId int) error {
	err := es.db.QueryRowContext(ctx, "SELECT api_key_id, original_url FROM endpoints WHERE throttlr_url = ? and api_key_id = ?", endpoint.ThrottlrPath, apiKeyId).Scan(&endpoint.ApiKey, &endpoint.OriginalUrl)
	return err
}
