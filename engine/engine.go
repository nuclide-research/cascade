package engine

import (
	"fmt"
	"sync"
)

// runKey deduplicates tool runs per unique input set.
type runKey struct {
	tool  string
	input string // first required key value at run time
}

// Engine resolves and executes the DAG of tools.
type Engine struct {
	state    *State
	nodes    []Node
	ran      map[runKey]bool
	results  []*Result
	mu       sync.Mutex
	OnResult func(*Result)
}

func New() *Engine {
	return &Engine{
		state: NewState(),
		ran:   make(map[runKey]bool),
	}
}

func (e *Engine) Register(nodes ...Node) {
	e.nodes = append(e.nodes, nodes...)
}

func (e *Engine) Seed(key DataKey, values ...string) {
	e.state.Set(key, values...)
}

// Run executes wave after wave until nothing new can fire.
func (e *Engine) Run() []*Result {
	for {
		ready := e.findReady()
		if len(ready) == 0 {
			break
		}

		var wg sync.WaitGroup
		wave := make([]*Result, len(ready))

		for i, node := range ready {
			wg.Add(1)
			go func(i int, n Node) {
				defer wg.Done()
				r, err := n.Run(e.state)
				if r == nil {
					r = &Result{Tool: n.Name()}
				}
				if err != nil {
					r.Err = err
				}
				wave[i] = r
			}(i, node)
		}
		wg.Wait()

		for _, r := range wave {
			if r == nil {
				continue
			}
			e.mu.Lock()
			e.results = append(e.results, r)
			e.mu.Unlock()

			if e.OnResult != nil {
				e.OnResult(r)
			}

			for key, vals := range r.Produces {
				e.state.Set(key, vals...)
			}
		}
	}
	return e.results
}

func (e *Engine) State() *State {
	return e.state
}

func (e *Engine) findReady() []Node {
	e.mu.Lock()
	defer e.mu.Unlock()

	var ready []Node
	for _, node := range e.nodes {
		rk := runKey{tool: node.Name(), input: e.inputKey(node)}
		if e.ran[rk] {
			continue
		}
		if e.canRun(node) {
			e.ran[rk] = true
			ready = append(ready, node)
		}
	}
	return ready
}

func (e *Engine) canRun(node Node) bool {
	for _, req := range node.Requires() {
		if !e.state.Has(req) {
			return false
		}
	}
	return true
}

func (e *Engine) inputKey(node Node) string {
	reqs := node.Requires()
	if len(reqs) == 0 {
		return ""
	}
	return fmt.Sprintf("%s=%s", reqs[0], e.state.First(reqs[0]))
}
