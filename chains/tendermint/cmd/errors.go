package cmd

import (
	"fmt"
)

func wrapInitFailed(err error) error {
	return fmt.Errorf("init failed: %w", err)
}

func errKeyExists(name string) error {
	return fmt.Errorf("a key with name %s already exists", name)
}

func errKeyDoesntExist(name string) error {
	return fmt.Errorf("a key with name %s doesn't exist", name)
}
