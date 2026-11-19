package synthesis

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/arahe-dev/hertasdk"
)

type entry = func(context.Context, string) (uint64, error)

type arm struct {
	name  string
	build func(*hertasdk.Runtime) (publish, checkout entry, err error)
}

func hertaArm(name string, plan func(*hertasdk.Runtime) (entry, entry, error)) arm {
	return arm{name, plan}
}

var arms = []arm{
	{"plain Go SDK", GoArm{}.build},
	{"typed Go SDK", GoArm{trackPublish: true, dispatch: true}.build},
	{"typed Go SDK + caller annotation", GoArm{trackPublish: true, dispatch: true, trackCheckout: true}.build},
	hertaArm("Herta, no evidence clause", func(rt *hertasdk.Runtime) (entry, entry, error) {
		p, err := NewBarePlan(rt)
		if err != nil {
			return nil, nil, err
		}
		return func(ctx context.Context, id string) (uint64, error) {
				return p.Publish.Do(ctx, BarePublishInput{Id: id})
			},
			func(ctx context.Context, id string) (uint64, error) {
				return p.Checkout.Do(ctx, BareCheckoutInput{Id: id})
			}, nil
	}),
	hertaArm("Herta plan, evidence dispatch", func(rt *hertasdk.Runtime) (entry, entry, error) {
		p, err := NewMainPlan(rt)
		if err != nil {
			return nil, nil, err
		}
		return func(ctx context.Context, id string) (uint64, error) {
				return p.Publish.Do(ctx, MainPublishInput{Id: id})
			},
			func(ctx context.Context, id string) (uint64, error) {
				return p.Checkout.Do(ctx, MainCheckoutInput{Id: id})
			}, nil
	}),
}

type scenario struct {
	name   string
	fault  Fault
	nested bool
	// want is commits/outcome per arm, fixed before the first run.
	want []string
}

var scenarios = []scenario{
	{"pre-send refusal", PreSend, false, []string{"1 Success", "1 Success", "1 Success", "1 Success", "1 Success"}},
	{"lost ack, honest adapter", AckHonest, false, []string{"1 Uncertain", "1 Uncertain", "1 Uncertain", "1 Uncertain", "1 Uncertain"}},
	{"lost ack, adapter says not sent", AckMislabel, false, []string{"2 Success", "1 Uncertain", "1 Uncertain", "2 Success", "1 Uncertain"}},
	{"nested: checkout -> publish, mislabel", AckMislabel, true, []string{"2 Success", "2 Success", "1 Uncertain", "2 Success", "1 Uncertain"}},
	{"adapter sends before dispatch point", EarlySend, false, []string{"2 Success", "2 Success", "2 Success", "2 Success", "2 Success"}},
}

// TestMatrix runs every scenario on every arm against a fresh runtime and
// provider; committed writes are counted by the provider, not the SDK.
func TestMatrix(t *testing.T) {
	var table strings.Builder
	table.WriteString("| scenario |")
	for _, a := range arms {
		table.WriteString(" " + a.name + " |")
	}
	table.WriteString("\n|---|" + strings.Repeat("---|", len(arms)) + "\n")
	for _, sc := range scenarios {
		table.WriteString("| " + sc.name + " |")
		for i, a := range arms {
			rt, err := NewMainRuntime()
			if err != nil {
				t.Fatal(err)
			}
			publish, checkout, err := a.build(rt)
			if err != nil {
				t.Fatal(err)
			}
			prov = &provider{fault: sc.fault}
			call := publish
			if sc.nested {
				call = checkout
			}
			_, err = call(context.Background(), "order-1")
			got := fmt.Sprintf("%d %s", prov.commits, hertasdk.OutcomeOf(err))
			if got != sc.want[i] {
				t.Errorf("%s / %s: got %q, want %q", sc.name, a.name, got, sc.want[i])
			}
			fmt.Fprintf(&table, " %s |", strings.Replace(got, " ", " commit(s), ", 1))
		}
		table.WriteString("\n")
	}
	t.Log("\n" + table.String())
	if out := os.Getenv("SYNTHESIS_TABLE"); out != "" {
		if err := os.WriteFile(out, []byte(table.String()), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// BenchmarkPublish is the cost of dispatch evidence on an uncontended call.
func BenchmarkPublish(b *testing.B) {
	for _, a := range arms[:2] {
		b.Run(a.name, func(b *testing.B) {
			rt, _ := NewMainRuntime()
			publish, _, _ := a.build(rt)
			prov = &provider{fault: -1}
			b.ReportAllocs()
			for b.Loop() {
				if _, err := publish(context.Background(), "order-1"); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
