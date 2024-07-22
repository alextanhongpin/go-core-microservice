package rate

import (
	"math"
	"sync"
	"time"

	"golang.org/x/exp/constraints"
)

func NewRate(period time.Duration) *Rate {
	return &Rate{
		period: period,
		Now:    time.Now,
	}
}

type Rate struct {
	mu     sync.Mutex
	count  float64
	last   time.Time
	period time.Duration
	Now    func() time.Time
}

func (r *Rate) Reset() {
	r.mu.Lock()
	r.reset()
	r.mu.Unlock()
}

func (r *Rate) reset() {
	r.count = 0
	r.last = time.Time{}
}

func (r *Rate) Inc(n int64) int64 {
	r.mu.Lock()
	f := r.inc(n)
	r.mu.Unlock()
	return int64(math.Ceil(f))
}

func (r *Rate) inc(n int64) float64 {
	f := dampFactor(r.Now().Sub(r.last), r.period)
	r.count *= f
	r.count += float64(n)
	r.last = time.Now()
	return r.count
}

func dampFactor(elapsed time.Duration, period time.Duration) float64 {
	f := float64(period-elapsed) / float64(period)
	return clip(0.0, 1.0, f)
}

type Number interface {
	constraints.Integer | constraints.Float
}

func clip[T Number](lo, hi, v T) T {
	return min(hi, max(lo, v))
}
