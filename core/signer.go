package core

import (
	"crypto/ecdsa"

	"github.com/cosmos/gogoproto/proto"
)

type SignerConfig interface {
	proto.Message
	Build() (Signer, error)
	Validate() error
}

type Signer interface {
	Sign(digest []byte) (signature []byte, err error)
	GetPublicKey() (ecdsa.PublicKey, error)
}
