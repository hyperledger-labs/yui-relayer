package config

import (
	"encoding/json"

	"github.com/datachainlab/relayer/core"
	"github.com/gogo/protobuf/proto"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/types"
)

func MarshalJSON(config Config) ([]byte, error) {
	return json.Marshal(config)
}

func UnmarshalJSON(m codec.Codec, bz []byte, config *Config) error {
	if err := json.Unmarshal(bz, config); err != nil {
		return err
	}
	for _, c := range config.Chains {
		var cfg core.ChainConfigI
		if err := UnmarshalJSONAny(m, &cfg, c); err != nil {
			return err
		}
		config.chains = append(config.chains, cfg.GetChain())
	}
	return nil
}

// MarshalJSONAny is a convenience function for packing the provided value in an
// Any and then proto marshaling it to bytes
func MarshalJSONAny(m codec.JSONCodec, msg proto.Message) ([]byte, error) {
	any, err := types.NewAnyWithValue(msg)
	if err != nil {
		return nil, err
	}
	return m.MarshalJSON(any)
}

// UnmarshalJSONAny is a convenience function for proto unmarshaling an Any from
// bz and then unpacking it to the interface pointer passed in as iface using
// the provided AnyUnpacker or returning an error
//
// Ex:
//		var x MyInterface
//		err := UnmarshalJSONAny(unpacker, &x, bz)
func UnmarshalJSONAny(m codec.Codec, iface interface{}, bz []byte) error {
	any := &types.Any{}

	err := m.UnmarshalJSON(bz, any)
	if err != nil {
		return err
	}

	return m.UnpackAny(any, iface)
}
