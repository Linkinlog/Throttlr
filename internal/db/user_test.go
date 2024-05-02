package db_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/linkinlog/throttlr/internal/db"
	"github.com/linkinlog/throttlr/internal/models"
	"github.com/stretchr/testify/assert"
)

func testDb() *sql.DB {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		panic(err)
	}

	return db
}

func withUser(t *testing.T, db *sql.DB) *sql.DB {
	t.Helper()

	_, err := db.Exec(`CREATE TABLE users (
		id INTEGER PRIMARY KEY,
		email TEXT NOT NULL UNIQUE,
	)`)
	if err != nil {
		t.Fatal(err)
	}

	_, err = db.Exec(`INSERT INTO users (id, email) VALUES (1, 'foo@bar.com')`)
	if err != nil {
		t.Fatal(err)
	}

	_, err = db.Exec(`INSERT INTO users (id, email) VALUES (2, 'bar@foo.com')`)
	if err != nil {
		t.Fatal(err)
	}

	return db
}

func TestStore(t *testing.T) {
	t.Parallel()

	d := withUser(t, testDb())
	s := db.NewUserStore(d)
	user := models.User{}
	err := s.Store(context.Background(), user)
	assert.Nil(t, err)
}

func TestById(t *testing.T) {
	t.Parallel()

	d := withUser(t, testDb())
	s := db.NewUserStore(d)
	id := "420-gitlab"

	user, err := s.ById(context.Background(), id)

	assert.Nil(t, err)
	assert.Equal(t, models.User{
		Id:    id,
		Email: "bar@foo.com",
	}, user)
}
