package cmd

import (
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubAPILockWriter struct {
	writeErr   error
	closeErr   error
	closeCalls int
}

func (w *stubAPILockWriter) Write(p []byte) (int, error) {
	if w.writeErr != nil {
		return 0, w.writeErr
	}
	return len(p), nil
}

func (w *stubAPILockWriter) Close() error {
	w.closeCalls++
	return w.closeErr
}

// TestAcquireAPILockMarshalWriteCloseErrorHooks covers defensive lock error paths via test hooks.
func TestAcquireAPILockMarshalWriteCloseErrorHooks(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	prevMarshal := marshalServiceJSON
	prevOpen := openAPILockForCreate
	t.Cleanup(func() {
		marshalServiceJSON = prevMarshal
		openAPILockForCreate = prevOpen
	})

	marshalWriter := &stubAPILockWriter{}
	openAPILockForCreate = func() (apiLockWriter, error) { return marshalWriter, nil }
	marshalServiceJSON = func(any) ([]byte, error) { return nil, errors.New("marshal fail") }
	err := acquireAPILock()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "marshal api lock")
	assert.Equal(t, 1, marshalWriter.closeCalls)

	marshalServiceJSON = prevMarshal
	writeWriter := &stubAPILockWriter{writeErr: errors.New("write fail")}
	openAPILockForCreate = func() (apiLockWriter, error) { return writeWriter, nil }
	err = acquireAPILock()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "write api lock")
	assert.Equal(t, 1, writeWriter.closeCalls)

	closeWriter := &stubAPILockWriter{closeErr: errors.New("close fail")}
	openAPILockForCreate = func() (apiLockWriter, error) { return closeWriter, nil }
	err = acquireAPILock()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "close api lock")
	assert.Equal(t, 1, closeWriter.closeCalls)
}

// TestUpdateAPILockPIDMarshalErrorHook ensures marshal failures are surfaced.
func TestUpdateAPILockPIDMarshalErrorHook(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	prevMarshal := marshalServiceJSON
	t.Cleanup(func() {
		marshalServiceJSON = prevMarshal
	})
	marshalServiceJSON = func(any) ([]byte, error) { return nil, errors.New("marshal fail") }

	err := updateAPILockPID(123)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "marshal api lock")

	_, statErr := os.Stat(apiLockPath())
	assert.ErrorIs(t, statErr, os.ErrNotExist)
}

// TestProcessAliveFindProcessErrorHook ensures find-process errors are treated as not alive.
func TestProcessAliveFindProcessErrorHook(t *testing.T) {
	prevFind := findProcessByPID
	t.Cleanup(func() {
		findProcessByPID = prevFind
	})
	findProcessByPID = func(int) (*os.Process, error) {
		return nil, errors.New("find fail")
	}

	assert.False(t, processAlive(os.Getpid()))
}
