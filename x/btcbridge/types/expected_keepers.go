package types

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	ibctransfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	ibcconnectiontypes "github.com/cosmos/ibc-go/v8/modules/core/03-connection/types"
	ibcchanneltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	ibcexported "github.com/cosmos/ibc-go/v8/modules/core/exported"

	oracletypes "github.com/bitwaylabs/bitway/x/oracle/types"
)

// AccountKeeper defines the expected account keeper interface
type AccountKeeper interface {
	GetAccount(ctx context.Context, addr sdk.AccAddress) sdk.AccountI

	GetModuleAddress(name string) sdk.AccAddress
	GetModuleAccount(ctx context.Context, moduleName string) sdk.ModuleAccountI
}

// BankKeeper defines the expected interface needed to retrieve account balances.
type BankKeeper interface {
	SpendableCoin(ctx context.Context, addr sdk.AccAddress, denom string) sdk.Coin
	SpendableCoins(ctx context.Context, addr sdk.AccAddress) sdk.Coins

	SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error

	SendCoinsFromModuleToModule(ctx context.Context, senderModule, recipientModule string, amt sdk.Coins) error
	SendCoinsFromAccountToModule(ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error
	SendCoins(ctx context.Context, fromAddr sdk.AccAddress, toAddr sdk.AccAddress, amt sdk.Coins) error
	SetDenomMetaData(ctx context.Context, denomMetaData banktypes.Metadata)

	MintCoins(ctx context.Context, moduleName string, amounts sdk.Coins) error
	BurnCoins(ctx context.Context, moduleName string, amounts sdk.Coins) error

	HasSupply(ctx context.Context, denom string) bool
	GetSupply(ctx context.Context, denom string) sdk.Coin
	GetBalance(ctx context.Context, addr sdk.AccAddress, denom string) sdk.Coin
}

// StakingKeeper defines the expected staking keeper used to retrieve validator (noalias)
type StakingKeeper interface {
	GetValidator(ctx context.Context, addr sdk.ValAddress) (stakingtypes.Validator, error)
	GetValidatorByConsAddr(ctx context.Context, consAddr sdk.ConsAddress) (stakingtypes.Validator, error)
}

// OracleKeeper defines the expected oracle interfaces
type OracleKeeper interface {
	HasBlockHeader(ctx sdk.Context, hash string) bool

	GetBestBlockHeader(ctx sdk.Context) *oracletypes.BlockHeader
	GetBlockHeader(ctx sdk.Context, hash string) *oracletypes.BlockHeader
	GetBlockHeaderByHeight(ctx sdk.Context, height int32) *oracletypes.BlockHeader
}

// IncentiveKeeper defines the expected incentive keeper
type IncentiveKeeper interface {
	DepositIncentiveEnabled(ctx sdk.Context) bool
	WithdrawIncentiveEnabled(ctx sdk.Context) bool

	DistributeDepositReward(ctx sdk.Context, addr string) error
	DistributeWithdrawReward(ctx sdk.Context, addr string) error
}

// TSSKeeper defines the expected TSS keeper interfaces
type TSSKeeper interface {
	AllowedDKGParticipants(ctx sdk.Context) []string
}

// IBCClientKeeper defines the expected IBC client keeper
type IBCClientKeeper interface {
	GetClientState(ctx sdk.Context, clientID string) (ibcexported.ClientState, bool)
}

// IBCConnectionKeeper defines the expected IBC connection keeper
type IBCConnectionKeeper interface {
	GetConnection(ctx sdk.Context, connectionID string) (ibcconnectiontypes.ConnectionEnd, bool)
	GetTimestampAtHeight(ctx sdk.Context, connection ibcconnectiontypes.ConnectionEnd, height ibcexported.Height) (uint64, error)
}

// IBCChannelKeeper defines the expected IBC channel keeper
type IBCChannelKeeper interface {
	GetChannel(ctx sdk.Context, srcPort, srcChan string) (channel ibcchanneltypes.Channel, found bool)
}

// IBCTransferKeeper defines the expected IBC transfer interfaces
type IBCTransferKeeper interface {
	GetPort(ctx sdk.Context) string

	GetTotalEscrowForDenom(ctx sdk.Context, denom string) sdk.Coin
	SetTotalEscrowForDenom(ctx sdk.Context, coin sdk.Coin)

	Transfer(goCtx context.Context, msg *ibctransfertypes.MsgTransfer) (*ibctransfertypes.MsgTransferResponse, error)
}
