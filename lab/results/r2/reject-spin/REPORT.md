# DoE full: orig=orig vs recon=recon

`20261010-042410-full-orig-recon` · 2026-10-10T04:24:10+05:30 · go1.27.1 windows/amd64 · 20 CPUs (GOMAXPROCS 20) · 80 trials (0 failed) in 110s · 5 reps · warm-up 300ms + window 1000ms · seed 1

Command: `go run ./lab/cmd/doe -design full -factors A,D,E -reps 5 -dur 1s -warmup 300ms -spin -q -out lab/results/r2/reject-spin`

## Verdict

X is the implementation factor (−1 baseline, +1 candidate), crossed with every design run, so its main effect is the candidate's change averaged over the whole design. Log-scale responses show the ratio candidate/baseline of geometric means; raw responses show the difference. Significance is Bonferroni-corrected over all contrasts of a response; ratios within 5% count as *same*.

### Key responses

| response | arms | baseline | candidate | change [95% CI] | p | verdict | strongest X×factor |
|---|---|---|---|---|---|---|---|
| throughput | orig→recon | 833k | 1.55M | ×1.87 [1.84, 1.9] | <1e-4 | **better** | DX (×1.15) |
| goodput | orig→recon | 608k | 1.05M | ×1.73 [1.68, 1.78] | <1e-4 | **better** | EX (×0.656) |
| lat_p50 | orig→recon | 9.23µs | 2.95µs | ×0.319 [0.308, 0.331] | <1e-4 | **better** | EX (×0.546) |
| lat_p99 | orig→recon | 276µs | 142µs | ×0.515 [0.443, 0.6] | <1e-4 | **better** | EX (×1.57) |
| lat_p999 | orig→recon | 1.18ms | 328µs | ×0.278 [0.241, 0.321] | <1e-4 | **better** | EX (×1.96) |
| wait_p99 | orig→recon | 195µs | 118µs | ×0.604 [0.517, 0.705] | <1e-4 | **better** | EX (×1.44) |
| allocs_op | orig→recon | 11.9 | 0.745 | -11.2 [-11.2, -11.1] | <1e-4 | **better** | EX (-0.935) |
| bytes_op | orig→recon | 527 | 46.9 | -480 [-481, -480] | <1e-4 | **better** | EX (-71.4) |
| peak_goroutines | orig→recon | 5625 | 0 | -5.62e+03 [-6.6e+03, -4.64e+03] | <1e-4 | **better** | EX (+5.38e+03) |
| err_rate | orig→recon | 12.07% | 22.78% | +0.107 [+0.103, +0.112] | <1e-4 | **worse** | EX (+0.156) |
| cap_violations | orig→recon | 0 | 0 | +0 [+0, +0] | – | same | – |
| key_violations | orig→recon | 0 | 0 | +0 [+0, +0] | – | same | – |
| drain | orig→recon | 159µs | 42.7µs | ×0.269 [0.219, 0.329] | <1e-4 | **better** | EX (×1.75) |

<details><summary>All other responses</summary>

| response | arms | baseline | candidate | change [95% CI] | p | verdict | strongest X×factor |
|---|---|---|---|---|---|---|---|
| lat_p90 | orig→recon | 64.2µs | 51.2µs | ×0.798 [0.76, 0.837] | <1e-4 | **better** | AX (×1.47) |
| lat_max | orig→recon | 11ms | 4.39ms | ×0.398 [0.325, 0.487] | <1e-4 | **better** | DX (×0.562) |
| wait_p50 | orig→recon | 4.82µs | 1.34µs | ×0.279 [0.271, 0.287] | <1e-4 | **better** | EX (×0.597) |
| exec_p50 | orig→recon | 1.26µs | 458ns | ×0.363 [0.355, 0.372] | <1e-4 | **better** | DX (×1.61) |
| exec_p99 | orig→recon | 77.7µs | 42.5µs | ×0.547 [0.516, 0.579] | <1e-4 | **better** | DX (×3.03) |
| peak_heap | orig→recon | 39.8 | 35.2 | ×0.886 [0.639, 1.23] | 0.465 | same | – |
| gc_per_s | orig→recon | 35.8 | 3.82 | -32 [-50.9, -13.1] | 0.00125 | **better** | – |
| retries_op | orig→recon | 0 | 0 | +0 [+0, +0] | – | same | – |
| fairness | orig→recon | 0.98 | 0.99 | +0.0102 [-0.00455, +0.0249] | 0.172 | same | – |
| stats_lag | orig→recon | 0 | 0 | +0 [+0, +0] | – | same | – |
| residual_goroutines | orig→recon | 0 | 0 | +0 [+0, +0] | – | same | – |

