# DoE full: orig=orig vs recon=recon

`20261010-005319-full-orig-recon` · 2026-10-10T00:53:19+05:30 · go1.27.1 windows/amd64 · 20 CPUs (GOMAXPROCS 20) · 80 trials (0 failed) in 109s · 5 reps · warm-up 300ms + window 1000ms · seed 1

Command: `go run ./lab/cmd/doe -design full -factors A,D,E -reps 5 -dur 1s -warmup 300ms -q -out lab/results/r1/reject-interaction-fixed`

## Verdict

X is the implementation factor (−1 baseline, +1 candidate), crossed with every design run, so its main effect is the candidate's change averaged over the whole design. Log-scale responses show the ratio candidate/baseline of geometric means; raw responses show the difference. Significance is Bonferroni-corrected over all contrasts of a response; ratios within 5% count as *same*.

### Key responses

| response | arms | baseline | candidate | change [95% CI] | p | verdict | strongest X×factor |
|---|---|---|---|---|---|---|---|
| throughput | orig→recon | 766k | 1.48M | ×1.93 [1.88, 1.99] | <1e-4 | **better** | DX (×1.21) |
| goodput | orig→recon | 710k | 995k | ×1.4 [1.35, 1.45] | <1e-4 | **better** | AX (×0.749) |
| lat_p50 | orig→recon | 9.77µs | 3.04µs | ×0.311 [0.299, 0.324] | <1e-4 | **better** | EX (×0.529) |
| lat_p99 | orig→recon | 342µs | 156µs | ×0.455 [0.437, 0.473] | <1e-4 | **better** | EX (×1.75) |
| lat_p999 | orig→recon | 1.23ms | 334µs | ×0.272 [0.26, 0.284] | <1e-4 | **better** | EX (×1.82) |
| wait_p99 | orig→recon | 281µs | 129µs | ×0.459 [0.443, 0.476] | <1e-4 | **better** | EX (×1.78) |
| allocs_op | orig→recon | 11.7 | 0.746 | -11 [-11, -11] | <1e-4 | **better** | EX (-1.14) |
| bytes_op | orig→recon | 525 | 46.8 | -478 [-479, -478] | <1e-4 | **better** | EX (-75) |
| peak_goroutines | orig→recon | 923 | 0 | -923 [-1.02e+03, -828] | <1e-4 | **better** | EX (+376) |
| err_rate | orig→recon | 5.96% | 22.97% | +0.17 [+0.165, +0.176] | <1e-4 | **worse** | DX (+0.17) |
| cap_violations | orig→recon | 0 | 0 | +0 [+0, +0] | – | same | – |
| key_violations | orig→recon | 0 | 0 | +0 [+0, +0] | – | same | – |
| drain | orig→recon | 98.5µs | 43µs | ×0.437 [0.359, 0.531] | <1e-4 | **better** | DX (×0.729) |

<details><summary>All other responses</summary>

| response | arms | baseline | candidate | change [95% CI] | p | verdict | strongest X×factor |
|---|---|---|---|---|---|---|---|
| lat_p90 | orig→recon | 72.6µs | 49.5µs | ×0.682 [0.58, 0.802] | <1e-4 | **better** | – |
| lat_max | orig→recon | 6.4ms | 4.71ms | ×0.736 [0.557, 0.973] | 0.032 | same | AX (×1.78) |
| wait_p50 | orig→recon | 4.89µs | 1.36µs | ×0.279 [0.267, 0.292] | <1e-4 | **better** | EX (×0.604) |
| exec_p50 | orig→recon | 1.27µs | 480ns | ×0.378 [0.365, 0.391] | <1e-4 | **better** | DX (×1.61) |
| exec_p99 | orig→recon | 108µs | 45.1µs | ×0.419 [0.398, 0.44] | <1e-4 | **better** | DX (×2.21) |
| peak_heap | orig→recon | 8.07 | 7.57 | ×0.938 [0.89, 0.989] | 0.0186 | same | – |
| gc_per_s | orig→recon | 118 | 12.3 | -106 [-114, -98.2] | <1e-4 | **better** | AX (+31.8) |
| retries_op | orig→recon | 0 | 0 | +0 [+0, +0] | – | same | – |
| fairness | orig→recon | 1 | 0.976 | -0.024 [-0.0328, -0.0153] | <1e-4 | **worse** | DX (-0.0234) |
| stats_lag | orig→recon | 0 | 0 | +0 [+0, +0] | – | same | – |
| residual_goroutines | orig→recon | 0 | 0 | +0 [+0, +0] | – | same | – |

