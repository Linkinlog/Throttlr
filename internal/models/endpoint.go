package models

import (
	"crypto/rand"
	"errors"
	"fmt"
	"net/url"
)

var InvalidURL = errors.New("invalid URL")

func GeneratePath() string {
	bytes := make([]byte, 10)
	if _, err := rand.Read(bytes); err != nil {
		panic(err)
	}

	return fmt.Sprintf("%X", bytes)
}

func NewEndpoint(originalUrl string, b *Bucket) (*Endpoint, error) {
	url, err := url.Parse(originalUrl)
	if err != nil {
		return nil, err
	}

	if url.Scheme == "" || url.Host == "" {
		return nil, InvalidURL
	}

	return &Endpoint{
		OriginalUrl:  url,
		ThrottlrPath: GeneratePath(),
		Bucket:       b,
	}, nil
}

type Endpoint struct {
	Id           int
	OriginalUrl  *url.URL
	ThrottlrPath string
	Bucket       *Bucket
}

func (e *Endpoint) String() string {
	return fmt.Sprintf("%s %s", e.OriginalUrl.String(), e.ThrottlrPath)
}
