package workload

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/arahe-dev/hertasdk/lab/impl"
)

// The deterministic tiers run implementations inside testing/synctest
// bubbles. That needs every channel an implementation blocks on to belong
// to the bubble that blocks on it. A channel reused across bubbles (from a
// package-level sync.Pool, say) is a fatal runtime error on select, and on
// a plain receive it is not "durable", so the bubble hangs. Neither can be
// recovered in-process, so compatibility is probed once per implementation
// in a child process, and an incompatible implementation fails its bubble
// tests cleanly instead of killing or hanging the test binary.

const probeEnv = "LAB_SYNCTEST_PROBE"

// Probe is the body of TestSynctestProbe, which every package that uses
// bubbles declares. It does nothing except in the child process started
// by RequireBubbles.
func Probe(t *testing.T) {
	if name := os.Getenv(probeEnv); name != "" {
		probe(t, impl.ByName(name))
	}
}

var probes sync.Map // impl name → error (nil if compatible)

// RequireBubbles fails t unless im can run inside synctest bubbles.
func RequireBubbles(t testing.TB, im impl.Impl) {
	t.Helper()
	v, _ := probes.LoadOrStore(im.Name(), sync.OnceValue(func() error {
		cmd := exec.Command(os.Args[0], "-test.run=^TestSynctestProbe$", "-test.timeout=30s")
		cmd.Env = append(os.Environ(), probeEnv+"="+im.Name())
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("%w\n%s", err, firstLines(out, 12))
		}
		return nil
	}))
	if err := v.(func() error)(); err != nil {
		t.Fatalf("%s cannot run inside a testing/synctest bubble (it reuses a blocking channel across bubbles?):\n%v", im.Name(), err)
	}
}

// probe parks a resource waiter and a key waiter alternately outside any
// bubble and inside fresh bubbles, so any state carried between them is
// exercised.
func probe(t *testing.T, im impl.Impl) {
	park := func() {
		rt, err := im.NewRuntime(impl.Spec{Name: "r", Capacity: 1})
		if err != nil {
			panic(err)
		}
		release := make(chan struct{})
		key := func(int) string { return "k" }
		op, _ := rt.NewOp("p", impl.Policy{Effect: impl.Pure, Resources: []impl.Req{{Name: "r", Units: 1}}, Key: key},
			func(ctx context.Context, in int) (int, error) {
				if in == 0 {
					<-release
				}
				return in, nil
			})
		free, _ := rt.NewOp("q", impl.Policy{Effect: impl.Pure, Resources: []impl.Req{{Name: "r", Units: 1}}},
			func(ctx context.Context, in int) (int, error) { return in, nil })
		var wg sync.WaitGroup
		wg.Go(func() { op.Do(context.Background(), 0) })
		for rt.Stats().Started == 0 {
			runtime.Gosched()
		}
		wg.Go(func() { op.Do(context.Background(), 1) })   // parks on the key
		wg.Go(func() { free.Do(context.Background(), 1) }) // parks on the budget
		for rt.Stats().Admitted < 3 {
			runtime.Gosched()
		}
		time.Sleep(time.Millisecond) // in a bubble: until both have parked
		close(release)
		wg.Wait()
		rt.Shutdown(context.Background())
	}
	// One P makes per-P caches (sync.Pool) hand objects straight back.
	runtime.GOMAXPROCS(1)
	// A call blocked on a foreign channel is not durably blocked, so the
	// bubble's clock stalls; a real-time timer outside the bubble ends it.
	time.AfterFunc(5*time.Second, func() {
		fmt.Fprintln(os.Stderr, "a call parked inside a bubble is not durably blocked: the bubble hung")
		os.Exit(3)
	})
	for range 20 {
		park()
		synctest.Test(t, func(*testing.T) { park() })
	}
}

func firstLines(b []byte, n int) []byte {
	for i, c := range b {
		if c == '\n' {
			if n--; n == 0 {
				return b[:i]
			}
		}
	}
	return b
}