</details>

## Noise

Replicate error of a single trial (pooled within-cell SD) as a coefficient of variation: the ±1σ spread between identical trials.

| response | orig | recon |
|---|---|---|
| throughput | 4.0% | 3.0% |
| goodput | 8.9% | 4.9% |
| lat_p50 | 4.8% | 10.9% |
| lat_p90 | 5.7% | 15.2% |
| lat_p99 | 61.3% | 6.2% |
| lat_p999 | 54.7% | 12.2% |
| lat_max | 42.6% | 70.0% |
| wait_p50 | 4.0% | 8.9% |
| wait_p99 | 62.3% | 7.9% |
| exec_p50 | 5.1% | 5.9% |
| exec_p99 | 17.5% | 8.8% |
| allocs_op | 0.3% | 1.9% |
| bytes_op | 0.3% | 1.6% |
| peak_goroutines | 55.1% | – |
| peak_heap | 104.4% | 112.3% |
| gc_per_s | 166.2% | 174.3% |
| err_rate | 6.6% | 5.1% |
| retries_op | – | – |
| fairness | 1.8% | 4.4% |
| cap_violations | – | – |
| key_violations | – | – |
| drain | 79.3% | 30.0% |
| stats_lag | – | – |
| residual_goroutines | – | – |

## Error classes

Share of all calls, over all trials of each arm. A class is the implementation-neutral error class, the outcome what `OutcomeOf` reports.

| class/outcome | orig | recon |
|---|---|---|
| overloaded/throttled | 23.588% | 27.940% |

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
| orig | D | ×1.56 | [1.52, 1.6] | <1e-4 |  | **sig** |
| orig | E | ×0.642 | [0.626, 0.658] | <1e-4 |  | **sig** |
| orig | DE | ×0.647 | [0.631, 0.664] | <1e-4 |  | **sig** |
| orig | ADE | ×0.823 | [0.802, 0.844] | <1e-4 |  | **sig** |
| orig | AD | ×1.22 | [1.18, 1.25] | <1e-4 |  | **sig** |
| orig | AE | ×1.12 | [1.09, 1.15] | <1e-4 |  | **sig** |
| orig vs recon | X | ×1.87 | [1.84, 1.9] | <1e-4 | * | **sig** |
| orig vs recon | D | ×1.8 | [1.77, 1.83] | <1e-4 | * | **sig** |
| orig vs recon | E | ×0.651 | [0.641, 0.661] | <1e-4 | * | **sig** |
| orig vs recon | DE | ×0.672 | [0.662, 0.683] | <1e-4 |  | **sig** |
| orig vs recon | AE | ×1.16 | [1.15, 1.18] | <1e-4 |  | **sig** |
| orig vs recon | A | ×0.864 | [0.851, 0.877] | <1e-4 |  | **sig** |
| recon | D | ×2.08 | [2.04, 2.12] | <1e-4 |  | **sig** |
| recon | E | ×0.661 | [0.648, 0.674] | <1e-4 |  | **sig** |
| recon | DE | ×0.699 | [0.686, 0.712] | <1e-4 |  | **sig** |
| recon | A | ×0.789 | [0.774, 0.804] | <1e-4 |  | **sig** |
| recon | AE | ×1.21 | [1.19, 1.24] | <1e-4 |  | **sig** |
| recon | AD | ×1.05 | [1.03, 1.07] | <1e-4 |  | **sig** |

### goodput (ops/s)

