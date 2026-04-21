package mcp

type Tool struct {
	Name        string
	Description string
	InputSchema map[string]any
}

type rpcRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      any    `json:"id,omitempty"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

type rpcResponse struct {
	Result map[string]any `json:"result"`
	Error  *rpcError      `json:"error,omitempty"`
}

type rpcError struct {
	Message string `json:"message"`
}
