# DoE full: orig=orig vs recon=recon

`20261010-042221-full-orig-recon` · 2026-10-10T04:22:21+05:30 · go1.27.1 windows/amd64 · 20 CPUs (GOMAXPROCS 20) · 80 trials (0 failed) in 109s · 5 reps · warm-up 300ms + window 1000ms · seed 1

Command: `go run ./lab/cmd/doe -design full -factors A,D,E -reps 5 -dur 1s -warmup 300ms -q -out lab/results/r2/reject-realistic`

## Verdict

X is the implementation factor (−1 baseline, +1 candidate), crossed with every design run, so its main effect is the candidate's change averaged over the whole design. Log-scale responses show the ratio candidate/baseline of geometric means; raw responses show the difference. Significance is Bonferroni-corrected over all contrasts of a response; ratios within 5% count as *same*.

### Key responses

| response | arms | baseline | candidate | change [95% CI] | p | verdict | strongest X×factor |
|---|---|---|---|---|---|---|---|
| throughput | orig→recon | 804k | 1.46M | ×1.81 [1.79, 1.83] | <1e-4 | **better** | AX (×0.874) |
| goodput | orig→recon | 750k | 1.08M | ×1.44 [1.42, 1.46] | <1e-4 | **better** | AX (×0.765) |
| lat_p50 | orig→recon | 9.3µs | 2.81µs | ×0.302 [0.296, 0.309] | <1e-4 | **better** | EX (×0.545) |
| lat_p99 | orig→recon | 330µs | 124µs | ×0.377 [0.36, 0.394] | <1e-4 | **better** | EX (×1.55) |
| lat_p999 | orig→recon | 1.15ms | 253µs | ×0.22 [0.216, 0.225] | <1e-4 | **better** | EX (×1.6) |
| wait_p99 | orig→recon | 261µs | 101µs | ×0.389 [0.376, 0.403] | <1e-4 | **better** | EX (×1.63) |
| allocs_op | orig→recon | 11.7 | 0.704 | -11 [-11, -11] | <1e-4 | **better** | EX (-1.12) |
| bytes_op | orig→recon | 524 | 45.7 | -478 [-479, -478] | <1e-4 | **better** | EX (-74.1) |
| peak_goroutines | orig→recon | 1091 | 0 | -1.09e+03 [-1.32e+03, -861] | <1e-4 | **better** | AX (+394) |
| err_rate | orig→recon | 5.53% | 18.98% | +0.135 [+0.132, +0.137] | <1e-4 | **worse** | DX (+0.135) |
| cap_violations | orig→recon | 0 | 0 | +0 [+0, +0] | – | same | – |
| key_violations | orig→recon | 0 | 0 | +0 [+0, +0] | – | same | – |
| drain | orig→recon | 94.4µs | 42.8µs | ×0.453 [0.353, 0.581] | <1e-4 | **better** | – |

<details><summary>All other responses</summary>

| response | arms | baseline | candidate | change [95% CI] | p | verdict | strongest X×factor |
|---|---|---|---|---|---|---|---|
| lat_p90 | orig→recon | 67.9µs | 43.1µs | ×0.635 [0.621, 0.649] | <1e-4 | **better** | AX (×1.12) |
| lat_max | orig→recon | 7.02ms | 3.12ms | ×0.444 [0.34, 0.58] | <1e-4 | **better** | AX (×1.79) |
| wait_p50 | orig→recon | 4.72µs | 1.24µs | ×0.262 [0.254, 0.271] | <1e-4 | **better** | EX (×0.624) |
| exec_p50 | orig→recon | 1.25µs | 420ns | ×0.337 [0.331, 0.343] | <1e-4 | **better** | EX (×0.663) |
| exec_p99 | orig→recon | 108µs | 36.1µs | ×0.335 [0.321, 0.349] | <1e-4 | **better** | DX (×1.97) |
| peak_heap | orig→recon | 8.21 | 7.63 | ×0.93 [0.871, 0.993] | 0.0316 | same | – |
| gc_per_s | orig→recon | 122 | 12.1 | -110 [-118, -103] | <1e-4 | **better** | AX (+33) |
| retries_op | orig→recon | 0 | 0 | +0 [+0, +0] | – | same | – |
| fairness | orig→recon | 1 | 0.985 | -0.0145 [-0.0251, -0.00403] | 0.00744 | same | – |
| stats_lag | orig→recon | 0 | 0 | +0 [+0, +0] | – | same | – |
| residual_goroutines | orig→recon | 0 | 0 | +0 [+0, +0] | – | same | – |

