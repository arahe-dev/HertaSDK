# DoE full: orig=orig vs recon=recon

`20261010-020929-full-orig-recon` · 2026-10-10T02:09:29+05:30 · go1.27.1 windows/amd64 · 20 CPUs (GOMAXPROCS 20) · 80 trials (0 failed) in 110s · 5 reps · warm-up 300ms + window 1000ms · seed 1

Command: `go run ./lab/cmd/doe -design full -factors A,D,E -reps 5 -dur 1s -warmup 300ms -spin -q -out lab/results/r1/reject-interaction-spin`

## Verdict

X is the implementation factor (−1 baseline, +1 candidate), crossed with every design run, so its main effect is the candidate's change averaged over the whole design. Log-scale responses show the ratio candidate/baseline of geometric means; raw responses show the difference. Significance is Bonferroni-corrected over all contrasts of a response; ratios within 5% count as *same*.

### Key responses

| response | arms | baseline | candidate | change [95% CI] | p | verdict | strongest X×factor |
|---|---|---|---|---|---|---|---|
| throughput | orig→recon | 844k | 1.5M | ×1.78 [1.72, 1.83] | <1e-4 | **better** | DX (×1.17) |
| goodput | orig→recon | 613k | 1.01M | ×1.65 [1.57, 1.74] | <1e-4 | **better** | EX (×0.646) |
| lat_p50 | orig→recon | 9.32µs | 3.03µs | ×0.325 [0.307, 0.343] | <1e-4 | **better** | EX (×0.529) |
| lat_p99 | orig→recon | 264µs | 149µs | ×0.565 [0.533, 0.599] | <1e-4 | **better** | EX (×1.65) |
| lat_p999 | orig→recon | 1.23ms | 367µs | ×0.299 [0.26, 0.344] | <1e-4 | **better** | EX (×2) |
| wait_p99 | orig→recon | 176µs | 123µs | ×0.698 [0.636, 0.767] | <1e-4 | **better** | EX (×1.36) |
| allocs_op | orig→recon | 11.9 | 0.746 | -11.2 [-11.2, -11.1] | <1e-4 | **better** | EX (-0.956) |
| bytes_op | orig→recon | 528 | 46.9 | -481 [-482, -480] | <1e-4 | **better** | EX (-72.7) |
| peak_goroutines | orig→recon | 5817 | 0 | -5.82e+03 [-6.46e+03, -5.17e+03] | <1e-4 | **better** | EX (+5.44e+03) |
| err_rate | orig→recon | 12.17% | 22.76% | +0.106 [+0.101, +0.11] | <1e-4 | **worse** | EX (+0.158) |
| cap_violations | orig→recon | 0 | 0 | +0 [+0, +0] | – | same | – |
| key_violations | orig→recon | 0 | 0 | +0 [+0, +0] | – | same | – |
| drain | orig→recon | 140µs | 44.9µs | ×0.32 [0.252, 0.407] | <1e-4 | **better** | EX (×1.95) |

<details><summary>All other responses</summary>

| response | arms | baseline | candidate | change [95% CI] | p | verdict | strongest X×factor |
|---|---|---|---|---|---|---|---|
| lat_p90 | orig→recon | 62.8µs | 53.2µs | ×0.847 [0.802, 0.895] | <1e-4 | **better** | AX (×1.45) |
| lat_max | orig→recon | 13ms | 5.22ms | ×0.402 [0.318, 0.508] | <1e-4 | **better** | DX (×0.553) |
| wait_p50 | orig→recon | 4.73µs | 1.37µs | ×0.289 [0.276, 0.304] | <1e-4 | **better** | EX (×0.598) |
| exec_p50 | orig→recon | 1.22µs | 470ns | ×0.387 [0.368, 0.406] | <1e-4 | **better** | DX (×1.63) |
| exec_p99 | orig→recon | 70.3µs | 44.3µs | ×0.631 [0.599, 0.665] | <1e-4 | **better** | DX (×3.04) |
| peak_heap | orig→recon | 43.8 | 36.7 | ×0.839 [0.631, 1.12] | 0.223 | same | – |
| gc_per_s | orig→recon | 31.3 | 3.15 | -28.1 [-42.7, -13.6] | 0.00026 | **better** | – |
| retries_op | orig→recon | 0 | 0 | +0 [+0, +0] | – | same | – |
| fairness | orig→recon | 0.978 | 0.989 | +0.0111 [-0.00579, +0.028] | 0.194 | same | – |
| stats_lag | orig→recon | 0 | 0 | +0 [+0, +0] | – | same | – |
| residual_goroutines | orig→recon | 0 | 0 | +0 [+0, +0] | – | same | – |

