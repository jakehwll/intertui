package intercept

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

var debugWire = os.Getenv("INTERCEPT_DEBUG") != ""

// Credentials holds login details. Set either User/Pass or Token.
type Credentials struct {
	User  string
	Pass  string
	Token string
}

// Client talks to the Intercept game server.
type Client struct {
	transport Transport
	addr      string // host:port for TCP, URL for WebSocket
	cred      Credentials

	tcp    net.Conn
	ws     *websocket.Conn
	ctx    context.Context
	cancel context.CancelFunc

	waiter  Waiter
	sendQ   chan map[string]any
	msgs    chan ClientMsg
	once    sync.Once
	user    string
	writeMu sync.Mutex
	onStatus func(string)
}

// SetStatus sets a callback for connection progress updates.
func (c *Client) SetStatus(fn func(string)) {
	c.onStatus = fn
}

func (c *Client) status(line string) {
	if c.onStatus != nil {
		c.onStatus(line)
	}
}

// NewTCP creates a client for newline-delimited JSON over TCP.
func NewTCP(addr string, cred Credentials) *Client {
	return &Client{
		transport: TransportTCP,
		addr:      addr,
		cred:      cred,
		msgs:      make(chan ClientMsg, 64),
	}
}

// NewWebSocket creates a client for JSON over WebSocket.
func NewWebSocket(url string, cred Credentials) *Client {
	return &Client{
		transport: TransportWebSocket,
		addr:      url,
		cred:      cred,
		msgs:      make(chan ClientMsg, 64),
	}
}

// Messages returns the outbound message channel for the UI.
func (c *Client) Messages() <-chan ClientMsg {
	return c.msgs
}

// User returns the connected player name, if known.
func (c *Client) User() string {
	return c.user
}

// Start dials, logs in, and begins read/write loops. Blocks until login completes.
func (c *Client) Start(parent context.Context) error {
	c.ctx, c.cancel = context.WithCancel(parent)

	switch c.transport {
	case TransportTCP:
		c.status("Opening TCP connection to " + c.addr + "…")
		conn, err := net.DialTimeout("tcp", c.addr, 15*time.Second)
		if err != nil {
			return fmt.Errorf("dial %s: %w", c.addr, err)
		}
		c.tcp = conn
		c.status("TCP connected")
	default:
		c.status("Opening WebSocket to " + c.addr + "…")
		conn, _, err := websocket.Dial(c.ctx, c.addr, nil)
		if err != nil {
			return fmt.Errorf("dial %s: %w", c.addr, err)
		}
		c.ws = conn
		c.status("WebSocket connected")
	}

	c.sendQ = make(chan map[string]any, 32)
	c.status("Starting read/write loops…")
	go c.readLoop()
	go c.writeLoop()

	c.status("Logging in…")
	if err := c.login(); err != nil {
		c.Close()
		return err
	}

	c.status("Login complete")
	return nil
}

// SendCommand enqueues a game command.
func (c *Client) SendCommand(cmd string) {
	cmd = strings.TrimSpace(cmd)
	if cmd == "" || c.sendQ == nil {
		return
	}
	select {
	case c.sendQ <- map[string]any{"request": "command", "cmd": cmd}:
	case <-c.ctx.Done():
	}
}

// Close shuts down the connection.
func (c *Client) Close() {
	c.once.Do(func() {
		if c.cancel != nil {
			c.cancel()
		}
		if c.ws != nil {
			_ = c.ws.Close(websocket.StatusNormalClosure, "")
		}
		if c.tcp != nil {
			_ = c.tcp.Close()
		}
		close(c.msgs)
	})
}

func (c *Client) login() error {
	if c.transport == TransportTCP {
		return c.loginTCP()
	}

	if c.cred.Token != "" {
		if err := c.loginToken(c.cred.Token); err != nil {
			return err
		}
	} else if c.cred.User != "" {
		if err := c.loginUserPassWS(c.cred.User, c.cred.Pass); err != nil {
			return err
		}
	} else {
		return fmt.Errorf("no credentials provided")
	}
	return c.selectMainSystem()
}

