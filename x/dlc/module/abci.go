package dlc

import (
	"slices"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bitwaylabs/bitway/x/dlc/keeper"
	"github.com/bitwaylabs/bitway/x/dlc/types"
)

// EndBlocker called at the end of every block
func EndBlocker(ctx sdk.Context, k keeper.Keeper) error {
	if ctx.BlockHeight() == 35800 {
		removeDeprecatedEvents(ctx, k, []string{"1b8aaca58853886992dce321bbec5f3c2a8d20d9e7b6bb24b57fb7b53d47befb"})
	}

	generateLendingEventNonces(ctx, k)

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

func removeDeprecatedEvents(ctx sdk.Context, k keeper.Keeper, oraclePubKeys []string) {
	events := []*types.DLCEvent{}
	k.IteratePendingLendingEvents(ctx, func(event *types.DLCEvent) (stop bool) {
		events = append(events, event)

		return false
	})

	for _, event := range events {
		if slices.Contains(oraclePubKeys, event.Pubkey) {
			k.RemoveLendingEventFromPendingQueue(ctx, event)
		}
	}
}
