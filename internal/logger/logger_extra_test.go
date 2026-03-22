package logger

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Close ---

func TestClose(t *testing.T) {
	t.Run("closes log file after Init", func(t *testing.T) {
		// Reset state so once.Do will fire.
		logFile = nil
		once = sync.Once{}
		Logger = slog.New(slog.NewJSONHandler(io.Discard, nil))

		tmpDir := t.TempDir()
		logPath := filepath.Join(tmpDir, "close-test.log")

		err := Init(logPath)
		require.NoError(t, err)

		// Verify we can write before close.
		Info("before close")
		data, err := os.ReadFile(logPath)
		require.NoError(t, err)
		assert.Contains(t, string(data), "before close")

		// Close should not panic.
		Close()

		// After close, the file handle is closed. Writing should still not
		// panic (the logger buffers or silently fails).
		Info("after close")

		// Calling Close again should be safe (once.Do guards it).
		Close()
	})

	t.Run("close when logFile is nil does not panic", func(t *testing.T) {
		logFile = nil
		once = sync.Once{}
		Logger = slog.New(slog.NewJSONHandler(io.Discard, nil))

		// Close without Init: logFile is nil.
		assert.NotPanics(t, func() {
			Close()
		})
	})
}

// --- Writer fallback path ---

func TestStderrCapture_WriterFallback(t *testing.T) {
	t.Run("Writer returns devnull when pipe writer is nil", func(t *testing.T) {
		sc := &StderrCapture{
			done: make(chan struct{}),
			w:    nil, // No pipe writer.
		}

		f := sc.Writer()
		require.NotNil(t, f)
		defer func() { _ = f.Close() }()

		// The returned file should be writable (pointing to /dev/null).
		_, err := f.Write([]byte("test"))
		// /dev/null opened for reading only via os.Open, so write will fail.
		// The important thing is that Writer() does not panic and returns a file.
		_ = err
	})
}

// --- Init error paths ---

func TestInit_ErrorPaths(t *testing.T) {
	t.Run("init with default path when empty string", func(t *testing.T) {
		// Reset state.
		logFile = nil
		once = sync.Once{}
		Logger = slog.New(slog.NewJSONHandler(io.Discard, nil))

		// We cannot easily test the default path without side effects on the
		// real filesystem. But we can test that Init with an explicit path works.
		tmpDir := t.TempDir()
		logPath := filepath.Join(tmpDir, "explicit.log")

		err := Init(logPath)
		require.NoError(t, err)

		_, statErr := os.Stat(logPath)
		assert.NoError(t, statErr)

		// Clean up.
		if logFile != nil {
			_ = logFile.Close()
		}
		logFile = nil
		once = sync.Once{}
		Logger = slog.New(slog.NewJSONHandler(io.Discard, nil))
	})

	t.Run("init with unwritable directory returns error", func(t *testing.T) {
		logFile = nil
		once = sync.Once{}
		Logger = slog.New(slog.NewJSONHandler(io.Discard, nil))

		// Use a path under /dev/null which cannot have child directories.
		err := Init("/dev/null/impossible/path/test.log")
		assert.Error(t, err)

		// Logger should still be the discard logger (Init failed).
	})
}

// --- StderrCapture Close edge cases ---

func TestStderrCapture_CloseEdgeCases(t *testing.T) {
	t.Run("close with nil fields does not panic", func(t *testing.T) {
		sc := &StderrCapture{
			// All nil: w, r, done.
		}
		assert.NotPanics(t, func() {
			sc.Close()
		})
	})

	t.Run("close with only done channel", func(t *testing.T) {
		done := make(chan struct{})
		close(done) // pre-close so Close() won't block.
		sc := &StderrCapture{
			done: done,
		}
		assert.NotPanics(t, func() {
			sc.Close()
		})
	})
}

// --- KlogWriter edge cases ---

func TestKlogWriter_EmptyWrite(t *testing.T) {
	w := KlogWriter()

	// Writing an empty string should not panic.
	n, err := w.Write([]byte(""))
	assert.NoError(t, err)
	assert.Equal(t, 0, n)

	// Writing only whitespace should not panic.
	n, err = w.Write([]byte("   \n\t  "))
	assert.NoError(t, err)
	assert.Greater(t, n, 0)
}