</details>

## Noise

Replicate error of a single trial (pooled within-cell SD) as a coefficient of variation: the ±1σ spread between identical trials.

| response | orig | recon |
|---|---|---|
| throughput | 7.9% | 5.3% |
| goodput | 8.7% | 7.7% |
| lat_p50 | 10.7% | 7.7% |
| lat_p90 | 12.7% | 64.8% |
| lat_p99 | 8.3% | 10.1% |
| lat_p999 | 8.5% | 11.5% |
| lat_max | 49.3% | 119.5% |
| wait_p50 | 8.0% | 12.7% |
| wait_p99 | 7.2% | 9.4% |
| exec_p50 | 9.3% | 6.9% |
| exec_p99 | 13.3% | 10.2% |
| allocs_op | 0.3% | 2.3% |
| bytes_op | 0.2% | 1.6% |
| peak_goroutines | 32.3% | – |
| peak_heap | 7.6% | 16.1% |
| gc_per_s | 21.0% | 12.3% |
| err_rate | 15.5% | 6.3% |
| retries_op | – | – |
| fairness | 0.0% | 2.8% |
| cap_violations | – | – |
| key_violations | – | – |
| drain | 56.3% | 53.9% |
| stats_lag | – | – |
| residual_goroutines | – | – |

## Error classes

Share of all calls, over all trials of each arm. A class is the implementation-neutral error class, the outcome what `OutcomeOf` reports.

| class/outcome | orig | recon |
|---|---|---|
| overloaded/throttled | 10.184% | 29.180% |

## Design

8 runs × 2 arms × 5 reps, randomised within each replicate block. 2^3 full factorial: every interaction is estimable.

| symbol | factor | low (−1) | high (+1) |
|---|---|---|---|
| A | clients | 8 | 256 |
| D | admission | wait | reject |
| E | keys | none | 64·zipf1.2 |

<details><summary>Design matrix</summary>

```
run A D E
  0 - - -
  1 + - -
  2 - + -
  3 + + -
  4 - - +
  5 + - +
  6 - + +
  7 + + +
```

</details>

## Effects

Top effects per scope by |t|. An effect is the change from the low to the high level (a ratio for log-scale responses). `*`/`**`: beyond Lenth's ME/SME; **sig**: Bonferroni p < 0.05/m on replicate error. effects.csv has every contrast.

### throughput (ops/s)

| scope | effect | estimate | 95% CI | p | Lenth |  |
|---|---|---|---|---|---|---|
| orig | D | ×1.49 | [1.42, 1.56] | <1e-4 |  | **sig** |
| orig | DE | ×0.678 | [0.646, 0.712] | <1e-4 |  | **sig** |
| orig | E | ×0.691 | [0.658, 0.725] | <1e-4 |  | **sig** |
| orig | AE | ×1.2 | [1.14, 1.26] | <1e-4 |  | **sig** |
| orig | ADE | ×0.852 | [0.811, 0.894] | <1e-4 |  | **sig** |
| orig | AD | ×1.17 | [1.12, 1.23] | <1e-4 |  | **sig** |
| orig vs recon | X | ×1.93 | [1.88, 1.99] | <1e-4 | ** | **sig** |
| orig vs recon | D | ×1.81 | [1.75, 1.86] | <1e-4 | * | **sig** |
| orig vs recon | E | ×0.672 | [0.653, 0.692] | <1e-4 | * | **sig** |
| orig vs recon | DE | ×0.691 | [0.671, 0.711] | <1e-4 | * | **sig** |
| orig vs recon | DX | ×1.21 | [1.18, 1.25] | <1e-4 |  | **sig** |
| orig vs recon | AE | ×1.2 | [1.16, 1.23] | <1e-4 |  | **sig** |
| recon | D | ×2.19 | [2.12, 2.27] | <1e-4 |  | **sig** |
| recon | E | ×0.654 | [0.633, 0.676] | <1e-4 |  | **sig** |
| recon | DE | ×0.703 | [0.68, 0.727] | <1e-4 |  | **sig** |
| recon | A | ×0.82 | [0.793, 0.848] | <1e-4 |  | **sig** |
| recon | AE | ×1.19 | [1.15, 1.23] | <1e-4 |  | **sig** |
| recon | AD | ×1.09 | [1.06, 1.13] | <1e-4 |  | **sig** |