| scope | effect | estimate | 95% CI | p | Lenth |  |
|---|---|---|---|---|---|---|
| orig | AE | ×2.05 | [1.94, 2.16] | <1e-4 |  | **sig** |
| orig | A | ×0.517 | [0.49, 0.546] | <1e-4 |  | **sig** |
| orig | AD | ×0.664 | [0.629, 0.702] | <1e-4 |  | **sig** |
| orig | ADE | ×1.5 | [1.42, 1.59] | <1e-4 |  | **sig** |
| orig | DE | ×1.22 | [1.15, 1.28] | <1e-4 |  | **sig** |
| orig | E | ×1.2 | [1.14, 1.27] | <1e-4 |  | **sig** |
| orig vs recon | A | ×0.484 | [0.469, 0.499] | <1e-4 |  | **sig** |
| orig vs recon | X | ×1.73 | [1.68, 1.78] | <1e-4 |  | **sig** |
| orig vs recon | AD | ×0.634 | [0.615, 0.654] | <1e-4 |  | **sig** |
| orig vs recon | EX | ×0.656 | [0.636, 0.677] | <1e-4 |  | **sig** |
| orig vs recon | DEX | ×0.672 | [0.651, 0.693] | <1e-4 |  | **sig** |
| orig vs recon | AEX | ×0.687 | [0.666, 0.708] | <1e-4 |  | **sig** |
| recon | A | ×0.453 | [0.439, 0.467] | <1e-4 |  | **sig** |
| recon | E | ×0.518 | [0.503, 0.535] | <1e-4 |  | **sig** |
| recon | DE | ×0.548 | [0.531, 0.565] | <1e-4 |  | **sig** |
| recon | AD | ×0.605 | [0.587, 0.625] | <1e-4 |  | **sig** |
| recon | ADE | ×0.776 | [0.752, 0.8] | <1e-4 |  | **sig** |
| recon | D | ×0.95 | [0.921, 0.98] | 0.00189 |  | **sig** |

### lat_p50 (ns)

| scope | effect | estimate | 95% CI | p | Lenth |  |
|---|---|---|---|---|---|---|
| orig | A | ×5.28 | [5.12, 5.44] | <1e-4 |  | **sig** |
| orig | D | ×0.252 | [0.244, 0.26] | <1e-4 |  | **sig** |
| orig | DE | ×3.85 | [3.74, 3.97] | <1e-4 |  | **sig** |
| orig | AD | ×0.279 | [0.271, 0.288] | <1e-4 |  | **sig** |
| orig | ADE | ×3.55 | [3.45, 3.66] | <1e-4 |  | **sig** |
| orig | AE | ×0.492 | [0.477, 0.507] | <1e-4 |  | **sig** |
| orig vs recon | DE | ×6.22 | [6, 6.44] | <1e-4 |  | **sig** |
| orig vs recon | A | ×4.78 | [4.61, 4.95] | <1e-4 |  | **sig** |
| orig vs recon | ADE | ×3.43 | [3.31, 3.55] | <1e-4 |  | **sig** |
| orig vs recon | AD | ×0.303 | [0.292, 0.314] | <1e-4 |  | **sig** |
| orig vs recon | X | ×0.319 | [0.308, 0.331] | <1e-4 |  | **sig** |
| orig vs recon | D | ×0.323 | [0.312, 0.335] | <1e-4 |  | **sig** |
| recon | DE | ×10 | [9.39, 10.7] | <1e-4 |  | **sig** |
| recon | A | ×4.32 | [4.04, 4.62] | <1e-4 |  | **sig** |
| recon | E | ×0.269 | [0.252, 0.287] | <1e-4 |  | **sig** |
| recon | ADE | ×3.31 | [3.09, 3.53] | <1e-4 |  | **sig** |
| recon | AD | ×0.329 | [0.308, 0.352] | <1e-4 |  | **sig** |
| recon | D | ×0.414 | [0.387, 0.442] | <1e-4 |  | **sig** |

### lat_p99 (ns)

| scope | effect | estimate | 95% CI | p | Lenth |  |
|---|---|---|---|---|---|---|
| orig | A | ×30.3 | [22.3, 41.2] | <1e-4 | ** | **sig** |
| orig | E | ×2.03 | [1.49, 2.76] | <1e-4 |  | **sig** |
| orig | AE | ×1.46 | [1.07, 1.99] | 0.0177 |  |  |
| orig | D | ×0.773 | [0.568, 1.05] | 0.0974 |  |  |
| orig | DE | ×1.27 | [0.931, 1.72] | 0.128 |  |  |
| orig | AD | ×0.976 | [0.718, 1.33] | 0.875 |  |  |
| orig vs recon | A | ×26.4 | [22.7, 30.7] | <1e-4 | ** | **sig** |
| orig vs recon | E | ×3.19 | [2.74, 3.71] | <1e-4 | * | **sig** |
| orig vs recon | DE | ×2.21 | [1.9, 2.57] | <1e-4 | * | **sig** |
| orig vs recon | X | ×0.515 | [0.443, 0.6] | <1e-4 | * | **sig** |
| orig vs recon | DEX | ×1.74 | [1.5, 2.03] | <1e-4 |  | **sig** |
| orig vs recon | AE | ×1.68 | [1.44, 1.96] | <1e-4 |  | **sig** |
| recon | A | ×23 | [22.1, 23.9] | <1e-4 | * | **sig** |
| recon | E | ×5 | [4.81, 5.2] | <1e-4 |  | **sig** |
| recon | DE | ×3.85 | [3.7, 4] | <1e-4 |  | **sig** |
| recon | AE | ×1.93 | [1.86, 2.01] | <1e-4 |  | **sig** |
| recon | AD | ×1.36 | [1.3, 1.41] | <1e-4 |  | **sig** |
| recon | D | ×0.822 | [0.791, 0.854] | <1e-4 |  | **sig** |

