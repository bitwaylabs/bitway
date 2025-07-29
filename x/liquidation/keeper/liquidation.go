package keeper

import (
	errorsmod "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"

	"github.com/bitwaylabs/bitway/x/liquidation/types"
)

// HandleLiquidation performs the liquidation handling
func (k Keeper) HandleLiquidation(ctx sdk.Context, liquidator string, liquidationId uint64, debtAmount sdk.Coin) (*types.LiquidationRecord, error) {
	if !k.HasLiquidation(ctx, liquidationId) {
		return nil, types.ErrLiquidationDoesNotExist
	}

	liquidation := k.GetLiquidation(ctx, liquidationId)
	if liquidation.Status != types.LiquidationStatus_LIQUIDATION_STATUS_LIQUIDATING {
		return nil, errorsmod.Wrap(types.ErrInvalidLiquidationStatus, "non liquidating status")
	}

	if debtAmount.Denom != liquidation.DebtAsset.Denom {
		return nil, errorsmod.Wrap(types.ErrInvalidAmount, "mismatched debt amount denom")
	}

	remainingCollateralAmount := liquidation.UnliquidatedCollateralAmount
	remainingDebtAmount := liquidation.DebtAmount.Sub(liquidation.LiquidatedDebtAmount)

	// check if there is no collateral or debt remaining
	if remainingCollateralAmount.Amount.IsZero() || remainingDebtAmount.Amount.IsZero() {
		return nil, errorsmod.Wrap(types.ErrInvalidAmount, "no collateral or debt remaining")
	}

	// minimum liquidation debt amount if the remaining debt amount is sufficient
	minLiquidationDebtAmount := liquidation.DebtAmount.Amount.Mul(sdkmath.NewInt(int64(k.MinLiquidationFactor(ctx)))).Quo(sdkmath.NewInt(1000))

	// check remaining debt amount
	if remainingDebtAmount.Amount.GTE(minLiquidationDebtAmount) && debtAmount.Amount.LT(minLiquidationDebtAmount) {
		return nil, errorsmod.Wrapf(types.ErrInvalidAmount, "liquidation debt amount must be greater than or equal to the minimum liquidation debt amount %s", minLiquidationDebtAmount)
	}

	if remainingDebtAmount.Amount.LT(minLiquidationDebtAmount) && debtAmount.Amount.LT(remainingDebtAmount.Amount) {
		return nil, errorsmod.Wrapf(types.ErrInvalidAmount, "liquidation debt amount must be greater than or equal to the remaining debt amount %s", remainingDebtAmount.Amount)
	}

	if remainingDebtAmount.IsLT(debtAmount) {
		debtAmount = remainingDebtAmount
	}

	currentPrice, err := k.GetPrice(ctx, types.GetPricePair(liquidation))
	if err != nil {
		return nil, types.ErrInvalidPrice
	}

	collateralDecimals := int(liquidation.CollateralAsset.Decimals)
	debtDecimals := int(liquidation.DebtAsset.Decimals)
	collateralIsBaseAsset := liquidation.CollateralAsset.IsBasePriceAsset

	// calculate collateral amount
	collateralAmount := types.GetCollateralAmount(debtAmount.Amount, debtDecimals, collateralDecimals, currentPrice, collateralIsBaseAsset)

	// check remaining collateral amount
	if remainingCollateralAmount.Amount.LT(collateralAmount) {
		collateralAmount = remainingCollateralAmount.Amount
		debtAmount.Amount = types.GetDebtAmount(collateralAmount, collateralDecimals, debtDecimals, currentPrice, collateralIsBaseAsset)
	}

	if err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, sdk.MustAccAddressFromBech32(liquidator), types.ModuleName, sdk.NewCoins(debtAmount)); err != nil {
		return nil, err
	}

	remainingCollateralAmount = remainingCollateralAmount.SubAmount(collateralAmount)

	// calculate bonus
	bonusAmountInDebt := debtAmount.Amount.Mul(sdkmath.NewInt(int64(k.LiquidationBonusFactor(ctx)))).Quo(sdkmath.NewInt(1000))
	bonusAmount := types.GetCollateralAmount(bonusAmountInDebt, debtDecimals, collateralDecimals, currentPrice, collateralIsBaseAsset)

	// check if there is left collateral for bonus
	if bonusAmount.GT(remainingCollateralAmount.Amount) {
		bonusAmount = remainingCollateralAmount.Amount
	}

	// check if the total received collateral amount is dust
	if types.IsDustOut(collateralAmount.Add(bonusAmount).Int64(), liquidator) {
		return nil, errorsmod.Wrapf(types.ErrInvalidAmount, "dust collateral amount %s", collateralAmount)
	}

	protocolLiquidationFee := bonusAmount.Mul(sdkmath.NewInt(int64(k.ProtocolLiquidationFeeFactor(ctx)))).Quo(sdkmath.NewInt(1000))

	liquidation.LiquidatedCollateralAmount = liquidation.LiquidatedCollateralAmount.AddAmount(collateralAmount).AddAmount(bonusAmount)
	liquidation.LiquidatedDebtAmount = liquidation.LiquidatedDebtAmount.Add(debtAmount)
	liquidation.LiquidationBonusAmount = liquidation.LiquidationBonusAmount.AddAmount(bonusAmount)
	liquidation.ProtocolLiquidationFee = liquidation.ProtocolLiquidationFee.AddAmount(protocolLiquidationFee)
	liquidation.UnliquidatedCollateralAmount = liquidation.UnliquidatedCollateralAmount.SubAmount(collateralAmount).SubAmount(bonusAmount)

	record := &types.LiquidationRecord{
		Id:               k.IncrementLiquidationRecordId(ctx),
		LiquidationId:    liquidationId,
		Liquidator:       liquidator,
		DebtAmount:       debtAmount,
		CollateralAmount: sdk.NewCoin(liquidation.CollateralAsset.Denom, collateralAmount),
		BonusAmount:      sdk.NewCoin(liquidation.CollateralAsset.Denom, bonusAmount.Sub(protocolLiquidationFee)),
		Time:             ctx.BlockTime(),
	}

	k.SetLiquidation(ctx, liquidation)
	k.SetLiquidationRecord(ctx, record)

	return record, nil
}

