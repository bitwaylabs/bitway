package keeper

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/btcsuite/btcd/btcutil/psbt"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"github.com/bitwaylabs/bitway/bitcoin/crypto/schnorr"
	"github.com/bitwaylabs/bitway/x/btcbridge/types"
	tsstypes "github.com/bitwaylabs/bitway/x/tss/types"
)

type msgServer struct {
	Keeper
}

// UpdateTrustedNonBtcRelayers implements types.MsgServer.
func (m msgServer) UpdateTrustedNonBtcRelayers(goCtx context.Context, msg *types.MsgUpdateTrustedNonBtcRelayers) (*types.MsgUpdateTrustedNonBtcRelayersResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	if !m.IsTrustedNonBtcRelayer(ctx, msg.Sender) {
		return nil, types.ErrUntrustedNonBtcRelayer
	}

	// update non-btc relayers
	params := m.GetParams(ctx)
	params.TrustedNonBtcRelayers = msg.Relayers
	m.SetParams(ctx, params)

	return &types.MsgUpdateTrustedNonBtcRelayersResponse{}, nil
}

// UpdateTrustedFeeProviders implements types.MsgServer.
func (m msgServer) UpdateTrustedFeeProviders(goCtx context.Context, msg *types.MsgUpdateTrustedFeeProviders) (*types.MsgUpdateTrustedFeeProvidersResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	if !m.IsTrustedFeeProvider(ctx, msg.Sender) {
		return nil, types.ErrUntrustedFeeProvider
	}

	// update fee providers
	params := m.GetParams(ctx)
	params.TrustedFeeProviders = msg.FeeProviders
	m.SetParams(ctx, params)

	return &types.MsgUpdateTrustedFeeProvidersResponse{}, nil
}

// SubmitDepositTransaction implements types.MsgServer.
// No Permission check required for this message
// Since everyone can submit a transaction to mint voucher tokens
// This message is usually sent by relayers
func (m msgServer) SubmitDepositTransaction(goCtx context.Context, msg *types.MsgSubmitDepositTransaction) (*types.MsgSubmitDepositTransactionResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	if !m.DepositEnabled(ctx) {
		return nil, types.ErrDepositNotEnabled
	}

	txHash, recipient, err := m.ProcessBitcoinDepositTransaction(ctx, msg)
	if err != nil {
		return nil, err
	}

	// Emit Events
	m.EmitEvent(ctx, msg.Sender,
		sdk.NewAttribute("blockhash", msg.Blockhash),
		sdk.NewAttribute("txid", txHash.String()),
		sdk.NewAttribute("recipient", recipient.EncodeAddress()),
	)

	return &types.MsgSubmitDepositTransactionResponse{}, nil
}

// SubmitWithdrawTransaction implements types.MsgServer.
// No Permission check required for this message
// This message is usually sent by relayers
func (m msgServer) SubmitWithdrawTransaction(goCtx context.Context, msg *types.MsgSubmitWithdrawTransaction) (*types.MsgSubmitWithdrawTransactionResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	txHash, err := m.ProcessBitcoinWithdrawTransaction(ctx, msg)
	if err != nil {
		return nil, err
	}

	// Emit Events
	m.EmitEvent(ctx, msg.Sender,
		sdk.NewAttribute("blockhash", msg.Blockhash),
		sdk.NewAttribute("txid", txHash.String()),
	)

	return &types.MsgSubmitWithdrawTransactionResponse{}, nil
}

// SubmitFeeRate submits the bitcoin network fee rate
func (m msgServer) SubmitFeeRate(goCtx context.Context, msg *types.MsgSubmitFeeRate) (*types.MsgSubmitFeeRateResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	if !m.IsTrustedFeeProvider(ctx, msg.Sender) {
		return nil, types.ErrUntrustedFeeProvider
	}

	m.SetFeeRate(ctx, msg.FeeRate)

	// Emit Events
	m.EmitEvent(ctx, msg.Sender,
		sdk.NewAttribute("fee_rate", fmt.Sprintf("%d", msg.FeeRate)),
		sdk.NewAttribute("height", fmt.Sprintf("%d", ctx.BlockHeight())),
	)

	return &types.MsgSubmitFeeRateResponse{}, nil
}

// WithdrawToBitcoin withdraws the asset to the bitcoin chain.
func (m msgServer) WithdrawToBitcoin(goCtx context.Context, msg *types.MsgWithdrawToBitcoin) (*types.MsgWithdrawToBitcoinResponse, error) {
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	if !m.WithdrawEnabled(ctx) {
		return nil, types.ErrWithdrawNotEnabled
	}

	sender := sdk.MustAccAddressFromBech32(msg.Sender)

	amount, err := sdk.ParseCoinNormalized(msg.Amount)
	if err != nil {
		return nil, err
	}

	// handle rate limit
	if err := m.HandleRateLimit(ctx, msg.Sender, amount); err != nil {
		return nil, err
	}

	if m.ProtocolWithdrawFeeEnabled(ctx) {
		// deduct the protocol fee and get the actual withdrawal amount
		amount, err = m.HandleWithdrawProtocolFee(ctx, sender, amount)
		if err != nil {
			return nil, err
		}
	}

	withdrawRequest, err := m.HandleWithdrawal(ctx, msg.Sender, amount)
	if err != nil {
		return nil, err
	}

	// Emit events
	m.EmitEvent(ctx, msg.Sender,
		sdk.NewAttribute("amount", amount.String()),
		sdk.NewAttribute("sequence", fmt.Sprintf("%d", withdrawRequest.Sequence)),
		sdk.NewAttribute("txid", withdrawRequest.Txid),
	)

	return &types.MsgWithdrawToBitcoinResponse{}, nil
}

