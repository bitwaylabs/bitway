package oracle

import (
	"context"

	"github.com/bitwaylabs/bitway/x/oracle/providers/binance"
	"github.com/bitwaylabs/bitway/x/oracle/providers/bitget"
	"github.com/bitwaylabs/bitway/x/oracle/providers/bybit"
	"github.com/bitwaylabs/bitway/x/oracle/providers/coinbase"
	"github.com/bitwaylabs/bitway/x/oracle/providers/okex"
	"github.com/bitwaylabs/bitway/x/oracle/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/server"
	"golang.org/x/sync/errgroup"
)

// Start Oracle Price Service
// Subscrible Prices from providers
func Start(svrCtx *server.Context, clientCtx client.Context, ctx context.Context, g *errgroup.Group) error {

	if types.StartProviders {

		svrCtx.Logger.With("module", types.ModuleName).Info("price service", "module", "oracle", "msg", "Start Oracle Price Subscriber")

		go binance.Subscribe(svrCtx, ctx)
		go okex.Subscribe(svrCtx, ctx)
		go coinbase.Subscribe(svrCtx, ctx)
		go bybit.Subscribe(svrCtx, ctx)
		go bitget.Subscribe(svrCtx, ctx)
		// g.Go(func() error { return binance.Subscribe(svrCtx, ctx) })
		// g.Go(func() error { return okex.Subscribe(svrCtx, ctx) })
		// g.Go(func() error { return coinbase.Subscribe(svrCtx, ctx) })
		// g.Go(func() error { return bybit.Subscribe(svrCtx, ctx) })
		// g.Go(func() error { return bitget.Subscribe(svrCtx, ctx) })
	} else {
		svrCtx.Logger.With("module", types.ModuleName).Warn("Price service is disabled. It is required if your node is a validator. ")
	}

	return nil

}
