package core

import (
	"fmt"
	"log"
	"time"

	chantypes "github.com/cosmos/cosmos-sdk/x/ibc/core/04-channel/types"
)

// CreateChannel runs the channel creation messages on timeout until they pass
// TODO: add max retries or something to this function
func CreateChannel(src, dst ChainI, ordered bool, to time.Duration) error {
	var order chantypes.Order
	if ordered {
		order = chantypes.ORDERED
	} else {
		order = chantypes.UNORDERED
	}

	ticker := time.NewTicker(to)
	failures := 0
	for ; true; <-ticker.C {
		chanSteps, err := createChannelStep(src, dst, order)
		if err != nil {
			return err
		}

		if !chanSteps.Ready() {
			break
		}

		chanSteps.Send(src, dst)

		switch {
		// In the case of success and this being the last transaction
		// debug logging, log created connection and break
		case chanSteps.Success() && chanSteps.Last:
			// srch, dsth, err := GetLatestLightHeights(src, dst)
			// if err != nil {
			// 	return err
			// }
			// srcChan, dstChan, err := QueryChannelPair(src, dst, srch, dsth)
			// if err != nil {
			// 	return err
			// }
			// if c.debug {
			// 	logChannelStates(c, dst, srcChan, dstChan)
			// }
			log.Println(fmt.Sprintf("★ Channel created: [%s]chan{%s}port{%s} -> [%s]chan{%s}port{%s}",
				src.ChainID(), src.Path().ChannelID, src.Path().PortID,
				dst.ChainID(), dst.Path().ChannelID, dst.Path().PortID))
			return nil
		// In the case of success, reset the failures counter
		case chanSteps.Success():
			failures = 0
			continue
		// In the case of failure, increment the failures counter and exit if this is the 3rd failure
		case !chanSteps.Success():
			failures++
			log.Println(fmt.Sprintf("retrying transaction..."))
			time.Sleep(5 * time.Second)
			if failures > 2 {
				return fmt.Errorf("! Channel failed: [%s]chan{%s}port{%s} -> [%s]chan{%s}port{%s}",
					src.ChainID(), src.Path().ClientID, src.Path().ChannelID,
					dst.ChainID(), dst.Path().ClientID, dst.Path().ChannelID)
			}
		}
	}

	return nil
}

func createChannelStep(src, dst ChainI, ordering chantypes.Order) (*RelayMsgs, error) {
	panic("not implemented error")
}
