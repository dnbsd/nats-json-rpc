package natsjsonrpc

import (
	"context"
	"encoding/json"
	"fmt"
	natsrpc "github.com/dnbsd/nats-rpc"
	"github.com/nats-io/nats.go"
	"strings"
)

var _ natsrpc.Service = &Service{}

type Service struct {
	mapper *natsrpc.Mapper
}

func New(rs ...natsrpc.Receiver) *Service {
	mapper := natsrpc.NewMapper()
	for i := range rs {
		mapper.Register(rs[i].Name, rs[i].R)
	}
	return &Service{
		mapper: mapper,
	}
}

func (s *Service) Start(ctx context.Context, reqCh <-chan *nats.Msg, respCh chan<- *nats.Msg) error {
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

			if !s.mapper.IsDefined(receiver, method) {
				err = fmt.Errorf("method %s is not defined", req.Method)
				_ = sendErrorMessage(msg.Reply, req.ID, err)
				continue
			}

			params, _ := s.mapper.Params(receiver, method)

			err = json.Unmarshal(req.Params, params.Interface())
			if err != nil {
				_ = sendErrorMessage(msg.Reply, req.ID, err)
				continue
			}

			result, err := s.mapper.Call(receiver, method, params.Elem())
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

func (s *Service) newErrorMessage(subject string, reqID uint64, err error) (*nats.Msg, error) {
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

func (s *Service) newMessage(subject string, reqID uint64, result any) (*nats.Msg, error) {
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

func (s *Service) splitMethodName(name string) (receiver string, method string) {
	tokens := strings.SplitN(name, ".", 2)
	if len(tokens) == 1 {
		method = tokens[0]
		return
	}
	receiver = tokens[0]
	method = tokens[1]
	return
}
