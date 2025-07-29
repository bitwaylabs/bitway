package keeper

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"

	"github.com/bitwaylabs/bitway/bitcoin/crypto/adaptor"
	"github.com/bitwaylabs/bitway/bitcoin/crypto/schnorr"
	"github.com/bitwaylabs/bitway/x/tss/types"
)

// GetSigningRequestId returns the signing request id
func (k Keeper) GetSigningRequestId(ctx sdk.Context) uint64 {
	store := ctx.KVStore(k.storeKey)

	bz := store.Get(types.SigningRequestIdKey)
	if bz == nil {
		return 0
	}

	return sdk.BigEndianToUint64(bz)
}

// IncrementSigningRequestId increments the signing request id and returns the new id
func (k Keeper) IncrementSigningRequestId(ctx sdk.Context) uint64 {
	store := ctx.KVStore(k.storeKey)

	id := k.GetSigningRequestId(ctx) + 1
	store.Set(types.SigningRequestIdKey, sdk.Uint64ToBigEndian(id))

	return id
}

// SetSigningRequest sets the signing request
func (k Keeper) SetSigningRequest(ctx sdk.Context, signingRequest *types.SigningRequest) {
	store := ctx.KVStore(k.storeKey)

	bz := k.cdc.MustMarshal(signingRequest)

	k.SetSigningRequestStatus(ctx, signingRequest.Id, signingRequest.Status)

	store.Set(types.SigningRequestKey(signingRequest.Id), bz)
}

// SetSigningRequestStatus sets the status store of the given signing request
func (k Keeper) SetSigningRequestStatus(ctx sdk.Context, id uint64, status types.SigningStatus) {
	store := ctx.KVStore(k.storeKey)

	if k.HasSigningRequest(ctx, id) {
		k.RemoveSigningRequestStatus(ctx, id)
	}

	store.Set(types.SigningRequestByStatusKey(status, id), []byte{})
}

// RemoveSigningRequestStatus removes the status store of the given signing request
func (k Keeper) RemoveSigningRequestStatus(ctx sdk.Context, id uint64) {
	store := ctx.KVStore(k.storeKey)

	signingRequest := k.GetSigningRequest(ctx, id)

	store.Delete(types.SigningRequestByStatusKey(signingRequest.Status, id))
}

// HasSigningRequest returns true if the given signing request exists, false otherwise
func (k Keeper) HasSigningRequest(ctx sdk.Context, id uint64) bool {
	store := ctx.KVStore(k.storeKey)

	return store.Has(types.SigningRequestKey(id))
}

// GetSigningRequest returns the signing request by the given id
func (k Keeper) GetSigningRequest(ctx sdk.Context, id uint64) *types.SigningRequest {
	store := ctx.KVStore(k.storeKey)

	var signingRequest types.SigningRequest
	bz := store.Get(types.SigningRequestKey(id))
	k.cdc.MustUnmarshal(bz, &signingRequest)

	return &signingRequest
}

// GetSigningRequestsByStatus gets the signing requests by the given status
func (k Keeper) GetSigningRequestsByStatus(ctx sdk.Context, status types.SigningStatus) []*types.SigningRequest {
	requests := make([]*types.SigningRequest, 0)

	k.IterateSigningRequestsByStatus(ctx, status, func(req *types.SigningRequest) (stop bool) {
		requests = append(requests, req)
		return false
	})

	return requests
}

