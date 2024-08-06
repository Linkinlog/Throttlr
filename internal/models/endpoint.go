package models

import (
	"crypto/rand"
	"fmt"
)

func GeneratePath() string {
	bytes := make([]byte, 10)
	if _, err := rand.Read(bytes); err != nil {
		panic(err)
	}

	return fmt.Sprintf("%X", bytes)
}

func NewEndpoint(apiKey, originalUrl string) *Endpoint {
	return &Endpoint{
		ApiKey:       apiKey,
		OriginalUrl:  originalUrl,
		ThrottlrPath: GeneratePath(),
	}
}

type Endpoint struct {
	ApiKey       string
	OriginalUrl  string
	ThrottlrPath string
}
