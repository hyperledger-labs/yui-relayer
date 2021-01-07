package config

import "github.com/cosmos/cosmos-sdk/codec"

type Context struct {
	Marshaler codec.Marshaler
	Config    *Config
}
