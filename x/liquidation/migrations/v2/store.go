package v2

import (
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bitwaylabs/bitway/x/liquidation/types"
)

// MigrateStore migrates the x/liquidation module state from the consensus version 1 to
// version 2
func MigrateStore(ctx sdk.Context, storeKey storetypes.StoreKey, cdc codec.BinaryCodec) error {
	migrateLiquidations(ctx, storeKey, cdc)

	return nil
}

// migrateLiquidations performs the liquidation migration
func migrateLiquidations(ctx sdk.Context, storeKey storetypes.StoreKey, cdc codec.BinaryCodec) {
	store := ctx.KVStore(storeKey)

	iterator := storetypes.KVStorePrefixIterator(store, types.LiquidationKeyPrefix)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var liquidationV1 types.LiquidationV1
		cdc.MustUnmarshal(iterator.Value(), &liquidationV1)

		liquidation := &types.Liquidation{
			Id:                               liquidationV1.Id,
			LoanId:                           liquidationV1.LoanId,
			Debtor:                           liquidationV1.Debtor,
			DCM:                              liquidationV1.DCM,
			CollateralAmount:                 liquidationV1.CollateralAmount,
			ActualCollateralAmount:           liquidationV1.ActualCollateralAmount,
			DebtAmount:                       liquidationV1.DebtAmount,
			CollateralAsset:                  liquidationV1.CollateralAsset,
			DebtAsset:                        liquidationV1.DebtAsset,
			LiquidationPrice:                 liquidationV1.LiquidationPrice,
			LiquidationTime:                  liquidationV1.LiquidationTime,
			LiquidatedCollateralAmount:       liquidationV1.LiquidatedCollateralAmount,
			LiquidatedDebtAmount:             liquidationV1.LiquidatedDebtAmount,
			LiquidationBonusAmount:           liquidationV1.LiquidationBonusAmount,
			ProtocolLiquidationFee:           liquidationV1.ProtocolLiquidationFee,
			UnliquidatedCollateralAmount:     liquidationV1.UnliquidatedCollateralAmount,
			AccruedInterestDuringLiquidation: liquidationV1.AccruedInterestDuringLiquidation,
			LiquidationCet:                   liquidationV1.LiquidationCet,
			SettlementTx:                     liquidationV1.SettlementTx,
			SettlementTxId:                   liquidationV1.SettlementTxId,
			Status:                           liquidationV1.Status,
		}

		store.Set(types.LiquidationKey(liquidation.Id), cdc.MustMarshal(liquidation))
	}
}