// SubmitSignatures submits the signatures of the signing request.
func (m msgServer) SubmitSignatures(goCtx context.Context, msg *types.MsgSubmitSignatures) (*types.MsgSubmitSignaturesResponse, error) {
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	if !m.HasSigningRequestByTxHash(ctx, msg.Txid) {
		return nil, types.ErrSigningRequestDoesNotExist
	}

	signingRequest := m.GetSigningRequestByTxHash(ctx, msg.Txid)
	if signingRequest.Status != types.SigningStatus_SIGNING_STATUS_PENDING {
		// return no error
		return nil, nil
	}

	p, _ := psbt.NewFromRawBytes(bytes.NewReader([]byte(signingRequest.Psbt)), true)

	if len(msg.Signatures) != len(p.Inputs) {
		return nil, errorsmod.Wrap(types.ErrInvalidSignatures, "mismatched signature number")
	}

	for i, input := range p.Inputs {
		sigHash, err := types.CalcTaprootSigHash(p, i, input.SighashType)
		if err != nil {
			return nil, err
		}

		pubKeyBytes := input.WitnessUtxo.PkScript[2:34]
		sigBytes, _ := hex.DecodeString(msg.Signatures[i])

		if !schnorr.Verify(sigBytes, sigHash, pubKeyBytes) {
			return nil, types.ErrInvalidSignature
		}

		p.Inputs[i].TaprootKeySpendSig = sigBytes
	}

	if err := psbt.MaybeFinalizeAll(p); err != nil {
		return nil, err
	}

	psbtB64, err := p.B64Encode()
	if err != nil {
		return nil, types.ErrFailToSerializePsbt
	}

	// update the signing request
	signingRequest.Psbt = psbtB64
	signingRequest.Status = types.SigningStatus_SIGNING_STATUS_BROADCASTED

	m.SetSigningRequest(ctx, signingRequest)

	return &types.MsgSubmitSignaturesResponse{}, nil
}

// ConsolidateVaults performs the UTXO consolidation for the given vaults.
func (m msgServer) ConsolidateVaults(goCtx context.Context, msg *types.MsgConsolidateVaults) (*types.MsgConsolidateVaultsResponse, error) {
	if m.authority != msg.Authority {
		return nil, errorsmod.Wrapf(govtypes.ErrInvalidSigner, "invalid authority; expected %s, got %s", m.authority, msg.Authority)
	}

	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	if err := m.Keeper.ConsolidateVaults(ctx, msg.VaultVersion, msg.BtcConsolidation, msg.RunesConsolidations); err != nil {
		return nil, err
	}

	return &types.MsgConsolidateVaultsResponse{}, nil
}

// InitiateDKG initiates the DKG request.
func (m msgServer) InitiateDKG(goCtx context.Context, msg *types.MsgInitiateDKG) (*types.MsgInitiateDKGResponse, error) {
	if m.authority != msg.Authority {
		return nil, errorsmod.Wrapf(govtypes.ErrInvalidSigner, "invalid authority; expected %s, got %s", m.authority, msg.Authority)
	}

	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	req, err := m.Keeper.InitiateDKG(ctx, msg.Participants, msg.Threshold, msg.VaultTypes, msg.EnableTransfer, msg.TargetUtxoNum)
	if err != nil {
		return nil, err
	}

	// Emit events
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeInitiateDKG,
			sdk.NewAttribute(types.AttributeKeyId, fmt.Sprintf("%d", req.Id)),
			sdk.NewAttribute(types.AttributeKeyParticipants, strings.Join(types.GetParticipantPubKeys(req.Participants), types.AttributeValueSeparator)),
			sdk.NewAttribute(types.AttributeKeyThreshold, fmt.Sprintf("%d", req.Threshold)),
			sdk.NewAttribute(types.AttributeKeyBatchSize, fmt.Sprintf("%d", len(req.VaultTypes))),
			sdk.NewAttribute(types.AttributeKeyExpirationTime, req.Expiration.String()),
		),
	)

	return &types.MsgInitiateDKGResponse{}, nil
}

// CompleteDKG completes the DKG request by the DKG participant
func (m msgServer) CompleteDKG(goCtx context.Context, msg *types.MsgCompleteDKG) (*types.MsgCompleteDKGResponse, error) {
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	req := &types.DKGCompletionRequest{
		Id:              msg.Id,
		Sender:          msg.Sender,
		Vaults:          msg.Vaults,
		ConsensusPubkey: msg.ConsensusPubkey,
		Signature:       msg.Signature,
	}

	if err := m.Keeper.CompleteDKG(ctx, req); err != nil {
		return nil, err
	}

	// Emit events
	m.EmitEvent(ctx, msg.Sender,
		sdk.NewAttribute("id", fmt.Sprintf("%d", msg.Id)),
	)

	return &types.MsgCompleteDKGResponse{}, nil
}

