package lock

import (
	"errors"
	"os"
	"testing"
	"time"
)

func TestLock_AcquireAndRelease(t *testing.T) {
	tempDir := t.TempDir()

	l, reclaimed, err := Acquire(tempDir, 10*time.Minute)
	if err != nil {
		t.Fatalf("unexpected error acquiring lock: %v", err)
	}
	if reclaimed {
		t.Errorf("expected reclaimed = false on fresh lock")
	}

	// Second acquire must fail with ErrLocked
	_, _, err2 := Acquire(tempDir, 10*time.Minute)
	if !errors.Is(err2, ErrLocked) {
		t.Errorf("expected ErrLocked, got %v", err2)
	}

	// Release lock
	if err := l.Release(); err != nil {
		t.Fatalf("unexpected error releasing lock: %v", err)
	}

	// Now should acquire cleanly again
	l2, reclaimed2, err3 := Acquire(tempDir, 10*time.Minute)
	if err3 != nil {
		t.Fatalf("unexpected error acquiring lock after release: %v", err3)
	}
	if reclaimed2 {
		t.Errorf("expected reclaimed = false")
	}
	_ = l2.Release()
}

func TestLock_ReclaimDeadPID(t *testing.T) {
	tempDir := t.TempDir()

	// Simulate existing lock with dead PID
	isPIDAliveHook = func(pid int) bool {
		if pid == 999999 {
			return false // dead PID
		}
		return true
	}
	defer func() { isPIDAliveHook = nil }()

	// Write mock lock with dead PID
	lockPath := tempDir + "/gw.lock"
	_, _ = tryCreateLock(lockPath, 999999, time.Now().UTC())

	// Acquire should detect dead PID and reclaim
	l, reclaimed, err := Acquire(tempDir, 10*time.Minute)
	if err != nil {
		t.Fatalf("unexpected error during dead PID reclaim: %v", err)
	}
	if !reclaimed {
		t.Errorf("expected reclaimed = true when PID is dead")
	}
	_ = l.Release()
}

func TestLock_ReclaimStaleAge(t *testing.T) {
	tempDir := t.TempDir()

	isPIDAliveHook = func(pid int) bool { return true } // alive but old
	defer func() { isPIDAliveHook = nil }()

	// Write mock lock acquired 15 minutes ago
	lockPath := tempDir + "/gw.lock"
	_, _ = tryCreateLock(lockPath, 12345, time.Now().UTC().Add(-15*time.Minute))

	// Acquire with 10m threshold should reclaim
	l, reclaimed, err := Acquire(tempDir, 10*time.Minute)
	if err != nil {
		t.Fatalf("unexpected error reclaiming aged lock: %v", err)
	}
	if !reclaimed {
		t.Errorf("expected reclaimed = true when lock age > staleAfter")
	}
	_ = l.Release()
}

func TestLock_CheckPIDAlive(t *testing.T) {
	myPID := os.Getpid()
	if !isPIDAlive(myPID) {
		t.Errorf("expected current PID %d to be alive", myPID)
	}

	if isPIDAlive(-1) {
		t.Errorf("expected negative PID to not be alive")
	}

	if isPIDAlive(0) {
		t.Errorf("expected PID 0 to not be alive")
	}
}