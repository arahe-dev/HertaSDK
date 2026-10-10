package hertasdk

import (
	"context"
	"testing"
)

// White-box tests of per-key FIFO mutual exclusion (CONTRACT §4.2).

func keyWaiters(kl *KeyLocker, key string) int {
	kl.mu.Lock()
	defer kl.mu.Unlock()
	if e := kl.m[key]; e != nil {
		return e.q.len
	}
	return -1
}

func liveKeys(kl *KeyLocker) int {
	kl.mu.Lock()
	defer kl.mu.Unlock()
	return len(kl.m)
}

type lockResult struct {
	e  *keyEntry
	ok bool
}

// lockAsync starts a Wait-mode lock in a goroutine and waits until it is
// queued behind the holder.
func lockAsync(t *testing.T, kl *KeyLocker, ctx context.Context, key string) <-chan lockResult {
	t.Helper()
	before := keyWaiters(kl, key)
	got := make(chan lockResult, 1)
	go func() {
		e, ok := kl.lock(ctx, key, Wait)
		got <- lockResult{e, ok}
	}()
	waitFor(t, func() bool { return keyWaiters(kl, key) > before })
	return got
}

func mustLock(t *testing.T, kl *KeyLocker, key string) *keyEntry {
	t.Helper()
	e, ok := kl.lock(context.Background(), key, Wait)
	if !ok {
		t.Fatalf("lock %q failed", key)
	}
	return e
}

func TestKeysHandOffInArrivalOrder(t *testing.T) {
	var kl KeyLocker
	held := mustLock(t, &kl, "K")
	var got []<-chan lockResult
	for range 4 {
		got = append(got, lockAsync(t, &kl, context.Background(), "K"))
	}
	for i, ch := range got {
		kl.unlock(held)
		r := <-ch
		if !r.ok {
			t.Fatalf("waiter %d not handed the key", i)
		}
		for _, later := range got[i+1:] {
			if len(later) != 0 {
				t.Fatalf("a later waiter overtook waiter %d", i)
			}
		}
		held = r.e
	}
	kl.unlock(held)
	if n := liveKeys(&kl); n != 0 {
		t.Fatalf("%d entries after the last holder left", n)
	}
}

func TestKeysRejectFailsOnlyWhenHeld(t *testing.T) {
	var kl KeyLocker
	e, ok := kl.lock(context.Background(), "K", Reject)
	if !ok {
		t.Fatal("Reject failed on a free key")
	}
	if _, ok := kl.lock(context.Background(), "K", Reject); ok {
		t.Fatal("Reject acquired a held key")
	}
	if _, ok := kl.lock(context.Background(), "other", Reject); !ok {
		t.Fatal("Reject failed on an unrelated key")
	}
	kl.unlock(e)
}

func TestKeysCancelledWaiterLeavesHolderIntact(t *testing.T) {
	var kl KeyLocker
	held := mustLock(t, &kl, "K")
	ctx, cancel := context.WithCancel(context.Background())
	waiter := lockAsync(t, &kl, ctx, "K")
	cancel()
	if r := <-waiter; r.ok {
		t.Fatal("cancelled waiter got the key")
	}
	if n := keyWaiters(&kl, "K"); n != 0 {
		t.Fatalf("%d waiters after cancel, want 0 (entry still held)", n)
	}
	kl.unlock(held)
	if n := liveKeys(&kl); n != 0 {
		t.Fatalf("%d entries after release", n)
	}
}

func TestKeysDoneBeforeQueueingFailsAtOnce(t *testing.T) {
	var kl KeyLocker
	held := mustLock(t, &kl, "K")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, ok := kl.lock(ctx, "K", Wait); ok {
		t.Fatal("cancelled lock acquired a held key")
	}
	if n := keyWaiters(&kl, "K"); n != 0 {
		t.Fatalf("%d waiters queued by a cancelled lock", n)
	}
	kl.unlock(held)
}

func TestKeysCloseWithdrawsWaiters(t *testing.T) {
	var kl KeyLocker
	held := mustLock(t, &kl, "K")
	first := lockAsync(t, &kl, context.Background(), "K")
	second := lockAsync(t, &kl, context.Background(), "K")
	kl.close()
	if (<-first).ok || (<-second).ok {
		t.Fatal("a withdrawn waiter got the key")
	}
	if _, ok := kl.lock(context.Background(), "K", Wait); ok {
		t.Fatal("a closed locker queued a call for a held key")
	}
	kl.unlock(held)
	if n := liveKeys(&kl); n != 0 {
		t.Fatalf("%d entries after release", n)
	}
}

// A waiter handed the key at the instant it is cancelled passes it on.
func TestKeysHandOffRacingCancelPassesKeyOn(t *testing.T) {
	for range 500 {
		var kl KeyLocker
		held := mustLock(t, &kl, "K")
		ctx, cancel := context.WithCancel(context.Background())
		racer := lockAsync(t, &kl, ctx, "K")
		next := lockAsync(t, &kl, context.Background(), "K")

		kl.mu.Lock() // cancel, then hand off, before the racer can withdraw
		cancel()
		kl.handOff(held)
		kl.mu.Unlock()

		if r := <-racer; r.ok {
			t.Fatal("cancelled racer kept the key")
		}
		r := <-next
		if !r.ok {
			t.Fatal("key not passed on by the cancelled racer")
		}
		kl.unlock(r.e)
		if n := liveKeys(&kl); n != 0 {
			t.Fatalf("%d entries after release", n)
		}
	}
}

func TestKeysEntriesAreReclaimed(t *testing.T) {
	var kl KeyLocker
	held := make([]*keyEntry, 0, 1000)
	for i := range 1000 {
		held = append(held, mustLock(t, &kl, string(rune('a'+i%26))+string(rune(i))))
	}
	if n := liveKeys(&kl); n != 1000 {
		t.Fatalf("%d live entries while 1000 keys are held", n)
	}
	for _, e := range held {
		kl.unlock(e)
	}
	if n := liveKeys(&kl); n != 0 {
		t.Fatalf("%d entries survive release", n)
	}
	kl.unlock(nil) // no key: a no-op
}
