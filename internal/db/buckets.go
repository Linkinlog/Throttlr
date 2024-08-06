package db

import (
	"context"
	"database/sql"

	"github.com/linkinlog/throttlr/internal/models"
)

func NewBucketStore(db *sql.DB) *BucketStore {
	return &BucketStore{db: db}
}

type BucketStore struct{ db *sql.DB }

type BucketModel struct {
	Id         int
	EndpointId int
	*models.Bucket
}

func (bs *BucketStore) Exists(ctx context.Context, bucket BucketModel) (bool, error) {
	var exists bool
	err := bs.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT id from buckets where endpoint_id = ?)", bucket.EndpointId).Scan(&exists)
	return exists, err
}

func (bs *BucketStore) Store(ctx context.Context, b BucketModel) (int, error) {
	tx, err := bs.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	var id int
	err = tx.QueryRowContext(ctx, "INSERT INTO buckets (max, interval, endpoint_id) VALUES (?, ?, ?) Returning id", b.Max, b.Interval, b.EndpointId).Scan(&id)
	if err != nil {
		return 0, err
	}

	return id, tx.Commit()
}
