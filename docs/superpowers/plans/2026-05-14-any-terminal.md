# Any Terminal Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Allow `hmm` to spawn its TUI in new windows for various terminal emulators (Kitty, WezTerm, iTerm, etc.) and fallback to inline execution.

**Architecture:** We will create an `internal/terminal` package with a `Launcher` struct or function that detects the terminal via env vars (like `TERM_PROGRAM` or `HMM_TERMINAL_CMD`) and returns the correct `exec.Cmd`. `main.go` will use this to launch the new window. If it fails or is not supported, `main.go` will run the TUI inline.

**Tech Stack:** Go, standard library `os/exec`.

---

### Task 1: Create `internal/terminal` package and tests for terminal detection

**Files:**
- Create: `internal/terminal/terminal.go`
- Create: `internal/terminal/terminal_test.go`

- [ ] **Step 1: Write the failing tests**

```go
// internal/terminal/terminal_test.go
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
			wantArgs: []string{"tmux", "new-window", "sh -c 'echo '\\''hello'\\'''"}, // or simpler
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
			wantArgs: nil, // Indicates it should run inline
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
				continue
			}
			if cmd == nil {
				t.Fatalf("expected command %s, got nil", tt.wantCmd)
			}
			if cmd.Path != tt.wantCmd && cmd.Args[0] != tt.wantCmd {
				t.Errorf("expected cmd path/name %s, got %s", tt.wantCmd, cmd.Path)
			}
			// Only check first few args to simplify test, or exact match if needed.
			// Just a basic check here.
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/terminal/... -v`
Expected: FAIL due to missing package/function.

- [ ] **Step 3: Write minimal implementation**

```go
// internal/terminal/terminal.go
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
			// +new-window requires IPC, so just return the exact command
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
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/terminal/... -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/terminal/
git commit -m "feat: add terminal detection and command building logic"
```

### Task 2: Integrate `internal/terminal` in `main.go`

**Files:**
- Modify: `main.go`

- [ ] **Step 1: Replace `launchGhostty` in `main.go`**

```go
// main.go (around line 187)
// Remove the existing `launchGhostty` function.
// Replace the logic inside main() that calls `launchGhostty` with:

// ... around line 79:
cmd := terminal.BuildCommand(shellCmd)
if cmd != nil {
	if err := cmd.Start(); err != nil {
		log.Warn("Failed to open terminal window, falling back to inline", "error", err)
		// Don't return, let it fall through to inline execution
	} else {
		return // Successfully spawned new window
	}
}
// If cmd == nil or failed to start, it will just continue executing the rest of the main function inline.

// ... around line 115 (the non-wait mode):
cmd := terminal.BuildCommand(shellCmd)
if cmd != nil {
	if err := cmd.Start(); err != nil {
		log.Warn("Failed to open terminal window, falling back to inline", "error", err)
	} else {
		return
	}
}
// It will fall through to the rest of main() and run inline.
```

- [ ] **Step 2: Ensure imports are updated**

Add `"github.com/suntianxun/hmm/internal/terminal"` to imports. Run `go imports` or `go fmt`.

- [ ] **Step 3: Run the project tests**

Run: `go test ./... -v`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add main.go
git commit -m "feat: integrate agnostic terminal launching and fallback"
```

### Task 3: Update documentation

**Files:**
- Modify: `README.md`
- Modify: `python/README.md`

- [ ] **Step 1: Update README.md**

```markdown
// Update references from "new Ghostty terminal window" to "new terminal window".
// Add a section on Supported Terminals: Ghostty, Kitty, WezTerm, iTerm2, Apple Terminal, tmux.
// Document HMM_TERMINAL_CMD.
```

- [ ] **Step 2: Update python/README.md**

```markdown
// Remove specific mention of Ghostty, use generic "terminal".
```

- [ ] **Step 3: Commit**

```bash
git add README.md python/README.md
git commit -m "docs: document supported terminals and HMM_TERMINAL_CMD"
```
