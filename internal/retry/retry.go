// Package retry содержит утилиты повторных попыток.
package retry

import (
	"context"
	"math"
	"math/rand"
	"sync"
	"time"
)

// Backoff рассчитывает экспоненциальные задержки с опциональным джиттером.
type Backoff struct {
	Base   time.Duration
	Cap    time.Duration
	Jitter bool

	mu  sync.Mutex
	rnd *rand.Rand
}

// NewBackoff создает Backoff с собственным генератором случайных чисел.
func NewBackoff(base time.Duration, capDur time.Duration, jitter bool) *Backoff {
	if capDur > 0 && base > capDur {
		base = capDur
	}
	return &Backoff{
		Base:   base,
		Cap:    capDur,
		Jitter: jitter,
		rnd:    rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// WaitDuration возвращает задержку для попытки повтора (0-базовая).
func (b *Backoff) WaitDuration(attempt int) time.Duration {
	if b == nil || b.Base <= 0 || attempt < 0 {
		return 0
	}

	maxInt := time.Duration(math.MaxInt64)
	wait := b.Base
	for i := 0; i < attempt; i++ {
		if wait > maxInt/2 {
			wait = maxInt
			break
		}
		wait *= 2
		if b.Cap > 0 && wait >= b.Cap {
			wait = b.Cap
			break
		}
	}

	if b.Cap > 0 && wait > b.Cap {
		wait = b.Cap
	}
	if !b.Jitter || wait <= 0 {
		return wait
	}

	b.mu.Lock()
	defer b.mu.Unlock()
	return time.Duration(b.rnd.Int63n(int64(wait) + 1))
}

// Policy задает правила повторов.
type Policy struct {
	MaxRetries  int
	Backoff     *Backoff
	ShouldRetry func(err error) bool
}

// Do выполняет op с повторами. onRetry вызывается после неуспешной попытки (1-базовая).
func Do(ctx context.Context, policy Policy, op func() error, onRetry func(err error, attempt int, wait time.Duration)) error {
	if policy.MaxRetries < 0 {
		policy.MaxRetries = 0
	}
	var lastErr error
	for attempt := 0; attempt <= policy.MaxRetries; attempt++ {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		lastErr = op()
		if lastErr == nil {
			return nil
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if policy.ShouldRetry != nil && !policy.ShouldRetry(lastErr) {
			return lastErr
		}
		if attempt == policy.MaxRetries {
			break
		}
		wait := time.Duration(0)
		if policy.Backoff != nil {
			wait = policy.Backoff.WaitDuration(attempt)
		}
		if onRetry != nil {
			onRetry(lastErr, attempt+1, wait)
		}
		if wait > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(wait):
			}
		}
	}
	return lastErr
}
