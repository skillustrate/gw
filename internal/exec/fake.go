package exec

import (
	"context"
	"fmt"
	"strings"
	"sync"
)

// Invocation records an execution attempt for test assertion.
type Invocation struct {
	Dir  string
	Name string
	Args []string
}

// Key formats invocation Name and Args into a lookup key.
func (inv Invocation) Key() string {
	parts := append([]string{inv.Name}, inv.Args...)
	return strings.Join(parts, " ")
}

// FakeRunner provides deterministic canned responses for unit tests.
type FakeRunner struct {
	mu sync.Mutex
	// Responses holds a single static response returned for every call matching a key.
	Responses map[string]Result
	// Sequences holds an ordered list of responses returned for successive calls matching
	// a key (e.g. simulating a "before mutation" then "after mutation" observation). Once
	// exhausted, the last response in the sequence is returned for any further calls.
	// A key present in Sequences takes precedence over the same key in Responses.
	Sequences map[string][]Result
	seqIndex  map[string]int
	Calls     []Invocation
}

// NewFakeRunner creates a new initialized FakeRunner.
func NewFakeRunner(responses map[string]Result) *FakeRunner {
	if responses == nil {
		responses = make(map[string]Result)
	}
	return &FakeRunner{
		Responses: responses,
		Sequences: make(map[string][]Result),
		seqIndex:  make(map[string]int),
		Calls:     make([]Invocation, 0),
	}
}

// Run looks up a canned response and records the invocation call.
func (f *FakeRunner) Run(ctx context.Context, dir, name string, args ...string) (Result, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	inv := Invocation{
		Dir:  dir,
		Name: name,
		Args: append([]string(nil), args...),
	}
	f.Calls = append(f.Calls, inv)

	key := inv.Key()
	if seq, ok := f.Sequences[key]; ok && len(seq) > 0 {
		idx := f.seqIndex[key]
		if idx >= len(seq) {
			idx = len(seq) - 1
		}
		f.seqIndex[key] = idx + 1
		return seq[idx], nil
	}

	if res, ok := f.Responses[key]; ok {
		return res, nil
	}

	return Result{ExitCode: -1}, fmt.Errorf("FakeRunner: unmapped invocation: %q in dir %q", key, dir)
}