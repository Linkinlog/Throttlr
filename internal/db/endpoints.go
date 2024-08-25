package db

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/linkinlog/throttlr/internal/models"
)

func NewEndpointStore(db *pgxpool.Pool, l *slog.Logger) *EndpointStore {
	return &EndpointStore{db: db, l: l}
}

type EndpointStore struct {
	db *pgxpool.Pool
	l  *slog.Logger
}

func (es *EndpointStore) AllForUser(ctx context.Context, userId string) ([]*models.Endpoint, error) {
	allQuery := `
SELECT
  endpoints.id,
  original_url,
  throttlr_url,
  max,
  interval,
  current,
  window_opened_at
FROM
  endpoints
  JOIN api_keys on api_keys.user_id = endpoints.user_id
  JOIN buckets on buckets.id = endpoints.bucket_id
where
  api_keys.valid = true
  and api_keys.user_id = $1
`
	es.l.Debug("all for user", "query", allQuery, "userId", userId)
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
		err := rows.Scan(&e.Id, &URL, &e.ThrottlrPath, &e.Bucket.Max, &e.Bucket.Interval, &e.Bucket.Current, &e.Bucket.WindowOpenedAt)
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

	existsQuery := `
SELECT
  EXISTS (
    SELECT
      throttlr_url
    FROM
      endpoints
    WHERE
      throttlr_url = $1
      and endpoints.user_id = $2
  );
`
	// not a huge fan of any of this but it works for now
	for {
		es.l.Debug("store", "query", existsQuery, "originalUrl", e.OriginalUrl, "userId", userId)
		var exists bool
		err := es.db.QueryRow(
			ctx,
			existsQuery,
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

	insertBucketQuery := `
INSERT INTO
  buckets (max, interval, current)
VALUES
  ($1, $2, $3) Returning id
`

	es.l.Debug("store", "query", insertBucketQuery, "max", e.Bucket.Max, "interval", e.Bucket.Interval, "current", e.Bucket.Current)
	var bucketId int
	err = tx.QueryRow(ctx, insertBucketQuery, e.Bucket.Max, e.Bucket.Interval, e.Bucket.Current).Scan(&bucketId)
	if err != nil {
		return 0, err
	}

	insertEndpointQuery := `
INSERT INTO
  endpoints (user_id, bucket_id, original_url, throttlr_url)
VALUES
  ($1, $2, $3, $4) Returning id
`

	es.l.Debug("store", "query", insertEndpointQuery, "userId", userId, "bucketId", bucketId, "originalUrl", e.OriginalUrl.String(), "throttlrPath", e.ThrottlrPath)
	var id int
	err = tx.QueryRow(ctx, insertEndpointQuery, userId, bucketId, e.OriginalUrl.String(), e.ThrottlrPath).Scan(&id)
	if err != nil {
		return 0, err
	}

	return id, tx.Commit(ctx)
}

func (es *EndpointStore) ExistsByOriginal(ctx context.Context, endpoint *models.Endpoint, userId string) (bool, error) {
	var exists bool
	existsByOriginalQuery := `
SELECT
    EXISTS (
        SELECT
            1
        FROM
            endpoints
        WHERE
            original_url = $1
            and user_id = $2
    );
`
	es.l.Debug("exists by original", "query", existsByOriginalQuery, "originalUrl", endpoint.OriginalUrl, "userId", userId)
	err := es.db.QueryRow(ctx, existsByOriginalQuery, endpoint.OriginalUrl, userId).Scan(&exists)
	return exists, err
}

func (es *EndpointStore) ExistsByThrottlr(ctx context.Context, endpoint *models.Endpoint, userId string) (bool, error) {
	var exists bool
	existsByThrottlrQuery := `
SELECT
    EXISTS (
        SELECT
            1
        FROM
            endpoints
        WHERE
            throttlr_url = $1
            and user_id = $2
    );
`
	es.l.Debug("exists by throttlr", "query", existsByThrottlrQuery, "throttlrPath", endpoint.ThrottlrPath, "userId", userId)
	err := es.db.QueryRow(ctx, existsByThrottlrQuery, endpoint.ThrottlrPath, userId).Scan(&exists)
	return exists, err
}

func (es *EndpointStore) Fill(ctx context.Context, endpoint *models.Endpoint, userId string) error {
	query := `
SELECT
	endpoints.id,
	throttlr_url,
	original_url,
	max,
	interval,
    current,
    window_opened_at
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
	es.l.Debug("fill", "query", query, "throttlrPath", endpoint.ThrottlrPath, "userId", userId)

	var URL string
	err := es.db.QueryRow(ctx, query, endpoint.ThrottlrPath, userId).Scan(&endpoint.Id, &endpoint.ThrottlrPath, &URL, &endpoint.Bucket.Max, &endpoint.Bucket.Interval, &endpoint.Bucket.Current, &endpoint.Bucket.WindowOpenedAt)
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

func (es *EndpointStore) Get(ctx context.Context, throttlrPath, userId string) (*models.Endpoint, error) {
	query := `
SELECT
	endpoints.id,
	throttlr_url,
	original_url,
	max,
	interval,
    current,
    window_opened_at
FROM
	endpoints
JOIN
	buckets
ON
	buckets.id = endpoints.bucket_id
WHERE
	endpoints.throttlr_url = $1
	and user_id = $2
`
	es.l.Debug("get", "query", query, "throttlrPath", throttlrPath, "userId", userId)

	e := &models.Endpoint{Bucket: &models.Bucket{}}
	var URL string
	err := es.db.QueryRow(ctx, query, throttlrPath, userId).Scan(&e.Id, &e.ThrottlrPath, &URL, &e.Bucket.Max, &e.Bucket.Interval, &e.Bucket.Current, &e.Bucket.WindowOpenedAt)
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

	deleteQuery := `
DELETE FROM
    endpoints
WHERE
    throttlr_url = $1
    and user_id = $2
`
	es.l.Debug("delete", "query", deleteQuery, "throttlrPath", endpoint.ThrottlrPath, "userId", userId)
	_, err = tx.Exec(ctx, deleteQuery, endpoint.ThrottlrPath, userId)
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

	existsQuery := `
SELECT
    EXISTS (
        SELECT
            1
        FROM
            endpoints
        WHERE
            throttlr_url <> $1
            and original_url = $2
            and user_id = $3

    );
`
	es.l.Debug("update", "query", existsQuery, "throttlrPath", endpoint.ThrottlrPath, "originalUrl", endpoint.OriginalUrl.String(), "userId", userId)
	var exists bool
	err = tx.QueryRow(ctx, existsQuery, endpoint.ThrottlrPath, endpoint.OriginalUrl.String(), userId).Scan(&exists)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("endpoint with original URL %s already exists", endpoint.OriginalUrl.String())
	}

	_, err = tx.Exec(ctx, "UPDATE endpoints SET original_url = $1 WHERE throttlr_url = $2 and user_id = $3", endpoint.OriginalUrl, endpoint.ThrottlrPath, userId)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, "UPDATE buckets SET max = $1, interval = $2, current = $3 WHERE id = (SELECT bucket_id FROM endpoints WHERE throttlr_url = $4 and user_id = $5)", endpoint.Bucket.Max, endpoint.Bucket.Interval, endpoint.Bucket.Current, endpoint.ThrottlrPath, userId)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (es *EndpointStore) UpdateBucketCount(ctx context.Context, endpoint *models.Endpoint, userId string) error {
	tx, err := es.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}

	updateBucketCountQuery := `
UPDATE buckets
SET
  current = $1
WHERE
  id = (
    SELECT
      bucket_id
    FROM
      endpoints
    WHERE
      throttlr_url = $2
      and user_id = $3
  );
`
	es.l.Debug("update bucket count", "query", updateBucketCountQuery, "current", endpoint.Bucket.Current, "throttlrPath", endpoint.ThrottlrPath, "userId", userId)
	_, err = tx.Exec(ctx, updateBucketCountQuery, endpoint.Bucket.Current, endpoint.ThrottlrPath, userId)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (es *EndpointStore) UpdateWindowOpenedAt(ctx context.Context, endpoint *models.Endpoint, userId string) error {
	tx, err := es.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}

	updateWindowOpenedAtQuery := `
UPDATE buckets
SET
  window_opened_at = $1
WHERE
  id = (
    SELECT
      bucket_id
    FROM
      endpoints
    WHERE
      throttlr_url = $2
      and user_id = $3
  );
`
	es.l.Debug("update window opened at", "query", updateWindowOpenedAtQuery, "windowOpenedAt", endpoint.Bucket.WindowOpenedAt, "throttlrPath", endpoint.ThrottlrPath, "userId", userId)
	_, err = tx.Exec(ctx, updateWindowOpenedAtQuery, endpoint.Bucket.WindowOpenedAt, endpoint.ThrottlrPath, userId)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}
