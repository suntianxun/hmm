# Terminal Agnostic Window Launching Design

## Objective
Update `hmm` to support opening its TUI in new windows across various terminal emulators, moving away from being exclusively tied to Ghostty.

## Architecture

1. **Terminal Launch Logic (`main.go`)**:
   - The current `launchGhostty` function will be replaced with a generic `launchTerminal(shellCmd string) error`.
   - The `if os.Getenv("HMM_INNER") != "1"` block in `main()` will invoke this generic function instead.

2. **Detection & Execution Flow**:
   - **Step 1: Custom Configuration**: Check the `HMM_TERMINAL_CMD` environment variable. If the user provides a custom command, format it and execute it.
   - **Step 2: Auto-Detection**: Use environment variables to identify the current terminal:
     - Check `$TERM_PROGRAM` for `Ghostty`, `WezTerm`, `iTerm.app`, `Apple_Terminal`, `tmux`.
     - Check `$KITTY_PID` for Kitty.
   - **Step 3: Launch**: Based on detection, run the respective CLI command:
     - **Ghostty**: `ghostty +new-window --command=/bin/sh -c '...'` (fallback `ghostty -e ...`)
     - **Kitty**: `kitty -e sh -c '...'`
     - **WezTerm**: `wezterm start -- sh -c '...'`
     - **tmux**: `tmux new-window "sh -c '...'"`
     - **Apple Terminal**: `osascript -e 'tell application "Terminal" to do script "..."'`
     - **iTerm2**: `osascript -e 'tell application "iTerm" to create window with default profile command "..."'`
   - **Step 4: Fallback**: If no supported terminal is detected, or the launch command returns an error, log a warning and run the TUI inline (in the current terminal window) instead of failing fatally.

## Fallback Mechanism
Running inline will be the ultimate fallback. This ensures `hmm` remains usable even if window spawning fails, although it will block the current session (e.g. a Python debugger). 

## Documentation
- The `README.md` and `python/README.md` will be updated to explain the new behavior.
- Supported terminals will be listed.
- Instructions for configuring `HMM_TERMINAL_CMD` will be added.