### goodput (ops/s)

| scope | effect | estimate | 95% CI | p | Lenth |  |
|---|---|---|---|---|---|---|
| orig | AE | ×1.36 | [1.29, 1.44] | <1e-4 |  | **sig** |
| orig | D | ×1.28 | [1.21, 1.35] | <1e-4 |  | **sig** |
| orig | DE | ×0.79 | [0.749, 0.833] | <1e-4 |  | **sig** |
| orig | E | ×0.805 | [0.763, 0.849] | <1e-4 |  | **sig** |
| orig | A | ×0.829 | [0.786, 0.875] | <1e-4 |  | **sig** |
| orig | ADE | ×0.969 | [0.918, 1.02] | 0.239 |  |  |
| orig vs recon | A | ×0.621 | [0.599, 0.643] | <1e-4 |  | **sig** |
| orig vs recon | E | ×0.642 | [0.62, 0.665] | <1e-4 |  | **sig** |
| orig vs recon | DE | ×0.659 | [0.636, 0.683] | <1e-4 |  | **sig** |
| orig vs recon | X | ×1.4 | [1.35, 1.45] | <1e-4 |  | **sig** |
| orig vs recon | AX | ×0.749 | [0.723, 0.776] | <1e-4 |  | **sig** |
| orig vs recon | ADX | ×0.776 | [0.749, 0.803] | <1e-4 |  | **sig** |
| recon | A | ×0.465 | [0.443, 0.488] | <1e-4 |  | **sig** |
| recon | E | ×0.512 | [0.488, 0.537] | <1e-4 |  | **sig** |
| recon | DE | ×0.55 | [0.525, 0.577] | <1e-4 |  | **sig** |
| recon | AD | ×0.62 | [0.591, 0.65] | <1e-4 |  | **sig** |
| recon | ADE | ×0.791 | [0.754, 0.829] | <1e-4 |  | **sig** |
| recon | AE | ×0.951 | [0.906, 0.997] | 0.0388 |  |  |

### lat_p50 (ns)

| scope | effect | estimate | 95% CI | p | Lenth |  |
|---|---|---|---|---|---|---|
| orig | A | ×6.02 | [5.64, 6.43] | <1e-4 |  | **sig** |
| orig | D | ×0.255 | [0.239, 0.272] | <1e-4 |  | **sig** |
| orig | DE | ×3.91 | [3.67, 4.18] | <1e-4 |  | **sig** |
| orig | ADE | ×3.51 | [3.29, 3.75] | <1e-4 |  | **sig** |
| orig | AD | ×0.286 | [0.268, 0.306] | <1e-4 |  | **sig** |
| orig | AE | ×0.494 | [0.463, 0.527] | <1e-4 |  | **sig** |
| orig vs recon | DE | ×6.27 | [6.03, 6.53] | <1e-4 |  | **sig** |
| orig vs recon | A | ×5.03 | [4.83, 5.23] | <1e-4 |  | **sig** |
| orig vs recon | ADE | ×3.41 | [3.28, 3.55] | <1e-4 |  | **sig** |
| orig vs recon | AD | ×0.296 | [0.285, 0.308] | <1e-4 |  | **sig** |
| orig vs recon | X | ×0.311 | [0.299, 0.324] | <1e-4 |  | **sig** |
| orig vs recon | D | ×0.323 | [0.31, 0.336] | <1e-4 |  | **sig** |
| recon | DE | ×10.1 | [9.58, 10.5] | <1e-4 |  | **sig** |
| recon | A | ×4.19 | [4, 4.4] | <1e-4 |  | **sig** |
| recon | E | ×0.264 | [0.252, 0.277] | <1e-4 |  | **sig** |
| recon | ADE | ×3.31 | [3.15, 3.47] | <1e-4 |  | **sig** |
| recon | AD | ×0.306 | [0.292, 0.321] | <1e-4 |  | **sig** |
| recon | D | ×0.409 | [0.39, 0.429] | <1e-4 |  | **sig** |

