package store

type State struct {
	Pools  map[string]*PoolState
	Routes []RouteState
}

type PoolState struct {
	Name     string
	Strategy string
	Backends []BackendState
}

type BackendState struct {
	Name    string
	Address string
	Weight  uint32
}

type RouteState struct {
	Host       string
	PathPrefix string
	Pool       string
}

func NewState() *State {
	return &State{Pools: make(map[string]*PoolState)}
}
