package archiver

// Op represents an Operation that archiver can execute.
type Op struct {
	verbose           bool
	directoryToIgnore string
}

// OpOption configures archiver operations.
type OpOption func(*Op)

// WithVerbose configures verbose mode.
func WithVerbose() OpOption {
	return func(op *Op) { op.verbose = true }
}

// WithDirectoryToIgnore add directory to ignore.
func WithDirectoryToIgnore(dir string) OpOption {
	return func(op *Op) { op.directoryToIgnore = dir }
}

func (op *Op) applyOpts(opts []OpOption) {
	for _, opt := range opts {
		opt(op)
	}
}