</details>

## Noise

Replicate error of a single trial (pooled within-cell SD) as a coefficient of variation: the ±1σ spread between identical trials.

| response | orig | recon |
|---|---|---|
| throughput | 8.6% | 5.2% |
| goodput | 15.7% | 6.8% |
| lat_p50 | 14.1% | 12.4% |
| lat_p90 | 11.5% | 14.5% |
| lat_p99 | 18.7% | 7.0% |
| lat_p999 | 49.4% | 20.0% |
| lat_max | 44.0% | 91.1% |
| wait_p50 | 10.2% | 12.3% |
| wait_p99 | 33.2% | 8.1% |
| exec_p50 | 12.6% | 10.8% |
| exec_p99 | 15.0% | 8.8% |
| allocs_op | 0.6% | 2.1% |
| bytes_op | 0.6% | 2.2% |
| peak_goroutines | 35.2% | – |
| peak_heap | 73.6% | 103.9% |
| gc_per_s | 146.5% | 131.7% |
| err_rate | 9.2% | 4.0% |
| retries_op | – | – |
| fairness | 2.6% | 4.8% |
| cap_violations | – | – |
| key_violations | – | – |
| drain | 104.0% | 30.2% |
| stats_lag | – | – |
| residual_goroutines | – | – |

## Error classes

Share of all calls, over all trials of each arm. A class is the implementation-neutral error class, the outcome what `OutcomeOf` reports.

| class/outcome | orig | recon |
|---|---|---|
| overloaded/throttled | 23.031% | 28.639% |

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
| orig | D | ×1.6 | [1.52, 1.69] | <1e-4 |  | **sig** |
| orig | DE | ×0.644 | [0.611, 0.68] | <1e-4 |  | **sig** |
| orig | E | ×0.651 | [0.617, 0.687] | <1e-4 |  | **sig** |
| orig | AD | ×1.22 | [1.16, 1.29] | <1e-4 |  | **sig** |
| orig | ADE | ×0.835 | [0.792, 0.881] | <1e-4 |  | **sig** |
| orig | AE | ×1.18 | [1.12, 1.24] | <1e-4 |  | **sig** |
| orig vs recon | D | ×1.87 | [1.81, 1.93] | <1e-4 | * | **sig** |
| orig vs recon | X | ×1.78 | [1.72, 1.83] | <1e-4 | * | **sig** |
| orig vs recon | E | ×0.653 | [0.633, 0.673] | <1e-4 | * | **sig** |
| orig vs recon | DE | ×0.674 | [0.654, 0.695] | <1e-4 | * | **sig** |
| orig vs recon | AE | ×1.18 | [1.15, 1.22] | <1e-4 |  | **sig** |
| orig vs recon | DX | ×1.17 | [1.13, 1.2] | <1e-4 |  | **sig** |
| recon | D | ×2.18 | [2.11, 2.25] | <1e-4 |  | **sig** |
| recon | E | ×0.655 | [0.634, 0.677] | <1e-4 |  | **sig** |
| recon | DE | ×0.705 | [0.682, 0.728] | <1e-4 |  | **sig** |
| recon | A | ×0.807 | [0.781, 0.833] | <1e-4 |  | **sig** |
| recon | AE | ×1.18 | [1.14, 1.22] | <1e-4 |  | **sig** |
| recon | AD | ×1.08 | [1.05, 1.12] | <1e-4 |  | **sig** |

### goodput (ops/s)

