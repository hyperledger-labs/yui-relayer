package server

import (
	"context"
	"fmt"
)

func (srv APIServer) QueryPacket(ctx context.Context, req *QueryPacketRequest) (*QueryPacketResponse, error) {
	chains, _, _, err := srv.cliCtx.Config.ChainsFromPath(req.Path)
	if err != nil {
		return nil, err
	}
	chain, ok := chains[req.Chain]
	if !ok {
		return nil, fmt.Errorf("chain '%v' not found", req.Chain)
	}
	p, err := chain.QueryPacket(0, req.Sequence)
	if err != nil {
		return nil, err
	}
	return &QueryPacketResponse{Packet: p}, nil
}
