package doe

import (
	"fmt"
	"os"
	"strings"

	"github.com/arahe-dev/hertasdk/lab/impl"
)

// Arm is one compared treatment: an implementation under a report label.
// Two labels may name one implementation ("a=orig,b=orig"), which turns
// any experiment into an A/A calibration of the lab's own noise.
type Arm struct {
	Label string
	Impl  impl.Impl
}

// EnvArms is the arm list selected by LAB_IMPLS (default "orig,recon").
func EnvArms() string {
	if s := os.Getenv("LAB_IMPLS"); s != "" {
		return s
	}
	return "orig,recon"
}

// ParseArms parses "orig,recon" or "a=orig,b=orig". The first arm is the
// baseline every other arm is compared against.
func ParseArms(spec string) ([]Arm, error) {
	var arms []Arm
	seen := map[string]bool{}
	for _, part := range strings.Split(spec, ",") {
		label, name, ok := strings.Cut(strings.TrimSpace(part), "=")
		if !ok {
			name = label
		}
		im := impl.ByName(name)
		if im == nil {
			return nil, fmt.Errorf("doe: unknown implementation %q", name)
		}
		if seen[label] {
			return nil, fmt.Errorf("doe: duplicate arm label %q", label)
		}
		seen[label] = true
		arms = append(arms, Arm{Label: label, Impl: im})
	}
	if len(arms) == 0 {
		return nil, fmt.Errorf("doe: no implementations in %q", spec)
	}
	return arms, nil
}
