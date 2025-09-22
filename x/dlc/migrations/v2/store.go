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
	disableDCMs(ctx, storeKey, cdc)
	disableOracles(ctx, storeKey, cdc)

	removePendingEvents(ctx, storeKey, cdc)

	return nil
}

// disableDCMs disables all existing dcms
func disableDCMs(ctx sdk.Context, storeKey storetypes.StoreKey, cdc codec.BinaryCodec) {
	store := ctx.KVStore(storeKey)

	iterator := storetypes.KVStorePrefixIterator(store, types.DCMKeyPrefix)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var dcm types.DCM
		cdc.MustUnmarshal(iterator.Value(), &dcm)

		dcm.Status = types.DCMStatus_DCM_status_Disable
		store.Set(iterator.Key(), cdc.MustMarshal(&dcm))
	}
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

// removePendingEvents removes dlc events from the pending queue
func removePendingEvents(ctx sdk.Context, storeKey storetypes.StoreKey, cdc codec.BinaryCodec) {
	store := ctx.KVStore(storeKey)

	iterator := storetypes.KVStorePrefixIterator(store, types.PendingLendingEventKeyPrefix)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		store.Delete(iterator.Key())
	}

	store.Delete(types.PendingLendingEventCountKey)
}
