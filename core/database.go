package core

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

type Database struct {
	dbFile string
	conn   *sql.DB
}

func NewDatabase(dbFile string) *Database {
	if dbFile == "" {
		dbFile = "time_tracker.db"
	}

	var dbDir string
	if homeDir, err := os.UserHomeDir(); err == nil {
		dbDir = filepath.Join(homeDir, ".time-tracker")
	} else {
		panic(fmt.Sprintf("Failed to determine user home directory: %v", err))
	}
	err := os.MkdirAll(dbDir, os.ModePerm)
	if err != nil {
		panic(fmt.Sprintf("Failed to create database directory: %v", err))
	}
	return &Database{
		dbFile: filepath.Join(dbDir, dbFile),
	}
}

func (db *Database) Connect() error {
	conn, err := sql.Open("sqlite3", db.dbFile)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	db.conn = conn

	err = db.initDatabase()
	if err != nil {
		return err
	}

	err = db.checkAndUpdateSchema()
	if err != nil {
		return err
	}

	return nil
}

func (db *Database) initDatabase() error {
	query := `
    CREATE TABLE IF NOT EXISTS activities (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        task TEXT NOT NULL,
        start_time TEXT NOT NULL,
        end_time TEXT,
        duration INTEGER,
        screenshot_path TEXT,
        keyboard_event_count INTEGER DEFAULT 0,
        mouse_event_count INTEGER DEFAULT 0
    )`
	_, err := db.conn.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	return nil
}

func (db *Database) checkAndUpdateSchema() error {
	query := "PRAGMA table_info(activities)"
	rows, err := db.conn.Query(query)
	if err != nil {
		return fmt.Errorf("failed to fetch table info: %w", err)
	}
	defer rows.Close()

	columns := make(map[string]bool)
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dfltValue sql.NullString
		err := rows.Scan(&cid, &name, &ctype, &notnull, &dfltValue, &pk)
		if err != nil {
			return fmt.Errorf("failed to scan table info: %w", err)
		}
		columns[name] = true
	}

	if !columns["keyboard_event_count"] {
		_, err := db.conn.Exec(`
        ALTER TABLE activities
        ADD COLUMN keyboard_event_count INTEGER DEFAULT 0
        `)
		if err != nil {
			return fmt.Errorf("failed to add keyboard_event_count column: %w", err)
		}
	}

	if !columns["mouse_event_count"] {
		_, err := db.conn.Exec(`
        ALTER TABLE activities
        ADD COLUMN mouse_event_count INTEGER DEFAULT 0
        `)
		if err != nil {
			return fmt.Errorf("failed to add mouse_event_count column: %w", err)
		}
	}

	return nil
}

func (db *Database) SaveActivity(task, startTime, endTime string, duration int, screenshotPath string, keyboardEventCount, mouseEventCount int) error {
	query := `
    INSERT INTO activities (task, start_time, end_time, duration, screenshot_path, keyboard_event_count, mouse_event_count)
    VALUES (?, ?, ?, ?, ?, ?, ?)`
	_, err := db.conn.Exec(query, task, startTime, endTime, duration, screenshotPath, keyboardEventCount, mouseEventCount)
	if err != nil {
		return fmt.Errorf("failed to save activity: %w", err)
	}
	return nil
}

func (db *Database) GetActivities() ([]map[string]interface{}, error) {
	query := "SELECT * FROM activities"
	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve activities: %w", err)
	}
	defer rows.Close()

	var activities []map[string]interface{}
	for rows.Next() {
		var id, duration, keyboardEventCount, mouseEventCount sql.NullInt64
		var task, startTime, endTime, screenshotPath sql.NullString

		err := rows.Scan(&id, &task, &startTime, &endTime, &duration, &screenshotPath, &keyboardEventCount, &mouseEventCount)
		if err != nil {
			return nil, fmt.Errorf("failed to scan activity: %w", err)
		}

		activity := map[string]interface{}{
			"id":                   id.Int64,
			"task":                 task.String,
			"start_time":           startTime.String,
			"end_time":             endTime.String,
			"duration":             duration.Int64,
			"screenshot_path":      screenshotPath.String,
			"keyboard_event_count": keyboardEventCount.Int64,
			"mouse_event_count":    mouseEventCount.Int64,
		}
		activities = append(activities, activity)
	}

	return activities, nil
}

func (db *Database) ClearActivities() error {
	query := "DELETE FROM activities"
	_, err := db.conn.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to clear activities: %w", err)
	}
	return db.conn.Close()
}
