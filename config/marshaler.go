package config

import "encoding/json"

func MarshalJSON(config Config) ([]byte, error) {
	return json.Marshal(config)
}

func UnmarshalJSON(bz []byte, config *Config) error {
	return json.Unmarshal(bz, config)
}
