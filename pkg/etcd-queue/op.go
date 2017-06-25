package etcdqueue

import (
	"time"
)

// Op represents an operation that queue can execute.
type Op struct {
	ttl int64
}

// OpOption configures queue operations.
type OpOption func(*Op)

// WithTTL configures TTL.
func WithTTL(dur time.Duration) OpOption {
	return func(op *Op) { op.ttl = int64(dur.Seconds()) }
}

func (op *Op) applyOpts(opts []OpOption) {
	for _, opt := range opts {
		opt(op)
	}
}