### lat_p999 (ns)

| scope | effect | estimate | 95% CI | p | Lenth |  |
|---|---|---|---|---|---|---|
| orig | A | ×56.9 | [42.9, 75.3] | <1e-4 | * | **sig** |
| orig | DE | ×0.588 | [0.444, 0.779] | 0.000536 |  | **sig** |
| orig | ADE | ×0.622 | [0.469, 0.823] | 0.00161 |  | **sig** |
| orig | D | ×1.56 | [1.18, 2.07] | 0.00279 |  | **sig** |
| orig | AD | ×1.5 | [1.13, 1.99] | 0.00599 |  | **sig** |
| orig | AE | ×0.672 | [0.507, 0.89] | 0.00698 |  | **sig** |
| orig vs recon | A | ×41.5 | [36, 47.9] | <1e-4 | ** | **sig** |
| orig vs recon | X | ×0.278 | [0.241, 0.321] | <1e-4 |  | **sig** |
| orig vs recon | DEX | ×3.05 | [2.65, 3.52] | <1e-4 |  | **sig** |
| orig vs recon | EX | ×1.96 | [1.7, 2.26] | <1e-4 |  | **sig** |
| orig vs recon | DE | ×1.8 | [1.56, 2.07] | <1e-4 |  | **sig** |
| orig vs recon | ADEX | ×1.66 | [1.44, 1.92] | <1e-4 |  | **sig** |
| recon | A | ×30.3 | [28.1, 32.6] | <1e-4 | * | **sig** |
| recon | DE | ×5.48 | [5.09, 5.9] | <1e-4 |  | **sig** |
| recon | E | ×2.86 | [2.66, 3.08] | <1e-4 |  | **sig** |
| recon | ADE | ×1.72 | [1.6, 1.85] | <1e-4 |  | **sig** |
| recon | D | ×0.68 | [0.631, 0.732] | <1e-4 |  | **sig** |
| recon | AE | ×1.22 | [1.14, 1.32] | <1e-4 |  | **sig** |

### wait_p99 (ns)

| scope | effect | estimate | 95% CI | p | Lenth |  |
|---|---|---|---|---|---|---|
| orig | A | ×30.4 | [22.2, 41.5] | <1e-4 | * | **sig** |
| orig | E | ×2.45 | [1.79, 3.35] | <1e-4 |  | **sig** |
| orig | D | ×0.579 | [0.424, 0.791] | 0.00116 |  | **sig** |
| orig | DE | ×1.7 | [1.24, 2.32] | 0.00157 |  | **sig** |
| orig | AE | ×1.57 | [1.15, 2.14] | 0.00612 |  | **sig** |
| orig | AD | ×0.826 | [0.605, 1.13] | 0.221 |  |  |
| orig vs recon | A | ×27.9 | [23.9, 32.5] | <1e-4 | ** | **sig** |
| orig vs recon | E | ×3.53 | [3.02, 4.12] | <1e-4 | * | **sig** |
| orig vs recon | DE | ×2.51 | [2.15, 2.93] | <1e-4 |  | **sig** |
| orig vs recon | AE | ×1.71 | [1.47, 2] | <1e-4 |  | **sig** |
| orig vs recon | X | ×0.604 | [0.517, 0.705] | <1e-4 |  | **sig** |
| orig vs recon | D | ×0.604 | [0.517, 0.705] | <1e-4 |  | **sig** |
| recon | A | ×25.5 | [24.3, 26.8] | <1e-4 | * | **sig** |
| recon | E | ×5.08 | [4.84, 5.34] | <1e-4 |  | **sig** |
| recon | DE | ×3.7 | [3.53, 3.89] | <1e-4 |  | **sig** |
| recon | AE | ×1.87 | [1.78, 1.96] | <1e-4 |  | **sig** |
| recon | D | ×0.63 | [0.6, 0.662] | <1e-4 |  | **sig** |
| recon | AD | ×1.37 | [1.3, 1.44] | <1e-4 |  | **sig** |

