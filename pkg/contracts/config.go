package contracts

// ENSConfig handles the ENS configuration
type ENSConfig struct {
	ChainID               string
	ENSRegistryAddress    string
	FDSRegistrarAddress   string
	PublicResolverAddress string
	ProviderDomain        string
	ProviderBackend       string
}

// SubscriptionConfig handles the Subscription Management
type SubscriptionConfig struct {
	RPC              string
	SwarmMailAddress string
}

// TestnetConfig defines the configuration for goerli testnet
func TestnetConfig() (*ENSConfig, *SubscriptionConfig) {
	e := &ENSConfig{
		ChainID:               "5",
		ENSRegistryAddress:    "0x42B22483e3c8dF794f351939620572d1a3193c12",
		FDSRegistrarAddress:   "0xF4C9Cd25031E3BB8c5618299bf35b349c1aAb6A9",
		PublicResolverAddress: "0xbfeCC6c32B224F7D0026ac86506Fe40A9607BD14",
		ProviderDomain:        "fds",
	}

	s := &SubscriptionConfig{
		SwarmMailAddress: "0x7Aedf45B82924B2dBF9818c7cAaB6c7557Ba09c0",
	}
	return e, s
}

// PlayConfig defines the configuration for fdp-play
func PlayConfig() (*ENSConfig, *SubscriptionConfig) {
	s := &SubscriptionConfig{
		SwarmMailAddress: "0x86072CbFF48dA3C1F01824a6761A03F105BCC697",
	}
	return &ENSConfig{
		ChainID:               "4020",
		ENSRegistryAddress:    "0xDb56f2e9369E0D7bD191099125a3f6C370F8ed15",
		FDSRegistrarAddress:   "0xA94B7f0465E98609391C623d0560C5720a3f2D33",
		PublicResolverAddress: "0xFC628dd79137395F3C9744e33b1c5DE554D94882",
		ProviderDomain:        "fds",
	}, s
}
