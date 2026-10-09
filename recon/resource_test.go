package execution

import (
	"context"
	"math/rand/v2"
	"sync"
	"testing"
)

// White-box tests of the budget semaphore (CONTRACT §4.1).

// acquireAsync starts r.acquire in a goroutine, waits until it is queued
// (or granted), and returns its result channel.
func acquireAsync(t *testing.T, r *Resource, ctx context.Context, n int64) <-chan bool {
	t.Helper()
	before := queued(r)
	got := make(chan bool, 1)
	go func() { got <- r.acquire(ctx, n) }()
	waitFor(t, func() bool { return queued(r) > before || len(got) > 0 })
	return got
}

func assertAccount(t *testing.T, r *Resource, cur int64, waiting int) {
	t.Helper()
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.cur != cur || r.q.len != waiting {
		t.Fatalf("%v: in use %d, waiting %d; want %d, %d", r, r.cur, r.q.len, cur, waiting)
	}
	assertFull(t, r)
}

// assertFull checks the fast-reject flag against the state it mirrors.
// Caller holds r.mu.
func assertFull(t *testing.T, r *Resource) {
	t.Helper()
	want := r.q.len > 0 || r.closed || r.cur >= r.size
	if r.full.Load() != want || r.isFull != want {
		t.Fatalf("%v: full=%v (copy %v), want %v (in use %d, waiting %d, closed %v)",
			r, r.full.Load(), r.isFull, want, r.cur, r.q.len, r.closed)
	}
}

func TestResourceGrantsInArrivalOrder(t *testing.T) {
	r := newResource("r", 1)
	r.tryAcquire(1)
	var got []<-chan bool
	for range 5 {
		got = append(got, acquireAsync(t, r, context.Background(), 1))
	}
	assertAccount(t, r, 1, 5)
	for i, ch := range got {
		r.release(1)
		if !<-ch {
			t.Fatalf("waiter %d not granted", i)
		}
		for _, later := range got[i+1:] {
			if len(later) != 0 {
				t.Fatalf("a later waiter overtook waiter %d", i)
			}
		}
	}
	r.release(1)
	assertAccount(t, r, 0, 0)
}

func TestResourceHeadOfLineBlocks(t *testing.T) {
	r := newResource("r", 4)
	r.tryAcquire(1)
	heavy := acquireAsync(t, r, context.Background(), 4)
	if r.tryAcquire(1) {
		t.Fatal("a request barged past a queued waiter while 3 units were free")
	}
	light := acquireAsync(t, r, context.Background(), 1)
	assertAccount(t, r, 1, 2)

	r.release(1)
	if !<-heavy {
		t.Fatal("heavy head not granted when capacity freed")
	}
	assertAccount(t, r, 4, 1)
	r.release(4)
	if !<-light {
		t.Fatal("light waiter not granted after the head released")
	}
	r.release(1)
	assertAccount(t, r, 0, 0)
}

func TestResourceCancelledHeadReexaminesQueue(t *testing.T) {
	r := newResource("r", 4)
	r.tryAcquire(1)
	ctx, cancel := context.WithCancel(context.Background())
	heavy := acquireAsync(t, r, ctx, 4)
	light := acquireAsync(t, r, context.Background(), 2)
	tail := acquireAsync(t, r, ctx, 1)

	cancel()
	if <-heavy {
		t.Fatal("cancelled head was granted")
	}
	if !<-light {
		t.Fatal("waiter behind a cancelled head not re-examined")
	}
	if <-tail {
		t.Fatal("cancelled tail was granted")
	}
	assertAccount(t, r, 3, 0)
}

func TestResourceCancelledMiddleKeepsOrder(t *testing.T) {
	r := newResource("r", 1)
	r.tryAcquire(1)
	ctx, cancel := context.WithCancel(context.Background())
	first := acquireAsync(t, r, context.Background(), 1)
	middle := acquireAsync(t, r, ctx, 1)
	last := acquireAsync(t, r, context.Background(), 1)
	cancel()
	if <-middle {
		t.Fatal("cancelled middle waiter was granted")
	}
	assertAccount(t, r, 1, 2)
	for _, ch := range []<-chan bool{first, last} {
		r.release(1)
		if !<-ch {
			t.Fatal("queue order broken by a middle removal")
		}
	}
	assertAccount(t, r, 1, 0)
}

