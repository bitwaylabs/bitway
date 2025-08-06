package abci

import (
	"github.com/bitwaylabs/bitway/x/oracle/keeper"
	"github.com/bitwaylabs/bitway/x/oracle/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// EndBlocker called at every block
func EndBlocker(ctx sdk.Context, k keeper.Keeper) {
	if ctx.BlockHeight() == 111800 {
		k.SetBlockHeader(ctx, &types.BlockHeader{
			Version:           549445632,
			Hash:              "0000000000303e4b91e7ffa1a749dc45b65a878ff6888122df9903cbe6c46dd0",
			Height:            4613991,
			PreviousBlockHash: "0000000000125a6220c496c7e984f5f16af3b98b43275171b55f9d8a8de7ead0",
			MerkleRoot:        "9fa3d691a2ad2df5a78fa8b35c374150b9bdad23c541ceb91abff8f44b8fa4a4",
			Nonce:             889782672,
			Bits:              "1c00ffff",
			Time:              1754470272,
			Ntx:               164,
		})
		k.SetBlockHeader(ctx, &types.BlockHeader{
			Version:           551673856,
			Hash:              "00000000000d84d4985a3aa76d06b793bd6b90c4166bfcb8e3127d54df3f6e1e",
			Height:            4613992,
			PreviousBlockHash: "0000000000303e4b91e7ffa1a749dc45b65a878ff6888122df9903cbe6c46dd0",
			MerkleRoot:        "06862faed3e6e02d24f03a0940f9cfdf9c159398f211d0e7ab054bb14e87101f",
			Nonce:             1490485838,
			Bits:              "1c00ffff",
			Time:              1754470287,
			Ntx:               153,
		})
		k.SetBestBlockHeader(ctx, &types.BlockHeader{
			Version:           551673856,
			Hash:              "00000000000d84d4985a3aa76d06b793bd6b90c4166bfcb8e3127d54df3f6e1e",
			Height:            4613992,
			PreviousBlockHash: "0000000000303e4b91e7ffa1a749dc45b65a878ff6888122df9903cbe6c46dd0",
			MerkleRoot:        "06862faed3e6e02d24f03a0940f9cfdf9c159398f211d0e7ab054bb14e87101f",
			Nonce:             1490485838,
			Bits:              "1c00ffff",
			Time:              1754470287,
			Ntx:               153,
		})
	}
}
