package v2

import (
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bitwaylabs/bitway/x/dlc/types"
)

// MigrateStore migrates the x/dlc module state from the consensus version 1 to
// version 2
func MigrateStore(ctx sdk.Context, storeKey storetypes.StoreKey, cdc codec.BinaryCodec) error {
	disableOracles(ctx, storeKey, cdc)

	removePendingDLCEvents(ctx, storeKey, cdc)

	removeOracleParticipants(ctx, storeKey, cdc)

	return nil
}

// disableOracles disables all existing oracles
func disableOracles(ctx sdk.Context, storeKey storetypes.StoreKey, cdc codec.BinaryCodec) {
	store := ctx.KVStore(storeKey)

	iterator := storetypes.KVStorePrefixIterator(store, types.OracleKeyPrefix)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var oracle types.DLCOracle
		cdc.MustUnmarshal(iterator.Value(), &oracle)

		oracle.Status = types.DLCOracleStatus_Oracle_status_Disable
		store.Set(iterator.Key(), cdc.MustMarshal(&oracle))
	}
}

// removePendingDLCEvents removes dlc events from the pending queue
func removePendingDLCEvents(ctx sdk.Context, storeKey storetypes.StoreKey, _ codec.BinaryCodec) {
	store := ctx.KVStore(storeKey)

	iterator := storetypes.KVStorePrefixIterator(store, types.PendingLendingEventKeyPrefix)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		store.Delete(iterator.Key())
	}

	store.Delete(types.PendingLendingEventCountKey)
}

// removeOracleParticipants removes the oracle participants
func removeOracleParticipants(ctx sdk.Context, storeKey storetypes.StoreKey, cdc codec.BinaryCodec) {
	store := ctx.KVStore(storeKey)

	// get the current params
	var params types.Params
	bz := store.Get(types.ParamsKey)
	cdc.MustUnmarshal(bz, &params)

	// remove the oracle participants
	params.AllowedOracleParticipants = []string{}
	store.Set(types.ParamsKey, cdc.MustMarshal(&params))

	// remove the oracle participants liveness
	iterator := storetypes.KVStorePrefixIterator(store, types.OracleParticipantLivenessKeyPrefix)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		store.Delete(iterator.Key())
	}
}