| scope | effect | estimate | 95% CI | p | Lenth |  |
|---|---|---|---|---|---|---|
| orig | AE | ×2.16 | [1.97, 2.37] | <1e-4 |  | **sig** |
| orig | A | ×0.506 | [0.461, 0.556] | <1e-4 |  | **sig** |
| orig | ADE | ×1.53 | [1.39, 1.68] | <1e-4 |  | **sig** |
| orig | AD | ×0.669 | [0.609, 0.734] | <1e-4 |  | **sig** |
| orig | E | ×1.23 | [1.12, 1.35] | <1e-4 |  | **sig** |
| orig | DE | ×1.22 | [1.11, 1.34] | 0.000159 |  | **sig** |
| orig vs recon | A | ×0.483 | [0.459, 0.508] | <1e-4 |  | **sig** |
| orig vs recon | X | ×1.65 | [1.57, 1.74] | <1e-4 |  | **sig** |
| orig vs recon | AD | ×0.644 | [0.612, 0.677] | <1e-4 |  | **sig** |
| orig vs recon | EX | ×0.646 | [0.614, 0.679] | <1e-4 |  | **sig** |
| orig vs recon | AEX | ×0.661 | [0.628, 0.695] | <1e-4 |  | **sig** |
| orig vs recon | DEX | ×0.673 | [0.64, 0.708] | <1e-4 |  | **sig** |
| recon | A | ×0.461 | [0.442, 0.481] | <1e-4 |  | **sig** |
| recon | E | ×0.513 | [0.492, 0.535] | <1e-4 |  | **sig** |
| recon | DE | ×0.552 | [0.529, 0.575] | <1e-4 |  | **sig** |
| recon | AD | ×0.619 | [0.594, 0.646] | <1e-4 |  | **sig** |
| recon | ADE | ×0.789 | [0.756, 0.822] | <1e-4 |  | **sig** |
| recon | AE | ×0.944 | [0.905, 0.984] | 0.00844 |  |  |

### lat_p50 (ns)

| scope | effect | estimate | 95% CI | p | Lenth |  |
|---|---|---|---|---|---|---|
| orig | A | ×5.62 | [5.16, 6.12] | <1e-4 |  | **sig** |
| orig | D | ×0.244 | [0.224, 0.265] | <1e-4 |  | **sig** |
| orig | DE | ×3.94 | [3.62, 4.29] | <1e-4 |  | **sig** |
| orig | AD | ×0.271 | [0.249, 0.295] | <1e-4 |  | **sig** |
| orig | ADE | ×3.58 | [3.28, 3.89] | <1e-4 |  | **sig** |
| orig | AE | ×0.504 | [0.463, 0.549] | <1e-4 |  | **sig** |
| orig vs recon | DE | ×6.26 | [5.92, 6.62] | <1e-4 |  | **sig** |
| orig vs recon | A | ×4.94 | [4.67, 5.22] | <1e-4 |  | **sig** |
| orig vs recon | AD | ×0.292 | [0.276, 0.309] | <1e-4 |  | **sig** |
| orig vs recon | ADE | ×3.39 | [3.21, 3.58] | <1e-4 |  | **sig** |
| orig vs recon | D | ×0.312 | [0.295, 0.33] | <1e-4 |  | **sig** |
| orig vs recon | X | ×0.325 | [0.307, 0.343] | <1e-4 |  | **sig** |
| recon | DE | ×9.96 | [9.23, 10.7] | <1e-4 |  | **sig** |
| recon | A | ×4.34 | [4.02, 4.67] | <1e-4 |  | **sig** |
| recon | E | ×0.27 | [0.251, 0.292] | <1e-4 |  | **sig** |
| recon | ADE | ×3.21 | [2.98, 3.47] | <1e-4 |  | **sig** |
| recon | AD | ×0.314 | [0.291, 0.339] | <1e-4 |  | **sig** |
| recon | D | ×0.399 | [0.37, 0.43] | <1e-4 |  | **sig** |

### lat_p99 (ns)

| scope | effect | estimate | 95% CI | p | Lenth |  |
|---|---|---|---|---|---|---|
| orig | A | ×30.6 | [27.4, 34.2] | <1e-4 | * | **sig** |
| orig | E | ×1.88 | [1.68, 2.1] | <1e-4 |  | **sig** |
| orig | D | ×0.745 | [0.667, 0.832] | <1e-4 |  | **sig** |
| orig | DE | ×1.31 | [1.17, 1.46] | <1e-4 |  | **sig** |
| orig | AE | ×1.28 | [1.15, 1.44] | <1e-4 |  | **sig** |
| orig | AD | ×1.04 | [0.929, 1.16] | 0.499 |  |  |
| orig vs recon | A | ×26.4 | [24.9, 28] | <1e-4 | ** | **sig** |
| orig vs recon | E | ×3.1 | [2.93, 3.29] | <1e-4 | * | **sig** |
| orig vs recon | DE | ×2.2 | [2.07, 2.33] | <1e-4 |  | **sig** |
| orig vs recon | X | ×0.565 | [0.533, 0.599] | <1e-4 |  | **sig** |
| orig vs recon | DEX | ×1.68 | [1.59, 1.78] | <1e-4 |  | **sig** |
| orig vs recon | EX | ×1.65 | [1.56, 1.75] | <1e-4 |  | **sig** |
| recon | A | ×22.8 | [21.9, 23.8] | <1e-4 | * | **sig** |
| recon | E | ×5.12 | [4.9, 5.35] | <1e-4 |  | **sig** |
| recon | DE | ×3.7 | [3.54, 3.86] | <1e-4 |  | **sig** |
| recon | AE | ×2.01 | [1.93, 2.1] | <1e-4 |  | **sig** |
| recon | AD | ×1.32 | [1.27, 1.38] | <1e-4 |  | **sig** |
| recon | D | ×0.761 | [0.728, 0.794] | <1e-4 |  | **sig** |

