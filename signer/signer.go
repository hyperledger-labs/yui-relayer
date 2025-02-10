package signer

import (
	"context"

	"github.com/cosmos/gogoproto/proto"
)

type SignerConfig interface {
	proto.Message
	Build() (Signer, error)
	Validate() error
}

type Signer interface {
	Sign(ctx context.Context, digest []byte) (signature []byte, err error)
	GetPublicKey(ctx context.Context) ([]byte, error)
}
