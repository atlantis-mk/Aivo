package domain

import "time"

func NowString(now time.Time) string {
	return now.UTC().Format(time.RFC3339Nano)
}