### lat_p999 (ns)

| scope | effect | estimate | 95% CI | p | Lenth |  |
|---|---|---|---|---|---|---|
| orig | A | ×64.9 | [50.1, 84] | <1e-4 | * | **sig** |
| orig | ADE | ×0.597 | [0.461, 0.773] | 0.000295 |  | **sig** |
| orig | DE | ×0.599 | [0.462, 0.775] | 0.000312 |  | **sig** |
| orig | AD | ×1.63 | [1.26, 2.12] | 0.00051 |  | **sig** |
| orig | AE | ×0.664 | [0.513, 0.86] | 0.00293 |  | **sig** |
| orig | D | ×1.39 | [1.08, 1.8] | 0.0134 |  |  |
| orig vs recon | A | ×45.6 | [39.7, 52.5] | <1e-4 | ** | **sig** |
| orig vs recon | X | ×0.299 | [0.26, 0.344] | <1e-4 |  | **sig** |
| orig vs recon | DEX | ×2.9 | [2.52, 3.33] | <1e-4 |  | **sig** |
| orig vs recon | EX | ×2 | [1.74, 2.3] | <1e-4 |  | **sig** |
| orig vs recon | DE | ×1.74 | [1.51, 2] | <1e-4 |  | **sig** |
| orig vs recon | ADEX | ×1.69 | [1.47, 1.95] | <1e-4 |  | **sig** |
| recon | A | ×32.1 | [28.5, 36.1] | <1e-4 | * | **sig** |
| recon | DE | ×5.03 | [4.47, 5.66] | <1e-4 |  | **sig** |
| recon | E | ×2.87 | [2.55, 3.23] | <1e-4 |  | **sig** |
| recon | D | ×0.569 | [0.506, 0.64] | <1e-4 |  | **sig** |
| recon | ADE | ×1.71 | [1.52, 1.93] | <1e-4 |  | **sig** |
| recon | AE | ×1.29 | [1.15, 1.46] | <1e-4 |  | **sig** |

### wait_p99 (ns)

| scope | effect | estimate | 95% CI | p | Lenth |  |
|---|---|---|---|---|---|---|
| orig | A | ×25.8 | [21.5, 31.1] | <1e-4 |  | **sig** |
| orig | E | ×2.81 | [2.34, 3.38] | <1e-4 |  | **sig** |
| orig | DE | ×2 | [1.67, 2.41] | <1e-4 |  | **sig** |
| orig | D | ×0.502 | [0.417, 0.603] | <1e-4 |  | **sig** |
| orig | AE | ×1.67 | [1.39, 2.01] | <1e-4 |  | **sig** |
| orig | AD | ×0.689 | [0.572, 0.828] | 0.000252 |  | **sig** |
| orig vs recon | A | ×25.7 | [23.4, 28.2] | <1e-4 | ** | **sig** |
| orig vs recon | E | ×3.83 | [3.49, 4.21] | <1e-4 | * | **sig** |
| orig vs recon | DE | ×2.67 | [2.43, 2.94] | <1e-4 |  | **sig** |
| orig vs recon | D | ×0.541 | [0.492, 0.594] | <1e-4 |  | **sig** |
| orig vs recon | AE | ×1.81 | [1.64, 1.98] | <1e-4 |  | **sig** |
| orig vs recon | X | ×0.698 | [0.636, 0.767] | <1e-4 |  | **sig** |
| recon | A | ×25.6 | [24.3, 26.9] | <1e-4 |  | **sig** |
| recon | E | ×5.22 | [4.96, 5.48] | <1e-4 |  | **sig** |
| recon | DE | ×3.56 | [3.39, 3.74] | <1e-4 |  | **sig** |
| recon | AE | ×1.95 | [1.86, 2.05] | <1e-4 |  | **sig** |
| recon | D | ×0.583 | [0.555, 0.613] | <1e-4 |  | **sig** |
| recon | AD | ×1.34 | [1.27, 1.41] | <1e-4 |  | **sig** |