### lat_p99 (ns)

| scope | effect | estimate | 95% CI | p | Lenth |  |
|---|---|---|---|---|---|---|
| orig | A | ×30.6 | [29.1, 32.2] | <1e-4 | ** | **sig** |
| orig | E | ×1.67 | [1.58, 1.76] | <1e-4 |  | **sig** |
| orig | D | ×0.82 | [0.779, 0.863] | <1e-4 |  | **sig** |
| orig | DE | ×1.2 | [1.14, 1.27] | <1e-4 |  | **sig** |
| orig | AD | ×1.16 | [1.1, 1.22] | <1e-4 |  | **sig** |
| orig | ADE | ×0.877 | [0.833, 0.923] | <1e-4 |  | **sig** |
| orig vs recon | A | ×26.8 | [25.7, 27.9] | <1e-4 | ** | **sig** |
| orig vs recon | E | ×2.92 | [2.8, 3.03] | <1e-4 |  | **sig** |
| orig vs recon | X | ×0.455 | [0.437, 0.473] | <1e-4 |  | **sig** |
| orig vs recon | DE | ×2.17 | [2.09, 2.26] | <1e-4 |  | **sig** |
| orig vs recon | DEX | ×1.8 | [1.73, 1.88] | <1e-4 |  | **sig** |
| orig vs recon | EX | ×1.75 | [1.68, 1.82] | <1e-4 |  | **sig** |
| recon | A | ×23.4 | [22, 24.9] | <1e-4 | * | **sig** |
| recon | E | ×5.1 | [4.79, 5.42] | <1e-4 |  | **sig** |
| recon | DE | ×3.92 | [3.68, 4.16] | <1e-4 |  | **sig** |
| recon | AE | ×1.91 | [1.79, 2.03] | <1e-4 |  | **sig** |
| recon | D | ×0.744 | [0.699, 0.791] | <1e-4 |  | **sig** |
| recon | AD | ×1.3 | [1.22, 1.38] | <1e-4 |  | **sig** |

### lat_p999 (ns)

| scope | effect | estimate | 95% CI | p | Lenth |  |
|---|---|---|---|---|---|---|
| orig | A | ×7.25 | [6.88, 7.65] | <1e-4 | * | **sig** |
| orig | D | ×1.23 | [1.17, 1.3] | <1e-4 |  | **sig** |
| orig | AD | ×1.23 | [1.16, 1.29] | <1e-4 |  | **sig** |
| orig | DE | ×0.815 | [0.773, 0.859] | <1e-4 |  | **sig** |
| orig | ADE | ×0.822 | [0.779, 0.866] | <1e-4 |  | **sig** |
| orig | E | ×1.16 | [1.1, 1.23] | <1e-4 |  | **sig** |
| orig vs recon | A | ×12.9 | [12.3, 13.5] | <1e-4 | ** | **sig** |
| orig vs recon | X | ×0.272 | [0.26, 0.284] | <1e-4 | * | **sig** |
| orig vs recon | DEX | ×2.2 | [2.11, 2.3] | <1e-4 |  | **sig** |
| orig vs recon | E | ×2.12 | [2.04, 2.22] | <1e-4 |  | **sig** |
| orig vs recon | EX | ×1.82 | [1.75, 1.9] | <1e-4 |  | **sig** |
| orig vs recon | DE | ×1.8 | [1.72, 1.88] | <1e-4 |  | **sig** |
| recon | A | ×22.9 | [21.4, 24.6] | <1e-4 | * | **sig** |
| recon | DE | ×3.96 | [3.69, 4.25] | <1e-4 |  | **sig** |
| recon | E | ×3.88 | [3.61, 4.16] | <1e-4 |  | **sig** |
| recon | AE | ×1.66 | [1.55, 1.78] | <1e-4 |  | **sig** |
| recon | ADE | ×1.31 | [1.23, 1.41] | <1e-4 |  | **sig** |
| recon | D | ×0.786 | [0.733, 0.843] | <1e-4 |  | **sig** |

### wait_p99 (ns)

