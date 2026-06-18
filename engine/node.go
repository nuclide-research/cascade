package engine

// DataKey is a typed key in the shared state store.
type DataKey string

const (
	KeyIPv4     DataKey = "ipv4"
	KeyIPv6     DataKey = "ipv6"
	KeyDomain   DataKey = "domain"
	KeyHostname DataKey = "hostname"
	KeyASN      DataKey = "asn"
	KeyOrg      DataKey = "org"
	KeyMAC      DataKey = "mac"
	KeyEmail    DataKey = "email"
	KeyMXHost   DataKey = "mx_host"
	KeyNSHost   DataKey = "ns_host"
	KeyAbuse    DataKey = "abuse_email"
	KeyCIDR     DataKey = "cidr"
	KeyRegistrantEmail DataKey = "registrant_email"
)

// Finding is a single discovered fact from a tool run.
type Finding struct {
	Label string
	Value string
	Sub   []Finding
}

// Result is what a tool returns after running.
type Result struct {
	Tool     string
	Findings []Finding
	Produces map[DataKey][]string
	Err      error
}

// Node is the interface every tool implements.
type Node interface {
	Name() string
	Requires() []DataKey
	Produces() []DataKey
	Run(state *State) (*Result, error)
}
