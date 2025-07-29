package dlc_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	keepertest "github.com/bitwaylabs/bitway/testutil/keeper"
	"github.com/bitwaylabs/bitway/testutil/nullify"
	dlc "github.com/bitwaylabs/bitway/x/dlc/module"
	"github.com/bitwaylabs/bitway/x/dlc/types"
)

func TestGenesis(t *testing.T) {
	mnemonic := "sunny bamboo garlic fold reopen exile letter addict forest vessel square lunar shell number deliver cruise calm artist fire just kangaroo suit wheel extend"
	println(mnemonic)

	genesisState := types.GenesisState{
		Params: types.DefaultParams(),

		// this line is used by starport scaffolding # genesis/test/state
	}

	k, ctx := keepertest.DLCKeeper(t)
	dlc.InitGenesis(ctx, k, genesisState)
	got := dlc.ExportGenesis(ctx, k)
	require.NotNil(t, got)

	nullify.Fill(&genesisState)
	nullify.Fill(got)

	// this line is used by starport scaffolding # genesis/test/assert
}
