package intercept

import "context"

// PluginTransport is an optional wire transport (browser builds only).
// Desktop clients never set this.
type PluginTransport interface {
	Dial(ctx context.Context, addr string) error
	Close() error
	Write([]byte) error
	Read(ctx context.Context, frame func([]byte)) error
	UsesTCPLogin() bool
}

// NewPlugin creates a client using a plugin transport.
func NewPlugin(addr string, cred Credentials, plugin PluginTransport) *Client {
	return &Client{
		transport: TransportPlugin,
		addr:      addr,
		cred:      cred,
		plugin:    plugin,
		msgs:      make(chan ClientMsg, 64),
	}
}
