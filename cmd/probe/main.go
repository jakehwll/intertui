package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/coder/serpent"
	"github.com/spf13/pflag"

)

func main() {
	var (
		mode  string
		user  string
		pass  string
		addr  string
		delay time.Duration
	)

	cmd := &serpent.Command{
		Use:   "probe",
		Short: "Low-level Intercept protocol probe.",
		Options: serpent.OptionSet{
			{
				Name:        "mode",
				Flag:        "mode",
				Default:     "raw",
				Value:       serpent.StringOf(&mode),
				Description: "Probe mode: raw or client.",
			},
			{
				Name:        "user",
				Flag:        "user",
				Env:         "INTERCEPT_USER",
				Value:       serpent.StringOf(&user),
				Description: "Intercept username.",
			},
			{
				Name:        "pass",
				Flag:        "pass",
				Env:         "INTERCEPT_PASS",
				Value:       serpent.StringOf(&pass),
				Description: "Intercept password.",
			},
			{
				Name:        "addr",
				Flag:        "addr",
				Env:         "INTERCEPT_ADDR",
				Value:       serpent.StringOf(&addr),
				Description: "TCP address to dial (host:port).",
			},
			{
				Name:        "delay",
				Flag:        "delay",
				Default:     "0",
				Value:       serpent.DurationOf(&delay),
				Description: "Delay before connect request.",
			},
		},
		Handler: func(inv *serpent.Invocation) error {
			if addr == "" {
				return fmt.Errorf("need --addr or INTERCEPT_ADDR")
			}

			if mode == "client" {
				runClient(addr, user, pass)
				return nil
			}

			if user == "" || pass == "" {
				return fmt.Errorf("need --user and --pass")
			}

			runRaw(addr, user, pass, delay)
			return nil
		},
	}

	if err := cmd.Invoke().WithOS().Run(); err != nil {
		if !errors.Is(err, pflag.ErrHelp) {
			fmt.Fprintln(os.Stderr, err)
		}
		os.Exit(1)
	}
}

func runRaw(addr, user, pass string, delay time.Duration) {
	conn, err := net.DialTimeout("tcp", addr, 15*time.Second)
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	lines := make(chan string, 64)
	go func() {
		sc := bufio.NewScanner(conn)
		for sc.Scan() {
			lines <- sc.Text()
		}
		if err := sc.Err(); err != nil {
			fmt.Println("scanner err:", err)
		}
		fmt.Println("scanner done")
		close(lines)
	}()

	send := func(v map[string]any) {
		b, _ := json.Marshal(v)
		fmt.Println(">>", string(b))
		n, err := conn.Write(append(b, '\n'))
		fmt.Println("wrote", n, "err", err)
	}

	readUntil := func(match func(map[string]any) bool, timeout time.Duration) map[string]any {
		deadline := time.After(timeout)
		for {
			select {
			case line, ok := <-lines:
				if !ok {
					return nil
				}
				fmt.Println("<<", line)
				var env map[string]any
				if json.Unmarshal([]byte(line), &env) != nil {
					continue
				}
				if match(env) {
					return env
				}
			case <-deadline:
				return nil
			}
		}
	}

	send(map[string]any{
		"request": "auth",
		"login":   map[string]string{"username": user, "password": pass},
	})
	auth := readUntil(func(e map[string]any) bool { return e["event"] == "auth" }, 10*time.Second)
	if auth == nil {
		fmt.Println("no auth response")
		return
	}
	token, _ := auth["token"].(string)
	fmt.Printf("token: %s delay: %s\n", token, delay)
	time.Sleep(delay)
	send(map[string]any{"request": "connect", "token": token})
	deadline := time.After(15 * time.Second)
	for {
		select {
		case line, ok := <-lines:
			if !ok {
				fmt.Println("lines closed")
				return
			}
			fmt.Println("<<", line)
			var env map[string]any
			if json.Unmarshal([]byte(line), &env) == nil {
				if clink, ok := env["clink"].([]any); ok && len(clink) > 0 {
					fmt.Println("   clink:", clink)
				}
			}
		case <-deadline:
			fmt.Println("(done)")
			return
		}
	}
}
