package keeper

import (
	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bitwaylabs/bitway/x/dlc/types"
)

type Keeper struct {
	cdc      codec.BinaryCodec
	storeKey storetypes.StoreKey
	memKey   storetypes.StoreKey

	oracleKeeper  types.OracleKeeper
	stakingKeeper types.StakingKeeper
	tssKeeper     types.TSSKeeper

	authority string
}

func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey,
	memKey storetypes.StoreKey,
	oracleKeeper types.OracleKeeper,
	stakingKeeper types.StakingKeeper,
	tssKeeper types.TSSKeeper,
	authority string,
) Keeper {
	k := Keeper{
		cdc:           cdc,
		storeKey:      storeKey,
		memKey:        memKey,
		oracleKeeper:  oracleKeeper,
		stakingKeeper: stakingKeeper,
		tssKeeper:     tssKeeper,
		authority:     authority,
	}

	// register DKG completion received handler
	tssKeeper.RegisterDKGCompletionReceivedHandler(types.ModuleName, k.DKGCompletionReceivedHandler)

	// register DKG request completed handler
	tssKeeper.RegisterDKGRequestCompletedHandler(types.ModuleName, k.DKGCompletedHandler)

	// register DKG request timeout handler
	tssKeeper.RegisterDKGRequestTimeoutHandler(types.ModuleName, k.DKGTimeoutHandler)

	// register signing request completed handler
	tssKeeper.RegisterSigningRequestCompletedHandler(types.ModuleName, k.SigningCompletedHandler)

	return k
}

func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	return sdkCtx.Logger().With("module", "x/"+types.ModuleName)
}

func (k Keeper) SetParams(ctx sdk.Context, params types.Params) {
	store := ctx.KVStore(k.storeKey)

	bz := k.cdc.MustMarshal(&params)

	store.Set(types.ParamsKey, bz)
}

func (k Keeper) GetParams(ctx sdk.Context) types.Params {
	store := ctx.KVStore(k.storeKey)

	var params types.Params
	bz := store.Get(types.ParamsKey)
	k.cdc.MustUnmarshal(bz, &params)

	return params
}

func (k Keeper) TSSKeeper() types.TSSKeeper {
	return k.tssKeeper
}