// GetSigningRequestsByStatusWithPagination gets the signing requests by the given status and module with pagination
func (k Keeper) GetSigningRequestsByStatusWithPagination(ctx sdk.Context, status types.SigningStatus, module string, pagination *query.PageRequest) ([]*types.SigningRequest, *query.PageResponse, error) {
	store := ctx.KVStore(k.storeKey)
	signingRequestStatusStore := prefix.NewStore(store, append(types.SigningRequestByStatusKeyPrefix, sdk.Uint64ToBigEndian(uint64(status))...))

	var signingRequests []*types.SigningRequest

	pageRes, err := query.Paginate(signingRequestStatusStore, pagination, func(key []byte, value []byte) error {
		id := sdk.BigEndianToUint64(key)
		signingRequest := k.GetSigningRequest(ctx, id)

		if len(module) == 0 || signingRequest.Module == module {
			signingRequests = append(signingRequests, signingRequest)
		}

		return nil
	})
	if err != nil {
		return nil, nil, err
	}

	return signingRequests, pageRes, nil
}

// GetSigningRequestsWithPagination gets the signing requests by the given module with pagination
func (k Keeper) GetSigningRequestsWithPagination(ctx sdk.Context, module string, pagination *query.PageRequest) ([]*types.SigningRequest, *query.PageResponse, error) {
	store := ctx.KVStore(k.storeKey)
	signingRequestStore := prefix.NewStore(store, types.SigningRequestKeyPrefix)

	var signingRequests []*types.SigningRequest

	pageRes, err := query.Paginate(signingRequestStore, pagination, func(key []byte, value []byte) error {
		var signingRequest types.SigningRequest
		k.cdc.MustUnmarshal(value, &signingRequest)

		if len(module) == 0 || signingRequest.Module == module {
			signingRequests = append(signingRequests, &signingRequest)
		}

		return nil
	})
	if err != nil {
		return nil, nil, err
	}

	return signingRequests, pageRes, nil
}

// GetAllSigningRequests gets all signing requests
func (k Keeper) GetAllSigningRequests(ctx sdk.Context) []*types.SigningRequest {
	requests := make([]*types.SigningRequest, 0)

	k.IterateSigningRequests(ctx, func(req *types.SigningRequest) (stop bool) {
		requests = append(requests, req)
		return false
	})

	return requests
}

// IterateSigningRequests iterates through all signing requests
func (k Keeper) IterateSigningRequests(ctx sdk.Context, cb func(signingRequest *types.SigningRequest) (stop bool)) {
	store := ctx.KVStore(k.storeKey)

	iterator := storetypes.KVStorePrefixIterator(store, types.SigningRequestKeyPrefix)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var signingRequest types.SigningRequest
		k.cdc.MustUnmarshal(iterator.Value(), &signingRequest)

		if cb(&signingRequest) {
			break
		}
	}
}

// IterateSigningRequestsByStatus iterates through signing requests by the given status
func (k Keeper) IterateSigningRequestsByStatus(ctx sdk.Context, status types.SigningStatus, cb func(req *types.SigningRequest) (stop bool)) {
	store := ctx.KVStore(k.storeKey)

	keyPrefix := append(types.SigningRequestByStatusKeyPrefix, sdk.Uint64ToBigEndian(uint64(status))...)

	iterator := storetypes.KVStorePrefixIterator(store, keyPrefix)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		key := iterator.Key()

		id := sdk.BigEndianToUint64(key[len(keyPrefix):])
		signingRequest := k.GetSigningRequest(ctx, id)

		if cb(signingRequest) {
			break
		}
	}
}