### allocs_op (allocs)

| scope | effect | estimate | 95% CI | p | Lenth |  |
|---|---|---|---|---|---|---|
| orig | E | +1.54 | [+1.52, +1.56] | <1e-4 |  | **sig** |
| orig | AE | -1.47 | [-1.5, -1.45] | <1e-4 |  | **sig** |
| orig | A | +1.07 | [+1.05, +1.09] | <1e-4 |  | **sig** |
| orig | DE | +0.66 | [+0.638, +0.681] | <1e-4 |  | **sig** |
| orig | D | -0.65 | [-0.672, -0.629] | <1e-4 |  | **sig** |
| orig | ADE | -0.0469 | [-0.0684, -0.0254] | 0.0001 |  | **sig** |
| orig vs recon | X | -11.2 | [-11.2, -11.1] | <1e-4 | ** | **sig** |
| orig vs recon | EX | -0.935 | [-0.946, -0.923] | <1e-4 |  | **sig** |
| orig vs recon | AE | -0.842 | [-0.854, -0.831] | <1e-4 |  | **sig** |
| orig vs recon | A | +0.808 | [+0.796, +0.819] | <1e-4 |  | **sig** |
| orig vs recon | AEX | +0.632 | [+0.62, +0.643] | <1e-4 |  | **sig** |
| orig vs recon | D | -0.615 | [-0.626, -0.603] | <1e-4 |  | **sig** |
| recon | D | -0.579 | [-0.588, -0.57] | <1e-4 |  | **sig** |
| recon | A | +0.546 | [+0.537, +0.555] | <1e-4 |  | **sig** |
| recon | DE | +0.471 | [+0.462, +0.481] | <1e-4 |  | **sig** |
| recon | E | -0.332 | [-0.341, -0.322] | <1e-4 |  | **sig** |
| recon | ADE | +0.329 | [+0.32, +0.338] | <1e-4 |  | **sig** |
| recon | AE | -0.211 | [-0.22, -0.202] | <1e-4 |  | **sig** |

### bytes_op (B)

| scope | effect | estimate | 95% CI | p | Lenth |  |
|---|---|---|---|---|---|---|
| orig | E | +112 | [+111, +114] | <1e-4 |  | **sig** |
| orig | DE | +98.1 | [+97, +99.3] | <1e-4 |  | **sig** |
| orig | D | -97.6 | [-98.7, -96.4] | <1e-4 |  | **sig** |
| orig | AE | -56 | [-57.2, -54.9] | <1e-4 |  | **sig** |
| orig | A | +34.8 | [+33.6, +35.9] | <1e-4 |  | **sig** |
| orig | AD | -26.6 | [-27.7, -25.4] | <1e-4 |  | **sig** |
| orig vs recon | X | -480 | [-481, -480] | <1e-4 | ** | **sig** |
| orig vs recon | D | -84.7 | [-85.3, -84.1] | <1e-4 |  | **sig** |
| orig vs recon | EX | -71.4 | [-72, -70.8] | <1e-4 |  | **sig** |
| orig vs recon | DE | +66 | [+65.4, +66.6] | <1e-4 |  | **sig** |
| orig vs recon | E | +41 | [+40.4, +41.6] | <1e-4 |  | **sig** |
| orig vs recon | AE | -38.1 | [-38.7, -37.5] | <1e-4 |  | **sig** |
| recon | D | -71.9 | [-72.3, -71.4] | <1e-4 |  | **sig** |
| recon | DE | +33.8 | [+33.3, +34.3] | <1e-4 |  | **sig** |
| recon | E | -30.4 | [-30.9, -30] | <1e-4 |  | **sig** |
| recon | A | +29.6 | [+29.1, +30.1] | <1e-4 |  | **sig** |
| recon | ADE | +23 | [+22.5, +23.5] | <1e-4 |  | **sig** |
| recon | AE | -20.2 | [-20.6, -19.7] | <1e-4 |  | **sig** |

