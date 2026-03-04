// Package log — JSON structured logger with size rotation and Tail().
package log

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Level string

const (
	LevelInfo  Level = "INFO"
	LevelWarn  Level = "WARN"
	LevelError Level = "ERROR"
	LevelDebug Level = "DEBUG"

	maxLogBytes int64 = 10 * 1024 * 1024 // 10 MB — rotaciona ao atingir
)

type Entry struct {
	Timestamp string                 `json:"ts"`
	Level     Level                  `json:"level"`
	Message   string                 `json:"msg"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
}

type Logger struct {
	mu      sync.Mutex
	out     io.Writer
	logPath string
	logFile *os.File
}

// New cria um Logger que escreve no arquivo logPath e em stderr.
func New(logPath string) *Logger {
	if logPath == "" {
		return &Logger{out: os.Stderr}
	}
	l := &Logger{logPath: logPath}
	l.openOrCreate()
	return l
}

func (l *Logger) openOrCreate() {
	if err := os.MkdirAll(filepath.Dir(l.logPath), 0750); err != nil {
		fmt.Fprintf(os.Stderr, "log: could not create directory: %v\n", err)
		l.out = os.Stderr
		return
	}
	f, err := os.OpenFile(l.logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0640)
	if err != nil {
		fmt.Fprintf(os.Stderr, "log: could not open file: %v\n", err)
		l.out = os.Stderr
		return
	}
	l.logFile = f
	// Write to file AND stderr (visible in journalctl).
	l.out = io.MultiWriter(f, os.Stderr)
}

func (l *Logger) Info(msg string, kvs ...interface{})  { l.write(LevelInfo, msg, kvs) }
func (l *Logger) Warn(msg string, kvs ...interface{})  { l.write(LevelWarn, msg, kvs) }
func (l *Logger) Error(msg string, kvs ...interface{}) { l.write(LevelError, msg, kvs) }
func (l *Logger) Debug(msg string, kvs ...interface{}) { l.write(LevelDebug, msg, kvs) }

func (l *Logger) write(level Level, msg string, kvs []interface{}) {
	entry := Entry{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Level:     level,
		Message:   msg,
	}
	if len(kvs) >= 2 {
		entry.Fields = make(map[string]interface{})
		for i := 0; i+1 < len(kvs); i += 2 {
			key := fmt.Sprintf("%v", kvs[i])
			val := kvs[i+1]
			if err, ok := val.(error); ok {
				val = err.Error()
			}
			entry.Fields[key] = val
		}
	}
	data, _ := json.Marshal(entry)
	data = append(data, '\n')

	l.mu.Lock()
	defer l.mu.Unlock()

	l.out.Write(data)
	l.rotateIfNeeded()
}

// rotateIfNeeded rotates if file exceeds maxLogBytes.
// Must be called with l.mu locked.
func (l *Logger) rotateIfNeeded() {
	if l.logFile == nil || l.logPath == "" {
		return
	}
	info, err := l.logFile.Stat()
	if err != nil || info.Size() < maxLogBytes {
		return
	}
	// Fecha arquivo atual.
	l.logFile.Close()

	// Renomeia para .1 (sobrescreve backup anterior).
	_ = os.Rename(l.logPath, l.logPath+".1")

	// Abre novo arquivo.
	f, err := os.OpenFile(l.logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0640)
	if err != nil {
		l.out = os.Stderr
		l.logFile = nil
		return
	}
	l.logFile = f
	l.out = io.MultiWriter(f, os.Stderr)
}

// Tail returns the last n lines of the log file.
// Thread-safe.
func (l *Logger) Tail(n int) []string {
	if l.logPath == "" {
		return nil
	}
	l.mu.Lock()
	path := l.logPath
	l.mu.Unlock()

	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if len(lines) <= n {
		return lines
	}
	return lines[len(lines)-n:]
}
