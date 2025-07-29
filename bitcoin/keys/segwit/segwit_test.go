package segwit_test

import (
	//"fmt"

	"encoding/base64"
	"encoding/hex"
	"strings"
	"testing"

	"github.com/bitwaylabs/bitway/bitcoin/keys/segwit"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cosmos/btcutil/bech32"

	"github.com/cometbft/cometbft/crypto"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/go-bip39"
)

func TestNumbers(t *testing.T) {
	t.Logf("Test numbers")
	msg := []byte("1234")
	prefix := segwit.VarintBufNum(uint64(len(segwit.MagicBytes)))
	t.Log("prefix1:", prefix)
	prefix2 := segwit.VarintBufNum(uint64(len(msg)))
	t.Log("prefix2:", prefix2)

	buf := append(prefix, segwit.MagicBytes...)
	buf = append(buf, prefix2...)
	buf = append(buf, msg...)

	t.Log("buf:", buf)

	buf = crypto.Sha256(buf)
	buf = crypto.Sha256(buf)
	// buf = crypto.SHA256.New().Sum(buf)

	t.Log("hash:", buf)
	t.Log("hash:", hex.EncodeToString(buf))

}

func TestSegwit(t *testing.T) {
	mnemonic := "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about"
	seed := bip39.NewSeed(mnemonic, "")
	expected_address := "bc1qcr8te4kr609gcawutmrza0j4xv80jy8z306fyu"

	BITWAY_HRP := "bc"
	sdk.GetConfig().SetBech32PrefixForAccount(BITWAY_HRP, BITWAY_HRP)
	sdk.GetConfig().Seal()
	bech32.CUSTOM_HRP = BITWAY_HRP

	masterKey, chParams := hd.ComputeMastersFromSeed(seed)
	derivedPrivKey, err := hd.DerivePrivateKeyForPath(masterKey, chParams, "m/84'/0'/0'/0/0")
	assert.NoError(t, err, "Private key derivation should not fail")
	privKey := segwit.PrivKey{Key: derivedPrivKey}

	sig, err := privKey.Sign([]byte("1234"))
	assert.NoError(t, err, "Sign should not fail")
	t.Log("sig:", base64.StdEncoding.EncodeToString(sig))

	pubKey := privKey.PubKey()
	assert.NotNil(t, pubKey, "Public key should not be nil")

	verify := pubKey.VerifySignature([]byte("1234"), sig)
	assert.True(t, verify, "Verify should be true")

	assert.Equal(t, 33, len(pubKey.Address().Bytes()), "Address length should be 33 bytes")
	bech32Address, err := bech32.Encode("bc", pubKey.Address().Bytes())
	// bech32Address, err := segwit.BitCoinAddr(pubKey.Bytes())
	assert.NoError(t, err)

	// bech32Address = sdk.AccAddress(pubKey.Address()).String()

	require.Equal(t, expected_address, bech32Address, "Address should be equal to expected address")
	t.Logf("Generated SegWit Address: %s", bech32Address)
	// Check if the Bech32 encoded address has the correct prefix and structure.
	assert.True(t, strings.HasPrefix(bech32Address, "bc1q"), "Address should start with 'bc1q'")
	t.Logf("Generated SegWit Address: %s", bech32Address)

	// data, err := sdk.GetFromBech32(bech32Address, "bc")

	hrp, data, version, err2 := bech32.DecodeNoLimitWithVersion(bech32Address)
	assert.NoError(t, err2)

	println(hrp, version, data)
	t.Log(hrp)

	hrp, bz, err := bech32.Decode(bech32Address, 1024)
	//hrp, bz, err := bech32.Decode("bc1qc2zm9xeje96yh6st7wmy60mmsteemsm3tfr2tn", 1000)
	assert.NoError(t, err)
	println(hrp, bz)

	acc, err := sdk.AccAddressFromBech32(bech32Address)
	require.NoError(t, err)
	t.Logf("Generated SegWit Address: %s", acc)
	// addr := []byte{123, 95, 226, 43, 84, 70, 247, 198, 46, 162, 123, 139, 215, 28, 239, 148, 224, 63, 61, 242}
	// _, err = sdkbech32.ConvertAndEncode("bc", addr)

	//t.Logf("parsed address", dd)
	require.NoError(t, err)

}
