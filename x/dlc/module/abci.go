package dlc

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bitwaylabs/bitway/x/dlc/keeper"
	"github.com/bitwaylabs/bitway/x/dlc/types"
)

// EndBlocker called at the end of every block
func EndBlocker(ctx sdk.Context, k keeper.Keeper) error {
	generateLendingEventNonces(ctx, k)

	checkOracleParticipantLiveness(ctx, k)

	return nil
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
	if len(k.GetOracleParticipantBaseSet(ctx)) < int(k.OracleParticipantNum(ctx)) {
		return
	}

	// get oracle participants
	participants := k.GetOracleParticipants(ctx)

	// initiate DKG
	k.TSSKeeper().InitiateDKG(ctx, types.ModuleName, types.DKG_TYPE_NONCE, int32(types.DKGIntent_DKG_INTENT_LENDING_EVENT_NONCE), participants, k.OracleParticipantThreshold(ctx), k.NonceGenerationBatchSize(ctx), k.NonceGenerationTimeoutDuration(ctx))
}

// checkOracleParticipantLiveness triggers DKG to check oracle participant liveness
func checkOracleParticipantLiveness(ctx sdk.Context, k keeper.Keeper) {
	// check block height
	if ctx.BlockHeight()%types.DefaultOracleParticipantLivenessCheckInterval != 0 {
		return
	}

	// check if there are sufficient oracle participants
	if len(k.GetOracleParticipantBaseSet(ctx)) < int(k.OracleParticipantNum(ctx)) {
		return
	}

	// get random oracle participants
	participants := k.GetRandomOracleParticipants(ctx)

	// initiate DKG
	k.TSSKeeper().InitiateDKG(ctx, types.ModuleName, types.DKG_TYPE_LIVENESS_CHECK, int32(types.DKGIntent_DKG_INTENT_DEFAULT), participants, k.OracleParticipantThreshold(ctx), 1, types.DefaultDKGTimeoutDuration)
}
