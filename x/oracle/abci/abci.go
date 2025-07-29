package abci

import (
	"github.com/bitwaylabs/bitway/x/oracle/keeper"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// EndBlocker called at every block
func EndBlocker(ctx sdk.Context, k keeper.Keeper) {

}
