package intercept

import "sync"

type waitEntry struct {
	match func(Envelope) bool
	ch    chan Envelope
}

// Waiter resolves synchronous RPC-style waits against inbound events.
type Waiter struct {
	mu    sync.Mutex
	waits []waitEntry
}

// Wait registers a matcher and returns a channel that receives the first match.
func (w *Waiter) Wait(match func(Envelope) bool) <-chan Envelope {
	ch := make(chan Envelope, 1)
	w.mu.Lock()
	w.waits = append(w.waits, waitEntry{match: match, ch: ch})
	w.mu.Unlock()
	return ch
}

// WaitEvent waits for a specific event name.
func (w *Waiter) WaitEvent(name string) <-chan Envelope {
	return w.Wait(func(e Envelope) bool { return e.Event == name })
}

// WaitCommand waits for a command response matching the first word of cmd.
func (w *Waiter) WaitCommand(firstWord string) <-chan Envelope {
	return w.Wait(func(e Envelope) bool {
		if e.Event == "command" && e.Cmd == firstWord {
			return true
		}
		if e.Event == "error" && e.Error == "Invalid command" {
			return true
		}
		return false
	})
}

// Cancel removes a pending wait so an abandoned matcher (e.g. a timed-out
// query) cannot claim a later, unrelated response.
func (w *Waiter) Cancel(ch <-chan Envelope) {
	w.mu.Lock()
	defer w.mu.Unlock()

	for i, entry := range w.waits {
		if entry.ch == ch {
			w.waits = append(w.waits[:i], w.waits[i+1:]...)
			return
		}
	}
}

// Deliver offers an event to the first matching waiter.
func (w *Waiter) Deliver(env Envelope) bool {
	w.mu.Lock()
	defer w.mu.Unlock()

	for i, entry := range w.waits {
		if entry.match(env) {
			entry.ch <- env
			w.waits = append(w.waits[:i], w.waits[i+1:]...)
			return true
		}
	}
	return false
}
