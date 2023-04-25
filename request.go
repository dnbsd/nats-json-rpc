package natsjsonrpc

import (
	"encoding/json"
	"fmt"
)

const version = "2.0"

type request struct {
	Version string          `json:"jsonrpc"`
	ID      uint64          `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

func newRequest() request {
	return request{
		Version: version,
	}
}

func (r request) IsNotification() bool {
	return r.ID == 0
}

func (r request) Validate() error {
	if r.Version != version {
		return fmt.Errorf("unsupported RPC version '%s'", r.Version)
	}

	if r.Method == "" {
		return fmt.Errorf("RPC method was not specified")
	}

	return nil
}
