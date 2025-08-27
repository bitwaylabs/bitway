package types

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"reflect"
	"slices"
	"time"

	"github.com/btcsuite/btcd/btcec/v2/schnorr"
	"github.com/btcsuite/btcd/txscript"

	errorsmod "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bitwaylabs/bitway/bitcoin/crypto/hash"
)

var (
	// minimum dkg participant number
	MinDKGParticipantNum = 2

	// minimum dkg threshold
	MinDKGThreshold = 2

	// minimum dkg threshold ratio
	MinDKGThresholdRatio = sdkmath.LegacyMustNewDecFromStr("0.5")

	// schnorr signature size
	SchnorrSignatureSize = 64

	// schnorr adaptor signature size
	SchnorrAdaptorSignatureSize = 65
)

// DKGCompletionReceivedHandler defines the callback handler on the DKG completion received
type DKGCompletionReceivedHandler func(ctx sdk.Context, id uint64, ty string, intent int32, participant string) error

// DKGRequestCompletedHandler defines the callback handler on the DKG request completed
type DKGRequestCompletedHandler func(ctx sdk.Context, id uint64, ty string, intent int32, pubKeys []string) error

// DKGRequestTimeoutHandler defines the callback handler on the DKG request timed out
type DKGRequestTimeoutHandler func(ctx sdk.Context, id uint64, ty string, intent int32, absentParticipants []string) error

// SigningRequestCompletedHandler defines the callback handler on the signing request completed
type SigningRequestCompletedHandler func(ctx sdk.Context, sender string, id uint64, scopedId string, ty SigningType, intent int32, pubKey string, signatures []string) error

// ParticipantExists returns true if the given participant is included in the authorized participants, false otherwise
func ParticipantExists(participants []string, participant string) bool {
	return slices.Contains(participants, participant)
}

// CheckDKGCompletions checks if public keys of all DKG completions are same
func CheckDKGCompletions(completions []*DKGCompletion) bool {
	if len(completions) == 0 {
		return false
	}

	pubKeys := completions[0].PubKeys

	for _, completion := range completions[1:] {
		if !reflect.DeepEqual(completion.PubKeys, pubKeys) {
			return false
		}
	}

	return true
}

// VerifySignature verifies the ed25519 signature against the given pub key and msg
// Assume that the signature is hex encoded and the pub key is base64 encoded
func VerifySignature(signature string, pubKey string, msg []byte) bool {
	sigBytes, _ := hex.DecodeString(signature)
	pubKeyBytes, _ := base64.StdEncoding.DecodeString(pubKey)

	return ed25519.Verify(pubKeyBytes, msg, sigBytes)
}

// GetDKGCompletionSigMsg gets the msg to be signed from the given data for the DKG completion
// Assume that the given pub keys are hex encoded
func GetDKGCompletionSigMsg(id uint64, pubKeys []string) []byte {
	msg := make([]byte, 8)
	binary.BigEndian.PutUint64(msg, id)

	for _, pubKey := range pubKeys {
		pubKeyBytes, _ := hex.DecodeString(pubKey)
		msg = append(msg, pubKeyBytes...)
	}

	return hash.Sha256(msg)
}

// GetRefreshingCompletionSigMsg gets the msg to be signed from the given data for the refreshing completion
// Assume that the given pub keys are hex encoded
func GetRefreshingCompletionSigMsg(id uint64, pubKeys []string) []byte {
	msg := make([]byte, 8)
	binary.BigEndian.PutUint64(msg, id)

	for _, pubKey := range pubKeys {
		pubKeyBytes, _ := hex.DecodeString(pubKey)
		msg = append(msg, pubKeyBytes...)
	}

	return hash.Sha256(msg)
}

// GetSigningOption gets the signing option according to the given signing type and options
// Assume that the options match the signing type
func GetSigningOption(signingType SigningType, options *SigningOptions) string {
	switch signingType {
	case SigningType_SIGNING_TYPE_SCHNORR_WITH_TWEAK:
		return options.Tweak

	case SigningType_SIGNING_TYPE_SCHNORR_WITH_COMMITMENT:
		return options.Nonce

	case SigningType_SIGNING_TYPE_SCHNORR_ADAPTOR:
		return options.AdaptorPoint

	default:
		return ""
	}
}

// GetTweakedPubKey gets the tweaked pub key by the given tweak
// Assume that the given pub key is valid
func GetTweakedPubKey(pubKeyBytes []byte, tweak []byte) []byte {
	pubKey, _ := schnorr.ParsePubKey(pubKeyBytes)
	tweakedPubKey := txscript.ComputeTaprootOutputKey(pubKey, tweak)

	return schnorr.SerializePubKey(tweakedPubKey)
}

// GetExpirationTime gets the expiration time according to the given timeout duration
func GetExpirationTime(currentTime time.Time, timeoutDuration time.Duration) time.Time {
	if timeoutDuration == 0 {
		return time.Time{}
	}

	return currentTime.Add(timeoutDuration)
}

// CheckDKGParticipantNum checks if the given participant num is valid
func CheckDKGParticipantNum(participantNum int) error {
	if participantNum < MinDKGParticipantNum {
		return errorsmod.Wrapf(ErrInvalidDKGParticipantNum, "dkg participant number cannot be less than min participant number %d", MinDKGParticipantNum)
	}

	return nil
}

// CheckDKGThreshold checks if the given threshold is valid
func CheckDKGThreshold(participantNum int, threshold int) error {
	if threshold < MinDKGThreshold {
		return errorsmod.Wrapf(ErrInvalidDKGThreshold, "dkg threshold cannot be less than min threshold %d", MinDKGThreshold)
	}

	if threshold > participantNum {
		return errorsmod.Wrapf(ErrInvalidDKGThreshold, "dkg threshold cannot be greater than participant number %d", participantNum)
	}

	if MinDKGThresholdRatio.IsPositive() {
		if int64(threshold) < MinDKGThresholdRatio.MulInt64(int64(participantNum)).Ceil().TruncateInt64() {
			return errorsmod.Wrapf(ErrInvalidDKGThreshold, "dkg threshold ratio cannot be less than %s", MinDKGThresholdRatio)
		}
	}

	return nil
}
