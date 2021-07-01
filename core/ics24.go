package core

import (
	"fmt"
	"strings"

	host "github.com/cosmos/ibc-go/modules/core/24-host"
)

// Vclient validates the client identifier in the path
func (pe *PathEnd) Vclient() error {
	return host.ClientIdentifierValidator(pe.ClientID)
}

// Vconn validates the connection identifier in the path
func (pe *PathEnd) Vconn() error {
	return host.ConnectionIdentifierValidator(pe.ConnectionID)
}

// Vchan validates the channel identifier in the path
func (pe *PathEnd) Vchan() error {
	return host.ChannelIdentifierValidator(pe.ChannelID)
}

// Vport validates the port identifier in the path
func (pe *PathEnd) Vport() error {
	return host.PortIdentifierValidator(pe.PortID)
}

// Vversion validates the version identifier in the path
func (pe *PathEnd) Vversion() error {
	// TODO: version validation
	return nil
}

func (pe PathEnd) String() string {
	return fmt.Sprintf("%s:cl(%s):co(%s):ch(%s):pt(%s)", pe.ChainID, pe.ClientID, pe.ConnectionID, pe.ChannelID, pe.PortID)
}

// Validate returns errors about invalid identifiers as well as
// unset path variables for the appropriate type
func (pe *PathEnd) Validate() error {
	if err := pe.Vclient(); err != nil {
		return err
	}
	if err := pe.Vconn(); err != nil {
		return err
	}
	if err := pe.Vchan(); err != nil {
		return err
	}
	if err := pe.Vport(); err != nil {
		return err
	}
	if !(strings.ToUpper(pe.Order) == "ORDERED" || strings.ToUpper(pe.Order) == "UNORDERED") {
		return fmt.Errorf("channel must be either 'ORDERED' or 'UNORDERED' is '%s'", pe.Order)
	}
	return nil
}
