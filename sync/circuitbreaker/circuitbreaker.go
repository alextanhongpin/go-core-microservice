// Package circuitbreaker is an in-memory implementation of circuit breaker.
// The idea is that each local node (server) should maintain it's own knowledge
// of the service availability, instead of depending on external infrastructure
// like distributed cache.

package circuitbreaker

import (
	"errors"
	"sync"
	"time"
)

const (
	breakDuration    = 5 * time.Second
	successThreshold = 5                // min 5 successThreshold before the circuit breaker becomes closed.
	failureThreshold = 10               // min 10 failures before the circuit breaker becomes open.
	failureRatio     = 0.9              // at least 90% of the requests fails.
	samplingDuration = 10 * time.Second // time window to measure the error rate.
)

// errorHandler returns a unit usage.
var errorHandler = func(err error) int64 {
	if err != nil {
		return 1
	}

	return 0
}

// Unavailable returns the error when the circuit breaker is not available.
var Unavailable = errors.New("circuit-breaker: unavailable")

// store implements a remote store to save the circuitbreaker state.
type store interface {
	Get() (Status, bool)
	Set(status Status)
}

type Option struct {
	// States.
	total   int64
	count   int64
	closeAt time.Time

	// Options.
	SuccessThreshold int64
	FailureThreshold int64
	BreakDuration    time.Duration
	Now              func() time.Time
	ErrorHandler     func(error) int64
	FailureRatio     float64
	SamplingDuration time.Duration
	Store            store
	OnStateChanged   func(from, to Status)
}

func NewOption() *Option {
	return &Option{
		SuccessThreshold: successThreshold,
		FailureThreshold: failureThreshold,
		BreakDuration:    breakDuration,
		Now:              time.Now,
		ErrorHandler:     errorHandler,
		FailureRatio:     failureRatio,
		SamplingDuration: samplingDuration,
		OnStateChanged:   func(from, to Status) {},
	}
}

// CircuitBreaker represents the circuit breaker.
type CircuitBreaker struct {
	mu             sync.RWMutex
	opt            *Option
	states         [3]state
	state          Status
	onStateChanged func(from, to Status)
}

func New(opt *Option) *CircuitBreaker {
	if opt == nil {
		opt = NewOption()
	}

	return &CircuitBreaker{
		opt: opt,
		states: [3]state{
			NewClosedState(opt),
			NewOpenState(opt),
			NewHalfOpenState(opt),
		},
		onStateChanged: opt.OnStateChanged,
	}
}

func (cb *CircuitBreaker) ResetIn() time.Duration {
	cb.mu.RLock()
	status := cb.state
	closeAt := cb.opt.closeAt
	cb.mu.RUnlock()

	if status.IsOpen() {
		delta := closeAt.Sub(cb.opt.Now())
		if delta > 0 {
			return delta
		}

		return 0
	}

	return 0

}

func (cb *CircuitBreaker) Status() Status {
	cb.mu.RLock()
	status := cb.state
	cb.mu.RUnlock()

	return status
}

func (cb *CircuitBreaker) Do(fn func() error) error {
	// Checks the remote store for the distributed state.
	// Failure in getting the remote state should not stop the circuitbreaker.
	storeState, ok := cb.storeState()

	cb.mu.Lock()
	defer cb.mu.Unlock()

	// If the local state is different from the remote state, sync them.
	if ok && storeState != cb.state {
		cb.onStateChanged(cb.state, storeState)
		cb.state = storeState
		cb.states[cb.state].Entry()

		return cb.states[cb.state].Do(fn)
	}

	state, ok := cb.states[cb.state].Next()
	if ok {
		// If the local state has changed, update the remote state.
		// Failure in updating the state should not stop the circuitbreaker.
		// Skip half-open, because it is just an intermediary state.
		if !state.IsHalfOpen() {
			cb.setStoreState(state)
		}
		cb.onStateChanged(cb.state, state)
		cb.state = state
		cb.states[cb.state].Entry()
	}

	return cb.states[cb.state].Do(fn)
}

