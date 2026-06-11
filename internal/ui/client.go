package ui

import (
	"context"

	tea "charm.land/bubbletea/v2"

	"intertui/internal/config"
	"intertui/internal/intercept"
	filelog "intertui/internal/log"
)

type clientReadyMsg struct {
	client *intercept.Client
	user   string
	err    error
}

type connectProgressMsg struct {
	statusCh <-chan string
	doneCh   <-chan clientReadyMsg
}

func startClient(cfg config.Config) tea.Cmd {
	statusCh := make(chan string, 32)
	doneCh := make(chan clientReadyMsg, 1)

	go func() {
		c := cfg.NewClient()
		c.SetStatus(func(line string) {
			filelog.Status(line)
			select {
			case statusCh <- line:
			default:
			}
		})

		err := c.Start(context.Background())
		c.SetStatus(nil)
		doneCh <- clientReadyMsg{
			client: c,
			user:   c.User(),
			err:    err,
		}
		close(statusCh)
	}()

	return pollConnect(statusCh, doneCh)
}

func pollConnect(statusCh <-chan string, doneCh <-chan clientReadyMsg) tea.Cmd {
	return func() tea.Msg {
		select {
		case line, ok := <-statusCh:
			if !ok {
				return <-doneCh
			}
			_ = line
			return connectProgressMsg{
				statusCh: statusCh,
				doneCh:   doneCh,
			}
		case result := <-doneCh:
			return result
		}
	}
}

func waitClientMsg(c *intercept.Client) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-c.Messages()
		if !ok {
			return intercept.DisconnectedMsg{}
		}
		return msg
	}
}