| scope | effect | estimate | 95% CI | p | Lenth |  |
|---|---|---|---|---|---|---|
| orig | A | ×39.6 | [37.8, 41.4] | <1e-4 | ** | **sig** |
| orig | E | ×1.63 | [1.56, 1.7] | <1e-4 |  | **sig** |
| orig | DE | ×1.24 | [1.19, 1.3] | <1e-4 |  | **sig** |
| orig | D | ×0.81 | [0.774, 0.846] | <1e-4 |  | **sig** |
| orig | ADE | ×0.857 | [0.82, 0.896] | <1e-4 |  | **sig** |
| orig | AD | ×1.15 | [1.1, 1.2] | <1e-4 |  | **sig** |
| orig vs recon | A | ×32.2 | [31, 33.3] | <1e-4 | ** | **sig** |
| orig vs recon | E | ×2.9 | [2.8, 3.01] | <1e-4 |  | **sig** |
| orig vs recon | X | ×0.459 | [0.443, 0.476] | <1e-4 |  | **sig** |
| orig vs recon | DE | ×2.15 | [2.08, 2.23] | <1e-4 |  | **sig** |
| orig vs recon | EX | ×1.78 | [1.72, 1.85] | <1e-4 |  | **sig** |
| orig vs recon | DEX | ×1.74 | [1.68, 1.8] | <1e-4 |  | **sig** |
| recon | A | ×26.1 | [24.7, 27.7] | <1e-4 |  | **sig** |
| recon | E | ×5.17 | [4.88, 5.48] | <1e-4 |  | **sig** |
| recon | DE | ×3.74 | [3.53, 3.97] | <1e-4 |  | **sig** |
| recon | AE | ×1.87 | [1.76, 1.98] | <1e-4 |  | **sig** |
| recon | D | ×0.575 | [0.543, 0.61] | <1e-4 |  | **sig** |
| recon | AD | ×1.31 | [1.23, 1.38] | <1e-4 |  | **sig** |

### allocs_op (allocs)

| scope | effect | estimate | 95% CI | p | Lenth |  |
|---|---|---|---|---|---|---|
| orig | E | +1.94 | [+1.92, +1.97] | <1e-4 |  | **sig** |
| orig | AE | -1.09 | [-1.11, -1.07] | <1e-4 |  | **sig** |
| orig | D | -1.04 | [-1.06, -1.02] | <1e-4 |  | **sig** |
| orig | DE | +1.02 | [+1, +1.05] | <1e-4 |  | **sig** |
| orig | A | +0.717 | [+0.696, +0.739] | <1e-4 |  | **sig** |
| orig | ADE | +0.333 | [+0.312, +0.354] | <1e-4 |  | **sig** |
| orig vs recon | X | -11 | [-11, -11] | <1e-4 | ** | **sig** |
| orig vs recon | EX | -1.14 | [-1.15, -1.13] | <1e-4 |  | **sig** |
| orig vs recon | E | +0.808 | [+0.796, +0.819] | <1e-4 |  | **sig** |
| orig vs recon | D | -0.805 | [-0.817, -0.793] | <1e-4 |  | **sig** |
| orig vs recon | DE | +0.747 | [+0.735, +0.759] | <1e-4 |  | **sig** |
| orig vs recon | AE | -0.655 | [-0.667, -0.643] | <1e-4 |  | **sig** |
| recon | D | -0.573 | [-0.584, -0.562] | <1e-4 |  | **sig** |
| recon | A | +0.553 | [+0.542, +0.564] | <1e-4 |  | **sig** |
| recon | DE | +0.47 | [+0.459, +0.481] | <1e-4 |  | **sig** |
| recon | E | -0.329 | [-0.34, -0.318] | <1e-4 |  | **sig** |
| recon | ADE | +0.327 | [+0.316, +0.338] | <1e-4 |  | **sig** |
| recon | AE | -0.216 | [-0.227, -0.205] | <1e-4 |  | **sig** |

### bytes_op (B)

