package config

import "github.com/cosmos/cosmos-sdk/codec"

type Context struct {
	Codec  codec.ProtoCodecMarshaler
	Config *Config
}
