package terminal

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"runtime"
)

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
}

func BuildCommand(shellCmd string) *exec.Cmd {
	if custom := os.Getenv("HMM_TERMINAL_CMD"); custom != "" {
		parts := strings.Fields(custom)
		if len(parts) > 0 {
			args := append(parts[1:], shellCmd)
			return exec.Command(parts[0], args...)
		}
	}

	if os.Getenv("KITTY_PID") != "" {
		return exec.Command("kitty", "-e", "sh", "-c", shellCmd)
	}

	switch os.Getenv("TERM_PROGRAM") {
	case "Ghostty":
		if runtime.GOOS == "darwin" {
			return exec.Command("ghostty", "+new-window", "--command=/bin/sh -c "+shellQuote(shellCmd))
		}
		return exec.Command("ghostty", "-e", "/bin/sh", "-c", shellCmd)
	case "WezTerm":
		return exec.Command("wezterm", "start", "--", "sh", "-c", shellCmd)
	case "tmux":
		return exec.Command("tmux", "new-window", fmt.Sprintf("sh -c %s", shellQuote(shellCmd)))
	case "Apple_Terminal":
		script := fmt.Sprintf(`tell application "Terminal" to do script "%s"`, strings.ReplaceAll(shellCmd, `"`, `\"`))
		return exec.Command("osascript", "-e", script)
	case "iTerm.app":
		script := fmt.Sprintf(`tell application "iTerm" to create window with default profile command "%s"`, strings.ReplaceAll(shellCmd, `"`, `\"`))
		return exec.Command("osascript", "-e", script)
	}

	return nil
}
