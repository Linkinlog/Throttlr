package db

import (
	"context"
	"fmt"
	"net/url"

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
  endpoints.id,
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
		var URL string
		err := rows.Scan(&e.Id, &URL, &e.ThrottlrPath, &e.Bucket.Max, &e.Bucket.Interval)
		if err != nil {
			return nil, fmt.Errorf("failed to scan endpoint: %w", err)
		}
		parsedURL, err := url.Parse(URL)
		if err != nil {
			return nil, fmt.Errorf("failed to parse URL: %w", err)
		}
		e.OriginalUrl = parsedURL
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
	err = tx.QueryRow(ctx, "INSERT INTO endpoints (user_id, bucket_id, original_url, throttlr_url) VALUES ($1, $2, $3, $4) Returning id", userId, bucketId, e.OriginalUrl.String(), e.ThrottlrPath).Scan(&id)
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
	endpoints.id,
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
	var URL string
	err := es.db.QueryRow(ctx, query, endpoint.ThrottlrPath, userId).Scan(&endpoint.Id, &endpoint.ThrottlrPath, &URL, &endpoint.Bucket.Max, &endpoint.Bucket.Interval)
	if err != nil {
		return err
	}
	parsedURL, err := url.Parse(URL)
	if err != nil {
		return err
	}

	endpoint.OriginalUrl = parsedURL

	return nil
}

func (es *EndpointStore) Get(ctx context.Context, id int, userId string) (*models.Endpoint, error) {
	query := `
SELECT
	endpoints.id,
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
	endpoints.id = $1
	and user_id = $2
`
	e := &models.Endpoint{Bucket: &models.Bucket{}}
	var URL string
	err := es.db.QueryRow(ctx, query, id, userId).Scan(&e.Id, &e.ThrottlrPath, &URL, &e.Bucket.Max, &e.Bucket.Interval)
	if err != nil {
		return nil, err
	}
	parsedURL, err := url.Parse(URL)
	if err != nil {
		return nil, err
	}

	e.OriginalUrl = parsedURL

	return e, nil
}

func (es *EndpointStore) Delete(ctx context.Context, endpoint *models.Endpoint, userId string) error {
	tx, err := es.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()
	_, err = tx.Exec(ctx, "DELETE FROM endpoints WHERE id = $1 and user_id = $2", endpoint.Id, userId)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (es *EndpointStore) Update(ctx context.Context, endpoint *models.Endpoint, userId string) error {
	tx, err := es.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()
	exists, err := es.ExistsByOriginal(ctx, endpoint, userId)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("endpoint already exists")
	}
	_, err = tx.Exec(ctx, "UPDATE endpoints SET original_url = $1 WHERE id = $2 and user_id = $3", endpoint.OriginalUrl, endpoint.Id, userId)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, "UPDATE buckets SET max = $1, interval = $2 WHERE id = (SELECT bucket_id FROM endpoints WHERE id = $3 and user_id = $4)", endpoint.Bucket.Max, endpoint.Bucket.Interval, endpoint.Id, userId)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}
