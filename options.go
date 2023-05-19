package natsjsonrpc

type Option func(*Options)

type Options struct{}

func (o Options) Validate() error {
	return nil
}