| scope | effect | estimate | 95% CI | p | Lenth |  |
|---|---|---|---|---|---|---|
| orig | E | +120 | [+119, +120] | <1e-4 |  | **sig** |
| orig | D | -104 | [-104, -103] | <1e-4 |  | **sig** |
| orig | DE | +103 | [+102, +104] | <1e-4 |  | **sig** |
| orig | AE | -50 | [-50.6, -49.3] | <1e-4 |  | **sig** |
| orig | ADE | +32.6 | [+31.9, +33.2] | <1e-4 |  | **sig** |
| orig | AD | -32.1 | [-32.8, -31.5] | <1e-4 |  | **sig** |
| orig vs recon | X | -478 | [-479, -478] | <1e-4 | ** | **sig** |
| orig vs recon | D | -87.7 | [-88.1, -87.3] | <1e-4 |  | **sig** |
| orig vs recon | EX | -75 | [-75.4, -74.6] | <1e-4 |  | **sig** |
| orig vs recon | DE | +68.4 | [+68, +68.8] | <1e-4 |  | **sig** |
| orig vs recon | E | +44.8 | [+44.4, +45.2] | <1e-4 |  | **sig** |
| orig vs recon | AE | -35.2 | [-35.6, -34.8] | <1e-4 |  | **sig** |
| recon | D | -71.6 | [-72.1, -71.1] | <1e-4 |  | **sig** |
| recon | DE | +33.7 | [+33.2, +34.1] | <1e-4 |  | **sig** |
| recon | E | -30.3 | [-30.7, -29.8] | <1e-4 |  | **sig** |
| recon | A | +29.9 | [+29.4, +30.4] | <1e-4 |  | **sig** |
| recon | ADE | +23 | [+22.6, +23.5] | <1e-4 |  | **sig** |
| recon | AE | -20.4 | [-20.9, -19.9] | <1e-4 |  | **sig** |

### peak_goroutines (goroutines)

| scope | effect | estimate | 95% CI | p | Lenth |  |
|---|---|---|---|---|---|---|
| orig | DE | -873 | [-1.07e+03, -681] | <1e-4 |  | **sig** |
| orig | E | -752 | [-944, -560] | <1e-4 |  | **sig** |
| orig | A | -585 | [-777, -393] | <1e-4 |  | **sig** |
| orig | D | +560 | [+368, +752] | <1e-4 |  | **sig** |
| orig | ADE | +346 | [+154, +538] | 0.00088 |  | **sig** |
| orig | AE | +290 | [+98.1, +482] | 0.00427 |  | **sig** |
| orig vs recon | X | -923 | [-1.02e+03, -828] | <1e-4 |  | **sig** |
| orig vs recon | DE | -436 | [-531, -342] | <1e-4 |  | **sig** |
| orig vs recon | DEX | +436 | [+342, +531] | <1e-4 |  | **sig** |
| orig vs recon | E | -376 | [-470, -282] | <1e-4 |  | **sig** |
| orig vs recon | EX | +376 | [+282, +470] | <1e-4 |  | **sig** |
| orig vs recon | A | -292 | [-387, -198] | <1e-4 |  | **sig** |
| recon | A | +0 | [+0, +0] | – |  |  |
| recon | D | +0 | [+0, +0] | – |  |  |
| recon | E | +0 | [+0, +0] | – |  |  |
| recon | AD | +0 | [+0, +0] | – |  |  |
| recon | AE | +0 | [+0, +0] | – |  |  |
| recon | DE | +0 | [+0, +0] | – |  |  |

### err_rate (fraction)

