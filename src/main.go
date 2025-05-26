package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal("Could not get home directory:", err)
	}

	dbPath := filepath.Join(homeDir, "romodoro", "data", "sessions.db")
	
	// Ensure data directory exists
	dataDir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		log.Fatal("Could not create data directory:", err)
	}

	db, err := InitDB(dbPath)
	if err != nil {
		log.Fatal("Could not initialize database:", err)
	}
	defer db.Close()

	app := NewApp(db)
	
	p := tea.NewProgram(app, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running program: %v", err)
		os.Exit(1)
	}
}
