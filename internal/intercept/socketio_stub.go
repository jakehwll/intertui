//go:build !js || !wasm

package intercept

import "fmt"

func (c *Client) dialSocketIO() error {
	return fmt.Errorf("socket.io transport requires WASM build")
}

func (c *Client) closeSocketIO() {}

func (c *Client) readLoopSocketIO() {}

func (c *Client) writeSocketIOFrame([]byte) error {
	return fmt.Errorf("socket.io transport requires WASM build")
}
