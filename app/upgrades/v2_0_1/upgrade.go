package v2_0_1

import (
	"context"

	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	slashingkeeper "github.com/cosmos/cosmos-sdk/x/slashing/keeper"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
)

// UpgradeName is the upgrade version name
const UpgradeName = "v2.0.1"

// StoreUpgrades defines the storage upgrades
var StoreUpgrades = storetypes.StoreUpgrades{
	Deleted: []string{
		"wasm",
	},
}

// CreateUpgradeHandler creates the upgrade handler
func CreateUpgradeHandler(
	mm *module.Manager,
	configurator module.Configurator,
	stakingKeeper *stakingkeeper.Keeper,
	slashingKeeper *slashingkeeper.Keeper,
) upgradetypes.UpgradeHandler {
	return func(c context.Context, _ upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		ctx := sdk.UnwrapSDKContext(c)

		// unjail validators slashed during the hard-fork upgrade
		if err := unjail(ctx, stakingKeeper, slashingKeeper, validatorsToUnjail); err != nil {
			return vm, err
		}

		return mm.RunMigrations(ctx, configurator, vm)
	}
}
