package lock

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// ErrLocked is returned when a valid, active lock is held by another live process.
var ErrLocked = errors.New("REPO_LOCKED")

// LockInfo describes the serializable payload stored inside .git/gw.lock.
type LockInfo struct {
	PID        int       `json:"pid"`
	AcquiredAt time.Time `json:"acquired_at"`
}

// Lock manages an acquired mutex lock file.
type Lock struct {
	path string
	pid  int
}

var (
	isPIDAliveHook func(pid int) bool
	lockMu         sync.Mutex
)

// Acquire atomically acquires the repository lock at <gitDir>/gw.lock.
// If an existing lock is held by a dead PID or is older than staleAfter, it is reclaimed.
func Acquire(gitDir string, staleAfter time.Duration) (*Lock, bool, error) {
	lockMu.Lock()
	defer lockMu.Unlock()

	lockPath := filepath.Join(gitDir, "gw.lock")
	myPID := os.Getpid()
	now := time.Now().UTC()

	// Try atomic creation
	l, err := tryCreateLock(lockPath, myPID, now)
	if err == nil {
		return l, false, nil
	}

	if !os.IsExist(err) {
		return nil, false, fmt.Errorf("failed to create lock file: %w", err)
	}

	// Lock file already exists. Inspect existing lock info for stale reclamation.
	existingInfo, readErr := readLockInfo(lockPath)
	isStale := false

	if readErr != nil {
		// Corrupted or empty lock file -> reclaim
		isStale = true
	} else {
		// Check age
		if staleAfter > 0 && time.Since(existingInfo.AcquiredAt) > staleAfter {
			isStale = true
		} else if !isPIDAlive(existingInfo.PID) {
			// PID is dead -> reclaim
			isStale = true
		}
	}

	if !isStale {
		return nil, false, ErrLocked
	}

	// Reclaim stale lock: remove and recreate
	if err := os.Remove(lockPath); err != nil && !os.IsNotExist(err) {
		return nil, false, fmt.Errorf("failed to remove stale lock: %w", err)
	}

	l, err = tryCreateLock(lockPath, myPID, now)
	if err != nil {
		if os.IsExist(err) {
			return nil, false, ErrLocked
		}
		return nil, false, fmt.Errorf("failed to acquire reclaimed lock: %w", err)
	}

	return l, true, nil
}

// Release releases the lock if it is still owned by the current process.
func (l *Lock) Release() error {
	lockMu.Lock()
	defer lockMu.Unlock()

	if l == nil || l.path == "" {
		return nil
	}

	info, err := readLockInfo(l.path)
	if err == nil && info.PID == l.pid {
		return os.Remove(l.path)
	}

	return nil
}

func tryCreateLock(path string, pid int, acquiredAt time.Time) (*Lock, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	info := LockInfo{
		PID:        pid,
		AcquiredAt: acquiredAt,
	}
	data, err := json.Marshal(info)
	if err != nil {
		_ = os.Remove(path)
		return nil, err
	}

	if _, err := f.Write(data); err != nil {
		_ = os.Remove(path)
		return nil, err
	}

	return &Lock{
		path: path,
		pid:  pid,
	}, nil
}

func readLockInfo(path string) (LockInfo, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return LockInfo{}, err
	}
	var info LockInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return LockInfo{}, err
	}
	return info, nil
}

func isPIDAlive(pid int) bool {
	if isPIDAliveHook != nil {
		return isPIDAliveHook(pid)
	}

	if pid <= 0 {
		return false
	}

	return checkPIDAlive(pid)
}