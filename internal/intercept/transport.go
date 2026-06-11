package intercept

// Transport selects how to reach the server.
type Transport int

const (
	// TransportTCP uses newline-delimited JSON over raw TCP.
	TransportTCP Transport = iota
	// TransportWebSocket uses JSON frames over WebSocket.
	TransportWebSocket
	// TransportMock uses an in-process mock backend (no network).
	TransportMock
	// TransportPlugin uses a PluginTransport implementation (browser only).
	TransportPlugin
)
