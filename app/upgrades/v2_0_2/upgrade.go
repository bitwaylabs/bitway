package v2_0_2

import (
	"context"

	hyperlanetypes "github.com/bcp-innovations/hyperlane-cosmos/x/core/types"
	hyperlanewarptypes "github.com/bcp-innovations/hyperlane-cosmos/x/warp/types"

	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/cosmos/cosmos-sdk/types/module"
)

// UpgradeName is the upgrade version name
const UpgradeName = "v2.0.2"

// StoreUpgrades defines the storage upgrades
var StoreUpgrades = storetypes.StoreUpgrades{
	Added: []string{
		hyperlanetypes.ModuleName,
		hyperlanewarptypes.ModuleName,
	},
}

// CreateUpgradeHandler creates the upgrade handler
func CreateUpgradeHandler(
	mm *module.Manager,
	configurator module.Configurator,
) upgradetypes.UpgradeHandler {
	return func(ctx context.Context, _ upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		return mm.RunMigrations(ctx, configurator, vm)
	}
}
