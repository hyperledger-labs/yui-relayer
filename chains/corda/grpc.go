package corda

import (
	transfertypes "github.com/cosmos/ibc-go/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/modules/core/02-client/types"
	conntypes "github.com/cosmos/ibc-go/modules/core/03-connection/types"
	chantypes "github.com/cosmos/ibc-go/modules/core/04-channel/types"
	cordatypes "github.com/datachainlab/corda-ibc/go/x/ibc/light-clients/xx-corda/types"
	"google.golang.org/grpc"
)

type cordaIbcClient struct {
	conn *grpc.ClientConn

	hostAndBank cordatypes.HostAndBankServiceClient

	clientQuery   clienttypes.QueryClient
	connQuery     conntypes.QueryClient
	chanQuery     chantypes.QueryClient
	transferQuery transfertypes.QueryClient

	clientTx   clienttypes.MsgClient
	connTx     conntypes.MsgClient
	chanTx     chantypes.MsgClient
	transferTx transfertypes.MsgClient
}

func createCordaIbcClient(addr string) (*cordaIbcClient, error) {
	conn, err := grpc.Dial(addr, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		return nil, err
	}

	return &cordaIbcClient{
		conn: conn,

		hostAndBank: cordatypes.NewHostAndBankServiceClient(conn),

		clientQuery:   clienttypes.NewQueryClient(conn),
		connQuery:     conntypes.NewQueryClient(conn),
		chanQuery:     chantypes.NewQueryClient(conn),
		transferQuery: transfertypes.NewQueryClient(conn),

		clientTx:   clienttypes.NewMsgClient(conn),
		connTx:     conntypes.NewMsgClient(conn),
		chanTx:     chantypes.NewMsgClient(conn),
		transferTx: transfertypes.NewMsgClient(conn),
	}, nil
}

func (gc *cordaIbcClient) shutdown() error {
	return gc.conn.Close()
}
