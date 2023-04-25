package natsjsonrpc

import "fmt"

type Option func(*Options)

type Options struct {
	subject string
}

func (o Options) Validate() error {
	if o.subject == "" {
		return fmt.Errorf("NATS subject not specified")
	}
	return nil
}

func WithSubject(subject string) Option {
	return func(opts *Options) {
		opts.subject = subject
	}
}
