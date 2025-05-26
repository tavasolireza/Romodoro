package main

import (
	"database/sql"
	"fmt"
	"os/exec"
	"strconv"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type AppState int

const (
	StateMainMenu AppState = iota
	StateTimerSetup
	StateTimer
	StatePaused
	StateSessionBrowser
)

type TimerPhase int

const (
	PhaseFocus TimerPhase = iota
	PhaseRest
)

type App struct {
	db           *sql.DB
	state        AppState
	session      *Session
	currentSplit *PomodoroSplit

	// Timer state
	phase            TimerPhase
	totalSeconds     int
	remainingSeconds int
	isPaused         bool

	// UI components
	textInput textinput.Model
	progress  progress.Model

	// Input state
	focusInput string
	restInput  string
	inputStep  int // 0: focus, 1: rest

	// Session browser state
	sessions        []Session
	selectedSession int

	width  int
	height int
}

type TickMsg time.Time

func NewApp(db *sql.DB) *App {
	ti := textinput.New()
	ti.Placeholder = "Enter focus time in minutes..."
	ti.Focus()
	ti.CharLimit = 3
	ti.Width = 20

	prog := progress.New(progress.WithDefaultGradient())
	prog.Width = 60

	return &App{
		db:        db,
		state:     StateMainMenu,
		textInput: ti,
		progress:  prog,
	}
}

func (m *App) Init() tea.Cmd {
	return textinput.Blink
}

func (m *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.progress.Width = min(60, msg.Width-4)
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			if m.session != nil {
				m.saveCurrentState()
				CloseSession(m.db, m.session.ID)
			}
			return m, tea.Quit
		}

		switch m.state {
		case StateMainMenu:
			return m.updateMainMenu(msg)
		case StateTimerSetup:
			return m.updateTimerSetup(msg)
		case StateTimer:
			return m.updateTimer(msg)
		case StatePaused:
			return m.updatePaused(msg)
		case StateSessionBrowser:
			return m.updateSessionBrowser(msg)
		}

	case TickMsg:
		if m.state == StateTimer && !m.isPaused {
			return m.updateTick()
		}
	}

	return m, nil
}

func (m *App) updateMainMenu(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "1":
		// Continue last session
		session, err := GetLastSession(m.db)
		if err != nil {
			// No previous session, create new one
			return m.createNewSession()
		}
		m.session = session
		m.state = StateTimerSetup
		m.textInput.Placeholder = "Enter focus time in minutes..."
		m.textInput.SetValue("")
		return m, textinput.Blink
	case "2":
		return m.loadSessionBrowser()
	case "3":
		return m.createNewSession()
	}
	return m, nil
}

func (m *App) createNewSession() (tea.Model, tea.Cmd) {
	sessionName := fmt.Sprintf("Session_%s", time.Now().Format("2006-01-02_15-04-05"))
	session, err := CreateSession(m.db, sessionName)
	if err != nil {
		return m, tea.Quit
	}
	m.session = session
	m.state = StateTimerSetup
	m.textInput.Placeholder = "Enter focus time in minutes..."
	m.textInput.SetValue("")
	return m, textinput.Blink
}

func (m *App) updateTimerSetup(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg.String() {
	case "enter":
		if m.inputStep == 0 {
			// Focus time entered
			m.focusInput = m.textInput.Value()
			m.inputStep = 1
			m.textInput.Placeholder = "Enter rest time in minutes..."
			m.textInput.SetValue("")
			return m, textinput.Blink
		} else {
			// Rest time entered
			m.restInput = m.textInput.Value()
			return m.startTimer()
		}
	case "m", "M":
		m.state = StateMainMenu
		return m, nil
	}

	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m *App) startTimer() (tea.Model, tea.Cmd) {
	focusMinutes, err := strconv.Atoi(m.focusInput)
	if err != nil || focusMinutes <= 0 {
		m.inputStep = 0
		m.textInput.Placeholder = "Invalid focus time. Enter focus time in minutes..."
		m.textInput.SetValue("")
		return m, textinput.Blink
	}

	restMinutes, err := strconv.Atoi(m.restInput)
	if err != nil || restMinutes < 0 {
		m.inputStep = 1
		m.textInput.Placeholder = "Invalid rest time. Enter rest time in minutes..."
		m.textInput.SetValue("")
		return m, textinput.Blink
	}

	split, err := CreatePomodoroSplit(m.db, m.session.ID, focusMinutes, restMinutes)
	if err != nil {
		return m, tea.Quit
	}

	m.currentSplit = split
	m.phase = PhaseFocus
	m.totalSeconds = focusMinutes * 60
	m.remainingSeconds = m.totalSeconds
	m.state = StateTimer
	m.isPaused = false

	// Reset for next split
	m.inputStep = 0
	m.focusInput = ""
	m.restInput = ""

	return m, m.tickCmd()
}