</details>

## Noise

Replicate error of a single trial (pooled within-cell SD) as a coefficient of variation: the ±1σ spread between identical trials.

| response | orig | recon |
|---|---|---|
| throughput | 2.9% | 2.0% |
| goodput | 3.6% | 2.2% |
| lat_p50 | 3.3% | 5.9% |
| lat_p90 | 6.4% | 3.7% |
| lat_p99 | 15.0% | 3.4% |
| lat_p999 | 4.6% | 5.2% |
| lat_max | 66.7% | 96.7% |
| wait_p50 | 2.7% | 10.1% |
| wait_p99 | 10.9% | 4.0% |
| exec_p50 | 3.0% | 4.9% |
| exec_p99 | 13.5% | 4.3% |
| allocs_op | 0.3% | 1.3% |
| bytes_op | 0.2% | 1.5% |
| peak_goroutines | 66.9% | – |
| peak_heap | 14.1% | 17.4% |
| gc_per_s | 19.0% | 10.4% |
| err_rate | 15.2% | 2.0% |
| retries_op | – | – |
| fairness | 0.0% | 3.4% |
| cap_violations | – | – |
| key_violations | – | – |
| drain | 111.7% | 28.2% |
| stats_lag | – | – |
| residual_goroutines | – | – |

## Error classes

Share of all calls, over all trials of each arm. A class is the implementation-neutral error class, the outcome what `OutcomeOf` reports.

| class/outcome | orig | recon |
|---|---|---|
| overloaded/throttled | 9.555% | 19.940% |

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
| orig | E | ×0.66 | [0.648, 0.672] | <1e-4 |  | **sig** |
| orig | D | ×1.48 | [1.46, 1.51] | <1e-4 |  | **sig** |
| orig | DE | ×0.681 | [0.669, 0.694] | <1e-4 |  | **sig** |
| orig | AE | ×1.19 | [1.17, 1.21] | <1e-4 |  | **sig** |
| orig | ADE | ×0.849 | [0.834, 0.865] | <1e-4 |  | **sig** |
| orig | AD | ×1.17 | [1.15, 1.19] | <1e-4 |  | **sig** |
| orig vs recon | X | ×1.81 | [1.79, 1.83] | <1e-4 | * | **sig** |
| orig vs recon | D | ×1.62 | [1.6, 1.64] | <1e-4 | * | **sig** |
| orig vs recon | E | ×0.65 | [0.643, 0.657] | <1e-4 | * | **sig** |
| orig vs recon | DE | ×0.676 | [0.668, 0.683] | <1e-4 |  | **sig** |
| orig vs recon | A | ×0.808 | [0.8, 0.817] | <1e-4 |  | **sig** |
| orig vs recon | AE | ×1.15 | [1.14, 1.16] | <1e-4 |  | **sig** |
| recon | D | ×1.77 | [1.75, 1.79] | <1e-4 |  | **sig** |
| recon | E | ×0.64 | [0.632, 0.648] | <1e-4 |  | **sig** |
| recon | DE | ×0.67 | [0.662, 0.678] | <1e-4 |  | **sig** |
| recon | A | ×0.706 | [0.697, 0.715] | <1e-4 |  | **sig** |
| recon | AE | ×1.11 | [1.1, 1.12] | <1e-4 |  | **sig** |
| recon | ADE | ×0.914 | [0.902, 0.925] | <1e-4 |  | **sig** |

### goodput (ops/s)

