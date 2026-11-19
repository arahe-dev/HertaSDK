package synthesis

import (
	"context"
	"errors"

	"github.com/arahe-dev/hertasdk"
)

func classify(err error) hertasdk.Outcome {
	switch {
	case errors.Is(err, ErrAckLost):
		return hertasdk.Uncertain
	case errors.Is(err, ErrNotSent):
		return hertasdk.Transient
	}
	return hertasdk.Permanent
}

// GoArm is the handwritten SDK equivalent of synthesis.he. trackPublish and
// dispatch are the typed evidence API; trackCheckout is the caller
// annotation a developer must remember in plain Go.
type GoArm struct{ trackPublish, dispatch, trackCheckout bool }

func (a GoArm) build(rt *hertasdk.Runtime) (publish, checkout func(context.Context, string) (uint64, error), err error) {
	retry := hertasdk.RetryPolicy{MaxAttempts: 2, On: map[hertasdk.Outcome]bool{hertasdk.Transient: true}}
	pub, err := hertasdk.NewOperation(rt, "go::publish", func(ctx context.Context, id string) (uint64, error) {
		req, err := prepareHost(ctx, id)
		if err != nil {
			return 0, err
		}
		if a.dispatch {
			hertasdk.Dispatch(ctx)
		}
		return sendHost(ctx, req)
	}, hertasdk.Policy[string]{
		Effect: hertasdk.NonIdempotent, Closed: true, Classify: classify, Retry: retry,
		Resources:     []hertasdk.Requirement{{Name: "writes", Units: 1}},
		TrackDispatch: a.trackPublish,
	})
	if err != nil {
		return nil, nil, err
	}
	co, err := hertasdk.NewOperation(rt, "go::checkout", func(ctx context.Context, id string) (uint64, error) {
		return pub.Do(ctx, id)
	}, hertasdk.Policy[string]{
		Effect: hertasdk.NonIdempotent, Closed: true, Calls: []string{"go::publish"},
		Classify: classify, Retry: retry, TrackDispatch: a.trackCheckout,
	})
	if err != nil {
		return nil, nil, err
	}
	return pub.Do, co.Do, nil
}
