package workload

import (
	"fmt"
	"sync"
	"testing"

	"github.com/arahe-dev/hertasdk/lab/impl"
)

// Monitor checks, from inside handlers, the two safety properties a
// runtime promises under any load: units in flight per resource never
// exceed capacity, and a key never has two holders.
type Monitor struct {
	caps map[string]int64

	mu         sync.Mutex
	units      map[string]int64
	peak       map[string]int64
	keys       map[string]int
	violations int
	first      []string
}

// NewMonitor watches the resources declared by specs.
func NewMonitor(specs ...impl.Spec) *Monitor {
	m := &Monitor{caps: map[string]int64{}, units: map[string]int64{}, peak: map[string]int64{}, keys: map[string]int{}}
	for _, s := range specs {
		m.caps[s.Name] = s.Capacity
	}
	return m
}

// Enter records a handler holding reqs under key ("" for none) and returns
// the matching exit.
func (m *Monitor) Enter(reqs []impl.Req, key string) (exit func()) {
	m.mu.Lock()
	for _, r := range reqs {
		m.units[r.Name] += r.Units
		m.peak[r.Name] = max(m.peak[r.Name], m.units[r.Name])
		if m.units[r.Name] > m.caps[r.Name] {
			m.violate("resource %s: %d units in flight, capacity %d", r.Name, m.units[r.Name], m.caps[r.Name])
		}
	}
	if key != "" {
		m.keys[key]++
		if m.keys[key] > 1 {
			m.violate("key %q: %d concurrent holders", key, m.keys[key])
		}
	}
	m.mu.Unlock()
	return func() {
		m.mu.Lock()
		for _, r := range reqs {
			m.units[r.Name] -= r.Units
		}
		if key != "" {
			if m.keys[key]--; m.keys[key] == 0 {
				delete(m.keys, key)
			}
		}
		m.mu.Unlock()
	}
}

func (m *Monitor) violate(format string, args ...any) {
	m.violations++
	if len(m.first) < 5 {
		m.first = append(m.first, fmt.Sprintf(format, args...))
	}
}

// Peak is the most units of name ever in flight at once.
func (m *Monitor) Peak(name string) int64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.peak[name]
}

// Check fails t if any invariant was ever violated.
func (m *Monitor) Check(t testing.TB) {
	t.Helper()
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.violations > 0 {
		t.Errorf("%d invariant violations, first: %v", m.violations, m.first)
	}
}
