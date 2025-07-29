package cli

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bitwaylabs/bitway/x/lending/types"
)

var DefaultRelativePacketTimeoutTimestamp = uint64((time.Duration(10) * time.Minute).Nanoseconds())

const (
	// flagPacketTimeoutTimestamp = "packet-timeout-timestamp"
	listSeparator = ","
)

// GetTxCmd returns the transaction commands for this module
func GetTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      fmt.Sprintf("%s transactions subcommands", types.ModuleName),
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(CmdAddLiquidity())
	cmd.AddCommand(CmdRemoveLiquidity())
	cmd.AddCommand(CmdApply())
	cmd.AddCommand(CmdSubmitCets())
	cmd.AddCommand(CmdSubmitDepositTransaction())
	cmd.AddCommand(CmdRedeem())
	cmd.AddCommand(CmdRepay())

	return cmd
}

func CmdAddLiquidity() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add-liquidity [pool id] [amount]",
		Short: "Add liquidity to the specified lending pool",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			amount, err := sdk.ParseCoinNormalized(args[1])
			if err != nil {
				return err
			}

			msg := types.NewMsgAddLiquidity(
				clientCtx.GetFromAddress().String(),
				args[0],
				amount,
			)

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

func CmdRemoveLiquidity() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove-liquidity [yToken amount]",
		Short: "Remove liquidity by the specified yToken amount",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			yTokens, err := sdk.ParseCoinNormalized(args[0])
			if err != nil {
				return err
			}

			msg := types.NewMsgRemoveLiquidity(
				clientCtx.GetFromAddress().String(),
				yTokens,
			)

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

func CmdApply() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "apply [borrower pub key] [borrower auth pub key] [pool id] [borrow amount] [maturity] [dcm id] [referrer]",
		Short: "Apply loan with the related params",
		Args:  cobra.RangeArgs(6, 7),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			borrowAmount, err := sdk.ParseCoinNormalized(args[3])
			if err != nil {
				return err
			}

			maturity, err := strconv.ParseInt(args[4], 10, 64)
			if err != nil {
				return err
			}

			dcmId, err := strconv.ParseUint(args[5], 10, 64)
			if err != nil {
				return err
			}

			referrer := ""
			if len(args) == 7 {
				referrer = args[6]
			}

			msg := types.NewMsgApply(
				clientCtx.GetFromAddress().String(),
				args[0],
				args[1],
				args[2],
				borrowAmount,
				maturity,
				dcmId,
				referrer,
			)

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

func CmdSubmitCets() *cobra.Command {
	cmd := &cobra.Command{
		Use: `submit-cets [loan id] [deposit txs] [liquidation cet] [liquidation adaptor signatures] 
		[default liquidation adaptor signatures] [repayment cet] [repayment signatures]`,
		Short: "Submit the related cets of the given loan",
		Args:  cobra.ExactArgs(7),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := types.NewMsgSubmitCets(
				clientCtx.GetFromAddress().String(),
				args[0],
				strings.Split(args[1], listSeparator),
				args[2],
				strings.Split(args[3], listSeparator),
				strings.Split(args[4], listSeparator),
				args[5],
				strings.Split(args[6], listSeparator),
			)

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

func CmdSubmitDepositTransaction() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "submit-deposit-tx [vault] [deposit tx] [block hash] [proof]",
		Short: "Submit the deposit tx along with the corresponding vault and proof",
		Args:  cobra.ExactArgs(4),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := types.NewMsgSubmitDepositTransaction(
				clientCtx.GetFromAddress().String(),
				args[0],
				args[1],
				args[2],
				strings.Split(args[3], listSeparator),
			)

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

func CmdRedeem() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "redeem [loan id] [tx] [signatures]",
		Short: "Redeem one or more collateral(s) along with the redemption tx and self-signatures",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := types.NewMsgRedeem(
				clientCtx.GetFromAddress().String(),
				args[0],
				args[1],
				strings.Split(args[2], listSeparator),
			)

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

func CmdRepay() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "repay [loan id]",
		Short: "Repay the given loan",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := types.NewMsgRepay(
				clientCtx.GetFromAddress().String(),
				args[0],
			)

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)

	return cmd
}
