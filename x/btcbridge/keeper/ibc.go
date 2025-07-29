package keeper

import (
	"fmt"
	"time"

	errorsmod "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	transfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	connectiontypes "github.com/cosmos/ibc-go/v8/modules/core/03-connection/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	ibcexported "github.com/cosmos/ibc-go/v8/modules/core/exported"

	"github.com/bitwaylabs/bitway/x/btcbridge/types"
)

// IBCTransfer performs the IBC transfer by the given params
func (k Keeper) IBCTransfer(ctx sdk.Context, sender string, recipient string, token sdk.Coin, channelId string) (uint64, error) {
	portId := k.ibctransferKeeper.GetPort(ctx)

	clientHeight, err := k.GetClientHeight(ctx, portId, channelId)
	if err != nil {
		return 0, err
	}

	msg := &transfertypes.MsgTransfer{
		SourcePort:       portId,
		SourceChannel:    channelId,
		Token:            token,
		Sender:           sender,
		Receiver:         recipient,
		TimeoutHeight:    getTimeoutHeight(clientHeight, k.IBCTimeoutHeightOffset(ctx)),
		TimeoutTimestamp: getTimeoutTimestamp(ctx.BlockTime(), k.IBCTimeoutDuration(ctx)),
		Memo:             types.DefaultMemo,
	}

	// total escrow before transfer
	totalEscrowBefore := k.ibctransferKeeper.GetTotalEscrowForDenom(ctx, token.Denom)

	resp, err := k.ibctransferKeeper.Transfer(ctx, msg)
	if err != nil {
		// total escrow after transfer
		totalEscrowAfter := k.ibctransferKeeper.GetTotalEscrowForDenom(ctx, token.Denom)
		if totalEscrowBefore.IsLT(totalEscrowAfter) {
			// unescrow token
			if err := k.UnescrowToken(ctx, portId, channelId, sender, token); err != nil {
				panic(err)
			}
		}

		return 0, err
	}

	return resp.Sequence, nil
}

// AddToIBCWithdrawRequestQueue adds the given withdrawal request to the IBC withdrawal queue for BTCT
func (k Keeper) AddToIBCWithdrawRequestQueue(ctx sdk.Context, channelId string, sequence uint64, recipient string, amount int64) {
	store := ctx.KVStore(k.storeKey)

	bz := k.cdc.MustMarshal(&types.IBCWithdrawRequest{
		ChannelId: channelId,
		Sequence:  sequence,
		Address:   recipient,
		Amount:    sdk.NewInt64Coin(k.BtcDenom(ctx), amount).String(),
	})

	store.Set(types.IBCWithdrawRequestQueueKey(channelId, sequence), bz)

	// Emit events
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeIBCWithdrawQueue,
			sdk.NewAttribute(types.AttributeKeyAddress, recipient),
			sdk.NewAttribute(types.AttributeKeyAmount, sdk.NewInt64Coin(k.BtcDenom(ctx), amount).String()),
			sdk.NewAttribute(types.AttributeKeyChannelId, channelId),
			sdk.NewAttribute(types.AttributeKeyPacketSequence, fmt.Sprintf("%d", sequence)),
		),
	)
}

// RemoveFromIBCWithdrawRequestQueue removes the given IBC withdrawal request from the IBC withdrawal request queue
func (k Keeper) RemoveFromIBCWithdrawRequestQueue(ctx sdk.Context, channelId string, sequence uint64) {
	store := ctx.KVStore(k.storeKey)

	store.Delete(types.IBCWithdrawRequestQueueKey(channelId, sequence))
}

// GetPendingIBCWithdrawRequests gets the pending IBC withdrawal requests up to the given maximum number
func (k Keeper) GetPendingIBCWithdrawRequests(ctx sdk.Context, maxNum uint32) []*types.IBCWithdrawRequest {
	requests := make([]*types.IBCWithdrawRequest, 0)

	k.IterateIBCWithdrawRequestQueue(ctx, func(req *types.IBCWithdrawRequest) (stop bool) {
		requests = append(requests, req)

		return maxNum != 0 && len(requests) >= int(maxNum)
	})

	return requests
}

// IterateIBCWithdrawRequestQueue iterates through the IBC withdrawal request queue
func (k Keeper) IterateIBCWithdrawRequestQueue(ctx sdk.Context, cb func(req *types.IBCWithdrawRequest) (stop bool)) {
	store := ctx.KVStore(k.storeKey)

	iterator := storetypes.KVStorePrefixIterator(store, types.IBCWithdrawRequestQueueKeyPrefix)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var request types.IBCWithdrawRequest
		k.cdc.MustUnmarshal(iterator.Value(), &request)

		if cb(&request) {
			break
		}
	}
}

// CheckBTCT returns true if the given packet is to receive native BTCT, false otherwise
func (k Keeper) CheckBTCT(ctx sdk.Context, packet ibcexported.PacketI, data transfertypes.FungibleTokenPacketData) bool {
	// check if the receiving chain is source
	if !transfertypes.ReceiverChainIsSource(packet.GetSourcePort(), packet.GetSourceChannel(), data.Denom) {
		return false
	}

	// sender prefix
	prefix := transfertypes.GetDenomPrefix(packet.GetSourcePort(), packet.GetSourceChannel())

	// remove sender prefix
	unprefixedDenom := data.Denom[len(prefix):]

	return unprefixedDenom == k.BtcDenom(ctx)
}

