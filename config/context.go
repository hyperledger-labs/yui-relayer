package config

import "github.com/cosmos/cosmos-sdk/codec"

type Context struct {
	Modules []ModuleI
	Codec   codec.ProtoCodecMarshaler
	Config  *Config
}