| scope | effect | estimate | 95% CI | p | Lenth |  |
|---|---|---|---|---|---|---|
| orig | D | +0.119 | [+0.113, +0.125] | <1e-4 |  | **sig** |
| orig | E | -0.119 | [-0.125, -0.113] | <1e-4 |  | **sig** |
| orig | DE | -0.119 | [-0.125, -0.113] | <1e-4 |  | **sig** |
| orig | A | +0.0959 | [+0.0899, +0.102] | <1e-4 |  | **sig** |
| orig | AD | +0.0959 | [+0.0899, +0.102] | <1e-4 |  | **sig** |
| orig | AE | -0.0958 | [-0.102, -0.0899] | <1e-4 |  | **sig** |
| orig vs recon | D | +0.289 | [+0.284, +0.295] | <1e-4 |  | **sig** |
| orig vs recon | A | +0.176 | [+0.17, +0.181] | <1e-4 |  | **sig** |
| orig vs recon | AD | +0.176 | [+0.17, +0.181] | <1e-4 |  | **sig** |
| orig vs recon | X | +0.17 | [+0.165, +0.176] | <1e-4 |  | **sig** |
| orig vs recon | DX | +0.17 | [+0.165, +0.176] | <1e-4 |  | **sig** |
| orig vs recon | EX | +0.0947 | [+0.0893, +0.1] | <1e-4 |  | **sig** |
| recon | D | +0.459 | [+0.45, +0.469] | <1e-4 | * | **sig** |
| recon | A | +0.255 | [+0.246, +0.265] | <1e-4 |  | **sig** |
| recon | AD | +0.255 | [+0.246, +0.265] | <1e-4 |  | **sig** |
| recon | E | +0.0703 | [+0.061, +0.0797] | <1e-4 |  | **sig** |
| recon | DE | +0.0703 | [+0.061, +0.0797] | <1e-4 |  | **sig** |
| recon | AE | +0.0554 | [+0.046, +0.0647] | <1e-4 |  | **sig** |

### cap_violations (count)

| scope | effect | estimate | 95% CI | p | Lenth |  |
|---|---|---|---|---|---|---|
| orig | A | +0 | [+0, +0] | – |  |  |
| orig | D | +0 | [+0, +0] | – |  |  |
| orig | E | +0 | [+0, +0] | – |  |  |
| orig | AD | +0 | [+0, +0] | – |  |  |
| orig | AE | +0 | [+0, +0] | – |  |  |
| orig | DE | +0 | [+0, +0] | – |  |  |
| orig vs recon | A | +0 | [+0, +0] | – |  |  |
| orig vs recon | D | +0 | [+0, +0] | – |  |  |
| orig vs recon | E | +0 | [+0, +0] | – |  |  |
| orig vs recon | X | +0 | [+0, +0] | – |  |  |
| orig vs recon | AD | +0 | [+0, +0] | – |  |  |
| orig vs recon | AE | +0 | [+0, +0] | – |  |  |
| recon | A | +0 | [+0, +0] | – |  |  |
| recon | D | +0 | [+0, +0] | – |  |  |
| recon | E | +0 | [+0, +0] | – |  |  |
| recon | AD | +0 | [+0, +0] | – |  |  |
| recon | AE | +0 | [+0, +0] | – |  |  |
| recon | DE | +0 | [+0, +0] | – |  |  |

### key_violations (count)

| scope | effect | estimate | 95% CI | p | Lenth |  |
|---|---|---|---|---|---|---|
| orig | A | +0 | [+0, +0] | – |  |  |
| orig | D | +0 | [+0, +0] | – |  |  |
| orig | E | +0 | [+0, +0] | – |  |  |
| orig | AD | +0 | [+0, +0] | – |  |  |
| orig | AE | +0 | [+0, +0] | – |  |  |
| orig | DE | +0 | [+0, +0] | – |  |  |
| orig vs recon | A | +0 | [+0, +0] | – |  |  |
| orig vs recon | D | +0 | [+0, +0] | – |  |  |
| orig vs recon | E | +0 | [+0, +0] | – |  |  |
| orig vs recon | X | +0 | [+0, +0] | – |  |  |
| orig vs recon | AD | +0 | [+0, +0] | – |  |  |
| orig vs recon | AE | +0 | [+0, +0] | – |  |  |
| recon | A | +0 | [+0, +0] | – |  |  |
| recon | D | +0 | [+0, +0] | – |  |  |
| recon | E | +0 | [+0, +0] | – |  |  |
| recon | AD | +0 | [+0, +0] | – |  |  |
| recon | AE | +0 | [+0, +0] | – |  |  |
| recon | DE | +0 | [+0, +0] | – |  |  |

### drain (ms)

