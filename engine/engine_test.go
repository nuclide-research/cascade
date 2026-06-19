package engine

import (
	"errors"
	"sort"
	"sync"
	"sync/atomic"
	"testing"
)

// fakeNode is a test-only Node (a Fake in the test-double taxonomy): a working
// in-memory tool with no network I/O. It records how many times it ran so we can
// assert the engine's per-input dedup behavior.
type fakeNode struct {
	name     string
	requires []DataKey
	produces []DataKey
	runs     int32                // atomic: Run fires from many goroutines
	emit     map[DataKey][]string // values to Produce on each run
	err      error                // returned from Run
	nilRes   bool                 // if true, Run returns (nil, err)
}

func (f *fakeNode) Name() string        { return f.name }
func (f *fakeNode) Requires() []DataKey { return f.requires }
func (f *fakeNode) Produces() []DataKey { return f.produces }
func (f *fakeNode) ranCount() int       { return int(atomic.LoadInt32(&f.runs)) }

func (f *fakeNode) Run(state *State) (*Result, error) {
	atomic.AddInt32(&f.runs, 1)
	if f.nilRes {
		return nil, f.err
	}
	r := &Result{Tool: f.name, Produces: map[DataKey][]string{}}
	for k, v := range f.emit {
		r.Produces[k] = append(r.Produces[k], v...)
	}
	return r, f.err
}

func TestEngineCascadesAcrossWaves(t *testing.T) {
	// a (domain -> ipv4) then b (ipv4 -> asn) then c (asn -> nothing).
	// Only the seed (domain) is present at start; b and c must fire on later waves.
	a := &fakeNode{name: "a", requires: []DataKey{KeyDomain}, produces: []DataKey{KeyIPv4},
		emit: map[DataKey][]string{KeyIPv4: {"9.9.9.9"}}}
	b := &fakeNode{name: "b", requires: []DataKey{KeyIPv4}, produces: []DataKey{KeyASN},
		emit: map[DataKey][]string{KeyASN: {"AS19281"}}}
	c := &fakeNode{name: "c", requires: []DataKey{KeyASN}}

	e := New()
	e.Register(a, b, c)
	e.Seed(KeyDomain, "example.com")
	results := e.Run()

	if a.ranCount() != 1 || b.ranCount() != 1 || c.ranCount() != 1 {
		t.Errorf("run counts a=%d b=%d c=%d, want 1,1,1", a.ranCount(), b.ranCount(), c.ranCount())
	}
	if len(results) != 3 {
		t.Errorf("got %d results, want 3", len(results))
	}
	if got := e.State().First(KeyASN); got != "AS19281" {
		t.Errorf("cascaded ASN = %q, want AS19281", got)
	}
}

// A node whose requirement never appears in state must never run.
func TestEngineSkipsUnsatisfiedNodes(t *testing.T) {
	never := &fakeNode{name: "never", requires: []DataKey{KeyMAC}}
	e := New()
	e.Register(never)
	e.Seed(KeyDomain, "example.com")
	e.Run()
	if never.ranCount() != 0 {
		t.Errorf("unsatisfied node ran %d times, want 0", never.ranCount())
	}
}

// A node with no requirements runs exactly once on the first wave.
func TestEngineRunsRootNodeOnce(t *testing.T) {
	root := &fakeNode{name: "root"}
	e := New()
	e.Register(root)
	e.Run()
	if root.ranCount() != 1 {
		t.Errorf("root node ran %d times, want 1", root.ranCount())
	}
}

// The runKey dedup must prevent a satisfied node from re-running across waves
// even when other nodes keep producing new state.
func TestEngineDedupsRepeatedRuns(t *testing.T) {
	// driver re-emits the SAME ipv4 value every run; consumer requires ipv4.
	// consumer must still run exactly once (same input key).
	driver := &fakeNode{name: "driver", requires: []DataKey{KeyDomain}, produces: []DataKey{KeyIPv4},
		emit: map[DataKey][]string{KeyIPv4: {"1.1.1.1"}}}
	consumer := &fakeNode{name: "consumer", requires: []DataKey{KeyIPv4}}
	e := New()
	e.Register(driver, consumer)
	e.Seed(KeyDomain, "example.com")
	e.Run()
	if consumer.ranCount() != 1 {
		t.Errorf("consumer ran %d times, want 1 (input value never changed)", consumer.ranCount())
	}
}

