package models

type Interval int

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
	}
}

type Bucket struct {
	Interval Interval
	Max      int
}
