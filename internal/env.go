package internal

import (
	"errors"
	"os"
)

func NewEnv() *Env {
	return &Env{}
}

type Env struct{}

var NotFound = errors.New("key not found")

func (e *Env) Get(key string) (string, error) {
	str, found := os.LookupEnv(key)
	if !found {
		return "", NotFound
	}
	return str, nil
}
