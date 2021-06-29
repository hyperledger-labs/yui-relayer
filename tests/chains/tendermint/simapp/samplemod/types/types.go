package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	txtypes "github.com/datachainlab/cross/x/core/tx/types"
)

// NewContractCallRequest creates a new instance of ContractCallRequest
func NewContractCallRequest(method string, args ...string) ContractCallRequest {
	return ContractCallRequest{
		Method: method,
		Args:   args,
	}
}

// ContractCallInfo converts the ContractCallRequest to a ContractCallInfo
func (r ContractCallRequest) ContractCallInfo(m codec.Marshaler) txtypes.ContractCallInfo {
	return m.MustMarshalJSON(&r)
}