// A node with multiple requirements fires exactly once when all are satisfied,
// and its dedup key incorporates every requirement (not just the first).
func TestEngineMultiRequirementNode(t *testing.T) {
	// feedA seeds ipv4, feedB seeds asn; multi requires BOTH.
	feedA := &fakeNode{name: "feedA", requires: []DataKey{KeyDomain}, produces: []DataKey{KeyIPv4},
		emit: map[DataKey][]string{KeyIPv4: {"1.1.1.1"}}}
	feedB := &fakeNode{name: "feedB", requires: []DataKey{KeyDomain}, produces: []DataKey{KeyASN},
		emit: map[DataKey][]string{KeyASN: {"AS65000"}}}
	multi := &fakeNode{name: "multi", requires: []DataKey{KeyIPv4, KeyASN}}

	e := New()
	e.Register(feedA, feedB, multi)
	e.Seed(KeyDomain, "example.com")
	e.Run()

	if multi.ranCount() != 1 {
		t.Errorf("multi-requirement node ran %d times, want 1", multi.ranCount())
	}
}

// Errors returned by a tool are attached to its Result and surfaced via OnResult.
func TestEnginePropagatesErrors(t *testing.T) {
	boom := errors.New("boom")
	bad := &fakeNode{name: "bad", err: boom}
	e := New()
	e.Register(bad)

	var seen []*Result
	var mu sync.Mutex
	e.OnResult = func(r *Result) { mu.Lock(); seen = append(seen, r); mu.Unlock() }

	results := e.Run()
	if len(results) != 1 || results[0].Err == nil {
		t.Fatalf("expected one result carrying an error, got %+v", results)
	}
	if !errors.Is(results[0].Err, boom) {
		t.Errorf("error = %v, want boom", results[0].Err)
	}
	if len(seen) != 1 {
		t.Errorf("OnResult fired %d times, want 1", len(seen))
	}
}

// A tool that returns (nil, err) must be normalized into a Result, not dropped.
func TestEngineNormalizesNilResult(t *testing.T) {
	boom := errors.New("nil-path")
	bad := &fakeNode{name: "bad", nilRes: true, err: boom}
	e := New()
	e.Register(bad)
	results := e.Run()
	if len(results) != 1 {
		t.Fatalf("nil result was dropped: got %d results, want 1", len(results))
	}
	if results[0].Tool != "bad" || !errors.Is(results[0].Err, boom) {
		t.Errorf("nil result not normalized: %+v", results[0])
	}
}

// OnResult fires from many goroutines; the engine must not race when many nodes
// finish in the same wave. Run under `go test -race`.
func TestEngineConcurrentWaveIsRaceFree(t *testing.T) {
	e := New()
	names := make([]string, 0, 20)
	for i := 0; i < 20; i++ {
		n := &fakeNode{name: string(rune('A' + i)), requires: []DataKey{KeyDomain}}
		e.Register(n)
		names = append(names, n.name)
	}
	e.Seed(KeyDomain, "example.com")

	var mu sync.Mutex
	var got []string
	e.OnResult = func(r *Result) { mu.Lock(); got = append(got, r.Tool); mu.Unlock() }

	e.Run()
	sort.Strings(got)
	sort.Strings(names)
	if len(got) != len(names) {
		t.Fatalf("got %d results, want %d", len(got), len(names))
	}
	for i := range names {
		if got[i] != names[i] {
			t.Errorf("result set mismatch at %d: %q vs %q", i, got[i], names[i])
		}
	}
}
