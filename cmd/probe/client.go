package main

import (
	"context"
	"fmt"
	"os"

	"intertui/internal/intercept"
)

func runClient(addr, user, pass string) {
	if user == "" || pass == "" {
		fmt.Fprintln(os.Stderr, "need --user and --pass")
		os.Exit(1)
	}
	c := intercept.NewTCP(addr, intercept.Credentials{User: user, Pass: pass})
	c.SetStatus(func(s string) { fmt.Println(">", s) })
	if err := c.Start(context.Background()); err != nil {
		fmt.Println("ERR:", err)
		os.Exit(1)
	}
	fmt.Println("OK user:", c.User())
}
