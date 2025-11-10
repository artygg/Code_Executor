package domain

import "time"

func intPtr(v int) *int {
	val := v
	return &val
}

func timePtr(t time.Time) *time.Time {
	tt := t.UTC()
	return &tt
}
