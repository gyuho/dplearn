package archiver

// Op represents an Operation that archiver can execute.
type Op struct {
	verbose bool
}

// OpOption configures archiver operations.
type OpOption func(*Op)

// WithVerbose configures verbose mode.
func WithVerbose() OpOption {
	return func(op *Op) { op.verbose = true }
}

func (op *Op) applyOpts(opts []OpOption) {
	for _, opt := range opts {
		opt(op)
	}
}
