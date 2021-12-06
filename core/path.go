package core

import (
	"encoding/json"
	"fmt"

	"golang.org/x/sync/errgroup"

	"github.com/cosmos/cosmos-sdk/codec"
	clienttypes "github.com/cosmos/ibc-go/modules/core/02-client/types"
	conntypes "github.com/cosmos/ibc-go/modules/core/03-connection/types"
	chantypes "github.com/cosmos/ibc-go/modules/core/04-channel/types"
	"github.com/hyperledger-labs/yui-relayer/utils"
)

const (
	check = "✔"
	xIcon = "✘"
)

// Paths represent connection paths between chains
type Paths map[string]*Path

// Get returns the configuration for a given path
func (p Paths) Get(name string) (path *Path, err error) {
	if pth, ok := p[name]; ok {
		path = pth
	} else {
		err = fmt.Errorf("path with name %s does not exist", name)
	}
	return
}

// MustGet panics if path is not found
func (p Paths) MustGet(name string) *Path {
	pth, err := p.Get(name)
	if err != nil {
		panic(err)
	}
	return pth
}

// Add adds a path by its name
func (p Paths) Add(name string, path *Path) error {
	if err := path.Validate(); err != nil {
		return err
	}
	if _, found := p[name]; found {
		return fmt.Errorf("path with name %s already exists", name)
	}
	p[name] = path
	return nil
}

// AddForce ignores existing paths and overwrites an existing path with that name
func (p Paths) AddForce(name string, path *Path) error {
	if err := path.Validate(); err != nil {
		return err
	}
	if _, found := p[name]; found {
		fmt.Printf("overwriting path %s with new path...\n", name)
	}
	p[name] = path
	return nil
}

// PathsFromChains returns a path from the config between two chains
func (p Paths) PathsFromChains(src, dst string) (Paths, error) {
	out := Paths{}
	for name, path := range p {
		if (path.Dst().ChainID() == src || path.Src().ChainID() == src) && (path.Dst().ChainID() == dst || path.Src().ChainID() == dst) {
			out[name] = path
		}
	}
	if len(out) == 0 {
		return Paths{}, fmt.Errorf("failed to find path in config between chains %s and %s", src, dst)
	}
	return out, nil
}

// Path represents a pair of chains and the identifiers needed to
// relay over them
type Path struct {
	SrcJSON  json.RawMessage `yaml:"src" json:"src"`
	DstJSON  json.RawMessage `yaml:"dst" json:"dst"`
	Strategy *StrategyCfg    `yaml:"strategy" json:"strategy"`

	// cache
	src PathEndI
	dst PathEndI
}

func UnmarshalPath(cdc codec.Codec, bz json.RawMessage) (*Path, error) {
	var path Path
	if err := json.Unmarshal(bz, &path); err != nil {
		return nil, err
	}
	if err := path.Init(cdc); err != nil {
		return nil, err
	}
	return &path, nil
}

func (p *Path) Init(cdc codec.Codec) error {
	var src, dst PathEndI
	if err := utils.UnmarshalJSONAny(cdc, &src, p.SrcJSON); err != nil {
		return err
	}
	if err := utils.UnmarshalJSONAny(cdc, &dst, p.DstJSON); err != nil {
		return err
	}
	p.src = src
	p.dst = dst
	return nil
}

func (p *Path) Src() PathEndI {
	if p.src == nil {
		panic("Path must be initialized with UnmarshalPath function")
	}
	return p.src
}

func (p *Path) Dst() PathEndI {
	if p.dst == nil {
		panic("Path must be initialized with UnmarshalPath function")
	}
	return p.dst
}

// Ordered returns true if the path is ordered and false if otherwise
func (p *Path) Ordered() bool {
	return p.Src().ChannelOrder() == chantypes.ORDERED
}

// Validate checks that a path is valid
func (p *Path) Validate() (err error) {
	if err = p.Src().Validate(); err != nil {
		return err
	}
	if p.Src().ChannelVersion() == "" {
		return fmt.Errorf("source must specify a version")
	}
	if err = p.Dst().Validate(); err != nil {
		return err
	}
	if _, err = p.GetStrategy(); err != nil {
		return err
	}
	if p.Src().ChannelOrder() != p.Dst().ChannelOrder() {
		return fmt.Errorf("both sides must have same order ('ORDERED' or 'UNORDERED'), got src(%s) and dst(%s)",
			p.Src().ChannelOrder(), p.Dst().ChannelOrder())
	}
	return nil
}