| scope | effect | estimate | 95% CI | p | Lenth |  |
|---|---|---|---|---|---|---|
| orig | AE | ×1.34 | [1.31, 1.37] | <1e-4 |  | **sig** |
| orig | E | ×0.758 | [0.741, 0.776] | <1e-4 |  | **sig** |
| orig | D | ×1.29 | [1.26, 1.32] | <1e-4 |  | **sig** |
| orig | DE | ×0.783 | [0.765, 0.801] | <1e-4 |  | **sig** |
| orig | A | ×0.822 | [0.803, 0.841] | <1e-4 |  | **sig** |
| orig | ADE | ×0.955 | [0.934, 0.978] | 0.000312 |  | **sig** |
| orig vs recon | E | ×0.605 | [0.598, 0.614] | <1e-4 |  | **sig** |
| orig vs recon | A | ×0.629 | [0.62, 0.637] | <1e-4 |  | **sig** |
| orig vs recon | DE | ×0.629 | [0.621, 0.638] | <1e-4 |  | **sig** |
| orig vs recon | X | ×1.44 | [1.42, 1.46] | <1e-4 |  | **sig** |
| orig vs recon | AX | ×0.765 | [0.755, 0.775] | <1e-4 |  | **sig** |
| orig vs recon | ADX | ×0.784 | [0.774, 0.794] | <1e-4 |  | **sig** |
| recon | A | ×0.481 | [0.474, 0.487] | <1e-4 |  | **sig** |
| recon | E | ×0.484 | [0.477, 0.49] | <1e-4 |  | **sig** |
| recon | DE | ×0.506 | [0.499, 0.513] | <1e-4 |  | **sig** |
| recon | AD | ×0.637 | [0.629, 0.646] | <1e-4 |  | **sig** |
| recon | ADE | ×0.71 | [0.7, 0.72] | <1e-4 |  | **sig** |
| recon | AE | ×0.862 | [0.851, 0.875] | <1e-4 |  | **sig** |

### lat_p50 (ns)

| scope | effect | estimate | 95% CI | p | Lenth |  |
|---|---|---|---|---|---|---|
| orig | A | ×5.9 | [5.78, 6.02] | <1e-4 |  | **sig** |
| orig | D | ×0.259 | [0.254, 0.265] | <1e-4 |  | **sig** |
| orig | DE | ×3.77 | [3.7, 3.85] | <1e-4 |  | **sig** |
| orig | AD | ×0.282 | [0.276, 0.288] | <1e-4 |  | **sig** |
| orig | ADE | ×3.47 | [3.4, 3.54] | <1e-4 |  | **sig** |
| orig | AE | ×0.483 | [0.473, 0.493] | <1e-4 |  | **sig** |
| orig vs recon | DE | ×6.41 | [6.28, 6.54] | <1e-4 |  | **sig** |
| orig vs recon | A | ×5.03 | [4.92, 5.13] | <1e-4 |  | **sig** |
| orig vs recon | ADE | ×3.42 | [3.35, 3.5] | <1e-4 |  | **sig** |
| orig vs recon | X | ×0.302 | [0.296, 0.309] | <1e-4 |  | **sig** |
| orig vs recon | AD | ×0.306 | [0.3, 0.312] | <1e-4 |  | **sig** |
| orig vs recon | D | ×0.327 | [0.321, 0.334] | <1e-4 |  | **sig** |
| recon | DE | ×10.9 | [10.5, 11.3] | <1e-4 |  | **sig** |
| recon | A | ×4.28 | [4.13, 4.44] | <1e-4 |  | **sig** |
| recon | E | ×0.277 | [0.267, 0.287] | <1e-4 |  | **sig** |
| recon | ADE | ×3.38 | [3.26, 3.51] | <1e-4 |  | **sig** |
| recon | AD | ×0.332 | [0.32, 0.345] | <1e-4 |  | **sig** |
| recon | D | ×0.413 | [0.398, 0.429] | <1e-4 |  | **sig** |

### lat_p99 (ns)

