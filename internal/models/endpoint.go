package models

import (
	"crypto/rand"
	"fmt"
)

func NewEndpoint(apiKey, originalUrl string) *Endpoint {
	bytes := make([]byte, 10)
	if _, err := rand.Read(bytes); err != nil {
		return nil
	}

	throttlrPath := fmt.Sprintf("%X", bytes)

	return &Endpoint{
		ApiKey:      apiKey,
		OriginalUrl: originalUrl,
		ThrottlrPath: throttlrPath,
	}
}

type Endpoint struct {
	ApiKey      string
	OriginalUrl string
	ThrottlrPath string
}
