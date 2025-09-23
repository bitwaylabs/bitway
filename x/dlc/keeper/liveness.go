package keeper

import (
	"slices"

	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bitwaylabs/bitway/x/dlc/types"
)

// SetOracleParticipantLiveness sets the oracle participant liveness
func (k Keeper) SetOracleParticipantLiveness(ctx sdk.Context, liveness *types.OracleParticipantLiveness) {
	store := ctx.KVStore(k.storeKey)

	bz := k.cdc.MustMarshal(liveness)

	store.Set(types.OracleParticipantLivenessKey(liveness.ConsensusPubkey), bz)
}

// RemoveOracleParticipantLiveness removes the given oracle participant liveness
func (k Keeper) RemoveOracleParticipantLiveness(ctx sdk.Context, consensusPubKey string) {
	store := ctx.KVStore(k.storeKey)

	store.Delete(types.OracleParticipantLivenessKey(consensusPubKey))
}

// HasOracleParticipantLiveness returns true if the given oracle participant liveness exists, false otherwise
func (k Keeper) HasOracleParticipantLiveness(ctx sdk.Context, consensusPubKey string) bool {
	store := ctx.KVStore(k.storeKey)

	return store.Has(types.OracleParticipantLivenessKey(consensusPubKey))
}

// GetOracleParticipantLiveness returns the given oracle participant liveness
func (k Keeper) GetOracleParticipantLiveness(ctx sdk.Context, consensusPubKey string) *types.OracleParticipantLiveness {
	store := ctx.KVStore(k.storeKey)

	var liveness types.OracleParticipantLiveness
	bz := store.Get(types.OracleParticipantLivenessKey(consensusPubKey))
	k.cdc.MustUnmarshal(bz, &liveness)

	return &liveness
}

// GetAllOracleParticipantsLiveness gets all oracle participants liveness
func (k Keeper) GetAllOracleParticipantsLiveness(ctx sdk.Context) []*types.OracleParticipantLiveness {
	participantsLiveness := make([]*types.OracleParticipantLiveness, 0)

	k.IterateOracleParticipantsLiveness(ctx, func(liveness *types.OracleParticipantLiveness) (stop bool) {
		participantsLiveness = append(participantsLiveness, liveness)
		return false
	})

	return participantsLiveness
}

// GetOracleParticipantsLiveness gets oracle participants liveness by the given status
func (k Keeper) GetOracleParticipantsLiveness(ctx sdk.Context, alive bool) []*types.OracleParticipantLiveness {
	participantsLiveness := make([]*types.OracleParticipantLiveness, 0)

	k.IterateOracleParticipantsLiveness(ctx, func(liveness *types.OracleParticipantLiveness) (stop bool) {
		if liveness.IsAlive == alive {
			participantsLiveness = append(participantsLiveness, liveness)
		}

		return false
	})

	return participantsLiveness
}

// IterateOracleParticipantsLiveness iterates through all oracle participants liveness
func (k Keeper) IterateOracleParticipantsLiveness(ctx sdk.Context, cb func(liveness *types.OracleParticipantLiveness) (stop bool)) {
	store := ctx.KVStore(k.storeKey)

	iterator := storetypes.KVStorePrefixIterator(store, types.OracleParticipantLivenessKeyPrefix)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var liveness types.OracleParticipantLiveness
		k.cdc.MustUnmarshal(iterator.Value(), &liveness)

		if cb(&liveness) {
			break
		}
	}
}

// IsOracleParticipantAlive returns true if the given oracle participant is alive, false otherwise
func (k Keeper) IsOracleParticipantAlive(ctx sdk.Context, consensusPubKey string) bool {
	if !k.HasOracleParticipantLiveness(ctx, consensusPubKey) {
		return false
	}

	return k.GetOracleParticipantLiveness(ctx, consensusPubKey).IsAlive
}

// UpdateOracleParticipantsLiveness updates oracle participants liveness
func (k Keeper) UpdateOracleParticipantsLiveness(ctx sdk.Context, participants []string, newParticipants []string) {
	for _, participant := range participants {
		// delete the liveness status for the removed participant
		if !slices.Contains(newParticipants, participant) {
			k.RemoveOracleParticipantLiveness(ctx, participant)
		}
	}

	for _, participant := range newParticipants {
		// set to initial status for the newly added participant
		if !slices.Contains(participants, participant) {
			k.SetOracleParticipantLiveness(ctx, &types.OracleParticipantLiveness{
				ConsensusPubkey: participant,
				IsAlive:         true,
			})
		}
	}
}
