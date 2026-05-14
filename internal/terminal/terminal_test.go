package terminal_test

import (
	"os"
	"testing"
	"github.com/suntianxun/hmm/internal/terminal"
)

func TestCommandBuilder(t *testing.T) {
	tests := []struct {
		name     string
		env      map[string]string
		shellCmd string
		wantCmd  string
		wantArgs []string
	}{
		{
			name:     "Custom HMM_TERMINAL_CMD",
			env:      map[string]string{"HMM_TERMINAL_CMD": "myterm -x"},
			shellCmd: "echo 'hello'",
			wantCmd:  "myterm",
			wantArgs: []string{"myterm", "-x", "echo 'hello'"},
		},
		{
			name:     "Ghostty detection",
			env:      map[string]string{"TERM_PROGRAM": "Ghostty"},
			shellCmd: "echo 'hello'",
			wantCmd:  "ghostty",
			wantArgs: []string{"ghostty", "+new-window", "--command=/bin/sh -c 'echo '\"'\"'hello'\"'\"''"},
		},
		{
			name:     "Kitty detection",
			env:      map[string]string{"KITTY_PID": "123"},
			shellCmd: "echo 'hello'",
			wantCmd:  "kitty",
			wantArgs: []string{"kitty", "-e", "sh", "-c", "echo 'hello'"},
		},
		{
			name:     "WezTerm detection",
			env:      map[string]string{"TERM_PROGRAM": "WezTerm"},
			shellCmd: "echo 'hello'",
			wantCmd:  "wezterm",
			wantArgs: []string{"wezterm", "start", "--", "sh", "-c", "echo 'hello'"},
		},
		{
			name:     "tmux detection",
			env:      map[string]string{"TERM_PROGRAM": "tmux"},
			shellCmd: "echo 'hello'",
			wantCmd:  "tmux",
			wantArgs: []string{"tmux", "new-window", "sh -c 'echo '\\''hello'\\'''"},
		},
		{
			name:     "Apple Terminal detection",
			env:      map[string]string{"TERM_PROGRAM": "Apple_Terminal"},
			shellCmd: "echo 'hello'",
			wantCmd:  "osascript",
			wantArgs: []string{"osascript", "-e", `tell application "Terminal" to do script "echo 'hello'"`},
		},
		{
			name:     "iTerm.app detection",
			env:      map[string]string{"TERM_PROGRAM": "iTerm.app"},
			shellCmd: "echo 'hello'",
			wantCmd:  "osascript",
			wantArgs: []string{"osascript", "-e", `tell application "iTerm" to create window with default profile command "echo 'hello'"`},
		},
		{
			name:     "Unsupported fallback",
			env:      map[string]string{"TERM_PROGRAM": "Unknown"},
			shellCmd: "echo 'hello'",
			wantCmd:  "",
			wantArgs: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Clearenv()
			for k, v := range tt.env {
				os.Setenv(k, v)
			}
			cmd := terminal.BuildCommand(tt.shellCmd)
			if tt.wantCmd == "" {
				if cmd != nil {
					t.Errorf("expected nil command for fallback, got %v", cmd)
				}
				return // changed continue to return
			}
			if cmd == nil {
				t.Fatalf("expected command %s, got nil", tt.wantCmd)
			}
			if cmd.Path != tt.wantCmd && cmd.Args[0] != tt.wantCmd {
				t.Errorf("expected cmd path/name %s, got %s", tt.wantCmd, cmd.Path)
			}
		})
	}
}
