//go:build js && wasm

package intercept

import (
	"fmt"
	"io"
	"syscall/js"
)

func (c *Client) dialSocketIO() error {
	net := js.Global().Get("intertuiNet")
	if net.IsUndefined() || net.IsNull() {
		return fmt.Errorf("intertuiNet bridge not loaded (missing socketio.js)")
	}

	c.sioWake = make(chan struct{}, 64)

	errCh := make(chan string, 1)
	connectCb := js.FuncOf(func(_ js.Value, args []js.Value) any {
		if len(args) > 0 && !args[0].IsNull() && args[0].Truthy() {
			errCh <- args[0].String()
		} else {
			errCh <- ""
		}
		return nil
	})
	defer connectCb.Release()

	net.Call("connect", c.addr, connectCb)

	select {
	case errMsg := <-errCh:
		if errMsg != "" {
			return fmt.Errorf("%s", errMsg)
		}
	case <-c.ctx.Done():
		return c.ctx.Err()
	}

	wake := js.FuncOf(func(_ js.Value, _ []js.Value) any {
		select {
		case c.sioWake <- struct{}{}:
		default:
		}
		return nil
	})
	net.Call("setWake", wake)

	c.sioCleanup = func() {
		wake.Release()
		net.Call("close")
	}

	return nil
}

func (c *Client) closeSocketIO() {
	if c.sioCleanup != nil {
		c.sioCleanup()
		c.sioCleanup = nil
	}
}

func (c *Client) readLoopSocketIO() {
	net := js.Global().Get("intertuiNet")
	for {
		select {
		case <-c.ctx.Done():
			return
		case <-c.sioWake:
			for {
				raw := net.Call("recv").String()
				if raw == "" {
					break
				}
				c.handleFrame([]byte(raw))
			}
			if !net.Call("connected").Bool() && c.ctx.Err() == nil {
				c.emit(DisconnectedMsg{Err: io.EOF})
				return
			}
		}
	}
}

func (c *Client) writeSocketIOFrame(raw []byte) error {
	net := js.Global().Get("intertuiNet")
	v := net.Call("send", string(raw))
	if v.IsNull() {
		return nil
	}
	return fmt.Errorf("%s", v.String())
}
