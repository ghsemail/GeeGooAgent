package chatrepl

import (
	"os"
	"os/exec"
	"runtime"

	"golang.org/x/term"
)

// ttyState holds the pre-prompt terminal state so we can undo go-prompt raw mode.
type ttyState struct {
	fd    int
	state *term.State
}

func saveTTY() *ttyState {
	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		return nil
	}
	state, err := term.GetState(fd)
	if err != nil || state == nil {
		return nil
	}
	return &ttyState{fd: fd, state: state}
}

func restoreTTY(s *ttyState) {
	if s == nil || s.state == nil {
		return
	}
	_ = term.Restore(s.fd, s.state)
	// go-prompt v0.2.6 can leave POSIX tty flags corrupted even after its own TearDown.
	if runtime.GOOS != "windows" {
		cmd := exec.Command("stty", "sane")
		cmd.Stdin = os.Stdin
		_ = cmd.Run()
	}
}
