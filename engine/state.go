package engine

import "sync"

// State is the shared key-value store all tools read from and write to.
// Each key holds a deduplicated list of values (a tool may discover multiple IPs, etc.).
type State struct {
	mu   sync.RWMutex
	data map[DataKey][]string
}

func NewState() *State {
	return &State{data: make(map[DataKey][]string)}
}

func (s *State) Set(key DataKey, values ...string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, v := range values {
		if v == "" {
			continue
		}
		dup := false
		for _, existing := range s.data[key] {
			if existing == v {
				dup = true
				break
			}
		}
		if !dup {
			s.data[key] = append(s.data[key], v)
		}
	}
}

func (s *State) Get(key DataKey) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	cp := make([]string, len(s.data[key]))
	copy(cp, s.data[key])
	return cp
}

func (s *State) First(key DataKey) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(s.data[key]) == 0 {
		return ""
	}
	return s.data[key][0]
}

func (s *State) Has(key DataKey) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.data[key]) > 0
}

// Snapshot returns a copy of all state data (for JSON output).
func (s *State) Snapshot() map[DataKey][]string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[DataKey][]string, len(s.data))
	for k, v := range s.data {
		cp := make([]string, len(v))
		copy(cp, v)
		out[k] = cp
	}
	return out
}
