package app

import (
	"errors"
	"time"

	"github.com/creack/pty"
)

func (a *terminalAttachment) Replay() []byte {
	return append([]byte(nil), a.replay...)
}

func (a *terminalAttachment) Cursor() int64 {
	return a.cursor
}

func (a *terminalAttachment) Write(data []byte) error {
	a.record.mu.Lock()
	file := a.record.ptyFile
	running := a.record.info.Status == TerminalStatusRunning
	a.record.mu.Unlock()
	if file == nil || !running {
		return errors.New("terminal is not running")
	}
	_, err := file.Write(data)
	return err
}

func (a *terminalAttachment) Resize(rows int, cols int) error {
	rows, cols = normalizeTerminalSize(rows, cols)
	a.record.mu.Lock()
	file := a.record.ptyFile
	a.record.info.Rows = rows
	a.record.info.Cols = cols
	a.record.info.TimeUpdated = time.Now()
	a.record.mu.Unlock()
	if file == nil {
		return errors.New("terminal is not running")
	}
	return pty.Setsize(file, &pty.Winsize{Rows: uint16(rows), Cols: uint16(cols)})
}

func (a *terminalAttachment) Data() <-chan []byte {
	return a.data
}

func (a *terminalAttachment) Detach() {
	a.once.Do(func() {
		a.record.mu.Lock()
		_, ok := a.record.subscribers[a.data]
		if ok {
			delete(a.record.subscribers, a.data)
		}
		a.record.mu.Unlock()
		if ok {
			close(a.data)
		}
	})
}
