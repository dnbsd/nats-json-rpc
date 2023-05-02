package natsjsonrpc

import (
	"context"
	"fmt"
	natsrpc "github.com/dnbsd/nats-rpc"
	natsserver "github.com/nats-io/nats-server/v2/test"
	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

type echo struct{}

type EchoParams struct {
	Message string `json:"message"`
}

type EchoResult struct {
	Message string `json:"message"`
}

func (s *echo) Echo(params *EchoParams) (*EchoResult, error) {
	println("echo:", params.Message)
	return &EchoResult{
		Message: params.Message,
	}, nil
}

func newNatsServerAndConnection(t *testing.T) *nats.Conn {
	opts := natsserver.DefaultTestOptions
	opts.NoLog = false
	opts.Port = 14444
	s := natsserver.RunServer(&opts)
	uri := fmt.Sprintf("nats://%s:%d", opts.Host, opts.Port)
	nc, err := nats.Connect(uri)
	assert.NoError(t, err)
	t.Cleanup(func() {
		nc.Close()
		s.Shutdown()
	})
	return nc
}

func TestService_Start(t *testing.T) {
	const rpcSubject = "test_service.rpc"
	nc := newNatsServerAndConnection(t)
	s := New(nc, WithSubject(rpcSubject))
	s.Bind(natsrpc.Receiver{
		Name: "Echo",
		R:    &echo{},
	})
	go func() {
		err := s.Start(context.Background())
		assert.NoError(t, err)
	}()

	time.Sleep(1 * time.Second)

	reqMsg := &nats.Msg{
		Subject: rpcSubject,
		Reply:   nats.NewInbox(),
		Data:    []byte(`{"jsonrpc": "2.0", "method": "Echo.Echo", "params": {"message": "hello world!"}, "id": 3}`),
	}
	respMsg, err := nc.RequestMsg(reqMsg, 5*time.Second)
	assert.NoError(t, err)

	expectedData := []byte(`{"jsonrpc": "2.0", "result": {"message": "hello world!"}, "id": 3}`)
	assert.JSONEq(t, string(expectedData), string(respMsg.Data))
}
