package app

import (
	"github.com/cosmos/btcutil/bech32"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bitwaylabs/bitway/bitcoin"
)

func init() {
	// Set prefixes
	accountPubKeyPrefix := AccountAddressPrefix + "pub"
	validatorAddressPrefix := AccountAddressPrefix + "valoper"
	validatorPubKeyPrefix := AccountAddressPrefix + "valoperpub"
	consNodeAddressPrefix := AccountAddressPrefix + "valcons"
	consNodePubKeyPrefix := AccountAddressPrefix + "valconspub"

	// Set and seal config
	config := sdk.GetConfig()
	config.SetBech32PrefixForAccount(AccountAddressPrefix, accountPubKeyPrefix)
	config.SetBech32PrefixForValidator(validatorAddressPrefix, validatorPubKeyPrefix)
	config.SetBech32PrefixForConsensusNode(consNodeAddressPrefix, consNodePubKeyPrefix)
	// config.SetBtcChainCfg(&chaincfg.TestNet3Params)
	config.Seal()

	bech32.BITCOIN_HRP = bitcoin.Network.Bech32HRPSegwit
	bech32.CUSTOM_HRP = AccountAddressPrefix
}
