package v2

import (
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bitwaylabs/bitway/x/btcbridge/types"
)

// MigrateStore migrates the x/btcbridge module state from the consensus version 1 to
// version 2
func MigrateStore(ctx sdk.Context, storeKey storetypes.StoreKey, cdc codec.BinaryCodec) error {
	migrateParams(ctx, storeKey, cdc)

	return nil
}

// migrateParams performs the params migration
func migrateParams(ctx sdk.Context, storeKey storetypes.StoreKey, cdc codec.BinaryCodec) {
	store := ctx.KVStore(storeKey)

	// get current params
	var paramsV1 types.ParamsV1
	bz := store.Get(types.ParamsStoreKey)
	cdc.MustUnmarshal(bz, &paramsV1)

	// build new params
	params := &types.Params{
		DepositConfirmationDepth:  types.DefaultDepositConfirmationDepth,
		WithdrawConfirmationDepth: types.DefaultWithdrawConfirmationDepth,
		MaxAcceptableBlockDepth:   paramsV1.MaxAcceptableBlockDepth,
		BtcVoucherDenom:           paramsV1.BtcVoucherDenom,
		DepositEnabled:            paramsV1.DepositEnabled,
		WithdrawEnabled:           paramsV1.WithdrawEnabled,
		TrustedNonBtcRelayers:     paramsV1.TrustedNonBtcRelayers,
		TrustedFeeProviders:       paramsV1.TrustedFeeProviders,
		FeeRateValidityPeriod:     paramsV1.FeeRateValidityPeriod,
		Vaults:                    paramsV1.Vaults,
		WithdrawParams:            paramsV1.WithdrawParams,
		ProtocolLimits:            paramsV1.ProtocolLimits,
		ProtocolFees:              paramsV1.ProtocolFees,
		TssParams:                 paramsV1.TssParams,
		RateLimitParams:           paramsV1.RateLimitParams,
		IbcParams:                 paramsV1.IbcParams,
		FeeSponsorshipParams:      paramsV1.FeeSponsorshipParams,
	}

	bz = cdc.MustMarshal(params)
	store.Set(types.ParamsStoreKey, bz)
}