func (m *App) updateTimer(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "p", "P":
		m.state = StatePaused
		m.isPaused = true
		return m, nil
	case "b", "B":
		m.saveCurrentState()
		m.refreshSessionData() // Add this line
		m.state = StateTimerSetup
		m.textInput.Placeholder = "Enter focus time in minutes..."
		m.textInput.SetValue("")
		return m, textinput.Blink
	case "m", "M":
		m.saveCurrentState()
		m.state = StateMainMenu
		return m, nil
	}
	return m, nil
}

func (m *App) updatePaused(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "s", "S", "c", "C":
		m.state = StateTimer
		m.isPaused = false
		return m, m.tickCmd()
	case "b", "B":
		m.saveCurrentState()
		m.refreshSessionData() // Add this line
		m.state = StateTimerSetup
		m.textInput.Placeholder = "Enter focus time in minutes..."
		m.textInput.SetValue("")
		return m, textinput.Blink
	case "m", "M":
		m.saveCurrentState()
		m.state = StateMainMenu
		return m, nil
	}
	return m, nil
}

func (m *App) refreshSessionData() {
	if m.session == nil {
		return
	}

	// Get updated session data from database
	var session Session
	var endTime time.Time

	err := m.db.QueryRow(`
		SELECT id, name, start_time, end_time, total_focus_seconds, total_rest_seconds
		FROM sessions
		WHERE id = ?
	`, m.session.ID).Scan(&session.ID, &session.Name, &session.StartTime, &endTime,
		&session.TotalFocusSeconds, &session.TotalRestSeconds)

	if err == nil {
		session.EndTime = &endTime
		m.session = &session
	}
}

func (m *App) loadSessionBrowser() (tea.Model, tea.Cmd) {
	sessions, err := GetAllSessions(m.db)
	if err != nil {
		return m, tea.Quit
	}
	m.sessions = sessions
	m.selectedSession = 0
	m.state = StateSessionBrowser
	return m, nil
}

func (m *App) updateSessionBrowser(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.selectedSession > 0 {
			m.selectedSession--
		}
	case "down", "j":
		if m.selectedSession < len(m.sessions)-1 {
			m.selectedSession++
		}
	case "x", "X":
		if len(m.sessions) > 0 {
			sessionID := m.sessions[m.selectedSession].ID
			err := DeleteSession(m.db, sessionID)
			if err != nil {
				return m, nil
			}
			// Reload sessions
			return m.loadSessionBrowser()
		}
	case "b", "B", "m", "M":
		m.state = StateMainMenu
		return m, nil
	}
	return m, nil
}

func (m *App) updateTick() (tea.Model, tea.Cmd) {
	m.remainingSeconds--

	if m.remainingSeconds <= 0 {
		// Phase completed
		if m.phase == PhaseFocus {
			// Focus phase completed, start rest
			m.currentSplit.ActualFocusSeconds = m.totalSeconds
			m.phase = PhaseRest
			m.totalSeconds = m.currentSplit.RestMinutes * 60
			m.remainingSeconds = m.totalSeconds
			m.playSound()
			return m, m.tickCmd()
		} else {
			// Rest phase completed, split finished
			m.currentSplit.ActualRestSeconds = m.totalSeconds
			m.finishSplit()
			m.playSound()
			m.state = StateTimerSetup
			m.textInput.Placeholder = "Enter focus time in minutes..."
			m.textInput.SetValue("")
			return m, textinput.Blink
		}
	}

	return m, m.tickCmd()
}

func (m *App) finishSplit() {
	now := time.Now()
	m.currentSplit.EndTime = &now
	m.currentSplit.Status = "completed"

	UpdatePomodoroSplit(m.db, m.currentSplit)
	UpdateSessionTotals(m.db, m.session.ID)

	// Refresh session totals
	session, _ := GetLastSession(m.db)
	if session != nil && session.ID == m.session.ID {
		m.session = session
	}
}

func (m *App) saveCurrentState() {
	if m.currentSplit == nil {
		return
	}

	now := time.Now()
	m.currentSplit.EndTime = &now
	m.currentSplit.Status = "cancelled"

	if m.phase == PhaseFocus {
		m.currentSplit.ActualFocusSeconds = m.totalSeconds - m.remainingSeconds
	} else {
		m.currentSplit.ActualFocusSeconds = m.currentSplit.FocusMinutes * 60
		m.currentSplit.ActualRestSeconds = m.totalSeconds - m.remainingSeconds
	}

	UpdatePomodoroSplit(m.db, m.currentSplit)
	UpdateSessionTotals(m.db, m.session.ID)
}

func (m *App) tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

func (m *App) playSound() {
	// Use macOS built-in sound
	go func() {
		cmd := exec.Command("afplay", "/System/Library/Sounds/Blow.aiff")
		cmd.Run()
	}()
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
