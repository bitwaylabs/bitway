package dlc

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bitwaylabs/bitway/x/dlc/keeper"
	"github.com/bitwaylabs/bitway/x/dlc/types"
)

// EndBlocker called at the end of every block
func EndBlocker(ctx sdk.Context, k keeper.Keeper) error {
	resetOracleParticipantsLiveness(ctx, k)
	generateLendingEventNonces(ctx, k)

	return nil
}

// resetOracleParticipantsLiveness resets oracle participants liveness
func resetOracleParticipantsLiveness(ctx sdk.Context, k keeper.Keeper) {
	if ctx.BlockHeight()%types.OracleParticipantsLivenessResetInterval == 0 {
		k.ResetOracleParticipantsLiveness(ctx)
	}
}

// generateLendingEventNonces generates nonces events for dlc lending events
func generateLendingEventNonces(ctx sdk.Context, k keeper.Keeper) {
	// check block height
	if ctx.BlockHeight()%k.NonceGenerationInterval(ctx) != 0 {
		return
	}

	// check if lending event nonces need to be generated
	pendingLendingEventCount := k.GetPendingLendingEventCount(ctx)
	if pendingLendingEventCount >= uint64(k.NonceQueueSize(ctx)) {
		return
	}

	// check if there are sufficient oracle participants
	oracleParticipantBaseSet := k.GetOracleParticipantBaseSet(ctx)
	if len(oracleParticipantBaseSet) < int(k.OracleParticipantNum(ctx)) {
		k.Logger(ctx).Warn("insufficient oracle participants", "num", len(oracleParticipantBaseSet), "required num", k.OracleParticipantNum(ctx))
		return
	}

	// get oracle participants
	participants := k.GetOracleParticipants(ctx)

	// initiate DKG
	k.TSSKeeper().InitiateDKG(ctx, types.ModuleName, types.DKG_TYPE_NONCE, int32(types.DKGIntent_DKG_INTENT_LENDING_EVENT_NONCE), participants, k.OracleParticipantThreshold(ctx), k.NonceGenerationBatchSize(ctx), k.NonceGenerationTimeoutDuration(ctx))
}
