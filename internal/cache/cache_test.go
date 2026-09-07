package cache

import (
	"testing"
	"time"
)

func TestCache_SetAndGet(t *testing.T) {
	c := New[string](2 * time.Second)
	now := time.Now()

	c.Set("repo1", "state_payload", now)

	val, found := c.Get("repo1")
	if !found || val != "state_payload" {
		t.Fatalf("expected to find 'state_payload', got %q (found=%v)", val, found)
	}

	c.Invalidate("repo1")
	_, found = c.Get("repo1")
	if found {
		t.Fatalf("expected entry to be invalidated")
	}
}

func TestCache_Expiration(t *testing.T) {
	c := New[string](50 * time.Millisecond)
	pastTime := time.Now().Add(-100 * time.Millisecond)

	c.Set("stale_key", "old_val", pastTime)

	_, found := c.Get("stale_key")
	if found {
		t.Errorf("expected expired entry to not be returned")
	}
}

func TestCache_InvalidatePrefix(t *testing.T) {
	c := New[string](2 * time.Second)
	now := time.Now()

	c.Set("/repo/a|main", "data1", now)
	c.Set("/repo/a|feature", "data2", now)
	c.Set("/repo/b|main", "data3", now)

	c.InvalidatePrefix("/repo/a|")

	if _, found := c.Get("/repo/a|main"); found {
		t.Errorf("expected /repo/a|main to be invalidated")
	}
	if _, found := c.Get("/repo/a|feature"); found {
		t.Errorf("expected /repo/a|feature to be invalidated")
	}
	if _, found := c.Get("/repo/b|main"); !found {
		t.Errorf("expected /repo/b|main to remain cached")
	}
}

func TestCache_Clear(t *testing.T) {
	c := New[string](2 * time.Second)
	now := time.Now()

	c.Set("k1", "v1", now)
	c.Set("k2", "v2", now)

	c.Clear()

	if _, found := c.Get("k1"); found {
		t.Errorf("expected k1 to be cleared")
	}
	if _, found := c.Get("k2"); found {
		t.Errorf("expected k2 to be cleared")
	}
}