package btcbridge

import (
	"fmt"
	"sort"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bitwaylabs/bitway/x/btcbridge/keeper"
	"github.com/bitwaylabs/bitway/x/btcbridge/types"
)

// InitGenesis initializes the module's state from a provided genesis state.
func InitGenesis(ctx sdk.Context, k keeper.Keeper, genState types.GenesisState) {
	// this line is used by starport scaffolding # genesis/module/init
	k.SetParams(ctx, genState.Params)

	// import utxos
	for _, utxo := range genState.Utxos {
		k.SaveUTXO(ctx, utxo)
	}

	// set dkg requests
	for _, req := range genState.DkgRequests {
		k.SetDKGRequest(ctx, req)
		k.SetDKGRequestID(ctx, req.Id)
	}

	// set dkg completions
	for _, completion := range genState.DkgCompletions {
		k.SetDKGCompletionRequest(ctx, completion)
	}

	// set signing requests
	for _, req := range genState.SigningRequests {
		k.SetSigningRequest(ctx, req)
		k.IncrementSigningRequestSequence(ctx)
	}

	// set withdrawal requests
	for _, req := range genState.WithdrawRequests {
		k.SetWithdrawRequest(ctx, req)
		k.IncreaseWithdrawRequestSequence(ctx)
	}

	// set pending btc withdrawal requests
	for _, req := range genState.PendingBtcWithdrawRequests {
		k.AddToBtcWithdrawRequestQueue(ctx, req)
	}

	// set minted tx hashes
	for _, txHash := range genState.MintedTxHashes {
		k.AddToMintHistory(ctx, txHash)
	}

	// sort vaults and set the latest vault version
	if len(genState.Params.Vaults) > 0 {
		vaults := genState.Params.Vaults
		sort.Slice(vaults, func(i, j int) bool { return vaults[i].Version < vaults[j].Version })

		k.SetVaultVersion(ctx, vaults[len(vaults)-1].Version)
	}

	// set the rate limit
	k.SetRateLimit(ctx, k.NewRateLimit(ctx))

	// check if the module account exists
	moduleAcc := k.GetModuleAccount(ctx)
	if moduleAcc == nil {
		panic(fmt.Sprintf("%s module account has not been set", types.ModuleName))
	}

	// check if the fee sponsor module account exists
	moduleAcc = k.GetFeeSponsorAccount(ctx)
	if moduleAcc == nil {
		panic(fmt.Sprintf("%s module account has not been set", types.FeeSponsorName))
	}
}

// ExportGenesis returns the module's exported genesis
func ExportGenesis(ctx sdk.Context, k keeper.Keeper) *types.GenesisState {
	genesis := types.DefaultGenesis()
	genesis.Params = k.GetParams(ctx)
	genesis.Utxos = k.GetAllUTXOs(ctx)

	genesis.DkgRequests = k.GetAllDKGRequests(ctx)
	for _, dkgRequest := range genesis.DkgRequests {
		genesis.DkgCompletions = append(genesis.DkgCompletions, k.GetDKGCompletionRequests(ctx, dkgRequest.Id)...)
	}

	genesis.SigningRequests = k.GetAllSigningRequests(ctx)
	genesis.WithdrawRequests = k.GetAllWithdrawRequests(ctx)
	genesis.PendingBtcWithdrawRequests = k.GetPendingBtcWithdrawRequests(ctx, 0)

	genesis.MintedTxHashes = k.GetAllMintHistories(ctx)

	// this line is used by starport scaffolding # genesis/module/export

	return genesis
}
