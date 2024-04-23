package secrets

import (
	"errors"
	"os"
)

func NewDev() *Dev {
	return &Dev{}
}

type Dev struct{}

var NotFound = errors.New("key not found")

func (d *Dev) Get(key string) (string, error) {
	str, found := os.LookupEnv(key)
	if !found {
		return "", NotFound
	}
	return str, nil
}
