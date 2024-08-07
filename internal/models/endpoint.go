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

func NewEndpoint(originalUrl string) *Endpoint {
	return &Endpoint{
		OriginalUrl:  originalUrl,
		ThrottlrPath: GeneratePath(),
	}
}

type Endpoint struct {
	OriginalUrl  string
	ThrottlrPath string
}

func (e *Endpoint) String() string {
	return fmt.Sprintf("%s %s", e.OriginalUrl, e.ThrottlrPath)
}
