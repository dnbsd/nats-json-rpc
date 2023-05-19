package natsjsonrpc

import (
	"context"
	"encoding/json"
	"fmt"
	natsrpc "github.com/dnbsd/nats-rpc"
	"github.com/nats-io/nats.go"
	"strings"
)

var _ natsrpc.Service = &service{}

type Service struct {
	server  *natsrpc.Server
	service *service
	subject string
	opts    Options
}

func New(nc *nats.Conn, subject string, opts ...Option) *Service {
	var o Options
	for _, opt := range opts {
		opt(&o)
	}

	return &Service{
		server:  natsrpc.NewServer(nc),
		service: newService(),
		subject: subject,
		opts:    o,
	}
}

func (s *Service) Start(ctx context.Context) error {
	err := s.opts.Validate()
	if err != nil {
		return err
	}

	s.server.Register(s.subject, s.service)
	return s.server.StartWithContext(ctx)
}

func (s *Service) Bind(rs ...natsrpc.Receiver) {
	s.service.Bind(rs...)
}

type service struct {
	receivers *natsrpc.Mapper
}

func newService() *service {
	return &service{
		receivers: natsrpc.NewMapper(),
	}
}

func (s *service) Start(ctx context.Context, reqCh <-chan *nats.Msg, respCh chan<- *nats.Msg) error {
	sendErrorMessage := func(subject string, reqID uint64, err error) error {
		msg, err := s.newErrorMessage(subject, reqID, err)
		if err != nil {
			return err
		}

		select {
		case respCh <- msg:
		case <-ctx.Done():
		}

		return nil
	}

	sendMessage := func(subject string, reqID uint64, result any) error {
		msg, err := s.newMessage(subject, reqID, result)
		if err != nil {
			return err
		}

		select {
		case respCh <- msg:
		case <-ctx.Done():
		}

		return nil
	}

	for {
		select {
		case msg := <-reqCh:
			var req request
			err := json.Unmarshal(msg.Data, &req)
			if err != nil {
				_ = sendErrorMessage(msg.Reply, 0, err)
				continue
			}

			err = req.Validate()
			if err != nil {
				_ = sendErrorMessage(msg.Reply, 0, err)
				continue
			}

			receiver, method := s.splitMethodName(req.Method)

			if !s.receivers.IsDefined(receiver, method) {
				err = fmt.Errorf("method %s is not defined", req.Method)
				_ = sendErrorMessage(msg.Reply, req.ID, err)
				continue
			}

			params, _ := s.receivers.Params(receiver, method)

			err = json.Unmarshal(req.Params, params.Interface())
			if err != nil {
				_ = sendErrorMessage(msg.Reply, req.ID, err)
				continue
			}

			result, err := s.receivers.Call(receiver, method, params.Elem())
			if err != nil {
				_ = sendErrorMessage(msg.Reply, req.ID, err)
				continue
			}

			if req.IsNotification() {
				continue
			}

			err = sendMessage(msg.Reply, req.ID, result.Interface())
			if err != nil {
				_ = sendErrorMessage(msg.Reply, req.ID, err)
				continue
			}

		case <-ctx.Done():
			return nil
		}
	}
}

func (s *service) Bind(rs ...natsrpc.Receiver) {
	for i := range rs {
		s.receivers.Register(rs[i].Name, rs[i].R)
	}
}

func (s *service) newErrorMessage(subject string, reqID uint64, err error) (*nats.Msg, error) {
	resp := newErrorResponse(reqID, err)
	b, err := json.Marshal(resp)
	if err != nil {
		return nil, err
	}

	return &nats.Msg{
		Subject: subject,
		Data:    b,
	}, nil
}

func (s *service) newMessage(subject string, reqID uint64, result any) (*nats.Msg, error) {
	resp := newResponse(reqID, result)
	b, err := json.Marshal(resp)
	if err != nil {
		return nil, err
	}

	return &nats.Msg{
		Subject: subject,
		Data:    b,
	}, nil
}

func (s *service) splitMethodName(name string) (receiver string, method string) {
	tokens := strings.SplitN(name, ".", 2)
	if len(tokens) == 1 {
		method = tokens[0]
		return
	}
	receiver = tokens[0]
	method = tokens[1]
	return
}
