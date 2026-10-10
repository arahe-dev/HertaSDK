package hertasdk

// waiter is one blocked acquisition, queued intrusively in a waitq. Its
// owner (a Resource or a KeyLocker) mutates it under the owner's lock and
// wakes it at most once, recording why in state before sending the token.
//
// A waiter and its channel are allocated per blocked call (2 allocations,
// inside the §10 waiter budget) and never pooled: a channel belongs to the
// testing/synctest bubble that made it, so a pooled one would make every
// caller that tests with synctest crash or hang.
type waiter struct {
	n          int64
	state      waitState
	ready      chan struct{} // cap 1: the single wake-up token
	prev, next *waiter
}

type waitState uint8

const (
	waiting   waitState = iota
	granted             // the owner handed this waiter its acquisition
	withdrawn           // Shutdown closed the queue under this waiter
)

func newWaiter(n int64) *waiter {
	return &waiter{n: n, ready: make(chan struct{}, 1)}
}

// wake records s and sends the token. The caller holds the owner's lock
// and has removed w from its queue.
func (w *waiter) wake(s waitState) {
	w.state = s
	w.ready <- struct{}{}
}

// park blocks until w is woken or done closes. It reports whether it took
// the token; if so, w.state is the owner's verdict and safe to read, since
// the owner wrote it before sending. Waiters select only on their own
// channels, so parked calls share no lock.
func (w *waiter) park(done <-chan struct{}) (woken bool) {
	if done == nil {
		<-w.ready
		return true
	}
	select {
	case <-w.ready:
		return true
	case <-done:
		return false
	}
}

// isClosed reports whether ch is closed; a nil channel never is.
func isClosed(ch <-chan struct{}) bool {
	select {
	case <-ch:
		return true
	default:
		return false
	}
}

// waitq is a FIFO of waiters with an exact length.
type waitq struct {
	head, tail *waiter
	len        int
}

func (q *waitq) push(w *waiter) {
	w.prev, w.next = q.tail, nil
	if q.tail == nil {
		q.head = w
	} else {
		q.tail.next = w
	}
	q.tail = w
	q.len++
}

func (q *waitq) remove(w *waiter) {
	if w.prev == nil {
		q.head = w.next
	} else {
		w.prev.next = w.next
	}
	if w.next == nil {
		q.tail = w.prev
	} else {
		w.next.prev = w.prev
	}
	w.prev, w.next = nil, nil
	q.len--
}

// closeAll withdraws every waiter, in order. Caller holds the owner's lock.
func (q *waitq) closeAll() {
	for w := q.head; w != nil; w = q.head {
		q.remove(w)
		w.wake(withdrawn)
	}
}