| scope | effect | estimate | 95% CI | p | Lenth |  |
|---|---|---|---|---|---|---|
| orig | A | ×31.2 | [28.5, 34.2] | <1e-4 | ** | **sig** |
| orig | E | ×1.76 | [1.61, 1.93] | <1e-4 |  | **sig** |
| orig | AD | ×1.22 | [1.11, 1.33] | <1e-4 |  | **sig** |
| orig | D | ×0.822 | [0.751, 0.899] | 0.000102 |  | **sig** |
| orig | DE | ×1.19 | [1.09, 1.31] | 0.000336 |  | **sig** |
| orig | ADE | ×0.877 | [0.801, 0.959] | 0.00551 |  | **sig** |
| orig vs recon | A | ×23.6 | [22.5, 24.7] | <1e-4 | ** | **sig** |
| orig vs recon | E | ×2.73 | [2.61, 2.86] | <1e-4 |  | **sig** |
| orig vs recon | X | ×0.377 | [0.36, 0.394] | <1e-4 |  | **sig** |
| orig vs recon | DE | ×2.06 | [1.97, 2.16] | <1e-4 |  | **sig** |
| orig vs recon | DEX | ×1.73 | [1.65, 1.81] | <1e-4 |  | **sig** |
| orig vs recon | EX | ×1.55 | [1.48, 1.62] | <1e-4 |  | **sig** |
| recon | A | ×17.8 | [17.4, 18.2] | <1e-4 |  | **sig** |
| recon | E | ×4.23 | [4.14, 4.32] | <1e-4 |  | **sig** |
| recon | DE | ×3.57 | [3.49, 3.64] | <1e-4 |  | **sig** |
| recon | AE | ×1.81 | [1.77, 1.85] | <1e-4 |  | **sig** |
| recon | D | ×0.595 | [0.582, 0.608] | <1e-4 |  | **sig** |
| recon | ADE | ×1.1 | [1.08, 1.13] | <1e-4 |  | **sig** |

### lat_p999 (ns)

| scope | effect | estimate | 95% CI | p | Lenth |  |
|---|---|---|---|---|---|---|
| orig | A | ×7.93 | [7.71, 8.16] | <1e-4 | * | **sig** |
| orig | AD | ×1.28 | [1.25, 1.32] | <1e-4 |  | **sig** |
| orig | D | ×1.28 | [1.25, 1.32] | <1e-4 |  | **sig** |
| orig | E | ×1.28 | [1.24, 1.31] | <1e-4 |  | **sig** |
| orig | ADE | ×0.802 | [0.779, 0.825] | <1e-4 |  | **sig** |
| orig | DE | ×0.816 | [0.793, 0.84] | <1e-4 |  | **sig** |
| orig vs recon | A | ×11.6 | [11.4, 11.9] | <1e-4 | ** | **sig** |
| orig vs recon | X | ×0.22 | [0.216, 0.225] | <1e-4 | * | **sig** |
| orig vs recon | E | ×2.04 | [2, 2.08] | <1e-4 |  | **sig** |
| orig vs recon | DEX | ×2.02 | [1.97, 2.06] | <1e-4 |  | **sig** |
| orig vs recon | DE | ×1.65 | [1.61, 1.68] | <1e-4 |  | **sig** |
| orig vs recon | EX | ×1.6 | [1.56, 1.63] | <1e-4 |  | **sig** |
| recon | A | ×17 | [16.5, 17.6] | <1e-4 | * | **sig** |
| recon | DE | ×3.32 | [3.21, 3.43] | <1e-4 |  | **sig** |
| recon | E | ×3.26 | [3.15, 3.36] | <1e-4 |  | **sig** |
| recon | AE | ×1.57 | [1.52, 1.62] | <1e-4 |  | **sig** |
| recon | D | ×0.673 | [0.651, 0.695] | <1e-4 |  | **sig** |
| recon | ADE | ×1.1 | [1.07, 1.14] | <1e-4 |  | **sig** |

### wait_p99 (ns)

| scope | effect | estimate | 95% CI | p | Lenth |  |
|---|---|---|---|---|---|---|
| orig | A | ×41 | [38.4, 43.9] | <1e-4 | ** | **sig** |
| orig | E | ×1.67 | [1.57, 1.79] | <1e-4 |  | **sig** |
| orig | DE | ×1.26 | [1.18, 1.35] | <1e-4 |  | **sig** |
| orig | D | ×0.816 | [0.763, 0.872] | <1e-4 |  | **sig** |
| orig | ADE | ×0.826 | [0.772, 0.883] | <1e-4 |  | **sig** |
| orig | AD | ×1.2 | [1.13, 1.29] | <1e-4 |  | **sig** |
| orig vs recon | A | ×28 | [27, 29] | <1e-4 | ** | **sig** |
| orig vs recon | E | ×2.73 | [2.64, 2.83] | <1e-4 |  | **sig** |
| orig vs recon | X | ×0.389 | [0.376, 0.403] | <1e-4 |  | **sig** |
| orig vs recon | DE | ×2.11 | [2.04, 2.19] | <1e-4 |  | **sig** |
| orig vs recon | DEX | ×1.67 | [1.62, 1.73] | <1e-4 |  | **sig** |
| orig vs recon | D | ×0.603 | [0.582, 0.625] | <1e-4 |  | **sig** |
| recon | A | ×19.1 | [18.6, 19.6] | <1e-4 |  | **sig** |
| recon | E | ×4.46 | [4.34, 4.57] | <1e-4 |  | **sig** |
| recon | DE | ×3.54 | [3.45, 3.63] | <1e-4 |  | **sig** |
| recon | D | ×0.446 | [0.434, 0.457] | <1e-4 |  | **sig** |
| recon | AE | ×1.83 | [1.79, 1.88] | <1e-4 |  | **sig** |
| recon | ADE | ×1.17 | [1.14, 1.2] | <1e-4 |  | **sig** |