// loginTCP authenticates and joins via auth → connect (token).
func (c *Client) loginTCP() error {
	if c.cred.Token != "" {
		return fmt.Errorf("API token login requires WebSocket mode (--ws)")
	}
	if c.cred.User == "" {
		return fmt.Errorf("no credentials provided")
	}

	c.status("Sending auth (username/password)…")
	auth, err := c.sendAndWait(
		map[string]any{
			"request": "auth",
			"login": map[string]string{
				"username": c.cred.User,
				"password": c.cred.Pass,
			},
		},
		"auth response",
		func(e Envelope) bool { return e.Event == "auth" || e.Event == "error" },
	)
	if err != nil {
		return err
	}
	if auth.Event == "error" {
		return fmt.Errorf("auth failed: %s", auth.Error)
	}
	if !auth.Success || auth.Token == "" {
		return fmt.Errorf("auth failed")
	}
	if name := auth.PlayerName(); name != "" {
		c.user = name
	} else {
		c.user = c.cred.User
	}
	c.status("Authenticated as " + c.user + ", joining game…")

	c.status("Sending connect request…")
	connect, err := c.sendAndWait(
		map[string]any{"request": "connect", "token": auth.Token},
		"connect response",
		func(e Envelope) bool {
			return e.Event == "connect" || e.Event == "connected" || e.Event == "error"
		},
	)
	if err != nil {
		return err
	}
	if !connect.ConnectOK() {
		return connect.ConnectErr()
	}
	if line, ok := connect.DisplayLine(); ok {
		c.emit(GameLineMsg{Line: line})
	}
	c.status("In game")

	return nil
}

func (c *Client) loginUserPassWS(user, pass string) error {
	c.status("Sending auth (username/password)…")
	auth, err := c.sendAndWait(
		map[string]any{
			"request": "auth",
			"login": map[string]string{
				"username": user,
				"password": pass,
			},
		},
		"auth response",
		func(e Envelope) bool { return e.Event == "auth" || e.Event == "error" },
	)
	if err != nil {
		return err
	}
	if !auth.Success {
		return fmt.Errorf("auth failed")
	}
	if name := auth.PlayerName(); name != "" {
		c.user = name
	} else {
		c.user = user
	}
	c.status("Authenticated as " + c.user)
	if auth.Token != "" {
		return c.loginTokenConfirm(auth.Token)
	}
	return nil
}

func (c *Client) loginToken(key string) error {
	c.status("Sending auth (API token)…")
	auth, err := c.sendAndWait(
		map[string]any{"request": "auth", "key": key},
		"auth response",
		func(e Envelope) bool { return e.Event == "auth" || e.Event == "error" },
	)
	if err != nil {
		return err
	}
	if !auth.Success || auth.Token == "" {
		return fmt.Errorf("auth failed")
	}
	if name := auth.PlayerName(); name != "" {
		c.user = name
	}
	return c.loginTokenConfirm(auth.Token)
}

func (c *Client) loginTokenConfirm(token string) error {
	c.status("Confirming auth token…")
	auth, err := c.sendAndWait(
		map[string]any{"request": "auth", "token": token},
		"token confirmation",
		func(e Envelope) bool { return e.Event == "auth" || e.Event == "error" },
	)
	if err != nil {
		return err
	}
	if !auth.Success {
		return fmt.Errorf("token auth failed")
	}
	if name := auth.PlayerName(); name != "" && c.user == "" {
		c.user = name
	}
	return nil
}

func (c *Client) selectMainSystem() error {
	c.status("Fetching game systems…")
	systems, err := c.sendAndWait(
		map[string]any{"request": "systems"},
		"systems list",
		func(e Envelope) bool { return e.Event == "systems" || e.Event == "error" },
	)
	if err != nil {
		return err
	}
	if !systems.Success {
		return fmt.Errorf("systems request failed")
	}

	var mainID string
	for _, sys := range systems.Systems {
		if sys.Type == "main" {
			mainID = sys.ID
			break
		}
	}
	if mainID == "" && len(systems.Systems) > 0 {
		mainID = systems.Systems[0].ID
	}
	if mainID == "" {
		return fmt.Errorf("no game systems available")
	}

	c.status("Connecting to system " + mainID + "…")
	connect, err := c.sendAndWait(
		map[string]any{"request": "connect", "system": mainID},
		"game connection",
		func(e Envelope) bool {
			return e.Event == "connect" || e.Event == "connected" || e.Event == "error"
		},
	)
	if err != nil {
		return err
	}
	if !connect.ConnectOK() {
		return connect.ConnectErr()
	}
	if connect.User != "" && c.user == "" {
		c.user = connect.User
	}
	if line, ok := connect.DisplayLine(); ok {
		c.emit(GameLineMsg{Line: line})
	}

	return nil
}

