package transport

import (
	"github.com/richard-senior/mcp/pkg/protocol"
)

// Transport defines the interface for communication methods
type Transport interface {
	ReadRequest() (*protocol.JsonRpcRequest, error)
	WriteResponse(*protocol.JsonRpcResponse) error
}
