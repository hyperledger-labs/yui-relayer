package relayer

type ChainI interface {
	ClientType() string
	QueryLatestHeader() (out HeaderI, err error)

	StartEventListener(dst ChainI, strategy StrategyI)
}
