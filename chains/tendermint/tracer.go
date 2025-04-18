package tendermint

import "go.opentelemetry.io/otel"

var (
	tracer = otel.Tracer("github.com/hyperledger-labs/yui-relayer/chains/tendermint")
)
