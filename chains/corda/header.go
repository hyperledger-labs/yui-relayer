package corda

import (
	clienttypes "github.com/cosmos/cosmos-sdk/x/ibc/core/02-client/types"
	ibcexported "github.com/cosmos/cosmos-sdk/x/ibc/core/exported"
	"github.com/datachainlab/relayer/core"
)

type cordaHeader struct{}

var _ core.HeaderI = (*cordaHeader)(nil)

func (*cordaHeader) ClientType() string {
	return "corda"
}

func (*cordaHeader) GetHeight() ibcexported.Height {
	return clienttypes.Height{}
}

func (*cordaHeader) ValidateBasic() error {
	return nil
}