// GetLiquidationId gets the current liquidation id
func (k Keeper) GetLiquidationId(ctx sdk.Context) uint64 {
	store := ctx.KVStore(k.storeKey)

	bz := store.Get(types.LiquidationIdKey)
	if bz == nil {
		return 0
	}

	return sdk.BigEndianToUint64(bz)
}

// IncrementLiquidationId increments the liquidation id and returns the new id
func (k Keeper) IncrementLiquidationId(ctx sdk.Context) uint64 {
	store := ctx.KVStore(k.storeKey)

	id := k.GetLiquidationId(ctx) + 1
	store.Set(types.LiquidationIdKey, sdk.Uint64ToBigEndian(id))

	return id
}

// HasLiquidation returns true if the given liquidation exists, false otherwise
func (k Keeper) HasLiquidation(ctx sdk.Context, id uint64) bool {
	store := ctx.KVStore(k.storeKey)

	return store.Has(types.LiquidationKey(id))
}

// GetLiquidation gets the liquidation by the given id
func (k Keeper) GetLiquidation(ctx sdk.Context, id uint64) *types.Liquidation {
	store := ctx.KVStore(k.storeKey)

	bz := store.Get(types.LiquidationKey(id))
	var liquidation types.Liquidation
	k.cdc.MustUnmarshal(bz, &liquidation)

	return &liquidation
}

// SetLiquidation sets the given liquidation
func (k Keeper) SetLiquidation(ctx sdk.Context, liquidation *types.Liquidation) {
	store := ctx.KVStore(k.storeKey)

	bz := k.cdc.MustMarshal(liquidation)

	k.SetLiquidationStatus(ctx, liquidation.Id, liquidation.Status)

	store.Set(types.LiquidationKey(liquidation.Id), bz)
}

// SetLiquidationStatus sets the status store of the given liquidation
func (k Keeper) SetLiquidationStatus(ctx sdk.Context, id uint64, status types.LiquidationStatus) {
	store := ctx.KVStore(k.storeKey)

	if k.HasLiquidation(ctx, id) {
		k.RemoveLiquidationStatus(ctx, id)
	}

	store.Set(types.LiquidationByStatusKey(status, id), []byte{})
}

// RemoveLiquidationStatus removes the status store of the given liquidation
func (k Keeper) RemoveLiquidationStatus(ctx sdk.Context, id uint64) {
	store := ctx.KVStore(k.storeKey)

	liquidation := k.GetLiquidation(ctx, id)

	store.Delete(types.LiquidationByStatusKey(liquidation.Status, id))
}

