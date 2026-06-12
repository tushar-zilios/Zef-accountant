package logger

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
)

var (
	// HandlersLogFile holds the file handle for handlers.log
	HandlersLogFile *os.File
	// DBLogFile holds the file handle for db.log
	DBLogFile *os.File

	// HandlersLogger is a logger instance that writes to handlers.log
	HandlersLogger *log.Logger
	// DBLogger is a logger instance that writes to db.log
	DBLogger *log.Logger

	logsDir = "logs"
)

// Init initializes the logging module by creating handlers.log and db.log
func Init() error {
	// Ensure the logs directory exists
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		return fmt.Errorf("failed to create logs directory: %w", err)
	}

	handlersPath := filepath.Join(logsDir, "handlers.log")
	hFile, err := os.OpenFile(handlersPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to create handlers.log: %w", err)
	}
	HandlersLogFile = hFile
	HandlersLogger = log.New(HandlersLogFile, "[HANDLERS] ", log.LstdFlags)

	dbPath := filepath.Join(logsDir, "db.log")
	dFile, err := os.OpenFile(dbPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		// Clean up the handlers log file if db log file fails to create
		HandlersLogFile.Close()
		os.Remove(handlersPath)
		return fmt.Errorf("failed to create db.log: %w", err)
	}
	DBLogFile = dFile
	DBLogger = log.New(DBLogFile, "[DB] ", log.LstdFlags)

	return nil
}

// Cleanup closes the log files and deletes them from the logs directory
func Cleanup() {
	if HandlersLogFile != nil {
		HandlersLogFile.Close()
		HandlersLogFile = nil
	}
	if DBLogFile != nil {
		DBLogFile.Close()
		DBLogFile = nil
	}

	handlersPath := filepath.Join(logsDir, "handlers.log")
	if err := os.Remove(handlersPath); err != nil && !os.IsNotExist(err) {
		log.Printf("Warning: failed to delete handlers.log: %v", err)
	}

	dbPath := filepath.Join(logsDir, "db.log")
	if err := os.Remove(dbPath); err != nil && !os.IsNotExist(err) {
		log.Printf("Warning: failed to delete db.log: %v", err)
	}
}

// LogHandler logs a formatted message to handlers.log and console
func LogHandler(format string, v ...interface{}) {
	if HandlersLogger != nil {
		HandlersLogger.Printf(format, v...)
	}
	log.Printf("[HANDLERS] "+format, v...)
}

// LogDB logs a formatted message to db.log and console
func LogDB(format string, v ...interface{}) {
	if DBLogger != nil {
		DBLogger.Printf(format, v...)
	}
	log.Printf("[DB] "+format, v...)
}