// InitiateSigningRequest initiates the signing request with the specified params
func (k Keeper) InitiateSigningRequest(ctx sdk.Context, module string, scopedId string, ty types.SigningType, intent int32, pubKey string, sigHashes []string, options *types.SigningOptions) *types.SigningRequest {
	req := &types.SigningRequest{
		Id:           k.IncrementSigningRequestId(ctx),
		Module:       module,
		ScopedId:     scopedId,
		Type:         ty,
		Intent:       intent,
		PubKey:       pubKey,
		SigHashes:    sigHashes,
		Options:      options,
		CreationTime: ctx.BlockTime(),
		Status:       types.SigningStatus_SIGNING_STATUS_PENDING,
	}

	k.SetSigningRequest(ctx, req)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeInitiateSigning,
			sdk.NewAttribute(types.AttributeKeyId, fmt.Sprintf("%d", req.Id)),
			sdk.NewAttribute(types.AttributeKeyModule, module),
			sdk.NewAttribute(types.AttributeKeyScopedId, scopedId),
			sdk.NewAttribute(types.AttributeKeyType, fmt.Sprintf("%d", ty)),
			sdk.NewAttribute(types.AttributeKeyIntent, fmt.Sprintf("%d", intent)),
			sdk.NewAttribute(types.AttributeKeyPubKey, pubKey),
			sdk.NewAttribute(types.AttributeKeySigHashes, strings.Join(sigHashes, types.AttributeValueSeparator)),
			sdk.NewAttribute(types.AttributeKeyOption, types.GetSigningOption(ty, options)),
		),
	)

	return req
}

// VerifySignatures verifies the given signatures against the specified signing request
// Assume that the signing request is valid and signatures are hex encoded
func (k Keeper) VerifySignatures(ctx sdk.Context, signingRequest *types.SigningRequest, signatures []string) error {
	if len(signatures) != len(signingRequest.SigHashes) {
		return errorsmod.Wrap(types.ErrInvalidSignatures, "signatures do not match sig hashes")
	}

	pubKey, _ := hex.DecodeString(signingRequest.PubKey)

	options := signingRequest.Options

	for i, signature := range signatures {
		sigBytes, _ := hex.DecodeString(signature)
		sigHash, _ := base64.StdEncoding.DecodeString(signingRequest.SigHashes[i])

		switch signingRequest.Type {
		case types.SigningType_SIGNING_TYPE_SCHNORR:
			if len(sigBytes) != types.SchnorrSignatureSize {
				return errorsmod.Wrap(types.ErrInvalidSignature, "invalid schnorr signature size")
			}

			if !schnorr.Verify(sigBytes, sigHash, pubKey) {
				return errorsmod.Wrap(types.ErrInvalidSignature, "invalid schnorr signature")
			}

		case types.SigningType_SIGNING_TYPE_SCHNORR_WITH_TWEAK:
			if len(sigBytes) != types.SchnorrSignatureSize {
				return errorsmod.Wrap(types.ErrInvalidSignature, "invalid schnorr signature size")
			}

			tweak, _ := hex.DecodeString(options.Tweak)
			tweakedPubKey := types.GetTweakedPubKey(pubKey, tweak)

			if !schnorr.Verify(sigBytes, sigHash, tweakedPubKey) {
				return errorsmod.Wrap(types.ErrInvalidSignature, "invalid schnorr signature")
			}

		case types.SigningType_SIGNING_TYPE_SCHNORR_WITH_COMMITMENT:
			if len(sigBytes) != types.SchnorrSignatureSize {
				return errorsmod.Wrap(types.ErrInvalidSignature, "invalid schnorr signature size")
			}

			nonce, _ := hex.DecodeString(options.Nonce)
			if !bytes.Equal(sigBytes[:32], nonce) {
				return errorsmod.Wrap(types.ErrInvalidSignature, "invalid signature r")
			}

			if !schnorr.Verify(sigBytes, sigHash, pubKey) {
				return errorsmod.Wrap(types.ErrInvalidSignature, "invalid schnorr signature")
			}

		case types.SigningType_SIGNING_TYPE_SCHNORR_ADAPTOR:
			if len(sigBytes) != types.SchnorrAdaptorSignatureSize {
				return errorsmod.Wrap(types.ErrInvalidSignature, "invalid schnorr adaptor signature size")
			}

			adaptorPoint, _ := hex.DecodeString(options.AdaptorPoint)
			if !adaptor.Verify(sigBytes, sigHash, pubKey, adaptorPoint) {
				return errorsmod.Wrap(types.ErrInvalidSignature, "invalid schnorr adaptor signature")
			}
		}
	}

	return nil
}