### allocs_op (allocs)

| scope | effect | estimate | 95% CI | p | Lenth |  |
|---|---|---|---|---|---|---|
| orig | E | +1.58 | [+1.54, +1.63] | <1e-4 |  | **sig** |
| orig | AE | -1.43 | [-1.47, -1.39] | <1e-4 |  | **sig** |
| orig | A | +1.09 | [+1.05, +1.14] | <1e-4 |  | **sig** |
| orig | D | -0.665 | [-0.708, -0.623] | <1e-4 |  | **sig** |
| orig | DE | +0.637 | [+0.595, +0.679] | <1e-4 |  | **sig** |
| orig | ADE | -0.0205 | [-0.0629, +0.0219] | 0.332 |  |  |
| orig vs recon | X | -11.2 | [-11.2, -11.1] | <1e-4 | ** | **sig** |
| orig vs recon | EX | -0.956 | [-0.977, -0.935] | <1e-4 |  | **sig** |
| orig vs recon | A | +0.821 | [+0.8, +0.843] | <1e-4 |  | **sig** |
| orig vs recon | AE | -0.821 | [-0.843, -0.8] | <1e-4 |  | **sig** |
| orig vs recon | E | +0.627 | [+0.606, +0.649] | <1e-4 |  | **sig** |
| orig vs recon | D | -0.623 | [-0.645, -0.602] | <1e-4 |  | **sig** |
| recon | D | -0.581 | [-0.591, -0.571] | <1e-4 |  | **sig** |
| recon | A | +0.55 | [+0.54, +0.56] | <1e-4 |  | **sig** |
| recon | DE | +0.472 | [+0.462, +0.482] | <1e-4 |  | **sig** |
| recon | E | -0.329 | [-0.339, -0.319] | <1e-4 |  | **sig** |
| recon | ADE | +0.324 | [+0.315, +0.334] | <1e-4 |  | **sig** |
| recon | AE | -0.212 | [-0.222, -0.202] | <1e-4 |  | **sig** |

### bytes_op (B)

| scope | effect | estimate | 95% CI | p | Lenth |  |
|---|---|---|---|---|---|---|
| orig | E | +115 | [+113, +117] | <1e-4 |  | **sig** |
| orig | D | -98.6 | [-101, -96.5] | <1e-4 |  | **sig** |
| orig | DE | +96.9 | [+94.9, +99] | <1e-4 |  | **sig** |
| orig | AE | -54.1 | [-56.1, -52] | <1e-4 |  | **sig** |
| orig | A | +36.6 | [+34.5, +38.7] | <1e-4 |  | **sig** |
| orig | AD | -27.4 | [-29.5, -25.3] | <1e-4 |  | **sig** |
| orig vs recon | X | -481 | [-482, -480] | <1e-4 | ** | **sig** |
| orig vs recon | D | -85.3 | [-86.3, -84.2] | <1e-4 |  | **sig** |
| orig vs recon | EX | -72.7 | [-73.7, -71.6] | <1e-4 |  | **sig** |
| orig vs recon | DE | +65.4 | [+64.3, +66.4] | <1e-4 |  | **sig** |
| orig vs recon | E | +42.4 | [+41.3, +43.4] | <1e-4 |  | **sig** |
| orig vs recon | AE | -37.1 | [-38.2, -36] | <1e-4 |  | **sig** |
| recon | D | -72 | [-72.6, -71.3] | <1e-4 |  | **sig** |
| recon | DE | +33.8 | [+33.1, +34.4] | <1e-4 |  | **sig** |
| recon | E | -30.3 | [-31, -29.6] | <1e-4 |  | **sig** |
| recon | A | +29.7 | [+29.1, +30.4] | <1e-4 |  | **sig** |
| recon | ADE | +22.8 | [+22.2, +23.5] | <1e-4 |  | **sig** |
| recon | AE | -20.1 | [-20.8, -19.5] | <1e-4 |  | **sig** |