func (cb *CircuitBreaker) storeState() (Status, bool) {
	if cb.opt.Store != nil {
		return cb.opt.Store.Get()
	}

	return 0, false
}

func (cb *CircuitBreaker) setStoreState(status Status) {
	if cb.opt.Store != nil {
		cb.opt.Store.Set(status)
	}
}

type state interface {
	Next() (Status, bool)
	Entry()
	Do(func() error) error
}

type ClosedState struct {
	opt     *Option
	resetAt time.Time
}

func NewClosedState(opt *Option) *ClosedState {
	return &ClosedState{opt: opt}
}

func (c *ClosedState) Entry() {
	now := c.opt.Now()
	c.resetAt = now.Add(c.opt.SamplingDuration)
	c.resetFailureCounter()
}

func (c *ClosedState) Next() (Status, bool) {
	return Open, c.isFailureThresholdReached()
}

func (c *ClosedState) Do(fn func() error) error {
	err := fn()
	c.incrementFailureCounter(err)

	return err
}

func (c *ClosedState) resetFailureCounter() {
	c.opt.count = 0
	c.opt.total = 0
}

func (c *ClosedState) isFailureThresholdReached() bool {
	o := c.opt

	// The state transition is only valid if the failures
	// count and error rate exceeds the threshold within the
	// error time window.
	//
	// now >= resetAt
	now := o.Now()
	if !now.Before(c.resetAt) {
		c.resetAt = now.Add(c.opt.SamplingDuration)
		c.resetFailureCounter()
	}

	return o.count > o.FailureThreshold && Ratio(o.count, o.total) >= o.FailureRatio
}

func (c *ClosedState) incrementFailureCounter(err error) {
	o := c.opt

	now := o.Now()
	if !now.Before(c.resetAt) {
		c.resetAt = now.Add(c.opt.SamplingDuration)
		c.resetFailureCounter()
	}

	o.total++
	o.count += o.ErrorHandler(err)
}

type OpenState struct {
	opt *Option
}

func NewOpenState(opt *Option) *OpenState {
	return &OpenState{opt}
}

func (s *OpenState) Entry() {
	s.startTimeoutTimer()
}

func (s *OpenState) Next() (Status, bool) {
	return HalfOpen, s.isTimeoutTimerExpired()
}

func (s *OpenState) Do(fn func() error) error {
	return Unavailable
}

func (s *OpenState) startTimeoutTimer() {
	s.opt.closeAt = s.opt.Now().Add(s.opt.BreakDuration)
}

func (s *OpenState) isTimeoutTimerExpired() bool {
	return !s.opt.Now().Before(s.opt.closeAt)
}

type HalfOpenState struct {
	opt    *Option
	failed bool
}

func NewHalfOpenState(opt *Option) *HalfOpenState {
	return &HalfOpenState{opt: opt}
}

func (s *HalfOpenState) Entry() {
	s.resetSuccessCounter()
}

func (s *HalfOpenState) Next() (Status, bool) {
	if s.isOperationFailed() {
		return Open, true
	}

	return Closed, s.isSuccessCountThresholdExceeded()
}

func (s *HalfOpenState) Do(fn func() error) error {
	err := fn()

	s.failed = s.opt.ErrorHandler(err) > 0
	if !s.failed {
		s.incrementSuccessCounter()
	}

	return err
}

func (s *HalfOpenState) resetSuccessCounter() {
	s.opt.count = 0
}

func (s *HalfOpenState) isOperationFailed() bool {
	return s.failed
}

func (s *HalfOpenState) isSuccessCountThresholdExceeded() bool {
	return s.opt.count > s.opt.SuccessThreshold
}

func (s *HalfOpenState) incrementSuccessCounter() {
	s.opt.count++
}

func Ratio(n, total int64) float64 {
	return float64(n) / float64(total)
}
