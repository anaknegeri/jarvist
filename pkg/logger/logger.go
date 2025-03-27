package logger

import (
	"fmt"
	"os"
	"time"
)

const (
	TRACE = iota
	DEBUG
	INFO
	WARNING
	ERROR
	FATAL
)

type Logger struct {
	logFile *os.File
	level   int
}

func NewLogger() *Logger {
	logFile, err := os.OpenFile("app.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		fmt.Println("Failed to open log file:", err)
	}

	return &Logger{
		logFile: logFile,
	}
}

func (l *Logger) SetLogLevel(level int) {
	l.level = level
}

func (l *Logger) Print(message string) {
	l.log("PRINT", message)
}

func (l *Logger) Trace(message string) {
	l.log("TRACE", message)
}

func (l *Logger) Debug(message string) {
	l.log("DEBUG", message)
}

func (l *Logger) Info(message string) {
	l.log("INFO", message)
}

func (l *Logger) Warning(message string) {
	l.log("WARNING", message)
}

func (l *Logger) Error(message string) {
	l.log("ERROR", message)
}

func (l *Logger) Fatal(message string) {
	l.log("FATAL", message)
}

func (l *Logger) log(level, message string) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	logMessage := fmt.Sprintf("[%s] [%s] %s\n", timestamp, level, message)

	fmt.Print(logMessage)

	if l.logFile != nil {
		_, err := l.logFile.WriteString(logMessage)
		if err != nil {
			fmt.Println("Failed to write to log file:", err)
		}
	}
}
