package main

import (
	"database/sql"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type Session struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	StartTime time.Time `json:"start_time"`
	EndTime   *time.Time `json:"end_time"`
	TotalFocusSeconds int `json:"total_focus_seconds"`
	TotalRestSeconds  int `json:"total_rest_seconds"`
}

type PomodoroSplit struct {
	ID              int       `json:"id"`
	SessionID       int       `json:"session_id"`
	FocusMinutes    int       `json:"focus_minutes"`
	RestMinutes     int       `json:"rest_minutes"`
	StartTime       time.Time `json:"start_time"`
	EndTime         *time.Time `json:"end_time"`
	Status          string    `json:"status"` // "completed", "cancelled", "in_progress"
	ActualFocusSeconds int    `json:"actual_focus_seconds"`
	ActualRestSeconds  int    `json:"actual_rest_seconds"`
}

func InitDB(dbPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	// Create sessions table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS sessions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			start_time DATETIME NOT NULL,
			end_time DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			total_focus_seconds INTEGER DEFAULT 0,
			total_rest_seconds INTEGER DEFAULT 0
		)
	`)
	if err != nil {
		return nil, err
	}

	// Create pomodoro_splits table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS pomodoro_splits (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			session_id INTEGER NOT NULL,
			focus_minutes INTEGER NOT NULL,
			rest_minutes INTEGER NOT NULL,
			start_time DATETIME NOT NULL,
			end_time DATETIME,
			status TEXT NOT NULL DEFAULT 'in_progress',
			actual_focus_seconds INTEGER DEFAULT 0,
			actual_rest_seconds INTEGER DEFAULT 0,
			FOREIGN KEY (session_id) REFERENCES sessions (id)
		)
	`)
	if err != nil {
		return nil, err
	}

	return db, nil
}

func CreateSession(db *sql.DB, name string) (*Session, error) {
	now := time.Now()
	result, err := db.Exec(
		"INSERT INTO sessions (name, start_time, end_time) VALUES (?, ?, ?)",
		name, now, now,
	)
	if err != nil {
		return nil, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	endTime := now
	return &Session{
		ID:        int(id),
		Name:      name,
		StartTime: now,
		EndTime:   &endTime,
	}, nil
}

func GetLastSession(db *sql.DB) (*Session, error) {
	var session Session
	var endTime time.Time

	err := db.QueryRow(`
		SELECT id, name, start_time, end_time, total_focus_seconds, total_rest_seconds
		FROM sessions
		ORDER BY start_time DESC
		LIMIT 1
	`).Scan(&session.ID, &session.Name, &session.StartTime, &endTime,
		&session.TotalFocusSeconds, &session.TotalRestSeconds)

	if err != nil {
		return nil, err
	}

	session.EndTime = &endTime
	return &session, nil
}

func GetAllSessions(db *sql.DB) ([]Session, error) {
	rows, err := db.Query(`
		SELECT id, name, start_time, end_time, total_focus_seconds, total_rest_seconds
		FROM sessions
		ORDER BY start_time DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []Session
	for rows.Next() {
		var session Session
		var endTime time.Time

		err := rows.Scan(&session.ID, &session.Name, &session.StartTime, &endTime,
			&session.TotalFocusSeconds, &session.TotalRestSeconds)
		if err != nil {
			return nil, err
		}

		session.EndTime = &endTime
		sessions = append(sessions, session)
	}

	return sessions, nil
}

func DeleteSession(db *sql.DB, sessionID int) error {
	// Delete pomodoro splits first (foreign key constraint)
	_, err := db.Exec("DELETE FROM pomodoro_splits WHERE session_id = ?", sessionID)
	if err != nil {
		return err
	}

	// Delete session
	_, err = db.Exec("DELETE FROM sessions WHERE id = ?", sessionID)
	return err
}

func CreatePomodoroSplit(db *sql.DB, sessionID, focusMinutes, restMinutes int) (*PomodoroSplit, error) {
	now := time.Now()
	result, err := db.Exec(`
		INSERT INTO pomodoro_splits (session_id, focus_minutes, rest_minutes, start_time)
		VALUES (?, ?, ?, ?)
	`, sessionID, focusMinutes, restMinutes, now)

	if err != nil {
		return nil, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	return &PomodoroSplit{
		ID:           int(id),
		SessionID:    sessionID,
		FocusMinutes: focusMinutes,
		RestMinutes:  restMinutes,
		StartTime:    now,
		Status:       "in_progress",
	}, nil
}

func UpdatePomodoroSplit(db *sql.DB, split *PomodoroSplit) error {
	_, err := db.Exec(`
		UPDATE pomodoro_splits
		SET end_time = ?, status = ?, actual_focus_seconds = ?, actual_rest_seconds = ?
		WHERE id = ?
	`, split.EndTime, split.Status, split.ActualFocusSeconds, split.ActualRestSeconds, split.ID)

	return err
}

func UpdateSessionTotals(db *sql.DB, sessionID int) error {
	now := time.Now()
	_, err := db.Exec(`
		UPDATE sessions
		SET end_time = ?,
		total_focus_seconds = (
			SELECT COALESCE(SUM(actual_focus_seconds), 0)
			FROM pomodoro_splits
			WHERE session_id = ?
		),
		total_rest_seconds = (
			SELECT COALESCE(SUM(actual_rest_seconds), 0)
			FROM pomodoro_splits
			WHERE session_id = ?
		)
		WHERE id = ?
	`, now, sessionID, sessionID, sessionID)

	return err
}

func CloseSession(db *sql.DB, sessionID int) error {
	now := time.Now()
	_, err := db.Exec("UPDATE sessions SET end_time = ? WHERE id = ?", now, sessionID)
	return err
}