// End returns the proper end given a chainID
func (p *Path) End(chainID string) PathEndI {
	if p.Dst().ChainID() == chainID {
		return p.Dst()
	}
	if p.Src().ChainID() == chainID {
		return p.Src()
	}
	return &PathEnd{}
}

func (p *Path) String() string {
	return fmt.Sprintf("[ ] %s ->\n %s", p.Src().String(), p.Dst().String())
}

// PathStatus holds the status of the primatives in the path
type PathStatus struct {
	Chains     bool `yaml:"chains" json:"chains"`
	Clients    bool `yaml:"clients" json:"clients"`
	Connection bool `yaml:"connection" json:"connection"`
	Channel    bool `yaml:"channel" json:"channel"`
}

// PathWithStatus is used for showing the status of the path
type PathWithStatus struct {
	Path   *Path      `yaml:"path" json:"chains"`
	Status PathStatus `yaml:"status" json:"status"`
}

// QueryPathStatus returns an instance of the path struct with some attached data about
// the current status of the path
func (p *Path) QueryPathStatus(src, dst *ProvableChain) *PathWithStatus {
	var (
		err              error
		eg               errgroup.Group
		srch, dsth       int64
		srcCs, dstCs     *clienttypes.QueryClientStateResponse
		srcConn, dstConn *conntypes.QueryConnectionResponse
		srcChan, dstChan *chantypes.QueryChannelResponse

		out = &PathWithStatus{Path: p, Status: PathStatus{false, false, false, false}}
	)
	eg.Go(func() error {
		srch, err = src.GetLatestHeight()
		return err
	})
	eg.Go(func() error {
		dsth, err = dst.GetLatestHeight()
		return err
	})
	if eg.Wait(); err != nil {
		return out
	}
	out.Status.Chains = true

	eg.Go(func() error {
		srcCs, err = src.QueryClientStateWithProof(srch)
		return err
	})
	eg.Go(func() error {
		dstCs, err = dst.QueryClientStateWithProof(dsth)
		return err
	})
	if err = eg.Wait(); err != nil || srcCs == nil || dstCs == nil {
		return out
	}
	out.Status.Clients = true

	eg.Go(func() error {
		srcConn, err = src.QueryConnectionWithProof(srch)
		return err
	})
	eg.Go(func() error {
		dstConn, err = dst.QueryConnectionWithProof(dsth)
		return err
	})
	if err = eg.Wait(); err != nil || srcConn.Connection.State != conntypes.OPEN || dstConn.Connection.State != conntypes.OPEN {
		return out
	}
	out.Status.Connection = true

	eg.Go(func() error {
		srcChan, err = src.QueryChannelWithProof(srch)
		return err
	})
	eg.Go(func() error {
		dstChan, err = dst.QueryChannelWithProof(dsth)
		return err
	})
	if err = eg.Wait(); err != nil || srcChan.Channel.State != chantypes.OPEN || dstChan.Channel.State != chantypes.OPEN {
		return out
	}
	out.Status.Channel = true
	return out
}

// PrintString prints a string representations of the path status
func (ps *PathWithStatus) PrintString(name string) string {
	pth := ps.Path
	return fmt.Sprintf(`Path "%s" strategy(%s):
  SRC(%s)
    ClientID:     %s
    ConnectionID: %s
    ChannelID:    %s
    PortID:       %s
  DST(%s)
    ClientID:     %s
    ConnectionID: %s
    ChannelID:    %s
    PortID:       %s
  STATUS:
    Chains:       %s
    Clients:      %s
    Connection:   %s
    Channel:      %s`, name, pth.Strategy.Type, pth.Src().ChainID(),
		pth.Src().ClientID(), pth.Src().ConnectionID(), pth.Src().ChannelID(), pth.Src().PortID(),
		pth.Dst().ChainID(), pth.Dst().ClientID(), pth.Dst().ConnectionID(), pth.Dst().ChannelID(), pth.Dst().PortID(),
		checkmark(ps.Status.Chains), checkmark(ps.Status.Clients), checkmark(ps.Status.Connection), checkmark(ps.Status.Channel))
}

func checkmark(status bool) string {
	if status {
		return check
	}
	return xIcon
}
