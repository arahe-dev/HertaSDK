package execution

import (
	"context"
	"sync"

	"golang.org/x/sync/semaphore"
)

// KeyLocker provides per-key mutual exclusion with ref-counted entries so
// unused keys are deleted. The ref-counting/lifecycle is hand-written
// because those semantics are central to Herta; the per-key primitive is a
// capacity-1 weighted semaphore so waiting is ctx-aware for free (no
// hand-rolled condition variables, no drain goroutines).
type KeyLocker struct {
	mu   sync.Mutex
	keys map[string]*keyLock
}

func newKeyLocker() *KeyLocker {
	return &KeyLocker{keys: make(map[string]*keyLock)}
}

type keyLock struct {
	name string
	// sem is capacity-1: the operation currently executing under this key
	// holds it; same-key waiters block on Acquire.
	sem *semaphore.Weighted
	// refs counts holders+waiters; at zero the key entry is deleted.
	refs int
}

// acquire locks the key, respecting ctx (caller cancellation AND runtime
// shutdown — both arrive as cancellation of the operation's admission
// context). On success it returns the unlock func; on failure nothing is
// held and the wait cost was zero.
func (kl *KeyLocker) acquire(ctx context.Context, key string) (func(), error) {
	kl.mu.Lock()
	l, ok := kl.keys[key]
	if !ok {
		l = &keyLock{name: key, sem: semaphore.NewWeighted(1)}
		kl.keys[key] = l
	}
	l.refs++
	kl.mu.Unlock()

	if err := l.sem.Acquire(ctx, 1); err != nil {
		// Did not acquire (Weighted releases any granted units on cancel —
		// there are none for capacity 1, but keep the invariant explicit).
		kl.release(l)
		return nil, err
	}
	return func() {
		l.sem.Release(1)
		kl.release(l)
	}, nil
}

// release drops one ref and deletes the key entry when the last user leaves.
func (kl *KeyLocker) release(l *keyLock) {
	kl.mu.Lock()
	defer kl.mu.Unlock()
	l.refs--
	if l.refs == 0 {
		delete(kl.keys, l.name)
	}
}
