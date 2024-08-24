package internal

import (
	"errors"
	"os"
)

var (
	DefaultEnv = NewEnv()
	NotFound   = errors.New("key not found")
)

func ClientDB() string {
	db := "postgres://username:password@localhost:5432/database_name"
	if d, err := DefaultEnv.Get("CLIENT_DB"); err == nil {
		db = d
	}
	return db
}

func ClientCallbackURL() string {
	callbackUrl := "http://localhost:8080"
	if url, err := DefaultEnv.Get("CLIENT_CALLBACK_URL"); err == nil {
		callbackUrl = url
	}
	return callbackUrl
}

func ServerCallbackURL() string {
	callbackUrl := "http://localhost:8091"
	if url, err := DefaultEnv.Get("SERVER_CALLBACK_URL"); err == nil {
		callbackUrl = url
	}
	return callbackUrl
}

func NewEnv() *Env {
	return &Env{}
}

type Env struct{}

func (e *Env) Get(key string) (string, error) {
	str, found := os.LookupEnv(key)
	if !found {
		return "", NotFound
	}
	return str, nil
}
