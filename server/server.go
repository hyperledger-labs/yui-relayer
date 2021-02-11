package server

import (
	"context"
	"net/http"
	"strings"

	grpctypes "github.com/cosmos/cosmos-sdk/types/grpc"
	"github.com/datachainlab/relayer/config"
	"github.com/datachainlab/relayer/core"
	"github.com/gogo/gateway"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
)

type APIServer struct {
	cliCtx *config.Context
	mux    *runtime.ServeMux
}

var _ QueryServer = (*APIServer)(nil)

func NewAPIServer(cliCtx *config.Context) APIServer {
	ctx := context.Background()
	mux := buildMux()
	srv := APIServer{cliCtx: cliCtx, mux: mux}
	if err := RegisterQueryHandlerServer(ctx, mux, srv); err != nil {
		panic(err)
	}
	return srv
}

func (m APIServer) Start(listenAddress string) error {
	return http.ListenAndServe(listenAddress, m.mux)
}

// CustomGRPCHeaderMatcher for mapping request headers to
// GRPC metadata.
// HTTP headers that start with 'Grpc-Metadata-' are automatically mapped to
// gRPC metadata after removing prefix 'Grpc-Metadata-'. We can use this
// CustomGRPCHeaderMatcher if headers don't start with `Grpc-Metadata-`
func CustomGRPCHeaderMatcher(key string) (string, bool) {
	switch strings.ToLower(key) {
	case grpctypes.GRPCBlockHeightHeader:
		return grpctypes.GRPCBlockHeightHeader, true
	default:
		return runtime.DefaultHeaderMatcher(key)
	}
}

func buildMux() *runtime.ServeMux {
	ec := core.MakeEncodingConfig()

	// The default JSON marshaller used by the gRPC-Gateway is unable to marshal non-nullable non-scalar fields.
	// Using the gogo/gateway package with the gRPC-Gateway WithMarshaler option fixes the scalar field marshalling issue.
	marshalerOption := &gateway.JSONPb{
		EmitDefaults: true,
		Indent:       "  ",
		OrigName:     true,
		AnyResolver:  ec.InterfaceRegistry,
	}
	return runtime.NewServeMux(
		// Custom marshaler option is required for gogo proto
		runtime.WithMarshalerOption(runtime.MIMEWildcard, marshalerOption),

		// This is necessary to get error details properly
		// marshalled in unary requests.
		runtime.WithProtoErrorHandler(runtime.DefaultHTTPProtoErrorHandler),

		// Custom header matcher for mapping request headers to
		// GRPC metadata
		runtime.WithIncomingHeaderMatcher(CustomGRPCHeaderMatcher),
	)
}
