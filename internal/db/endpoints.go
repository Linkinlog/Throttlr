package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/linkinlog/throttlr/internal/models"
)

func NewEndpointStore(db *pgx.Conn) *EndpointStore {
	return &EndpointStore{db: db}
}

type EndpointStore struct{ db *pgx.Conn }

func (es *EndpointStore) AllForUser(ctx context.Context, userId string) ([]*models.Endpoint, error) {
	allQuery := `
SELECT
  original_url,
  throttlr_url,
  max,
  interval
FROM
  endpoints
  JOIN api_keys on api_keys.user_id = endpoints.user_id
  JOIN buckets on buckets.id = endpoints.bucket_id
where
  api_keys.valid = true
  and api_keys.user_id = $1
`
	rows, err := es.db.Query(ctx, allQuery, userId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var endpoints []*models.Endpoint
	for rows.Next() {
		e := &models.Endpoint{
			Bucket: &models.Bucket{},
		}
		err := rows.Scan(&e.OriginalUrl, &e.ThrottlrPath, &e.Bucket.Max, &e.Bucket.Interval)
		if err != nil {
			return nil, fmt.Errorf("failed to scan endpoint: %w", err)
		}
		endpoints = append(endpoints, e)
	}

	return endpoints, nil
}

func (es *EndpointStore) Store(ctx context.Context, e *models.Endpoint, userId string) (int, error) {
	tx, err := es.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return 0, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	// not a huge fan of any of this but it works for now
	for {
		var exists bool
		err := es.db.QueryRow(
			ctx,
			"SELECT EXISTS(SELECT throttlr_url FROM endpoints WHERE throttlr_url = $1 and endpoints.user_id = $2)",
			e.OriginalUrl,
			userId,
		).Scan(&exists)
		if err != nil {
			return 0, err
		}

		if exists {
			e.ThrottlrPath = models.GeneratePath()
		} else {
			break
		}
	}

	var bucketId int
	err = tx.QueryRow(ctx, "INSERT INTO buckets (max, interval) VALUES ($1, $2) Returning id", e.Bucket.Max, e.Bucket.Interval).Scan(&bucketId)
	if err != nil {
		return 0, err
	}

	var id int
	err = tx.QueryRow(ctx, "INSERT INTO endpoints (user_id, bucket_id, original_url, throttlr_url) VALUES ($1, $2, $3, $4) Returning id", userId, bucketId, e.OriginalUrl, e.ThrottlrPath).Scan(&id)
	if err != nil {
		return 0, err
	}

	return id, tx.Commit(ctx)
}

func (es *EndpointStore) ExistsByOriginal(ctx context.Context, endpoint *models.Endpoint, userId string) (bool, error) {
	var exists bool
	err := es.db.QueryRow(ctx, "SELECT EXISTS(SELECT id FROM endpoints WHERE original_url = $1 and user_id = $2)", endpoint.OriginalUrl, userId).Scan(&exists)
	return exists, err
}

func (es *EndpointStore) ExistsByThrottlr(ctx context.Context, endpoint *models.Endpoint, userId string) (bool, error) {
	var exists bool
	err := es.db.QueryRow(ctx, "SELECT EXISTS(SELECT id FROM endpoints WHERE throttlr_url = $1 and user_id = $2)", endpoint.ThrottlrPath, userId).Scan(&exists)
	return exists, err
}

func (es *EndpointStore) Fill(ctx context.Context, endpoint *models.Endpoint, userId string) error {
	query := `
SELECT
	throttlr_url,
	original_url,
	max,
	interval
FROM
	endpoints
JOIN
	buckets
ON
	buckets.id = endpoints.bucket_id
WHERE
	throttlr_url = $1
	and user_id = $2
`
	err := es.db.QueryRow(ctx, query, endpoint.ThrottlrPath, userId).Scan(&endpoint.ThrottlrPath, &endpoint.OriginalUrl, &endpoint.Bucket.Max, &endpoint.Bucket.Interval)
	return err
}
