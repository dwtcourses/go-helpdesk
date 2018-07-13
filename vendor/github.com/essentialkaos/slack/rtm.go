package slack

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

const (
	WEBSOCKET_DEFAULT_TIMEOUT = 10 * time.Second
)

// StartRTM calls the "rtm.start" endpoint and returns the provided URL and the full Info block.
//
// To have a fully managed Websocket connection, use `NewRTM`, and call `ManageConnection()` on it.
func (api *Client) StartRTM() (info *Info, websocketURL string, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), WEBSOCKET_DEFAULT_TIMEOUT)

	defer cancel()

	return api.StartRTMContext(ctx)
}

// StartRTMContext calls the "rtm.start" endpoint and returns the provided URL and the full Info block with a custom context.
//
// To have a fully managed Websocket connection, use `NewRTM`, and call `ManageConnection()` on it.
func (api *Client) StartRTMContext(ctx context.Context) (info *Info, websocketURL string, err error) {
	response := &infoResponseFull{}

	err = post(ctx, "rtm.start", api.Config.toParams(), response, api.debug)

	if err != nil {
		return nil, "", fmt.Errorf("post: %s", err)
	}

	if !response.Ok {
		return nil, "", response.Error
	}

	// websocket.Dial does not accept url without the port (yet)
	// Fixed by: https://github.com/golang/net/commit/5058c78c3627b31e484a81463acd51c7cecc06f3
	// but slack returns the address with no port, so we have to fix it
	api.Debugln("Using URL:", response.Info.URL)

	return &response.Info, response.Info.URL, nil
}

// ConnectRTM calls the "rtm.connect" endpoint and returns the provided URL and the compact
// Info block.
// To have a fully managed Websocket connection, use `NewRTM`, and call `ManageConnection()` on it.
func (api *Client) ConnectRTM() (info *Info, websocketURL string, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), WEBSOCKET_DEFAULT_TIMEOUT)

	defer cancel()

	return api.ConnectRTMContext(ctx)
}

// ConnectRTMContext calls the "rtm.connect" endpoint and returns the provided URL and the
// compact Info block with a custom context.
// To have a fully managed Websocket connection, use `NewRTM`, and call `ManageConnection()` on it.
func (api *Client) ConnectRTMContext(ctx context.Context) (info *Info, websocketURL string, err error) {
	response := &infoResponseFull{}
	err = post(ctx, "rtm.connect", api.Config.toParams(), response, api.debug)

	if err != nil {
		api.Debugf("Failed to connect to RTM: %s", err)
		return nil, "", fmt.Errorf("post: %s", err)
	}

	if !response.Ok {
		return nil, "", response.Error
	}

	// websocket.Dial does not accept url without the port (yet)
	// Fixed by: https://github.com/golang/net/commit/5058c78c3627b31e484a81463acd51c7cecc06f3
	// but slack returns the address with no port, so we have to fix it
	api.Debugln("Using URL:", response.Info.URL)

	return &response.Info, response.Info.URL, nil
}

// NewRTM returns a RTM, which provides a fully managed connection to
// Slack's websocket-based Real-Time Messaging protocol.
func (api *Client) NewRTM() *RTM {
	return api.NewRTMWithOptions(nil)
}

// NewRTMWithOptions returns a RTM, which provides a fully managed connection to
// Slack's websocket-based Real-Time Messaging protocol.
// This also allows to configure various options available for RTM API.
func (api *Client) NewRTMWithOptions(options *RTMOptions) *RTM {
	result := &RTM{
		Client:           *api,
		IncomingEvents:   make(chan RTMEvent, 50),
		outgoingMessages: make(chan OutgoingMessage, 20),
		pings:            make(map[int]time.Time),
		isConnected:      false,
		wasIntentional:   true,
		killChannel:      make(chan bool),
		disconnected:     make(chan struct{}),
		forcePing:        make(chan bool),
		rawEvents:        make(chan json.RawMessage),
		idGen:            NewSafeID(1),
		mu:               &sync.Mutex{},
	}

	if options != nil && options.UseRTMStart {
		result.useRTMStart = true
	}

	return result
}