### allocs_op (allocs)

| scope | effect | estimate | 95% CI | p | Lenth |  |
|---|---|---|---|---|---|---|
| orig | E | +1.95 | [+1.93, +1.98] | <1e-4 |  | **sig** |
| orig | AE | -1.11 | [-1.13, -1.09] | <1e-4 |  | **sig** |
| orig | D | -1.05 | [-1.07, -1.03] | <1e-4 |  | **sig** |
| orig | DE | +1.03 | [+1.01, +1.05] | <1e-4 |  | **sig** |
| orig | A | +0.689 | [+0.669, +0.71] | <1e-4 |  | **sig** |
| orig | AD | -0.367 | [-0.387, -0.346] | <1e-4 |  | **sig** |
| orig vs recon | X | -11 | [-11, -11] | <1e-4 | ** | **sig** |
| orig vs recon | EX | -1.12 | [-1.13, -1.11] | <1e-4 |  | **sig** |
| orig vs recon | D | -0.85 | [-0.86, -0.839] | <1e-4 |  | **sig** |
| orig vs recon | E | +0.838 | [+0.827, +0.848] | <1e-4 |  | **sig** |
| orig vs recon | DE | +0.772 | [+0.762, +0.783] | <1e-4 |  | **sig** |
| orig vs recon | AE | -0.647 | [-0.657, -0.636] | <1e-4 |  | **sig** |
| recon | D | -0.649 | [-0.655, -0.643] | <1e-4 |  | **sig** |
| recon | DE | +0.512 | [+0.506, +0.518] | <1e-4 |  | **sig** |
| recon | A | +0.489 | [+0.483, +0.495] | <1e-4 |  | **sig** |
| recon | ADE | +0.37 | [+0.364, +0.376] | <1e-4 |  | **sig** |
| recon | E | -0.279 | [-0.285, -0.273] | <1e-4 |  | **sig** |
| recon | AE | -0.183 | [-0.189, -0.177] | <1e-4 |  | **sig** |

### bytes_op (B)

| scope | effect | estimate | 95% CI | p | Lenth |  |
|---|---|---|---|---|---|---|
| orig | E | +119 | [+119, +120] | <1e-4 |  | **sig** |
| orig | D | -103 | [-104, -103] | <1e-4 |  | **sig** |
| orig | DE | +102 | [+102, +103] | <1e-4 |  | **sig** |
| orig | AE | -51.7 | [-52.4, -51] | <1e-4 |  | **sig** |
| orig | AD | -33.7 | [-34.4, -33] | <1e-4 |  | **sig** |
| orig | ADE | +33.1 | [+32.4, +33.7] | <1e-4 |  | **sig** |
| orig vs recon | X | -478 | [-479, -478] | <1e-4 | ** | **sig** |
| orig vs recon | D | -88.3 | [-88.7, -87.9] | <1e-4 |  | **sig** |
| orig vs recon | EX | -74.1 | [-74.5, -73.7] | <1e-4 |  | **sig** |
| orig vs recon | DE | +68.4 | [+68, +68.8] | <1e-4 |  | **sig** |
| orig vs recon | E | +45.2 | [+44.8, +45.6] | <1e-4 |  | **sig** |
| orig vs recon | AE | -35.8 | [-36.2, -35.4] | <1e-4 |  | **sig** |
| recon | D | -73.2 | [-73.6, -72.8] | <1e-4 |  | **sig** |
| recon | DE | +34.5 | [+34, +34.9] | <1e-4 |  | **sig** |
| recon | E | -28.8 | [-29.3, -28.4] | <1e-4 |  | **sig** |
| recon | A | +28.5 | [+28.1, +28.9] | <1e-4 |  | **sig** |
| recon | ADE | +24.4 | [+24, +24.8] | <1e-4 |  | **sig** |
| recon | AE | -19.9 | [-20.3, -19.5] | <1e-4 |  | **sig** |

