package cli

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strconv"

	// "strings"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"

	"github.com/bitwaylabs/bitway/x/dlc/types"
)

var (
	FlagTriggered = "triggered"
)

// GetQueryCmd returns the cli query commands for this module
func GetQueryCmd(_ string) *cobra.Command {
	// Group yield queries under a subcommand
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      fmt.Sprintf("Querying commands for the %s module", types.ModuleName),
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(CmdQueryParams())
	cmd.AddCommand(CmdQueryDCM())
	cmd.AddCommand(CmdQueryDCMs())
	cmd.AddCommand(CmdQueryOracle())
	cmd.AddCommand(CmdQueryOracles())
	cmd.AddCommand(CmdQueryNonces())
	cmd.AddCommand(CmdQueryEvent())
	cmd.AddCommand(CmdQueryEvents())
	cmd.AddCommand(CmdQueryAttestation())
	cmd.AddCommand(CmdQueryAttestationByEvent())
	cmd.AddCommand(CmdQueryAttestations())
	cmd.AddCommand(CmdQueryOracleParticipantLiveness())
	// this line is used by starport scaffolding # 1

	return cmd
}

func CmdQueryParams() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "params",
		Short: "Query the parameters of the module",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			res, err := queryClient.Params(cmd.Context(), &types.QueryParamsRequest{})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}

func CmdQueryDCM() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dcm [id | pub key]",
		Short: "Query DCM by the given id or public key",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			id, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				_, err := hex.DecodeString(args[0])
				if err != nil {
					return fmt.Errorf("neither id nor pub key is provided")
				}

				res, err := queryClient.DCM(cmd.Context(), &types.QueryDCMRequest{PubKey: args[0]})
				if err != nil {
					return err
				}

				return clientCtx.PrintProto(res)
			}

			res, err := queryClient.DCM(cmd.Context(), &types.QueryDCMRequest{Id: id})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}

func CmdQueryDCMs() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dcms [status]",
		Short: "Query DCMs by the given status",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			status, err := strconv.ParseUint(args[0], 10, 32)
			if err != nil {
				return err
			}

			res, err := queryClient.DCMs(cmd.Context(), &types.QueryDCMsRequest{Status: types.DCMStatus(status)})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}

func CmdQueryOracle() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "oracle [id | pub key]",
		Short: "Query oracle by the given id or public key",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			id, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				_, err := hex.DecodeString(args[0])
				if err != nil {
					return fmt.Errorf("neither id nor pub key is provided")
				}

				res, err := queryClient.Oracle(cmd.Context(), &types.QueryOracleRequest{PubKey: args[0]})
				if err != nil {
					return err
				}

				return clientCtx.PrintProto(res)
			}

			res, err := queryClient.Oracle(cmd.Context(), &types.QueryOracleRequest{Id: id})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}

func CmdQueryOracles() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "oracles [status]",
		Short: "Query oracles by the given status",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			status, err := strconv.ParseUint(args[0], 10, 32)
			if err != nil {
				return err
			}

			res, err := queryClient.Oracles(cmd.Context(), &types.QueryOraclesRequest{Status: types.DLCOracleStatus(status)})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}

func CmdQueryNonces() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "nonces [oracle id]",
		Short: "Query all nonces of the given oracle",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			oracleId, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return err
			}

			res, err := queryClient.Nonces(cmd.Context(), &types.QueryNoncesRequest{OracleId: oracleId})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}

func CmdQueryEvent() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "event [id]",
		Short: "Query the event by the given id",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			id, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return err
			}

			res, err := queryClient.Event(cmd.Context(), &types.QueryEventRequest{Id: id})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}

func CmdQueryEvents() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "events [flag]",
		Short:   "Query events by the given status",
		Args:    cobra.NoArgs,
		Example: "events --triggered",
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			triggered, err := cmd.Flags().GetBool(FlagTriggered)
			if err != nil {
				return err
			}

			res, err := queryClient.Events(cmd.Context(), &types.QueryEventsRequest{Triggered: triggered})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	cmd.Flags().Bool(FlagTriggered, false, "Indicates if the events have been triggered")
	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}

func CmdQueryAttestation() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "attestation [id]",
		Short: "Query the attestation by the given id",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			id, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return err
			}

			res, err := queryClient.Attestation(cmd.Context(), &types.QueryAttestationRequest{Id: id})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}

func CmdQueryAttestationByEvent() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "attestation-by-event [event id]",
		Short: "Query the attestation by the given event id",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			eventId, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return err
			}

			res, err := queryClient.AttestationByEvent(cmd.Context(), &types.QueryAttestationByEventRequest{EventId: eventId})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}

func CmdQueryAttestations() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "attestations",
		Short: "Query all attestations",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			res, err := queryClient.Attestations(cmd.Context(), &types.QueryAttestationsRequest{})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}

func CmdQueryOracleParticipantLiveness() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "oracle-participant-liveness [ consensus pub key | liveness status (true|false)",
		Short: "Query oracle participant liveness with the consensus pub key or liveness status",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			alive, err := strconv.ParseBool(args[0])
			if err != nil {
				_, err := base64.StdEncoding.DecodeString(args[0])
				if err != nil {
					return fmt.Errorf("neither consensus pub key nor liveness status provided")
				}

				res, err := queryClient.OracleParticipantLiveness(cmd.Context(), &types.QueryOracleParticipantLivenessRequest{
					ConsensusPubkey: args[0],
				})
				if err != nil {
					return err
				}

				return clientCtx.PrintProto(res)
			}

			res, err := queryClient.OracleParticipantLiveness(cmd.Context(), &types.QueryOracleParticipantLivenessRequest{
				Alive: alive,
			})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
