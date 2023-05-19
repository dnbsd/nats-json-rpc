package natsjsonrpc

import (
	"fmt"
	natsrpc "github.com/dnbsd/nats-rpc"
)

type Option func(*Options)

type Options struct {
	receivers []natsrpc.Receiver
}

func (o Options) Validate() error {
	if len(o.receivers) == 0 {
		return fmt.Errorf("at least one receiver must be assigned")
	}
	return nil
}

func WithReceiver(name string, r any) Option {
	return func(opts *Options) {
		opts.receivers = append(opts.receivers, natsrpc.Receiver{
			Name: name,
			R:    r,
		})
	}
}