func (c *Client) sendAndWait(
	payload map[string]any,
	waiting string,
	match func(Envelope) bool,
) (Envelope, error) {
	ch := c.waiter.Wait(match)
	if err := c.sendNow(payload); err != nil {
		return Envelope{}, err
	}
	return c.await(ch, waiting)
}

// sendNow writes immediately (used for login RPCs). Game commands use send().
func (c *Client) sendNow(data map[string]any) error {
	raw, err := json.Marshal(data)
	if err != nil {
		return err
	}
	c.writeMu.Lock()
	err = c.writeFrame(raw)
	c.writeMu.Unlock()
	if debugWire {
		log.Printf("wire write: %s err=%v", raw, err)
	}
	return err
}

func (c *Client) send(data map[string]any) {
	select {
	case c.sendQ <- data:
	case <-c.ctx.Done():
	}
}

func (c *Client) await(ch <-chan Envelope, waiting string) (Envelope, error) {
	if waiting != "" {
		c.status("Waiting for " + waiting + " (30s timeout)…")
	}
	select {
	case env := <-ch:
		if waiting != "" {
			c.status("Received " + waiting)
		}
		return env, nil
	case <-time.After(30 * time.Second):
		return Envelope{}, fmt.Errorf("timed out waiting for %s", waiting)
	case <-c.ctx.Done():
		return Envelope{}, c.ctx.Err()
	}
}

func (c *Client) readLoop() {
	defer func() {
		if c.ctx.Err() == nil {
			c.emit(DisconnectedMsg{Err: io.EOF})
		}
		c.Close()
	}()

	switch c.transport {
	case TransportTCP:
		c.readLoopTCP()
	default:
		c.readLoopWS()
	}
}

func (c *Client) readLoopTCP() {
	scanner := bufio.NewScanner(c.tcp)
	scanner.Buffer(make([]byte, 0, 64*1024), 1<<20)
	for scanner.Scan() {
		c.handleFrame(scanner.Bytes())
	}
	if err := scanner.Err(); err != nil && c.ctx.Err() == nil {
		c.emit(DisconnectedMsg{Err: err})
	}
}

func (c *Client) readLoopWS() {
	for {
		var raw json.RawMessage
		if err := wsjson.Read(c.ctx, c.ws, &raw); err != nil {
			if c.ctx.Err() == nil {
				c.emit(DisconnectedMsg{Err: err})
			}
			return
		}
		c.handleFrame(raw)
	}
}

func (c *Client) handleFrame(raw []byte) {
	if debugWire {
		log.Printf("wire read: %s", raw)
	}
	env, err := ParseEnvelope(raw)
	if err != nil {
		if debugWire {
			log.Printf("wire parse err: %v", err)
		}
		return
	}
	if debugWire {
		log.Printf("wire event: %s success=%v", env.Event, env.Success)
	}
	if !c.waiter.Deliver(env) {
		if line, ok := env.DisplayLine(); ok {
			c.emit(GameLineMsg{Line: line})
		} else if env.Event != "" && !env.IsSilent() {
			c.emit(GameLineMsg{Line: env.Summarize()})
		}
	}
}

func (c *Client) writeLoop() {
	for {
		if c.ctx.Err() != nil {
			return
		}
		select {
		case data := <-c.sendQ:
			raw, err := json.Marshal(data)
			if err != nil {
				continue
			}
			c.writeMu.Lock()
			err = c.writeFrame(raw)
			c.writeMu.Unlock()
			if err != nil {
				return
			}
			delay := max(300*time.Millisecond, time.Duration(len(raw))*time.Millisecond/20)
			select {
			case <-time.After(delay):
			case <-c.ctx.Done():
				return
			}
		case <-c.ctx.Done():
			return
		}
	}
}

func (c *Client) writeFrame(raw []byte) error {
	switch c.transport {
	case TransportTCP:
		_, err := c.tcp.Write(append(raw, '\n'))
		return err
	default:
		return wsjson.Write(c.ctx, c.ws, json.RawMessage(raw))
	}
}

func (c *Client) emit(msg ClientMsg) {
	select {
	case c.msgs <- msg:
	case <-c.ctx.Done():
	}
}
