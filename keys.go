package hertasdk

import (
	"context"
	"sync"
)

// KeyLocker gives per-key mutual exclusion with FIFO hand-off. One mutex
// guards the map and every entry, so lock and unlock are one critical
// section each. An entry lives exactly while its key is held; waiters
// exist only behind a holder. An idle key has no entry, and entries are
// pooled. Once closed by Shutdown, no call queues for a key.
type KeyLocker struct {
	mu     sync.Mutex
	m      map[string]*keyEntry
	closed bool
}

type keyEntry struct {
	key string
	q   waitq // waiters behind the holder, in arrival order
}

var keyEntries = sync.Pool{New: func() any { return new(keyEntry) }}

// lock acquires key. Under Reject a held key fails at once (Δ1); under
// Wait the call queues until the key is handed to it, ctx ends, or the
// locker closes. On failure nothing is held.
func (kl *KeyLocker) lock(ctx context.Context, key string, mode AdmissionMode) (*keyEntry, bool) {
	kl.mu.Lock()
	e := kl.m[key]
	if e == nil {
		e = keyEntries.Get().(*keyEntry)
		e.key = key
		if kl.m == nil {
			kl.m = make(map[string]*keyEntry)
		}
		kl.m[key] = e
		kl.mu.Unlock()
		return e, true
	}
	if mode == Reject || kl.closed {
		kl.mu.Unlock()
		return nil, false
	}
	done := ctx.Done()
	if isClosed(done) {
		kl.mu.Unlock()
		return nil, false
	}
	w := newWaiter(1)
	e.q.push(w)
	kl.mu.Unlock()

	woken := w.park(done)
	ok := woken && w.state == granted && !isClosed(done)
	if !ok {
		kl.leave(e, w, woken)
		e = nil
	}
	return e, ok
}

// leave settles a waiter that is not keeping the key: one handed the key
// as its caller gave up passes it on, one still queued is removed.
func (kl *KeyLocker) leave(e *keyEntry, w *waiter, woken bool) {
	kl.mu.Lock()
	defer kl.mu.Unlock()
	if w.state == waiting {
		e.q.remove(w)
		return
	}
	if !woken {
		<-w.ready // the key arrived as the caller gave up
	}
	if w.state == granted {
		kl.handOff(e)
	}
}

// unlock releases a key from lock; nil is a no-op.
func (kl *KeyLocker) unlock(e *keyEntry) {
	if e == nil {
		return
	}
	kl.mu.Lock()
	kl.handOff(e)
	kl.mu.Unlock()
}

// handOff passes the held key to the next waiter, or retires the entry.
// Caller holds mu.
func (kl *KeyLocker) handOff(e *keyEntry) {
	if w := e.q.head; w != nil {
		e.q.remove(w)
		w.wake(granted)
		return
	}
	delete(kl.m, e.key)
	e.key = ""
	keyEntries.Put(e)
}

// close withdraws every key waiter and refuses new ones; holders keep
// their keys until they finish.
func (kl *KeyLocker) close() {
	kl.mu.Lock()
	kl.closed = true
	for _, e := range kl.m {
		e.q.closeAll()
	}
	kl.mu.Unlock()
}
