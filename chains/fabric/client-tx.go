package fabric

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/datachainlab/relayer/core"
)

func (dst *Chain) MakeMsgCreateClient(clientID string, dstHeader core.HeaderI, signer sdk.AccAddress) (sdk.Msg, error) {
	panic("not implemented error")
}
