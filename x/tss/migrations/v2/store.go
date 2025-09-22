package v2

import (
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bitwaylabs/bitway/x/tss/types"
)

// MigrateStore migrates the x/tss module state from the consensus version 1 to
// version 2
func MigrateStore(ctx sdk.Context, storeKey storetypes.StoreKey, cdc codec.BinaryCodec) error {
	markSigningRequestsFailed(ctx, storeKey, cdc)

	return nil
}

// markSigningRequestsFailed marks the pending signing requests as failed
func markSigningRequestsFailed(ctx sdk.Context, storeKey storetypes.StoreKey, cdc codec.BinaryCodec) {
	store := ctx.KVStore(storeKey)

	iterator := storetypes.KVStorePrefixIterator(store, types.SigningRequestKeyPrefix)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var signingRequest types.SigningRequest
		cdc.MustUnmarshal(iterator.Value(), &signingRequest)

		if signingRequest.Status == types.SigningStatus_SIGNING_STATUS_PENDING {
			signingRequest.Status = types.SigningStatus_SIGNING_STATUS_FAILED
			store.Set(types.SigningRequestKey(signingRequest.Id), cdc.MustMarshal(&signingRequest))

			store.Delete(types.SigningRequestByStatusKey(types.SigningStatus_SIGNING_STATUS_PENDING, signingRequest.Id))
			store.Set(types.SigningRequestByStatusKey(types.SigningStatus_SIGNING_STATUS_FAILED, signingRequest.Id), []byte{})
		}
	}
}
