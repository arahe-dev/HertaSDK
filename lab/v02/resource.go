package execution

import (
	"context"
	"fmt"

	"golang.org/x/sync/semaphore"
)

// Resource is a named weighted semaphore: a finite capacity shared across
// operations. Deliberately thin over golang.org/x/sync/semaphore — Herta
// does not write its own semaphore.
type Resource struct {
	name     string
	capacity int64
	sem      *semaphore.Weighted
}

func newResource(name string, capacity int64) *Resource {
	return &Resource{
		name:     name,
		capacity: capacity,
		sem:      semaphore.NewWeighted(capacity),
	}
}

// acquire blocks until units are available or ctx is done. For Wait
// admission. The ctx passed in is the caller's admission context, which
// the runtime cancels on shutdown — so a shutdown wakes waiters through
// this same ctx. On cancellation (including shutdown), semaphore.Weighted
// releases any units it had already granted before returning the error.
func (r *Resource) acquire(ctx context.Context, units int64) error {
	return r.sem.Acquire(ctx, units)
}

// tryAcquire attempts immediate acquisition (Reject admission).
func (r *Resource) tryAcquire(units int64) bool {
	return r.sem.TryAcquire(units)
}

func (r *Resource) release(units int64) {
	r.sem.Release(units)
}

// String renders "name(capacity)" for diagnostics.
func (r *Resource) String() string {
	return fmt.Sprintf("%s(%d)", r.name, r.capacity)
}
