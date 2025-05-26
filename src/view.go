package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4")).
			Padding(0, 1).
			Align(lipgloss.Center)

	sessionHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#04B575")).
				Border(lipgloss.RoundedBorder()).
				Padding(1, 2).
				Margin(1, 0).
				Align(lipgloss.Center)

	timerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#FF6B6B")).
			Padding(1, 2).
			Margin(1, 0).
			Align(lipgloss.Center)

	restTimerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#4ECDC4")).
			Padding(1, 2).
			Margin(1, 0).
			Align(lipgloss.Center)

	pausedStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#FFE66D")).
			Padding(1, 2).
			Margin(1, 0).
			Align(lipgloss.Center)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#626262")).
			Margin(1, 0).
			Align(lipgloss.Center)

	inputStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			Padding(1, 2).
			Margin(1, 0).
			Align(lipgloss.Center)

	menuStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			Padding(2, 4).
			Margin(1, 0).
			Align(lipgloss.Center)

	sessionRowStyle = lipgloss.NewStyle().
			Padding(0, 1).
			Margin(0, 0)

	selectedSessionRowStyle = lipgloss.NewStyle().
				Padding(0, 1).
				Margin(0, 0).
				Background(lipgloss.Color("#7D56F4")).
				Foreground(lipgloss.Color("#FFFFFF"))

	browserStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			Padding(1, 2).
			Margin(1, 0).
			Height(20).
			Align(lipgloss.Center)
)

func (m *App) View() string {
	var sections []string

	// Title
	title := titleStyle.Width(60).Render("🍅 ROMODORO")
	sections = append(sections, title)

	switch m.state {
	case StateMainMenu:
		sections = append(sections, m.viewMainMenu())
	case StateTimerSetup:
		sections = append(sections, m.viewSessionHeader())
		sections = append(sections, m.viewTimerSetup())
	case StateTimer:
		sections = append(sections, m.viewSessionHeader())
		sections = append(sections, m.viewTimer())
	case StatePaused:
		sections = append(sections, m.viewSessionHeader())
		sections = append(sections, m.viewPaused())
	case StateSessionBrowser:
		sections = append(sections, m.viewSessionBrowser())
	}

	content := lipgloss.JoinVertical(lipgloss.Center, sections...)

	// Center the entire content
	if m.width > 0 {
		content = lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
	}

	return content
}

func (m *App) viewMainMenu() string {
	var content strings.Builder

	content.WriteString("Welcome to Romodoro!\n\n")
	content.WriteString("1. Continue Session\n")
	content.WriteString("2. Browse Previous Sessions\n")
	content.WriteString("3. Create New Session\n\n")
	content.WriteString("Press 'q' or Ctrl+C to quit")

	return menuStyle.Width(50).Render(content.String())
}

func (m *App) viewSessionHeader() string {
	if m.session == nil {
		return ""
	}

	focusTime := m.formatDuration(m.session.TotalFocusSeconds)
	restTime := m.formatDuration(m.session.TotalRestSeconds)

	content := fmt.Sprintf(
		"📝 Session: %s\n"+
			"🎯 Total Focus Time: %s\n"+
			"☕ Total Rest Time: %s",
		m.session.Name,
		focusTime,
		restTime,
	)

	return sessionHeaderStyle.Width(60).Render(content)
}

func (m *App) viewTimerSetup() string {
	var content strings.Builder

	if m.inputStep == 0 {
		content.WriteString("🎯 Set Focus Time\n\n")
		content.WriteString(m.textInput.View())
		content.WriteString("\n\nEnter focus time in minutes and press Enter\n")
		content.WriteString("Press 'm' to go back to main menu")
	} else {
		content.WriteString("☕ Set Rest Time\n\n")
		content.WriteString(fmt.Sprintf("Focus: %s minutes\n", m.focusInput))
		content.WriteString(m.textInput.View())
		content.WriteString("\n\nEnter rest time in minutes and press Enter\n")
		content.WriteString("Press 'm' to go back to main menu")
	}

	return inputStyle.Width(60).Render(content.String())
}

func (m *App) viewTimer() string {
	var content strings.Builder
	var style lipgloss.Style

	if m.phase == PhaseFocus {
		content.WriteString("🎯 FOCUS TIME\n\n")
		style = timerStyle
	} else {
		content.WriteString("☕ REST TIME\n\n")
		style = restTimerStyle
	}

	// Progress bar
	progress := float64(m.totalSeconds-m.remainingSeconds) / float64(m.totalSeconds)
	progressBar := m.progress.ViewAs(progress)
	content.WriteString(progressBar)
	content.WriteString("\n\n")

	// Time remaining
	timeStr := m.formatDuration(m.remainingSeconds)
	content.WriteString(fmt.Sprintf("Time Remaining: %s\n\n", timeStr))

	// Current split info
	content.WriteString(fmt.Sprintf(
		"Current Split: %dm focus / %dm rest\n\n",
		m.currentSplit.FocusMinutes,
		m.currentSplit.RestMinutes,
	))

	content.WriteString("Press 'p' to pause • 'b' back to session • 'm' main menu")

	return style.Width(70).Render(content.String())
}

func (m *App) viewPaused() string {
	var content strings.Builder

	content.WriteString("⏸️  PAUSED\n\n")

	// Time remaining
	timeStr := m.formatDuration(m.remainingSeconds)
	content.WriteString(fmt.Sprintf("Time Remaining: %s\n\n", timeStr))

	if m.phase == PhaseFocus {
		content.WriteString("Phase: 🎯 Focus\n\n")
	} else {
		content.WriteString("Phase: ☕ Rest\n\n")
	}

	content.WriteString("Press 's' or 'c' to continue • 'b' back to session • 'm' main menu")

	return pausedStyle.Width(70).Render(content.String())
}

func (m *App) viewSessionBrowser() string {
	var content strings.Builder

	content.WriteString("📊 Session History\n\n")

	if len(m.sessions) == 0 {
		content.WriteString("No sessions found.\n\n")
		content.WriteString("Press 'm' to go back to main menu")
		return browserStyle.Width(80).Render(content.String())
	}

	// Header - removed "Session Name" and "Status"
	header := fmt.Sprintf("%-15s %-15s %-12s %-12s",
		"Started", "Ended", "Focus", "Rest")
	content.WriteString(header + "\n")
	content.WriteString(strings.Repeat("─", 60) + "\n")

	// Sessions
	for i, session := range m.sessions {
		endedStr := session.EndTime.Format("01-02 15:04")
		focusStr := m.formatDuration(session.TotalFocusSeconds)
		restStr := m.formatDuration(session.TotalRestSeconds)
		startedStr := session.StartTime.Format("01-02 15:04")

		row := fmt.Sprintf("%-15s %-15s %-12s %-12s",
			startedStr, endedStr, focusStr, restStr)

		if i == m.selectedSession {
			content.WriteString(selectedSessionRowStyle.Render("→ "+row) + "\n")
		} else {
			content.WriteString(sessionRowStyle.Render("  "+row) + "\n")
		}
	}

	content.WriteString("\n")
	content.WriteString("↑/↓ or j/k to navigate • 'x' to delete • 'm' to go back")

	return browserStyle.Width(70).Render(content.String())
}

func (m *App) formatDuration(seconds int) string {
	duration := time.Duration(seconds) * time.Second
	minutes := int(duration.Minutes())
	secs := int(duration.Seconds()) % 60
	return fmt.Sprintf("%02d:%02d", minutes, secs)
}
