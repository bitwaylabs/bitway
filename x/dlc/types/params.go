package types

import (
	"encoding/base64"
	"time"

	errorsmod "cosmossdk.io/errors"
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"

	tsstypes "github.com/bitwaylabs/bitway/x/tss/types"
)

var (
	// default nonce queue size
	DefaultNonceQueueSize = uint32(1000)

	// default nonce generation batch size
	DefaultNonceGenerationBatchSize = uint32(100)

	// default nonce generation interval in blocks
	DefaultNonceGenerationInterval = int64(50) // 50 blocks

	// default nonce generation timeout duration
	DefaultNonceGenerationTimeoutDuration = 24 * time.Hour // 24 hours

	// default oracle participant number
	DefaultOracleParticipantNum = uint32(3)

	// default oracle participant threshold
	DefaultOracleParticipantThreshold = uint32(2)
)

// NewParams creates a new Params instance
func NewParams() Params {
	return Params{
		NonceQueueSize:                 DefaultNonceQueueSize,
		NonceGenerationBatchSize:       DefaultNonceGenerationBatchSize,
		NonceGenerationInterval:        DefaultNonceGenerationInterval,
		NonceGenerationTimeoutDuration: DefaultNonceGenerationTimeoutDuration,
		AllowedOracleParticipants:      []string{},
		OracleParticipantNum:           DefaultOracleParticipantNum,
		OracleParticipantThreshold:     DefaultOracleParticipantThreshold,
	}
}

// DefaultParams returns a default set of parameters
func DefaultParams() Params {
	return NewParams()
}

// Validate validates params
func (p Params) Validate() error {
	if p.NonceQueueSize == 0 {
		return errorsmod.Wrap(ErrInvalidParams, "nonce queue size must be greater than 0")
	}

	if p.NonceGenerationBatchSize < 2 {
		return errorsmod.Wrapf(ErrInvalidParams, "nonce generation batch size cannot be less than 2")
	}

	if p.NonceGenerationInterval <= 0 {
		return errorsmod.Wrapf(ErrInvalidParams, "nonce generation interval must be greater than 0")
	}

	if p.NonceGenerationTimeoutDuration < 0 {
		return errorsmod.Wrapf(ErrInvalidParams, "nonce generation timeout duration cannot be negative")
	}

	if err := validateOracleParticipants(p.AllowedOracleParticipants); err != nil {
		return err
	}

	if err := validateOracleParticipantNum(p); err != nil {
		return err
	}

	return tsstypes.CheckDKGThreshold(int(p.OracleParticipantNum), int(p.OracleParticipantThreshold))
}

// validateOracleParticipants validates the given oracle participants
// Note: the participant is the ed25519 consensus pub key
func validateOracleParticipants(participants []string) error {
	for _, p := range participants {
		consensusPubKey, err := base64.StdEncoding.DecodeString(p)
		if err != nil {
			return errorsmod.Wrap(ErrInvalidParams, "failed to decode the participant consensus pub key")
		}

		if len(consensusPubKey) != ed25519.PubKeySize {
			return errorsmod.Wrap(ErrInvalidParams, "incorrect participant consensus pub key size")
		}
	}

	return nil
}

// validateOracleParticipantNum validates the given oracle participant num
func validateOracleParticipantNum(p Params) error {
	if p.OracleParticipantNum < uint32(tsstypes.MinDKGParticipantNum) {
		return errorsmod.Wrapf(ErrInvalidParams, "oracle participant number cannot be less than min dkg participant number %d", tsstypes.MinDKGParticipantNum)
	}

	if len(p.AllowedOracleParticipants) > 0 && p.OracleParticipantNum > uint32(len(p.AllowedOracleParticipants)) {
		return errorsmod.Wrapf(ErrInvalidParams, "oracle participant number cannot be greater than allowed oracle participant number %d", len(p.AllowedOracleParticipants))
	}

	return nil
}
