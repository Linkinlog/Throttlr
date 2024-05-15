package internal

import (
	"errors"
	"os"
)

var (
	DefaultEnv = NewEnv()
	NotFound   = errors.New("key not found")
)

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
