package fetcher

import (
	"time"
)

type FetcherOption func(*Fetcher)
type JobBoardName string

func WithTickerTime(duration time.Duration) FetcherOption {
	return func(f *Fetcher) {
		f.tickerTime = duration
	}
}