// Refresh refreshes the key shares
func (m msgServer) Refresh(goCtx context.Context, msg *types.MsgRefresh) (*types.MsgRefreshResponse, error) {
	if m.authority != msg.Authority {
		return nil, errorsmod.Wrapf(govtypes.ErrInvalidSigner, "invalid authority; expected %s, got %s", m.authority, msg.Authority)
	}

	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	for i, dkgId := range msg.DkgIds {
		if !m.HasDKGRequest(ctx, dkgId) {
			return nil, errorsmod.Wrapf(types.ErrDKGRequestDoesNotExist, "dkg %d", dkgId)
		}

		dkgRequest := m.GetDKGRequest(ctx, dkgId)
		if dkgRequest.Status != types.DKGRequestStatus_DKG_REQUEST_STATUS_COMPLETED {
			return nil, errorsmod.Wrapf(types.ErrInvalidDKGStatus, "dkg %d not completed", dkgId)
		}

		remainingParticipantNum := len(dkgRequest.Participants) - len(msg.RemovedParticipants)
		if remainingParticipantNum < tsstypes.MinDKGParticipantNum {
			return nil, errorsmod.Wrapf(types.ErrInvalidParticipants, "remaining participants %d cannot be less than min participants %d", remainingParticipantNum, tsstypes.MinDKGParticipantNum)
		}

		for _, p := range msg.RemovedParticipants {
			if !types.ParticipantExists(dkgRequest.Participants, p) {
				return nil, errorsmod.Wrapf(types.ErrInvalidParticipants, "participant %s does not exist for dkg %d", p, dkgId)
			}
		}

		if msg.Thresholds[i] > uint32(remainingParticipantNum) {
			return nil, errorsmod.Wrapf(types.ErrInvalidThresholds, "threshold %d cannot be greater than participants %d for dkg %d", msg.Thresholds[i], remainingParticipantNum, dkgId)
		}

		m.InitiateRefreshingRequest(ctx, dkgId, msg.RemovedParticipants, msg.Thresholds[i], msg.TimeoutDuration)
	}

	return &types.MsgRefreshResponse{}, nil
}

// CompleteRefreshing completes the refreshing request by the participant
func (m msgServer) CompleteRefreshing(goCtx context.Context, msg *types.MsgCompleteRefreshing) (*types.MsgCompleteRefreshingResponse, error) {
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	if err := m.Keeper.CompleteRefreshing(ctx, msg.Sender, msg.Id, msg.ConsensusPubkey, msg.Signature); err != nil {
		return nil, err
	}

	// Emit events
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeCompleteRefreshing,
			sdk.NewAttribute(types.AttributeKeySender, msg.Sender),
			sdk.NewAttribute(types.AttributeKeyId, fmt.Sprintf("%d", msg.Id)),
			sdk.NewAttribute(types.AttributeKeyParticipant, msg.ConsensusPubkey),
		),
	)

	return &types.MsgCompleteRefreshingResponse{}, nil
}

// TransferVault performs the vault asset transfer from the source version to the destination version
func (m msgServer) TransferVault(goCtx context.Context, msg *types.MsgTransferVault) (*types.MsgTransferVaultResponse, error) {
	if m.authority != msg.Authority {
		return nil, errorsmod.Wrapf(govtypes.ErrInvalidSigner, "invalid authority; expected %s, got %s", m.authority, msg.Authority)
	}

	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	if err := m.Keeper.TransferVault(ctx, msg.SourceVersion, msg.DestVersion, msg.AssetType, msg.Psbts, msg.TargetUtxoNum); err != nil {
		return nil, err
	}

	// Emit events
	m.EmitEvent(ctx, msg.Authority,
		sdk.NewAttribute("source_version", fmt.Sprintf("%d", msg.SourceVersion)),
		sdk.NewAttribute("dest_version", fmt.Sprintf("%d", msg.DestVersion)),
		sdk.NewAttribute("asset_type", msg.AssetType.String()),
	)

	return &types.MsgTransferVaultResponse{}, nil
}

// UpdateParams updates the module params.
func (m msgServer) UpdateParams(goCtx context.Context, msg *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	if m.authority != msg.Authority {
		return nil, errorsmod.Wrapf(govtypes.ErrInvalidSigner, "invalid authority; expected %s, got %s", m.authority, msg.Authority)
	}

	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	m.SetParams(ctx, msg.Params)

	// update total quotas of the rate limit if any
	if m.HasRateLimit(ctx) {
		m.UpdateRateLimitTotalQuotas(ctx, m.GlobalRateLimitSupplyPercentageQuota(ctx), m.AddressRateLimitQuota(ctx))
	}

	return &types.MsgUpdateParamsResponse{}, nil
}

// NewMsgServerImpl returns an implementation of the MsgServer interface
// for the provided Keeper.
func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

var _ types.MsgServer = msgServer{}
