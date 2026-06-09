package intercept

// ClientMsg is a message emitted by the intercept client for the UI layer.
type ClientMsg interface {
	clientMsg()
}

// DisconnectedMsg signals the connection closed.
type DisconnectedMsg struct {
	Err error
}

func (DisconnectedMsg) clientMsg() {}

// GameLineMsg is a line of game output to display.
type GameLineMsg struct {
	Line string
}

func (GameLineMsg) clientMsg() {}
