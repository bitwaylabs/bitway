package coinbase

import (
	"context"
	"encoding/json"
	"time"

	"github.com/cosmos/cosmos-sdk/server"

	"github.com/bitwaylabs/bitway/x/oracle/types"
)

// ▼ {"type":"ticker","sequence":1566204232,"product_id":"BTC-USD","price":"1.11","open_24h":"1","volume_24h":"318996.26881102","low_24h":"0.38","high_24h":"1.48","volume_30d":"3743179.26781576","best_bid":"1.09","best_bid_size":"11.99966940","best_ask":"1.11","best_ask_size":"16.34610601","side":"buy","time":"2025-03-01T13:25:17.042052Z","trade_id":130278532,"last_size":"1.98198198"}

var (
	ProviderName = "coinbase"
	// url := "wss://ws-feed-public.sandbox.exchange.coinbase.com"
	URL          = "wss://ws-feed.exchange.coinbase.com"
	SubscribeMsg = `{"type":"subscribe","product_ids":["BTC-USD"],"channels":[{"name":"ticker","product_ids":["BTC-USD"]}]}`
	SymbolMap    = map[string]string{
		"BTC-USD": types.BTCUSD,
	}
)

func symbol(source string) string {
	if target, ok := SymbolMap[source]; ok {
		return target
	} else {
		return source
	}
}

type Subscription struct {
	Type   string `json:"type"`
	Symbol string `json:"product_id,omitempty"`
	Price  string `json:"price,omitempty"`
	Time   string `json:"time,omitempty"`
}

func Subscribe(svrCtx *server.Context, ctx context.Context) error {
	return types.Subscribe(ProviderName, svrCtx, ctx, URL, SubscribeMsg, func(msg []byte) []types.Price {
		prices := make([]types.Price, 1)
		subscription := &Subscription{}
		if err := json.Unmarshal(msg, subscription); err == nil {
			if subscription.Type == "ticker" {
				// svrCtx.Logger.Info("Websocket Received", "provider", ProviderName, "message", subscription, "symbol", subscription.Symbol, "price", subscription.Price)

				// sample time: 2025-03-01T03:42:43.951417Z
				if t, err := time.Parse(time.RFC3339Nano, subscription.Time); err == nil {
					price := types.Price{
						Symbol: symbol(subscription.Symbol),
						Price:  subscription.Price,
						Time:   t.UnixMilli(),
					}
					prices = append(prices, price)
				} else {
					svrCtx.Logger.Error("Parse time error")
				}

			}
		}

		return prices
	})
}

// {"type":"subscribe","product_ids":["BTC-USD"],"channels":[{"name":"ticker","product_ids":["BTC-USD"]}]}
// func subscribe(conn *websocket.Conn) {
// 	conn.WriteMessage(websocket.TextMessage, []byte("{\"type\":\"subscribe\",\"product_ids\":[\"BTC-USD\"],\"channels\":[{\"name\":\"ticker\",\"product_ids\":[\"BTC-USD\"]}]}"))
// }

// func Subscribe(svrCtx *server.Context) error {
// 	// url := "wss://ws-feed-public.sandbox.exchange.coinbase.com"
// 	url := "wss://ws-feed.exchange.coinbase.com"
// 	c, re, err := websocket.DefaultDialer.Dial(url, nil)
// 	if err != nil {
// 		svrCtx.Logger.Error("price provider connection", "url", url)
// 		return err
// 	}
// 	defer c.Close()

// 	subscribe(c)
// 	reconnect := false

// 	for {

// 		if reconnect {
// 			for {
// 				time.Sleep(5 * time.Second)
// 				if c, _, err = websocket.DefaultDialer.Dial(url, nil); err == nil {
// 					reconnect = false
// 					subscribe(c)
// 					svrCtx.Logger.Info("reconnected price provider", "url", url, "status", re.Status, "body", re.Body)
// 					break
// 				}
// 			}
// 		}

// 		subscription := &Subscription{}
// 		if err = c.ReadJSON(subscription); err == nil {
// 			if subscription.Type == "ticker" {
// 				// svrCtx.Logger.Info("Websocket Received", "provider", ProviderName, "message", subscription, "symbol", subscription.Symbol, "price", subscription.Price)

// 				// sample time: 2025-03-01T03:42:43.951417Z
// 				if t, err := time.Parse(time.RFC3339Nano, subscription.Time); err == nil {
// 					price := types.Price{
// 						Symbol: symbol(subscription.Symbol),
// 						Price:  subscription.Price,
// 						Time:   t.UnixMilli(),
// 					}
// 					types.CachePrice(ProviderName, price)
// 				} else {
// 					svrCtx.Logger.Error("Parse time error")
// 				}

// 			}
// 		} else {
// 			c.Close()
// 			svrCtx.Logger.Error("Read Error", "error", err, "provider", ProviderName)
// 			reconnect = true

// 		}

// 		// adaptor(steam)

// 	}
// }