// GetClientHeight gets the current client height by the given source port and channel
func (k Keeper) GetClientHeight(ctx sdk.Context, sourcePort string, sourceChannel string) (ibcexported.Height, error) {
	channel, found := k.ibcchannelKeeper.GetChannel(ctx, sourcePort, sourceChannel)
	if !found {
		return nil, errorsmod.Wrap(channeltypes.ErrChannelNotFound, sourceChannel)
	}

	connectionEnd, found := k.ibcconnectionKeeper.GetConnection(ctx, channel.ConnectionHops[0])
	if !found {
		return nil, errorsmod.Wrap(connectiontypes.ErrConnectionNotFound, channel.ConnectionHops[0])
	}

	clientState, found := k.ibcclientKeeper.GetClientState(ctx, connectionEnd.GetClientID())
	if !found {
		return nil, errorsmod.Wrap(clienttypes.ErrClientNotFound, connectionEnd.GetClientID())
	}

	return clientState.GetLatestHeight(), nil
}

// UnescrowToken unescrows the given token from the IBC transfer module
// NOTE: This method is called only if the IBC transfer failed while the token is escrowed
func (k Keeper) UnescrowToken(ctx sdk.Context, portId string, channelId string, recipient string, token sdk.Coin) error {
	escrowAddress := transfertypes.GetEscrowAddress(portId, channelId)
	if err := k.bankKeeper.SendCoins(ctx, escrowAddress, sdk.MustAccAddressFromBech32(recipient), sdk.NewCoins(token)); err != nil {
		return errorsmod.Wrap(err, "failed to unescrow token")
	}

	currentTotalEscrow := k.ibctransferKeeper.GetTotalEscrowForDenom(ctx, token.Denom)
	newTotalEscrow := currentTotalEscrow.Sub(token)
	k.ibctransferKeeper.SetTotalEscrowForDenom(ctx, newTotalEscrow)

	return nil
}

// IBCSendPacketCallback implements IBC callbacks
func (k Keeper) IBCSendPacketCallback(
	ctx sdk.Context,
	sourcePort string,
	sourceChannel string,
	timeoutHeight clienttypes.Height,
	timeoutTimestamp uint64,
	packetData []byte,
	contractAddress,
	packetSenderAddress string,
) error {
	// no-op
	return nil
}

// IBCOnAcknowledgementPacketCallback implements IBC callbacks
func (k Keeper) IBCOnAcknowledgementPacketCallback(
	ctx sdk.Context,
	packet channeltypes.Packet,
	acknowledgement []byte,
	relayer sdk.AccAddress,
	contractAddress,
	packetSenderAddress string,
) error {
	// no-op
	return nil
}

// IBCOnTimeoutPacketCallback implements IBC callbacks
func (k Keeper) IBCOnTimeoutPacketCallback(
	ctx sdk.Context,
	packet channeltypes.Packet,
	relayer sdk.AccAddress,
	contractAddress,
	packetSenderAddress string,
) error {
	// no-op
	return nil
}

// IBCReceivePacketCallback implements IBC callbacks
func (k Keeper) IBCReceivePacketCallback(
	ctx sdk.Context,
	packet ibcexported.PacketI,
	ack ibcexported.Acknowledgement,
	contractAddress string,
) error {
	// check if the callback address is the expected address
	if contractAddress != types.CallbackAddress {
		return nil
	}

	// check if withdrawal is enabled
	if !k.WithdrawEnabled(ctx) {
		return nil
	}

	// check if the packet is BTCT token transfer and auto-pegout enabled
	data, ok := tryGetFungibleTokenPacketData(packet)
	if !ok || !k.CheckBTCT(ctx, packet, data) {
		return nil
	}

	// check if the recipient address is valid btc address
	if !types.IsValidBtcAddress(data.Receiver) {
		return nil
	}

	// check amount
	amount, ok := sdkmath.NewIntFromString(data.Amount)
	if !ok || !amount.IsInt64() {
		return nil
	}

	// check rate limit
	if err := k.CheckRateLimit(ctx, data.Receiver, amount.Int64()); err != nil {
		return err
	}

	// add to IBC withdrawal request queue
	k.AddToIBCWithdrawRequestQueue(ctx, packet.GetDestChannel(), packet.GetSequence(), data.Receiver, amount.Int64())

	return nil
}

// tryGetFungibleTokenPacketData attempts to parse the IBC transfer packet data from the given packet
func tryGetFungibleTokenPacketData(packet ibcexported.PacketI) (transfertypes.FungibleTokenPacketData, bool) {
	var data transfertypes.FungibleTokenPacketData
	if err := transfertypes.ModuleCdc.UnmarshalJSON(packet.GetData(), &data); err != nil {
		return data, false
	}

	return data, true
}

// getTimeoutHeight gets the timeout height
func getTimeoutHeight(clientHeight ibcexported.Height, timeoutHeightOffset uint64) clienttypes.Height {
	if timeoutHeightOffset == 0 {
		return clienttypes.ZeroHeight()
	}

	return clienttypes.NewHeight(clientHeight.GetRevisionNumber(), clientHeight.GetRevisionHeight()+timeoutHeightOffset)
}

// getTimeoutTimestamp gets the timeout timestamp
func getTimeoutTimestamp(currentTime time.Time, timeoutDuration time.Duration) uint64 {
	if timeoutDuration == 0 {
		return 0
	}

	return uint64(currentTime.UnixNano()) + uint64(timeoutDuration.Nanoseconds())
}
