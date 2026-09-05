// Command quickstart demonstrates the Herta V0 execution model: one
// Runtime with a finite resource budget shared by heterogeneous
// operations with different policies.
package main

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/arahe-dev/hertasdk"
)

func main() {
	ctx := context.Background()

	rt, err := execution.NewRuntime(execution.ResourceSpec{Name: "worker", Capacity: 4})
	if err != nil {
		panic(err)
	}

	// Idempotent operation: safe to retry Transient failures, waits for capacity.
	render, err := execution.NewOperation(rt, "render",
		func(ctx context.Context, job string) (string, error) {
			if job == "flaky" {
				return "", execution.Fail(execution.Transient, errors.New("upstream hiccup"))
			}
			return "rendered:" + job, nil
		},
		execution.Policy[string]{
			Effect:    execution.Idempotent,
			Resources: []execution.Requirement{{Name: "worker", Units: 1}},
			Admission: execution.Wait,
			Retry:     execution.RetryPolicy{MaxAttempts: 3, On: map[execution.Outcome]bool{execution.Transient: true}},
		})
	if err != nil {
		panic(err)
	}

	// Non-idempotent operation: fails fast when busy, retries nothing.
	charge, err := execution.NewOperation(rt, "charge",
		func(ctx context.Context, customer string) (string, error) {
			return "charged:" + customer, nil
		},
		execution.Policy[string]{
			Effect:    execution.NonIdempotent,
			Resources: []execution.Requirement{{Name: "worker", Units: 1}},
			Admission: execution.Reject,
		})
	if err != nil {
		panic(err)
	}

	jobs := []string{"a", "b", "flaky", "c", "d", "e"}
	var wg sync.WaitGroup
	for _, j := range jobs {
		wg.Add(1)
		go func(job string) {
			defer wg.Done()
			out, err := render.Do(ctx, job)
			fmt.Printf("render(%s) -> %q, err=%v (outcome=%s)\n", job, out, err, execution.OutcomeOf(err))
		}(j)
	}
	for _, c := range []string{"alice", "bob"} {
		wg.Add(1)
		go func(customer string) {
			defer wg.Done()
			out, err := charge.Do(ctx, customer)
			fmt.Printf("charge(%s) -> %q, err=%v (outcome=%s)\n", customer, out, err, execution.OutcomeOf(err))
		}(c)
	}
	wg.Wait()

	s := rt.Stats()
	fmt.Printf("stats: admitted=%d rejected=%d started=%d finished=%d retried=%d\n",
		s.Admitted, s.Rejected, s.Started, s.Finished, s.Retried)

	if err := rt.Shutdown(ctx); err != nil {
		panic(err)
	}
}
