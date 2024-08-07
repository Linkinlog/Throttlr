package db

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/linkinlog/throttlr/internal/models"
)

func NewBucketStore(db *pgx.Conn) *BucketStore {
	return &BucketStore{db: db}
}

type BucketStore struct{ db *pgx.Conn }

type BucketModel struct {
	Id         int
	*models.Bucket
}

func (bs *BucketStore) Exists(ctx context.Context, bucket BucketModel) (bool, error) {
	var exists bool
	err := bs.db.QueryRow(ctx, "SELECT EXISTS(SELECT id from buckets where id = $1)", bucket.Id).Scan(&exists)
	return exists, err
}

func (bs *BucketStore) Store(ctx context.Context, b BucketModel) (int, error) {
	tx, err := bs.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return 0, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var id int
	err = tx.QueryRow(ctx, "INSERT INTO buckets (max, interval) VALUES ($1, $2) Returning id", b.Max, b.Interval).Scan(&id)
	if err != nil {
		return 0, err
	}

	return id, tx.Commit(ctx)
}
