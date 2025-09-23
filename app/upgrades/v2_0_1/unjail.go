package v2_0_1

import (
	"time"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	slashingkeeper "github.com/cosmos/cosmos-sdk/x/slashing/keeper"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// validators to be unjailed
var validatorsToUnjail = []string{
	"bitwayvaloper1qqr3wzgzqvpqxycjzyr3kzcequxskxqxzydsjqgeqygq2xqxqgtpuskm8pm", // Ping Pub
	"bitwayvaloper1qqwqursez5x3xygyzg8pu9scruzs59qsqcxpqqqepy2qyrckzypqg8dvpkx", // HashKey Cloud
}

// unjail unjails the given validators
func unjail(ctx sdk.Context, stakingKeeper *stakingkeeper.Keeper, slashingKeeper *slashingkeeper.Keeper, validatorAddresses []string) error {
	for _, validatorAddress := range validatorAddresses {
		valAddr, err := sdk.ValAddressFromBech32(validatorAddress)
		if err != nil {
			return err
		}

		// get validator
		validator, err := stakingKeeper.Validator(ctx, valAddr)
		if err != nil {
			return err
		}

		if validator == nil {
			return errorsmod.Wrap(stakingtypes.ErrNoValidatorFound, validatorAddress)
		}

		consAddr, err := validator.GetConsAddr()
		if err != nil {
			return err
		}

		// get the signing info
		signingInfo, err := slashingKeeper.GetValidatorSigningInfo(ctx, consAddr)
		if err != nil {
			return err
		}

		// update signing info
		signingInfo.JailedUntil = time.Unix(0, 0)
		signingInfo.Tombstoned = false
		slashingKeeper.SetValidatorSigningInfo(ctx, consAddr, signingInfo)

		// unjail
		if err := slashingKeeper.Unjail(ctx, valAddr); err != nil {
			return err
		}
	}

	return nil
}
