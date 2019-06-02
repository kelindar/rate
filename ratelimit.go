// Simple, thread-safe Go rate-limiter. This is a fork of https://github.com/bsm/ratelimit
// Inspired by Antti Huima's algorithm on http://stackoverflow.com/a/668327
// Changes Copyright (c) 2019 Misakai Limited
// Original Copyright (c) 2017 Black Square Media

package ratelimit

import (
	"sync/atomic"
	"time"
)

// Limiter instances are thread-safe.
type Limiter struct {
	rate, allowance, max, unit, lastCheck uint64
}

// New creates a new rate limiter instance
func New(rate int, per time.Duration) *Limiter {
	nano := uint64(per)
	if nano < 1 {
		nano = uint64(time.Second)
	}
	if rate < 1 {
		rate = 1
	}

	return &Limiter{
		rate:      uint64(rate),        // store the rate
		allowance: uint64(rate) * nano, // set our allowance to max in the beginning
		max:       uint64(rate) * nano, // remember our maximum allowance
		unit:      nano,                // remember our unit size
		lastCheck: unixNano(),
	}
}

// UpdateRate allows to update the allowed rate
func (rl *Limiter) UpdateRate(rate int) {
	atomic.StoreUint64(&rl.rate, uint64(rate))
	atomic.StoreUint64(&rl.max, uint64(rate)*rl.unit)
}

// Limit returns true if rate was exceeded
func (rl *Limiter) Limit() bool {
	// Calculate the number of ns that have passed since our last call
	now := unixNano()
	passed := now - atomic.SwapUint64(&rl.lastCheck, now)

	// Add them to our allowance
	rate := atomic.LoadUint64(&rl.rate)
	current := atomic.AddUint64(&rl.allowance, passed*rate)

	// Ensure our allowance is not over maximum
	if max := atomic.LoadUint64(&rl.max); current > max {
		atomic.AddUint64(&rl.allowance, max-current)
		current = max
	}

	// If our allowance is less than one unit, rate-limit!
	if current < rl.unit {
		return true
	}

	// Not limited, subtract a unit
	atomic.AddUint64(&rl.allowance, -rl.unit)
	return false
}

// Undo reverts the last Limit() call, returning consumed allowance
func (rl *Limiter) Undo() {
	current := atomic.AddUint64(&rl.allowance, rl.unit)

	// Ensure our allowance is not over maximum
	if max := atomic.LoadUint64(&rl.max); current > max {
		atomic.AddUint64(&rl.allowance, max-current)
	}
}

// now as unix nanoseconds
func unixNano() uint64 {
	return uint64(time.Now().UnixNano())
}
