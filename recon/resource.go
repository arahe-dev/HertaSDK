package execution

import (
	"context"
	"strconv"
	"sync"
)

// Resource is a named finite capacity: a weighted semaphore with a strict
// FIFO queue, the same semantics as golang.org/x/sync/semaphore made
// explicit (CONTRACT §4.1).
//
//   - A request is granted at once only if it fits and nobody is queued.
//   - Release grants from the head for as long as each waiter fits; a head
//     that does not fit blocks everyone behind it, so heavy requests never
//     starve.
//   - A waiter that leaves is removed; if it was the head, the queue is
//     re-examined.
//
// Units in use (cur) and queued waiters (q.len) are exact at every unlock.
// Once closed by Shutdown, the queue is empty and admits no new waiter.
type Resource struct {
	mu     sync.Mutex
	cur    int64
	q      waitq
	closed bool
	size   int64
	name   string
}

func newResource(name string, capacity int64) *Resource {
	return &Resource{name: name, size: capacity}
}

// String renders "name(capacity)".
func (r *Resource) String() string {
	return r.name + "(" + strconv.FormatInt(r.size, 10) + ")"
}

// tryAcquire takes n units only if they fit and nobody is queued.
func (r *Resource) tryAcquire(n int64) bool {
	r.mu.Lock()
	ok := r.q.len == 0 && r.size-r.cur >= n
	if ok {
		r.cur += n
	}
	r.mu.Unlock()
	return ok
}

// acquire takes n units (0 < n ≤ size), queueing behind earlier waiters.
// It fails, holding nothing, if ctx ends while it waits or the resource is
// closed. The uncontended path never touches ctx, so it never allocates.
func (r *Resource) acquire(ctx context.Context, n int64) bool {
	r.mu.Lock()
	if r.q.len == 0 && r.size-r.cur >= n {
		r.cur += n
		r.mu.Unlock()
		return true
	}
	done := ctx.Done() // only now: Done allocates on a fresh cancelCtx
	if r.closed || isClosed(done) {
		r.mu.Unlock()
		return false
	}
	w := newWaiter(n)
	r.q.push(w)
	r.mu.Unlock()

	woken := w.park(done)
	ok := woken && w.state == granted && !isClosed(done)
	if !ok {
		r.leave(w, woken)
	}
	return ok
}

// leave settles a waiter that is not keeping a grant: one granted as its
// caller gave up returns the units, one still queued is removed.
func (r *Resource) leave(w *waiter, woken bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if w.state == waiting {
		head := r.q.head == w
		r.q.remove(w)
		if head {
			r.grant()
		}
		return
	}
	if !woken {
		<-w.ready // the owner woke w as its caller gave up
	}
	if w.state == granted {
		r.cur -= w.n
		r.grant()
	}
}

func (r *Resource) release(n int64) {
	r.mu.Lock()
	r.cur -= n
	r.grant()
	r.mu.Unlock()
}

// grant admits waiters from the head while each one fits. Caller holds mu.
func (r *Resource) grant() {
	for w := r.q.head; w != nil && r.size-r.cur >= w.n; w = r.q.head {
		r.cur += w.n
		r.q.remove(w)
		w.wake(granted)
	}
}

// close withdraws every waiter and refuses new ones.
func (r *Resource) close() {
	r.mu.Lock()
	r.closed = true
	r.q.closeAll()
	r.mu.Unlock()
}
