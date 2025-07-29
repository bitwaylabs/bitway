package keeper

import (
	"context"
	"fmt"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"github.com/bitwaylabs/bitway/x/liquidation/types"
)

type msgServer struct {
	Keeper
}

// Liquidate implements types.MsgServer.
func (m msgServer) Liquidate(goCtx context.Context, msg *types.MsgLiquidate) (*types.MsgLiquidateResponse, error) {
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	record, err := m.Keeper.HandleLiquidation(ctx, msg.Liquidator, msg.LiquidationId, msg.DebtAmount)
	if err != nil {
		return nil, err
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeLiquidate,
			sdk.NewAttribute(types.AttributeKeyLiquidator, msg.Liquidator),
			sdk.NewAttribute(types.AttributeKeyLiquidationId, fmt.Sprintf("%d", msg.LiquidationId)),
			sdk.NewAttribute(types.AttributeKeyLiquidationRecordId, fmt.Sprintf("%d", record.Id)),
			sdk.NewAttribute(types.AttributeKeyDebtAmount, record.DebtAmount.String()),
			sdk.NewAttribute(types.AttributeKeyCollateralAmount, record.CollateralAmount.String()),
			sdk.NewAttribute(types.AttributeKeyBonusAmount, record.BonusAmount.String()),
		),
	)

	return &types.MsgLiquidateResponse{}, nil
}

// UpdateParams updates the module params.
func (m msgServer) UpdateParams(goCtx context.Context, msg *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	if m.authority != msg.Authority {
		return nil, errorsmod.Wrapf(govtypes.ErrInvalidSigner, "invalid authority; expected %s, got %s", m.authority, msg.Authority)
	}

	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	m.SetParams(ctx, msg.Params)

	return &types.MsgUpdateParamsResponse{}, nil
}

// NewMsgServerImpl returns an implementation of the MsgServer interface
// for the provided Keeper.
func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

var _ types.MsgServer = msgServer{}