### peak_goroutines (goroutines)

| scope | effect | estimate | 95% CI | p | Lenth |  |
|---|---|---|---|---|---|---|
| orig | E | -1.08e+04 | [-1.28e+04, -8.76e+03] | <1e-4 |  | **sig** |
| orig | D | +1.07e+04 | [+8.72e+03, +1.27e+04] | <1e-4 |  | **sig** |
| orig | DE | -1.07e+04 | [-1.27e+04, -8.71e+03] | <1e-4 |  | **sig** |
| orig | A | +1.04e+04 | [+8.38e+03, +1.24e+04] | <1e-4 |  | **sig** |
| orig | ADE | -1.03e+04 | [-1.23e+04, -8.29e+03] | <1e-4 |  | **sig** |
| orig | AD | +1.03e+04 | [+8.27e+03, +1.23e+04] | <1e-4 |  | **sig** |
| orig vs recon | X | -5.62e+03 | [-6.6e+03, -4.64e+03] | <1e-4 |  | **sig** |
| orig vs recon | E | -5.38e+03 | [-6.36e+03, -4.4e+03] | <1e-4 |  | **sig** |
| orig vs recon | EX | +5.38e+03 | [+4.4e+03, +6.36e+03] | <1e-4 |  | **sig** |
| orig vs recon | D | +5.36e+03 | [+4.38e+03, +6.34e+03] | <1e-4 |  | **sig** |
| orig vs recon | DX | -5.36e+03 | [-6.34e+03, -4.38e+03] | <1e-4 |  | **sig** |
| orig vs recon | DE | -5.35e+03 | [-6.33e+03, -4.37e+03] | <1e-4 |  | **sig** |
| recon | A | +0 | [+0, +0] | – |  |  |
| recon | D | +0 | [+0, +0] | – |  |  |
| recon | E | +0 | [+0, +0] | – |  |  |
| recon | AD | +0 | [+0, +0] | – |  |  |
| recon | AE | +0 | [+0, +0] | – |  |  |
| recon | DE | +0 | [+0, +0] | – |  |  |

### err_rate (fraction)

| scope | effect | estimate | 95% CI | p | Lenth |  |
|---|---|---|---|---|---|---|
| orig | D | +0.241 | [+0.236, +0.247] | <1e-4 |  | **sig** |
| orig | E | -0.241 | [-0.246, -0.236] | <1e-4 |  | **sig** |
| orig | DE | -0.241 | [-0.246, -0.236] | <1e-4 |  | **sig** |
| orig | A | +0.216 | [+0.21, +0.221] | <1e-4 |  | **sig** |
| orig | AD | +0.216 | [+0.21, +0.221] | <1e-4 |  | **sig** |
| orig | AE | -0.215 | [-0.221, -0.21] | <1e-4 |  | **sig** |
| orig vs recon | D | +0.349 | [+0.344, +0.353] | <1e-4 |  | **sig** |
| orig vs recon | A | +0.234 | [+0.229, +0.238] | <1e-4 |  | **sig** |
| orig vs recon | AD | +0.234 | [+0.229, +0.238] | <1e-4 |  | **sig** |
| orig vs recon | EX | +0.156 | [+0.151, +0.16] | <1e-4 |  | **sig** |
| orig vs recon | DEX | +0.156 | [+0.151, +0.16] | <1e-4 |  | **sig** |
| orig vs recon | AEX | +0.137 | [+0.133, +0.142] | <1e-4 |  | **sig** |
| recon | D | +0.456 | [+0.448, +0.463] | <1e-4 | * | **sig** |
| recon | A | +0.252 | [+0.244, +0.259] | <1e-4 |  | **sig** |
| recon | AD | +0.252 | [+0.244, +0.259] | <1e-4 |  | **sig** |
| recon | E | +0.0699 | [+0.0624, +0.0774] | <1e-4 |  | **sig** |
| recon | DE | +0.0699 | [+0.0624, +0.0774] | <1e-4 |  | **sig** |
| recon | AE | +0.059 | [+0.0515, +0.0665] | <1e-4 |  | **sig** |

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
| orig | A | ×25.7 | [17.6, 37.4] | <1e-4 |  | **sig** |
| orig | D | ×3.03 | [2.08, 4.41] | <1e-4 |  | **sig** |
| orig | DE | ×0.445 | [0.305, 0.648] | 0.000118 |  | **sig** |
| orig | E | ×0.468 | [0.321, 0.682] | 0.000255 |  | **sig** |
| orig | ADE | ×0.548 | [0.376, 0.799] | 0.00268 |  | **sig** |
| orig | AE | ×0.59 | [0.405, 0.859] | 0.00744 |  |  |
| orig vs recon | A | ×17.2 | [14.1, 21.1] | <1e-4 | * | **sig** |
| orig vs recon | X | ×0.269 | [0.219, 0.329] | <1e-4 |  | **sig** |
| orig vs recon | DEX | ×1.9 | [1.56, 2.33] | <1e-4 |  | **sig** |
| orig vs recon | D | ×1.76 | [1.44, 2.16] | <1e-4 |  | **sig** |
| orig vs recon | EX | ×1.75 | [1.43, 2.14] | <1e-4 |  | **sig** |
| orig vs recon | DX | ×0.581 | [0.475, 0.712] | <1e-4 |  | **sig** |
| recon | A | ×11.5 | [9.74, 13.7] | <1e-4 | * | **sig** |
| recon | DE | ×1.61 | [1.36, 1.91] | <1e-4 |  | **sig** |
| recon | ADE | ×1.53 | [1.29, 1.81] | <1e-4 |  | **sig** |
| recon | E | ×1.43 | [1.2, 1.69] | 0.000159 |  | **sig** |
| recon | AE | ×1.24 | [1.05, 1.47] | 0.0138 |  |  |
| recon | AD | ×1.05 | [0.884, 1.24] | 0.585 |  |  |