### peak_goroutines (goroutines)

| scope | effect | estimate | 95% CI | p | Lenth |  |
|---|---|---|---|---|---|---|
| orig | DE | -1e+03 | [-1.47e+03, -535] | 0.000128 |  | **sig** |
| orig | A | -787 | [-1.26e+03, -318] | 0.00176 |  | **sig** |
| orig | E | -771 | [-1.24e+03, -301] | 0.00212 |  | **sig** |
| orig | D | +693 | [+223, +1.16e+03] | 0.00516 |  | **sig** |
| orig | ADE | +248 | [-222, +717] | 0.291 |  |  |
| orig | AE | +39.3 | [-431, +509] | 0.866 |  |  |
| orig vs recon | X | -1.09e+03 | [-1.32e+03, -861] | <1e-4 |  | **sig** |
| orig vs recon | DE | -502 | [-733, -272] | <1e-4 |  | **sig** |
| orig vs recon | DEX | +502 | [+272, +733] | <1e-4 |  | **sig** |
| orig vs recon | A | -394 | [-624, -163] | 0.00112 |  | **sig** |
| orig vs recon | AX | +394 | [+163, +624] | 0.00112 |  | **sig** |
| orig vs recon | E | -386 | [-616, -155] | 0.00139 |  | **sig** |
| recon | A | +0 | [+0, +0] | – |  |  |
| recon | D | +0 | [+0, +0] | – |  |  |
| recon | E | +0 | [+0, +0] | – |  |  |
| recon | AD | +0 | [+0, +0] | – |  |  |
| recon | AE | +0 | [+0, +0] | – |  |  |
| recon | DE | +0 | [+0, +0] | – |  |  |

### err_rate (fraction)

| scope | effect | estimate | 95% CI | p | Lenth |  |
|---|---|---|---|---|---|---|
| orig | D | +0.111 | [+0.105, +0.116] | <1e-4 |  | **sig** |
| orig | E | -0.111 | [-0.116, -0.105] | <1e-4 |  | **sig** |
| orig | DE | -0.111 | [-0.116, -0.105] | <1e-4 |  | **sig** |
| orig | A | +0.0902 | [+0.0847, +0.0956] | <1e-4 |  | **sig** |
| orig | AD | +0.0902 | [+0.0847, +0.0956] | <1e-4 |  | **sig** |
| orig | AE | -0.0901 | [-0.0955, -0.0847] | <1e-4 |  | **sig** |
| orig vs recon | D | +0.245 | [+0.242, +0.248] | <1e-4 |  | **sig** |
| orig vs recon | A | +0.14 | [+0.137, +0.143] | <1e-4 |  | **sig** |
| orig vs recon | AD | +0.14 | [+0.137, +0.143] | <1e-4 |  | **sig** |
| orig vs recon | X | +0.135 | [+0.132, +0.137] | <1e-4 |  | **sig** |
| orig vs recon | DX | +0.135 | [+0.132, +0.137] | <1e-4 |  | **sig** |
| orig vs recon | EX | +0.113 | [+0.11, +0.116] | <1e-4 |  | **sig** |
| recon | D | +0.38 | [+0.377, +0.382] | <1e-4 |  | **sig** |
| recon | A | +0.19 | [+0.188, +0.193] | <1e-4 |  | **sig** |
| recon | AD | +0.19 | [+0.188, +0.193] | <1e-4 |  | **sig** |
| recon | E | +0.116 | [+0.114, +0.119] | <1e-4 |  | **sig** |
| recon | DE | +0.116 | [+0.114, +0.119] | <1e-4 |  | **sig** |
| recon | AE | +0.0935 | [+0.0911, +0.0959] | <1e-4 |  | **sig** |

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
| orig | A | ×11.9 | [7.32, 19.3] | <1e-4 | * | **sig** |
| orig | AE | ×1.87 | [1.15, 3.03] | 0.0127 |  |  |
| orig | D | ×1.54 | [0.952, 2.5] | 0.0768 |  |  |
| orig | AD | ×0.741 | [0.457, 1.2] | 0.216 |  |  |
| orig | ADE | ×1.34 | [0.825, 2.17] | 0.23 |  |  |
| orig | DE | ×0.753 | [0.464, 1.22] | 0.241 |  |  |
| orig vs recon | A | ×13.5 | [10.5, 17.4] | <1e-4 | ** | **sig** |
| orig vs recon | X | ×0.453 | [0.353, 0.581] | <1e-4 |  | **sig** |
| orig vs recon | DEX | ×1.47 | [1.14, 1.88] | 0.0031 |  | **sig** |
| orig vs recon | AE | ×1.44 | [1.12, 1.85] | 0.00495 |  |  |
| orig vs recon | D | ×1.35 | [1.05, 1.73] | 0.0188 |  |  |
| orig vs recon | AEX | ×0.769 | [0.599, 0.987] | 0.0397 |  |  |
| recon | A | ×15.4 | [13.1, 18.1] | <1e-4 | * | **sig** |
| recon | DE | ×1.62 | [1.38, 1.91] | <1e-4 |  | **sig** |
| recon | E | ×1.39 | [1.18, 1.63] | 0.000227 |  | **sig** |
| recon | ADE | ×1.26 | [1.08, 1.48] | 0.00555 |  | **sig** |
| recon | AD | ×1.2 | [1.03, 1.41] | 0.0242 |  |  |
| recon | D | ×1.18 | [1.01, 1.39] | 0.0391 |  |  |

