package engine

import (
	"reflect"
	"sync"
	"testing"
)

func TestStateSetDeduplicates(t *testing.T) {
	tests := []struct {
		name string
		sets [][]string // each inner slice is one Set call
		want []string
	}{
		{
			name: "single value",
			sets: [][]string{{"1.1.1.1"}},
			want: []string{"1.1.1.1"},
		},
		{
			name: "duplicate in one call collapses, order preserved",
			sets: [][]string{{"a", "b", "a", "c", "b"}},
			want: []string{"a", "b", "c"},
		},
		{
			name: "duplicate across calls collapses",
			sets: [][]string{{"a", "b"}, {"b", "c"}},
			want: []string{"a", "b", "c"},
		},
		{
			name: "empty values are dropped",
			sets: [][]string{{"", "a", ""}},
			want: []string{"a"},
		},
		{
			name: "all empty yields nothing",
			sets: [][]string{{"", ""}},
			want: nil,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := NewState()
			for _, call := range tc.sets {
				s.Set(KeyIPv4, call...)
			}
			got := s.Get(KeyIPv4)
			if len(got) == 0 && len(tc.want) == 0 {
				return
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("Get = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestStateFirst(t *testing.T) {
	s := NewState()
	if got := s.First(KeyDomain); got != "" {
		t.Errorf("First on empty = %q, want empty string", got)
	}
	s.Set(KeyDomain, "example.com", "example.org")
	if got := s.First(KeyDomain); got != "example.com" {
		t.Errorf("First = %q, want example.com", got)
	}
}

func TestStateHas(t *testing.T) {
	s := NewState()
	if s.Has(KeyASN) {
		t.Error("Has on empty key = true, want false")
	}
	s.Set(KeyASN, "")
	if s.Has(KeyASN) {
		t.Error("Has after setting only empty values = true, want false")
	}
	s.Set(KeyASN, "AS13335")
	if !s.Has(KeyASN) {
		t.Error("Has after real set = false, want true")
	}
}

// Get must return a copy: mutating the returned slice must not corrupt state.
func TestStateGetReturnsCopy(t *testing.T) {
	s := NewState()
	s.Set(KeyDomain, "a", "b")
	got := s.Get(KeyDomain)
	got[0] = "MUTATED"
	if again := s.Get(KeyDomain); again[0] != "a" {
		t.Errorf("Get returned an aliased slice: state corrupted to %v", again)
	}
}

// Snapshot must be a deep copy: mutating it must not corrupt state.
func TestStateSnapshotIsDeepCopy(t *testing.T) {
	s := NewState()
	s.Set(KeyIPv4, "1.1.1.1")
	snap := s.Snapshot()
	snap[KeyIPv4][0] = "MUTATED"
	snap[KeyDomain] = []string{"injected"}
	if got := s.First(KeyIPv4); got != "1.1.1.1" {
		t.Errorf("Snapshot aliased the value slice: state corrupted to %q", got)
	}
	if s.Has(KeyDomain) {
		t.Error("Snapshot aliased the map: injected key leaked into state")
	}
}

// The store is accessed concurrently by engine worker goroutines; -race must
// stay clean under parallel writers and readers.
func TestStateConcurrentAccess(t *testing.T) {
	s := NewState()
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(2)
		go func() { defer wg.Done(); s.Set(KeyIPv4, "1.2.3.4") }()
		go func() { defer wg.Done(); _ = s.Get(KeyIPv4); _ = s.Has(KeyIPv4); _ = s.First(KeyIPv4) }()
	}
	wg.Wait()
	if got := s.Get(KeyIPv4); !reflect.DeepEqual(got, []string{"1.2.3.4"}) {
		t.Errorf("after concurrent dedup writes, Get = %v, want [1.2.3.4]", got)
	}
}
