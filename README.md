# Romodoro

A terminal-based Pomodoro timer application built with Go. Romodoro provides a clean, customizable interface for managing focus and rest sessions with persistent session tracking.

## Features

- Customizable focus and rest periods for each session
- Session persistence with SQLite database
- Beautiful terminal UI with progress bars and animations
- Session history browser with ability to delete old sessions
- Pause/resume functionality
- Sound notifications when timers complete
- Automatic session totals tracking

## Requirements

- Go 1.19 or higher
- macOS, Linux, or Windows
- Terminal emulator (tested with iTerm2, Ghostty, and standard terminals)

## Installation

1. Clone the repository:
```bash
git clone https://github.com/yourusername/romodoro.git
cd romodoro
```

2. Install dependencies:
```bash
go mod tidy
```

3. Build the application:
```bash
go build -o bin/romodoro .
```

4. (Optional) Add to your PATH:
```bash
# Add this line to your ~/.zshrc or ~/.bash_profile
export PATH="$PWD/bin:$PATH"
```

## Usage

Run the application:
```bash
./bin/romodoro
```

Or if added to PATH:
```bash
romodoro
```

### Controls

- **Main Menu**: Use number keys (1-3) to navigate options
- **Timer**:
  - `p` - Pause timer
  - `b` - Back to session setup (saves progress)
  - `m` - Return to main menu (saves progress)
  - `s` or `c` - Continue from pause
- **Session Browser**:
  - Arrow keys or `j`/`k` - Navigate sessions
  - `x` - Delete selected session
  - `m` - Return to main menu
- **Global**: `q` or `Ctrl+C` - Quit application

## Platform-Specific Configuration

### Sound Notifications

The application uses platform-specific sound commands:

**macOS** (default):
```go
cmd := exec.Command("afplay", "/System/Library/Sounds/Glass.aiff")
```

**Linux** (requires `aplay` or `paplay`):
```go
// For ALSA
cmd := exec.Command("aplay", "/usr/share/sounds/alsa/Front_Left.wav")
// For PulseAudio
cmd := exec.Command("paplay", "/usr/share/sounds/alsa/Front_Left.wav")
```

**Windows** (requires PowerShell):
```go
cmd := exec.Command("powershell", "-c", "(New-Object Media.SoundPlayer 'C:\\Windows\\Media\\notify.wav').PlaySync();")
```

To modify the sound, edit the `playSound()` function in `models.go`.

### Database Location

By default, the application creates a `data/` directory in your home folder at `~/romodoro/data/sessions.db`. To change this location, modify the `dbPath` variable in `main.go`.

## Development

### Project Structure

- `src/main.go` - Application entry point and initialization
- `src/database.go` - SQLite database operations and schema
- `src/models.go` - Application state management and business logic
- `src/view.go` - Terminal UI rendering and styling

### Dependencies

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - Terminal UI framework
- [Lipgloss](https://github.com/charmbracelet/lipgloss) - Styling and layout
- [Bubbles](https://github.com/charmbracelet/bubbles) - UI components
- [go-sqlite3](https://github.com/mattn/go-sqlite3) - SQLite driver

## License

MIT License - see LICENSE file for details.
