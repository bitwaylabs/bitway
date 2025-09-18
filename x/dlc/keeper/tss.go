package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bitwaylabs/bitway/x/dlc/types"
	tsstypes "github.com/bitwaylabs/bitway/x/tss/types"
)

// DKGCompletionReceivedHandler is callback handler when the DKG completion received by TSS
func (k Keeper) DKGCompletionReceivedHandler(ctx sdk.Context, id uint64, ty string, intent int32, participant string) error {
	switch ty {
	case types.DKG_TYPE_NONCE:
		if k.HasOracleParticipantLiveness(ctx, participant) {
			liveness := k.GetOracleParticipantLiveness(ctx, participant)

			// set to alive with the current dkg and block height
			liveness.IsAlive = true
			liveness.LastDkgId = id
			liveness.LastBlockHeight = ctx.BlockHeight()
			k.SetOracleParticipantLiveness(ctx, liveness)
		}
	}

	return nil
}

// DKGCompletedHandler is callback handler when the DKG request completed by TSS
func (k Keeper) DKGCompletedHandler(ctx sdk.Context, id uint64, ty string, intent int32, pubKeys []string) error {
	switch ty {
	case types.DKG_TYPE_DCM:
		return k.CreateDCM(ctx, id, pubKeys[0])

	case types.DKG_TYPE_NONCE:
		// the first pub key is oracle and the remaining are nonces

		if err := k.CreateOracle(ctx, id, pubKeys[0]); err != nil {
			return err
		}

		return k.HandleNonces(ctx, pubKeys[0], pubKeys[1:], intent)
	}

	return nil
}

// DKGTimeoutHandler is callback handler when the DKG timed out in TSS
func (k Keeper) DKGTimeoutHandler(ctx sdk.Context, id uint64, ty string, intent int32, absentParticipants []string) error {
	switch ty {
	case types.DKG_TYPE_NONCE:
		if len(absentParticipants) == len(k.tssKeeper.GetDKGRequest(ctx, id).Participants) {
			// remain current liveness if all participants are absent
			return nil
		}

		for _, participant := range absentParticipants {
			if k.HasOracleParticipantLiveness(ctx, participant) {
				liveness := k.GetOracleParticipantLiveness(ctx, participant)
				if liveness.LastDkgId > id {
					// skip if the last dkg is later than the current one
					continue
				}

				// set to non-alive
				liveness.IsAlive = false
				k.SetOracleParticipantLiveness(ctx, liveness)
			}
		}
	}

	return nil
}

// SigningCompletedHandler is callback handler when the signing request completed by TSS
func (k Keeper) SigningCompletedHandler(ctx sdk.Context, sender string, id uint64, scopedId string, ty tsstypes.SigningType, intent int32, pubKey string, signatures []string) error {
	return k.HandleAttestation(ctx, sender, types.FromScopedId(scopedId), signatures[0])
}