<details><summary>Half-normal plot: lat_p99, orig vs recon (15 contrasts)</summary>

```
 rank  z(half)  |effect|
    1    2.13     4.723  ██████████████████████████████ A **
    2    1.64     1.672  ██████████▋                    E *
    3    1.38     1.143  ███████▎                       DE *
    4    1.19     0.956  ██████▏                        X *
    5    1.04     0.802  █████▏                         DEX 
    6    0.90     0.748  ████▊                          AE 
    7    0.78     0.651  ████▏                          EX 
    8    0.67     0.328  ██▏                            D 
    9    0.57     0.237  █▌                             ADX 
   10    0.48     0.203  █▎                             AEX 
   11    0.39     0.202  █▎                             AD 
   12    0.30     0.198  █▎                             AX 
   13    0.21     0.115  ▊                              ADE 
   14    0.13     0.092  ▋                              ADEX 
   15    0.04     0.045  ▎                              DX 
```

</details>

<details><summary>Half-normal plot: throughput, orig vs recon (15 contrasts)</summary>

```
 rank  z(half)  |effect|
    1    2.13     0.901  ██████████████████████████████ X *
    2    1.64     0.849  ████████████████████████████▎  D *
    3    1.38     0.619  ████████████████████▋          E *
    4    1.19     0.573  ███████████████████▏           DE 
    5    1.04     0.220  ███████▍                       AE 
    6    0.90     0.211  ███████                        A 
    7    0.78     0.206  ██████▉                        DX 
    8    0.67     0.179  ██████                         AD 
    9    0.57     0.159  █████▎                         ADE 
   10    0.48     0.131  ████▍                          AX 
   11    0.39     0.123  ████▏                          ADEX 
   12    0.30     0.102  ███▍                           ADX 
   13    0.21     0.058  ██                             AEX 
   14    0.13     0.055  █▉                             DEX 
   15    0.04     0.022  ▊                              EX 
```

</details>

## Per-point medians

| pt | levels ADE | orig ops/s | orig p99 | recon ops/s | recon p99 |
|---|---|---|---|---|---|
| 0 | `---` | 896k | 45.3µs | 1.43M | 43.3µs |
| 1 | `+--` | 507k | 1.14ms | 868k | 436µs |
| 2 | `-+-` | 1.5M | 36.6µs | 3.89M | 8.26µs |
| 3 | `++-` | 1.8M | 1.2ms | 2.72M | 113µs |
| 4 | `--+` | 653k | 56.1µs | 1.09M | 33.5µs |
| 5 | `+-+` | 681k | 2.67ms | 1.01M | 954µs |
| 6 | `-++` | 661k | 55.6µs | 1.55M | 67.1µs |
| 7 | `+++` | 687k | 2.51ms | 1.51M | 4.75ms |

