package audit

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Logger struct {
	mu      sync.Mutex
	writer  io.WriteCloser
	file    string
	maxSize int64
	size    int64
}

func NewLogger(format, file string, maxSize int64) (*Logger, error) {
	if maxSize <= 0 {
		maxSize = 2 * 1024 * 1024 // 2MB default
	}

	if file == "" {
		return &Logger{writer: os.Stdout}, nil
	}

	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		return nil, err
	}

	f, err := os.OpenFile(file, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, err
	}

	// Get current file size for rotation tracking
	info, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, err
	}

	return &Logger{
		writer:  f,
		file:    file,
		maxSize: maxSize,
		size:    info.Size(),
	}, nil
}

func (l *Logger) Log(s *Session) {
	data, err := json.Marshal(s)
	if err != nil {
		slog.Error("failed to marshal session", "err", err)
		return
	}
	data = append(data, '\n')

	l.mu.Lock()
	defer l.mu.Unlock()

	l.writer.Write(data)
	l.size += int64(len(data))

	if l.file != "" && l.size >= l.maxSize {
		l.rotate()
	}
}

func (l *Logger) rotate() {
	l.writer.Close()

	// Rename current log to audit.log.<unix_epoch>
	rotated := fmt.Sprintf("%s.%d", l.file, time.Now().Unix())
	if err := os.Rename(l.file, rotated); err != nil {
		slog.Error("failed to rotate log file", "err", err)
	}

	f, err := os.OpenFile(l.file, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		l.writer = os.Stdout
		l.file = ""
		return
	}
	l.writer = f
	l.size = 0
}

func (l *Logger) Close() error {
	if l.writer != os.Stdout {
		return l.writer.Close()
	}
	return nil
}