func TestResourceCloseWithdrawsWaiters(t *testing.T) {
	r := newResource("r", 2)
	r.tryAcquire(2)
	first := acquireAsync(t, r, context.Background(), 1)
	second := acquireAsync(t, r, context.Background(), 2)
	r.close()
	if <-first || <-second {
		t.Fatal("a withdrawn waiter was granted")
	}
	assertAccount(t, r, 2, 0)
	if r.acquire(context.Background(), 1) {
		t.Fatal("a closed resource queued or granted a full request")
	}
	r.release(1)
	if !r.acquire(context.Background(), 1) {
		t.Fatal("a closed resource refused a request that fits: closing withdraws waiters only")
	}
}

func TestResourceDoneBeforeQueueingFailsAtOnce(t *testing.T) {
	r := newResource("r", 1)
	r.tryAcquire(1)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if r.acquire(ctx, 1) {
		t.Fatal("cancelled acquire granted")
	}
	assertAccount(t, r, 1, 0)
}

// A waiter granted at the same instant it is cancelled must return the
// units and pass them on, whichever wake-up its select observes.
func TestResourceGrantRacingCancelReturnsUnits(t *testing.T) {
	for range 500 {
		r := newResource("r", 1)
		r.tryAcquire(1)
		ctx, cancel := context.WithCancel(context.Background())
		racer := acquireAsync(t, r, ctx, 1)
		next := acquireAsync(t, r, context.Background(), 1)

		r.mu.Lock() // cancel, then grant, before the racer can withdraw
		cancel()
		r.cur--
		r.grant()
		r.mu.Unlock()

		if <-racer {
			t.Fatal("cancelled racer kept its grant")
		}
		if !<-next {
			t.Fatal("units returned by the racer were not passed on")
		}
		assertAccount(t, r, 1, 0)
	}
}

// Random weights and cancellations, then a close, never break the accounting.
func TestResourceAccountingUnderChurn(t *testing.T) {
	const size = 8
	r := newResource("r", size)
	var wg sync.WaitGroup
	for g := range 16 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			rng := rand.New(rand.NewPCG(uint64(g), 7))
			for i := range 2000 {
				n := 1 + rng.Int64N(size)
				ctx, cancel := context.WithCancel(context.Background())
				if rng.IntN(4) == 0 {
					cancel()
				}
				if g == 0 && i == 1000 {
					r.close() // withdraw whoever is queued mid-churn
				}
				got := false
				if rng.IntN(2) == 0 {
					got = r.tryAcquire(n)
				} else {
					got = r.acquire(ctx, n)
				}
				if got {
					r.mu.Lock()
					bad := r.cur < n || r.cur > size
					assertFull(t, r)
					r.mu.Unlock()
					if bad {
						t.Error("accounting out of range while holding")
					}
					r.release(n)
				}
				r.mu.Lock()
				assertFull(t, r)
				r.mu.Unlock()
				cancel()
			}
		}()
	}
	wg.Wait()
	assertAccount(t, r, 0, 0)
}

// TestResourceFastReject pins down when tryAcquire answers from the full
// flag without the lock: only when nothing at all can be granted. A request
// that merely does not fit still takes the locked path and still fails.
func TestResourceFastReject(t *testing.T) {
	r := newResource("r", 4)
	if !r.tryAcquire(3) {
		t.Fatal("3 of 4 free units refused")
	}
	assertAccount(t, r, 3, 0) // not full: one unit left
	if r.tryAcquire(2) {
		t.Fatal("granted 2 units with 1 free")
	}
	if !r.tryAcquire(1) {
		t.Fatal("refused the last free unit")
	}
	assertAccount(t, r, 4, 0) // full
	if r.tryAcquire(1) {
		t.Fatal("granted a unit while full")
	}
	r.release(2)
	assertAccount(t, r, 2, 0) // no longer full
	if !r.tryAcquire(1) {
		t.Fatal("refused a unit after release")
	}

	// A queued waiter makes the resource full even with units free:
	// Reject must not overtake it (CONTRACT §4.1).
	got := acquireAsync(t, r, context.Background(), 2)
	assertAccount(t, r, 3, 1)
	if r.tryAcquire(1) {
		t.Fatal("Reject overtook a queued waiter")
	}
	r.release(1)
	if !<-got {
		t.Fatal("waiter not granted")
	}
	assertAccount(t, r, 4, 0)
	r.release(4)
	assertAccount(t, r, 0, 0)

	r.close()
	r.mu.Lock()
	assertFull(t, r)
	r.mu.Unlock()
	if r.tryAcquire(1) {
		t.Fatal("granted a unit after close")
	}
}
