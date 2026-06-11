//go:build js && wasm

package browser

import (
	"context"
	"fmt"
	"io"
	"syscall/js"
)

type socketIO struct {
	wake    chan struct{}
	cleanup func()
}

func (s *socketIO) UsesTCPLogin() bool { return true }

func (s *socketIO) Dial(ctx context.Context, addr string) error {
	net := js.Global().Get("intertuiNet")
	if net.IsUndefined() || net.IsNull() {
		return fmt.Errorf("intertuiNet bridge not loaded (missing socketio.js)")
	}

	s.wake = make(chan struct{}, 64)

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

	net.Call("connect", addr, connectCb)

	select {
	case errMsg := <-errCh:
		if errMsg != "" {
			return fmt.Errorf("%s", errMsg)
		}
	case <-ctx.Done():
		return ctx.Err()
	}

	wake := js.FuncOf(func(_ js.Value, _ []js.Value) any {
		select {
		case s.wake <- struct{}{}:
		default:
		}
		return nil
	})
	net.Call("setWake", wake)

	s.cleanup = func() {
		wake.Release()
		net.Call("close")
	}

	return nil
}

func (s *socketIO) Close() error {
	if s.cleanup != nil {
		s.cleanup()
		s.cleanup = nil
	}
	return nil
}

func (s *socketIO) Write(raw []byte) error {
	net := js.Global().Get("intertuiNet")
	v := net.Call("send", string(raw))
	if v.IsNull() {
		return nil
	}
	return fmt.Errorf("%s", v.String())
}

func (s *socketIO) Read(ctx context.Context, frame func([]byte)) error {
	net := js.Global().Get("intertuiNet")
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-s.wake:
			for {
				raw := net.Call("recv").String()
				if raw == "" {
					break
				}
				frame([]byte(raw))
			}
			if !net.Call("connected").Bool() {
				return io.EOF
			}
		}
	}
}