// CreateLiquidation creates and returns the newly created liquidation
func (k Keeper) CreateLiquidation(ctx sdk.Context, liquidation *types.Liquidation) *types.Liquidation {
	// set the id
	liquidation.Id = k.IncrementLiquidationId(ctx)

	// initialize the unliquidated collateral amount
	liquidation.UnliquidatedCollateralAmount = liquidation.ActualCollateralAmount.Sub(liquidation.LiquidatedCollateralAmount).SubAmount(sdkmath.NewInt(types.LiquidationNetworkFeeReserve))

	// set the status to liquidating
	liquidation.Status = types.LiquidationStatus_LIQUIDATION_STATUS_LIQUIDATING

	k.SetLiquidation(ctx, liquidation)

	return liquidation
}

// GetAllLiquidations gets all liquidations
func (k Keeper) GetAllLiquidations(ctx sdk.Context) []*types.Liquidation {
	liquidations := make([]*types.Liquidation, 0)

	k.IterateLiquidations(ctx, func(liquidation *types.Liquidation) (stop bool) {
		liquidations = append(liquidations, liquidation)
		return false
	})

	return liquidations
}

// GetLiquidationsByStatus gets liquidations by the given status
func (k Keeper) GetLiquidationsByStatus(ctx sdk.Context, status types.LiquidationStatus) []*types.Liquidation {
	liquidations := make([]*types.Liquidation, 0)

	k.IterateLiquidationsByStatus(ctx, status, func(liquidation *types.Liquidation) (stop bool) {
		liquidations = append(liquidations, liquidation)
		return false
	})

	return liquidations
}

// GetLiquidationsByStatusWithPagination gets the liquidations by the given status with pagination
func (k Keeper) GetLiquidationsByStatusWithPagination(ctx sdk.Context, status types.LiquidationStatus, pagination *query.PageRequest) ([]*types.Liquidation, *query.PageResponse, error) {
	store := ctx.KVStore(k.storeKey)
	liquidationStatusStore := prefix.NewStore(store, append(types.LiquidationByStatusKeyPrefix, sdk.Uint64ToBigEndian(uint64(status))...))

	var liquidations []*types.Liquidation

	pageRes, err := query.Paginate(liquidationStatusStore, pagination, func(key []byte, value []byte) error {
		id := sdk.BigEndianToUint64(key)
		liquidation := k.GetLiquidation(ctx, id)

		liquidations = append(liquidations, liquidation)

		return nil
	})
	if err != nil {
		return nil, nil, err
	}

	return liquidations, pageRes, nil
}

// GetLiquidationsWithPagination gets the liquidations with pagination
func (k Keeper) GetLiquidationsWithPagination(ctx sdk.Context, pagination *query.PageRequest) ([]*types.Liquidation, *query.PageResponse, error) {
	store := ctx.KVStore(k.storeKey)
	liquidationStore := prefix.NewStore(store, types.LiquidationKeyPrefix)

	var liquidations []*types.Liquidation

	pageRes, err := query.Paginate(liquidationStore, pagination, func(key []byte, value []byte) error {
		var liquidation types.Liquidation
		k.cdc.MustUnmarshal(value, &liquidation)

		liquidations = append(liquidations, &liquidation)

		return nil
	})
	if err != nil {
		return nil, nil, err
	}

	return liquidations, pageRes, nil
}

// IterateLiquidations iterates through all liquidations
func (k Keeper) IterateLiquidations(ctx sdk.Context, cb func(liquidation *types.Liquidation) (stop bool)) {
	store := ctx.KVStore(k.storeKey)

	iterator := storetypes.KVStorePrefixIterator(store, types.LiquidationKeyPrefix)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var liquidation types.Liquidation
		k.cdc.MustUnmarshal(iterator.Value(), &liquidation)

		if cb(&liquidation) {
			break
		}
	}
}

// IterateLiquidationsByStatus iterates through liquidations by the given status
func (k Keeper) IterateLiquidationsByStatus(ctx sdk.Context, status types.LiquidationStatus, cb func(liquidation *types.Liquidation) (stop bool)) {
	store := ctx.KVStore(k.storeKey)

	keyPrefix := append(types.LiquidationByStatusKeyPrefix, sdk.Uint64ToBigEndian(uint64(status))...)

	iterator := storetypes.KVStorePrefixIterator(store, keyPrefix)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		id := sdk.BigEndianToUint64(iterator.Key()[len(keyPrefix):])
		liquidation := k.GetLiquidation(ctx, id)

		if cb(liquidation) {
			break
		}
	}
}
