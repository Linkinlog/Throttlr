package internal

import (
	"errors"
	"os"
)

var (
	DefaultEnv = NewEnv()
	NotFound   = errors.New("key not found")
)

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
