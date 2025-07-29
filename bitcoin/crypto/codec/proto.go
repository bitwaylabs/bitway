package codec

import (
	"github.com/bitwaylabs/bitway/bitcoin/keys/segwit"
	"github.com/bitwaylabs/bitway/bitcoin/keys/taproot"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
)

// RegisterInterfaces registers the sdk.Tx interface.
func RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	var pk *cryptotypes.PubKey
	// registry.RegisterInterface("cosmos.crypto.PubKey", pk)
	registry.RegisterImplementations(pk, &segwit.PubKey{})
	registry.RegisterImplementations(pk, &taproot.PubKey{})

	var priv *cryptotypes.PrivKey
	// registry.RegisterInterface("cosmos.crypto.PrivKey", priv)
	registry.RegisterImplementations(priv, &segwit.PrivKey{})
	registry.RegisterImplementations(priv, &taproot.PrivKey{})
}
