//go:build mirai_probe

// Consumer-level probes: what a mirai Wait consumer (events) observes of
// the CONTRACT §8 deltas through its own error mapping. mirai.sh copies
// this file into internal/events of both consumer copies, so it compiles
// against whichever implementation sits in internal/execution. Each probe
// passes on both and logs one "PROBE" line for the side-by-side table.
package events

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"testing"
	"time"

	"github.com/mirai/mirai-backend/internal/execution"
)

// TestProbeDelta6CancelledWaiterAtShutdown: a batch queued for db-write
// whose caller gives up just as the bridge starts draining. The handler's
// contract is "caller context wins first" (toConnectCodeExec), i.e. the
// client should see Canceled. v0.2 reports ErrRuntimeStopping instead,
// which events maps to Internal; recon reports ctx.Err() (Δ6).
func TestProbeDelta6CancelledWaiterAtShutdown(t *testing.T) {
	const runs = 200
	codes := map[string]int{}
	for range runs {
		codes[cancelledWaiterAtShutdown(t)]++
	}
	t.Logf("PROBE delta6 events code for a waiter cancelled as Shutdown begins (%d runs): %s", runs, tally(codes))
}

func cancelledWaiterAtShutdown(t *testing.T) string {
	store := &gatedStore{entered: make(chan struct{}, 1), gate: make(chan struct{})}
	svc, err := NewService(store, testLogger())
	if err != nil {
		t.Fatal(err)
	}
	rt, err := execution.NewRuntime(execution.ResourceSpec{Name: "db-write", Capacity: 1})
	if err != nil {
		t.Fatal(err)
	}
	op, err := NewOperation(rt, svc, Config{AttemptTimeout: time.Minute, MaxAttempts: 1})
	if err != nil {
		t.Fatal(err)
	}
	batch := []Event{validEvent("probe-1")}
	go op.Do(context.Background(), batch) // the holder: owns db-write until gate opens
	<-store.entered

	ctx, cancel := context.WithCancel(context.Background())
	res := make(chan error, 1)
	go func() { _, err := op.Do(ctx, batch); res <- err }()
	for rt.Stats().Admitted < 2 {
		time.Sleep(50 * time.Microsecond)
	}
	time.Sleep(time.Millisecond) // let it queue behind the holder

	cancel()
	expired, stop := context.WithCancel(context.Background())
	stop()
	_ = rt.Shutdown(expired) // begin draining without waiting for it
	code := codeOf(<-res)

	close(store.gate)
	if err := rt.Shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}
	return code
}

type gatedStore struct {
	entered chan struct{}
	gate    chan struct{}
}

func (s *gatedStore) RecordBatch(ctx context.Context, batch []Event) (RecordResult, error) {
	s.entered <- struct{}{}
	<-s.gate
	return RecordResult{Inserted: uint32(len(batch))}, nil
}

func codeOf(err error) string {
	if err == nil {
		return "ok"
	}
	return toConnectCodeExec(err).String()
}

func tally(m map[string]int) string {
	var s string
	for _, k := range slices.Sorted(maps.Keys(m)) {
		s += fmt.Sprintf("%s=%d ", k, m[k])
	}
	return s
}