### peak_goroutines (goroutines)

| scope | effect | estimate | 95% CI | p | Lenth |  |
|---|---|---|---|---|---|---|
| orig | AD | +1.09e+04 | [+9.56e+03, +1.22e+04] | <1e-4 |  | **sig** |
| orig | E | -1.09e+04 | [-1.22e+04, -9.56e+03] | <1e-4 |  | **sig** |
| orig | DE | -1.07e+04 | [-1.21e+04, -9.43e+03] | <1e-4 |  | **sig** |
| orig | ADE | -1.07e+04 | [-1.2e+04, -9.37e+03] | <1e-4 |  | **sig** |
| orig | D | +1.05e+04 | [+9.19e+03, +1.18e+04] | <1e-4 |  | **sig** |
| orig | AE | -1.05e+04 | [-1.18e+04, -9.17e+03] | <1e-4 |  | **sig** |
| orig vs recon | X | -5.82e+03 | [-6.46e+03, -5.17e+03] | <1e-4 |  | **sig** |
| orig vs recon | AD | +5.44e+03 | [+4.79e+03, +6.09e+03] | <1e-4 |  | **sig** |
| orig vs recon | ADX | -5.44e+03 | [-6.09e+03, -4.79e+03] | <1e-4 |  | **sig** |
| orig vs recon | E | -5.44e+03 | [-6.09e+03, -4.79e+03] | <1e-4 |  | **sig** |
| orig vs recon | EX | +5.44e+03 | [+4.79e+03, +6.09e+03] | <1e-4 |  | **sig** |
| orig vs recon | DE | -5.37e+03 | [-6.02e+03, -4.73e+03] | <1e-4 |  | **sig** |
| recon | A | +0 | [+0, +0] | – |  |  |
| recon | D | +0 | [+0, +0] | – |  |  |
| recon | E | +0 | [+0, +0] | – |  |  |
| recon | AD | +0 | [+0, +0] | – |  |  |
| recon | AE | +0 | [+0, +0] | – |  |  |
| recon | DE | +0 | [+0, +0] | – |  |  |

### err_rate (fraction)

| scope | effect | estimate | 95% CI | p | Lenth |  |
|---|---|---|---|---|---|---|
| orig | D | +0.243 | [+0.236, +0.251] | <1e-4 |  | **sig** |
| orig | E | -0.243 | [-0.251, -0.236] | <1e-4 |  | **sig** |
| orig | DE | -0.243 | [-0.251, -0.236] | <1e-4 |  | **sig** |
| orig | A | +0.213 | [+0.206, +0.22] | <1e-4 |  | **sig** |
| orig | AD | +0.213 | [+0.206, +0.22] | <1e-4 |  | **sig** |
| orig | AE | -0.213 | [-0.22, -0.206] | <1e-4 |  | **sig** |
| orig vs recon | D | +0.349 | [+0.345, +0.354] | <1e-4 |  | **sig** |
| orig vs recon | A | +0.234 | [+0.229, +0.238] | <1e-4 |  | **sig** |
| orig vs recon | AD | +0.234 | [+0.229, +0.238] | <1e-4 |  | **sig** |
| orig vs recon | EX | +0.158 | [+0.153, +0.162] | <1e-4 |  | **sig** |
| orig vs recon | DEX | +0.158 | [+0.153, +0.162] | <1e-4 |  | **sig** |
| orig vs recon | AEX | +0.135 | [+0.13, +0.139] | <1e-4 |  | **sig** |
| recon | D | +0.455 | [+0.449, +0.461] | <1e-4 | * | **sig** |
| recon | A | +0.254 | [+0.248, +0.26] | <1e-4 |  | **sig** |
| recon | AD | +0.254 | [+0.248, +0.26] | <1e-4 |  | **sig** |
| recon | E | +0.0717 | [+0.0658, +0.0776] | <1e-4 |  | **sig** |
| recon | DE | +0.0717 | [+0.0658, +0.0776] | <1e-4 |  | **sig** |
| recon | AE | +0.0562 | [+0.0503, +0.0621] | <1e-4 |  | **sig** |

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
| orig | A | ×21.5 | [13.6, 34] | <1e-4 |  | **sig** |
| orig | DE | ×0.353 | [0.223, 0.559] | <1e-4 |  | **sig** |
| orig | E | ×0.381 | [0.241, 0.603] | 0.000159 |  | **sig** |
| orig | D | ×2.28 | [1.44, 3.61] | 0.000918 |  | **sig** |
| orig | AE | ×0.585 | [0.37, 0.926] | 0.0236 |  |  |
| orig | AD | ×1.38 | [0.874, 2.19] | 0.16 |  |  |
| orig vs recon | A | ×14.5 | [11.4, 18.5] | <1e-4 | ** | **sig** |
| orig vs recon | X | ×0.32 | [0.252, 0.407] | <1e-4 |  | **sig** |
| orig vs recon | DEX | ×2.23 | [1.76, 2.84] | <1e-4 |  | **sig** |
| orig vs recon | EX | ×1.95 | [1.53, 2.48] | <1e-4 |  | **sig** |
| orig vs recon | D | ×1.69 | [1.33, 2.14] | <1e-4 |  | **sig** |
| orig vs recon | AEX | ×1.48 | [1.17, 1.89] | 0.00164 |  | **sig** |
| recon | A | ×9.85 | [8.31, 11.7] | <1e-4 | * | **sig** |
| recon | DE | ×1.76 | [1.49, 2.09] | <1e-4 |  | **sig** |
| recon | ADE | ×1.54 | [1.3, 1.82] | <1e-4 |  | **sig** |
| recon | E | ×1.45 | [1.22, 1.72] | 0.000106 |  | **sig** |
| recon | AE | ×1.29 | [1.09, 1.53] | 0.00463 |  | **sig** |
| recon | D | ×1.25 | [1.05, 1.48] | 0.0127 |  |  |