<details><summary>Half-normal plot: lat_p99, orig vs recon (15 contrasts)</summary>

```
 rank  z(half)  |effect|
    1    2.13     4.559  ██████████████████████████████ A **
    2    1.64     1.448  █████████▌                     E 
    3    1.38     1.408  █████████▎                     X 
    4    1.19     1.045  ██████▉                        DE 
    5    1.04     0.789  █████▎                         DEX 
    6    0.90     0.633  ████▏                          EX 
    7    0.78     0.516  ███▍                           D 
    8    0.67     0.445  ██▉                            AE 
    9    0.57     0.411  ██▊                            AEX 
   10    0.48     0.405  ██▋                            AX 
   11    0.39     0.233  █▌                             DX 
   12    0.30     0.167  █▏                             ADEX 
   13    0.21     0.144  █                              ADX 
   14    0.13     0.140  ▉                              AD 
   15    0.04     0.023  ▏                              ADE 
```

</details>

<details><summary>Half-normal plot: throughput, orig vs recon (15 contrasts)</summary>

```
 rank  z(half)  |effect|
    1    2.13     0.856  ██████████████████████████████ X *
    2    1.64     0.696  ████████████████████████▍      D *
    3    1.38     0.622  █████████████████████▊         E *
    4    1.19     0.566  ███████████████████▉           DE 
    5    1.04     0.307  ██████████▊                    A 
    6    0.90     0.202  ███████▏                       AE 
    7    0.78     0.195  ██████▉                        AX 
    8    0.67     0.183  ██████▍                        ADE 
    9    0.57     0.159  █████▋                         ADX 
   10    0.48     0.127  ████▌                          DX 
   11    0.39     0.064  ██▎                            AD 
   12    0.30     0.053  █▉                             ADEX 
   13    0.21     0.052  █▊                             AEX 
   14    0.13     0.022  ▊                              EX 
   15    0.04     0.012  ▍                              DEX 
```

</details>

## Per-point medians

| pt | levels ADE | orig ops/s | orig p99 | recon ops/s | recon p99 |
|---|---|---|---|---|---|
| 0 | `---` | 911k | 57.6µs | 1.43M | 44.8µs |
| 1 | `+--` | 509k | 1.38ms | 891k | 481µs |
| 2 | `-+-` | 1.43M | 30.8µs | 3.68M | 8.26µs |
| 3 | `++-` | 1.53M | 1.34ms | 2.4M | 73.2µs |
| 4 | `--+` | 612k | 71.2µs | 1.12M | 32.6µs |
| 5 | `+-+` | 678k | 2.44ms | 1.02M | 954µs |
| 6 | `-++` | 624k | 74.2µs | 1.56M | 61.2µs |
| 7 | `+++` | 685k | 2.51ms | 1.04M | 2.21ms |

