package models

type Interval int

const (
	Minute Interval = iota + 1
	Hour
	Day
	Week
	Month
)

func NewBucket(endpoint *Endpoint, interval Interval, max int) *Bucket {
	return &Bucket{
		Endpoint: endpoint,
		Interval: interval,
		Max:      max,
	}
}

type Bucket struct {
	Endpoint       *Endpoint
	Interval       Interval
	Max            int
	windowOpenedAt int64
}
