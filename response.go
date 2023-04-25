package natsjsonrpc

type response struct {
	Version string `json:"jsonrpc"`
	ID      uint64 `json:"id,omitempty"`
	Result  any    `json:"result,omitempty"`
	Error   any    `json:"error,omitempty"`
}

func newResponse(id uint64, result any) response {
	return response{
		Version: version,
		ID:      id,
		Result:  result,
	}
}

func newErrorResponse(id uint64, err error) response {
	return response{
		Version: version,
		ID:      id,
		Error:   err.Error(),
	}
}
