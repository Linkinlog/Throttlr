package models

import (
	"errors"
	"time"
)

var (
	ErrBucketFull = errors.New("Bucket is full")
	ErrBucketNil  = errors.New("Bucket is nil")
)

type Interval int

func (i Interval) String() string {
	switch i {
	case Minute:
		return "min"
	case Hour:
		return "hour"
	case Day:
		return "day"
	case Week:
		return "week"
	case Month:
		return "month"
	default:
		return "unknown"
	}
}

const (
	Minute Interval = iota + 1
	Hour
	Day
	Week
	Month
)

func NewBucket(interval Interval, max int) *Bucket {
	return &Bucket{
		Interval: interval,
		Max:      max,
		Current:  0,
	}
}

type Bucket struct {
	Interval       Interval
	Max            int
	Current        int
	WindowOpenedAt time.Time
}