<details><summary>Half-normal plot: lat_p99, orig vs recon (15 contrasts)</summary>

```
 rank  z(half)  |effect|
    1    2.13     4.724  ██████████████████████████████ A **
    2    1.64     1.633  ██████████▍                    E *
    3    1.38     1.137  ███████▎                       DE 
    4    1.19     0.825  █████▎                         X 
    5    1.04     0.750  ████▊                          DEX 
    6    0.90     0.723  ████▋                          EX 
    7    0.78     0.685  ████▍                          AE 
    8    0.67     0.410  ██▋                            D 
    9    0.57     0.324  ██                             AEX 
   10    0.48     0.230  █▌                             AD 
   11    0.39     0.212  █▍                             AX 
   12    0.30     0.176  █▏                             ADX 
   13    0.21     0.100  ▋                              ADEX 
   14    0.13     0.078  ▌                              ADE 
   15    0.04     0.015  ▏                              DX 
```

</details>

<details><summary>Half-normal plot: throughput, orig vs recon (15 contrasts)</summary>

```
 rank  z(half)  |effect|
    1    2.13     0.903  ██████████████████████████████ D *
    2    1.64     0.830  ███████████████████████████▋   X *
    3    1.38     0.615  ████████████████████▍          E *
    4    1.19     0.569  ██████████████████▉            DE *
    5    1.04     0.241  ████████                       AE 
    6    0.90     0.222  ███████▍                       DX 
    7    0.78     0.209  ███████                        A 
    8    0.67     0.204  ██████▊                        AD 
    9    0.57     0.138  ████▋                          ADE 
   10    0.48     0.122  ████                           ADEX 
   11    0.39     0.101  ███▍                           AX 
   12    0.30     0.088  ██▉                            ADX 
   13    0.21     0.065  ██▏                            DEX 
   14    0.13     0.005  ▏                              EX 
   15    0.04     0.002                                 AEX 
```

</details>

## Per-point medians

| pt | levels ADE | orig ops/s | orig p99 | recon ops/s | recon p99 |
|---|---|---|---|---|---|
| 0 | `---` | 853k | 48.9µs | 1.32M | 46.3µs |
| 1 | `+--` | 479k | 1.17ms | 824k | 453µs |
| 2 | `-+-` | 1.47M | 28µs | 3.73M | 8.51µs |
| 3 | `++-` | 1.69M | 692µs | 2.62M | 116µs |
| 4 | `--+` | 626k | 59.1µs | 1.04M | 35.6µs |
| 5 | `+-+` | 650k | 2.44ms | 928k | 1.12ms |
| 6 | `-++` | 628k | 57.6µs | 1.48M | 68.1µs |
| 7 | `+++` | 773k | 2.18ms | 1.47M | 4.75ms |