| scope | effect | estimate | 95% CI | p | Lenth |  |
|---|---|---|---|---|---|---|
| orig | A | ×10.3 | [7.74, 13.8] | <1e-4 | ** | **sig** |
| orig | AE | ×1.46 | [1.1, 1.95] | 0.0111 |  |  |
| orig | D | ×1.36 | [1.02, 1.82] | 0.0358 |  |  |
| orig | AD | ×0.9 | [0.675, 1.2] | 0.46 |  |  |
| orig | E | ×0.908 | [0.681, 1.21] | 0.499 |  |  |
| orig | DE | ×0.918 | [0.689, 1.22] | 0.551 |  |  |
| orig vs recon | A | ×10.9 | [8.98, 13.3] | <1e-4 | ** | **sig** |
| orig vs recon | X | ×0.437 | [0.359, 0.531] | <1e-4 |  | **sig** |
| orig vs recon | AE | ×1.55 | [1.27, 1.88] | <1e-4 |  | **sig** |
| orig vs recon | DEX | ×1.51 | [1.24, 1.83] | <1e-4 |  | **sig** |
| orig vs recon | DE | ×1.38 | [1.14, 1.68] | 0.00152 |  | **sig** |
| orig vs recon | DX | ×0.729 | [0.6, 0.887] | 0.00204 |  | **sig** |
| recon | A | ×11.6 | [8.77, 15.3] | <1e-4 |  | **sig** |
| recon | DE | ×2.09 | [1.58, 2.75] | <1e-4 |  | **sig** |
| recon | ADE | ×1.66 | [1.26, 2.19] | 0.000747 |  | **sig** |
| recon | AE | ×1.64 | [1.24, 2.16] | 0.000982 |  | **sig** |
| recon | E | ×1.46 | [1.1, 1.92] | 0.0094 |  |  |
| recon | D | ×0.725 | [0.549, 0.957] | 0.0245 |  |  |

<details><summary>Half-normal plot: lat_p99, orig vs recon (15 contrasts)</summary>

```
 rank  z(half)  |effect|
    1    2.13     4.743  ██████████████████████████████ A **
    2    1.64     1.544  █████████▊                     E 
    3    1.38     1.136  ███████▏                       X 
    4    1.19     1.119  ███████▏                       DE 
    5    1.04     0.850  █████▍                         DEX 
    6    0.90     0.806  █████▏                         EX 
    7    0.78     0.467  ███                            AEX 
    8    0.67     0.464  ███                            AE 
    9    0.57     0.357  ██▎                            D 
   10    0.48     0.292  █▉                             AD 
   11    0.39     0.193  █▎                             AX 
   12    0.30     0.188  █▏                             ADEX 
   13    0.21     0.084  ▌                              ADX 
   14    0.13     0.070  ▌                              DX 
   15    0.04     0.002                                 ADE 
```

</details>

<details><summary>Half-normal plot: throughput, orig vs recon (15 contrasts)</summary>

```
 rank  z(half)  |effect|
    1    2.13     0.951  ██████████████████████████████ X **
    2    1.64     0.853  ██████████████████████████▉    D *
    3    1.38     0.573  ██████████████████▏            E *
    4    1.19     0.534  ████████████████▉              DE *
    5    1.04     0.281  ████████▉                      DX 
    6    0.90     0.258  ████████▏                      AE 
    7    0.78     0.185  █████▉                         A 
    8    0.67     0.179  █████▋                         AD 
    9    0.57     0.121  ███▉                           ADE 
   10    0.48     0.110  ███▌                           ADEX 
   11    0.39     0.101  ███▏                           AX 
   12    0.30     0.050  █▋                             ADX 
   13    0.21     0.039  █▎                             EX 
   14    0.13     0.026  ▉                              DEX 
   15    0.04     0.004  ▏                              AEX 
```

</details>

## Per-point medians

| pt | levels ADE | orig ops/s | orig p99 | recon ops/s | recon p99 |
|---|---|---|---|---|---|
| 0 | `---` | 809k | 65.3µs | 1.28M | 48.9µs |
| 1 | `+--` | 461k | 1.58ms | 808k | 522µs |
| 2 | `-+-` | 1.28M | 34.6µs | 3.64M | 8.77µs |
| 3 | `++-` | 1.37M | 1.35ms | 2.68M | 119µs |
| 4 | `--+` | 582k | 81.4µs | 1M | 38.1µs |
| 5 | `+-+` | 649k | 2.51ms | 924k | 1.11ms |
| 6 | `-++` | 584k | 79.4µs | 1.43M | 74.2µs |
| 7 | `+++` | 665k | 2.57ms | 1.51M | 4.95ms |

