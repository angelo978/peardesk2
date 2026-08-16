// Package clipboard polls the system clipboard and notifies on changes.
package clipboard

import (
	"time"

	"github.com/atotto/clipboard"
)

// Monitor polls the clipboard every interval and calls onChange when the
// text content changes. Call Stop() to halt monitoring.
type Monitor struct {
	interval time.Duration
	last     string
	stopCh   chan struct{}
}

func New(interval time.Duration) *Monitor {
	return &Monitor{interval: interval, stopCh: make(chan struct{})}
}

// Start begins polling. onChange is called in the monitor goroutine.
func (m *Monitor) Start(onChange func(text string)) {
	// Seed initial value without triggering onChange.
	m.last, _ = clipboard.ReadAll()
	go func() {
		ticker := time.NewTicker(m.interval)
		defer ticker.Stop()
		for {
			select {
			case <-m.stopCh:
				return
			case <-ticker.C:
				text, err := clipboard.ReadAll()
				if err != nil || text == m.last {
					continue
				}
				m.last = text
				onChange(text)
			}
		}
	}()
}

// Stop halts the monitor goroutine.
func (m *Monitor) Stop() {
	select {
	case m.stopCh <- struct{}{}:
	default:
	}
}

// Write sets the clipboard content (and updates the internal snapshot so the
// same text is not echoed back to the remote).
func (m *Monitor) Write(text string) error {
	m.last = text
	return clipboard.WriteAll(text)
}

// Read returns the current clipboard content.
func Read() (string, error) {
	return clipboard.ReadAll()
}

// Write sets the clipboard without a monitor snapshot.
func Write(text string) error {
	return clipboard.WriteAll(text)
}
