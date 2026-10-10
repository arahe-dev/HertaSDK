# DoE full: orig=orig vs recon=recon

`20261009-232343-full-orig-recon` · 2026-10-09T23:23:43+05:30 · go1.27.1 windows/amd64 · 20 CPUs (GOMAXPROCS 20) · 80 trials (0 failed) in 110s · 5 reps · warm-up 300ms + window 1000ms · seed 1

Command: `go run ./lab/cmd/doe -design full -factors A,D,E -reps 5 -dur 1s -warmup 300ms -q -out lab/results/r1/reject-interaction`

## Verdict

X is the implementation factor (−1 baseline, +1 candidate), crossed with every design run, so its main effect is the candidate's change averaged over the whole design. Log-scale responses show the ratio candidate/baseline of geometric means; raw responses show the difference. Significance is Bonferroni-corrected over all contrasts of a response; ratios within 5% count as *same*.

### Key responses

| response | arms | baseline | candidate | change [95% CI] | p | verdict | strongest X×factor |
|---|---|---|---|---|---|---|---|
| throughput | orig→recon | 821k | 1.69M | ×2.06 [2, 2.13] | <1e-4 | **better** | DX (×1.34) |
| goodput | orig→recon | 608k | 754k | ×1.24 [1.18, 1.3] | <1e-4 | **better** | AX (×0.669) |
| lat_p50 | orig→recon | 9.52µs | 2.59µs | ×0.272 [0.257, 0.288] | <1e-4 | **better** | EX (×0.363) |
| lat_p99 | orig→recon | 277µs | 240µs | ×0.867 [0.833, 0.903] | <1e-4 | **better** | DX (×1.54) |
| lat_p999 | orig→recon | 1.18ms | 652µs | ×0.554 [0.487, 0.63] | <1e-4 | **better** | EX (×1.27) |
| wait_p99 | orig→recon | 194µs | 182µs | ×0.939 [0.892, 0.989] | 0.0185 | same | DX (×1.49) |
| allocs_op | orig→recon | 11.9 | 0.801 | -11.1 [-11.1, -11.1] | <1e-4 | **better** | EX (-0.985) |
| bytes_op | orig→recon | 528 | 48.2 | -479 [-480, -479] | <1e-4 | **better** | EX (-73.1) |
| peak_goroutines | orig→recon | 4858 | 0 | -4.86e+03 [-6.35e+03, -3.36e+03] | <1e-4 | **better** | EX (+4.53e+03) |
| err_rate | orig→recon | 12.08% | 28.41% | +0.163 [+0.16, +0.167] | <1e-4 | **worse** | DX (+0.163) |
| cap_violations | orig→recon | 0 | 0 | +0 [+0, +0] | – | same | – |
| key_violations | orig→recon | 0 | 0 | +0 [+0, +0] | – | same | – |
| drain | orig→recon | 156µs | 64.6µs | ×0.413 [0.337, 0.507] | <1e-4 | **better** | EX (×1.63) |

<details><summary>All other responses</summary>

| response | arms | baseline | candidate | change [95% CI] | p | verdict | strongest X×factor |
|---|---|---|---|---|---|---|---|
| lat_p90 | orig→recon | 65.5µs | 28.8µs | ×0.44 [0.422, 0.459] | <1e-4 | **better** | DX (×0.57) |
| lat_max | orig→recon | 12.6ms | 5.93ms | ×0.47 [0.416, 0.531] | <1e-4 | **better** | EX (×1.56) |
| wait_p50 | orig→recon | 4.83µs | 1.22µs | ×0.253 [0.239, 0.268] | <1e-4 | **better** | EX (×0.438) |
| exec_p50 | orig→recon | 1.24µs | 412ns | ×0.331 [0.315, 0.348] | <1e-4 | **better** | EX (×0.507) |
| exec_p99 | orig→recon | 71.7µs | 69.5µs | ×0.968 [0.925, 1.01] | 0.16 | same | DX (×4.79) |
| peak_heap | orig→recon | 41.7 | 35.3 | ×0.846 [0.628, 1.14] | 0.267 | same | – |
| gc_per_s | orig→recon | 33.5 | 4.52 | -29 [-45.4, -12.5] | 0.000809 | **better** | – |
| retries_op | orig→recon | 0 | 0 | +0 [+0, +0] | – | same | – |
| fairness | orig→recon | 0.98 | 0.998 | +0.0182 [+0.0156, +0.0207] | <1e-4 | **better** | EX (-0.0194) |
| stats_lag | orig→recon | 0 | 0 | +0 [+0, +0] | – | same | – |
| residual_goroutines | orig→recon | 0 | 0 | +0 [+0, +0] | – | same | – |

</details>

## Noise

Replicate error of a single trial (pooled within-cell SD) as a coefficient of variation: the ±1σ spread between identical trials.

| response | orig | recon |
|---|---|---|
| throughput | 9.2% | 5.5% |
| goodput | 15.0% | 8.3% |
| lat_p50 | 12.2% | 14.8% |
| lat_p90 | 11.0% | 8.6% |
| lat_p99 | 12.4% | 5.3% |
| lat_p999 | 47.7% | 12.4% |
| lat_max | 32.3% | 31.0% |
| wait_p50 | 9.5% | 17.0% |
| wait_p99 | 16.8% | 5.3% |
| exec_p50 | 11.2% | 11.9% |
| exec_p99 | 12.0% | 9.3% |
| allocs_op | 0.4% | 1.5% |
| bytes_op | 0.3% | 1.9% |
| peak_goroutines | 97.4% | – |
| peak_heap | 84.6% | 104.6% |
| gc_per_s | 154.0% | 163.6% |
| err_rate | 9.0% | 1.2% |
| retries_op | – | – |
| fairness | 0.8% | 0.1% |
| cap_violations | – | – |
| key_violations | – | – |
| drain | 80.5% | 31.2% |
| stats_lag | – | – |
| residual_goroutines | – | – |

## Error classes

Share of all calls, over all trials of each arm. A class is the implementation-neutral error class, the outcome what `OutcomeOf` reports.

| class/outcome | orig | recon |
|---|---|---|
| overloaded/throttled | 24.005% | 45.060% |

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
| orig | D | ×1.63 | [1.54, 1.72] | <1e-4 |  | **sig** |
| orig | DE | ×0.625 | [0.591, 0.662] | <1e-4 |  | **sig** |
| orig | E | ×0.659 | [0.623, 0.698] | <1e-4 |  | **sig** |
| orig | AD | ×1.24 | [1.17, 1.31] | <1e-4 |  | **sig** |
| orig | ADE | ×0.83 | [0.784, 0.879] | <1e-4 |  | **sig** |
| orig | AE | ×1.13 | [1.06, 1.19] | 0.000184 |  | **sig** |
| orig vs recon | D | ×2.18 | [2.11, 2.26] | <1e-4 | ** | **sig** |
| orig vs recon | X | ×2.06 | [2, 2.13] | <1e-4 | ** | **sig** |
| orig vs recon | E | ×0.661 | [0.639, 0.682] | <1e-4 | * | **sig** |
| orig vs recon | DE | ×0.661 | [0.64, 0.683] | <1e-4 | * | **sig** |
| orig vs recon | DX | ×1.34 | [1.3, 1.39] | <1e-4 | * | **sig** |
| orig vs recon | AD | ×1.26 | [1.22, 1.3] | <1e-4 | * | **sig** |
| recon | D | ×2.93 | [2.83, 3.04] | <1e-4 |  | **sig** |
| recon | E | ×0.662 | [0.639, 0.685] | <1e-4 |  | **sig** |
| recon | DE | ×0.699 | [0.675, 0.723] | <1e-4 |  | **sig** |
| recon | AD | ×1.27 | [1.23, 1.32] | <1e-4 |  | **sig** |
| recon | AE | ×1.18 | [1.14, 1.22] | <1e-4 |  | **sig** |
| recon | A | ×0.953 | [0.921, 0.987] | 0.00854 |  |  |

### goodput (ops/s)

| scope | effect | estimate | 95% CI | p | Lenth |  |
|---|---|---|---|---|---|---|
| orig | AE | ×1.98 | [1.81, 2.16] | <1e-4 |  | **sig** |
| orig | A | ×0.538 | [0.491, 0.588] | <1e-4 |  | **sig** |
| orig | ADE | ×1.46 | [1.33, 1.6] | <1e-4 |  | **sig** |
| orig | AD | ×0.705 | [0.645, 0.771] | <1e-4 |  | **sig** |
| orig | E | ×1.2 | [1.1, 1.31] | 0.000245 |  | **sig** |
| orig | DE | ×1.14 | [1.04, 1.24] | 0.00618 |  | **sig** |
| orig vs recon | A | ×0.36 | [0.342, 0.378] | <1e-4 | * | **sig** |
| orig vs recon | AD | ×0.476 | [0.453, 0.501] | <1e-4 |  | **sig** |
| orig vs recon | AE | ×1.99 | [1.9, 2.1] | <1e-4 |  | **sig** |
| orig vs recon | ADE | ×1.56 | [1.48, 1.64] | <1e-4 |  | **sig** |
| orig vs recon | AX | ×0.669 | [0.636, 0.704] | <1e-4 |  | **sig** |
| orig vs recon | ADX | ×0.675 | [0.642, 0.71] | <1e-4 |  | **sig** |
| recon | A | ×0.241 | [0.229, 0.253] | <1e-4 |  | **sig** |
| recon | AD | ×0.322 | [0.305, 0.339] | <1e-4 |  | **sig** |
| recon | AE | ×2.01 | [1.91, 2.12] | <1e-4 |  | **sig** |
| recon | D | ×0.581 | [0.551, 0.611] | <1e-4 |  | **sig** |
| recon | ADE | ×1.67 | [1.58, 1.75] | <1e-4 |  | **sig** |
| recon | DE | ×1.13 | [1.08, 1.19] | <1e-4 |  | **sig** |

### lat_p50 (ns)

| scope | effect | estimate | 95% CI | p | Lenth |  |
|---|---|---|---|---|---|---|
| orig | A | ×5.56 | [5.16, 5.99] | <1e-4 |  | **sig** |
| orig | D | ×0.236 | [0.219, 0.254] | <1e-4 |  | **sig** |
| orig | DE | ×4.05 | [3.76, 4.36] | <1e-4 |  | **sig** |
| orig | AD | ×0.271 | [0.252, 0.292] | <1e-4 |  | **sig** |
| orig | ADE | ×3.54 | [3.29, 3.81] | <1e-4 |  | **sig** |
| orig | AE | ×0.52 | [0.483, 0.56] | <1e-4 |  | **sig** |
| orig vs recon | A | ×5.5 | [5.19, 5.82] | <1e-4 |  | **sig** |
| orig vs recon | DE | ×4.67 | [4.41, 4.94] | <1e-4 |  | **sig** |
| orig vs recon | X | ×0.272 | [0.257, 0.288] | <1e-4 |  | **sig** |
| orig vs recon | D | ×0.276 | [0.261, 0.292] | <1e-4 |  | **sig** |
| orig vs recon | ADE | ×3.25 | [3.07, 3.44] | <1e-4 |  | **sig** |
| orig vs recon | AD | ×0.339 | [0.32, 0.359] | <1e-4 |  | **sig** |
| recon | E | ×0.125 | [0.114, 0.136] | <1e-4 |  | **sig** |
| recon | A | ×5.43 | [4.97, 5.94] | <1e-4 |  | **sig** |
| recon | DE | ×5.39 | [4.93, 5.89] | <1e-4 |  | **sig** |
| recon | D | ×0.322 | [0.295, 0.352] | <1e-4 |  | **sig** |
| recon | ADE | ×2.98 | [2.72, 3.25] | <1e-4 |  | **sig** |
| recon | AE | ×0.379 | [0.347, 0.414] | <1e-4 |  | **sig** |

### lat_p99 (ns)

| scope | effect | estimate | 95% CI | p | Lenth |  |
|---|---|---|---|---|---|---|
| orig | A | ×34.2 | [31.7, 36.9] | <1e-4 | ** | **sig** |
| orig | E | ×1.8 | [1.67, 1.94] | <1e-4 |  | **sig** |
| orig | D | ×0.785 | [0.728, 0.846] | <1e-4 |  | **sig** |
| orig | AE | ×1.27 | [1.17, 1.37] | <1e-4 |  | **sig** |
| orig | DE | ×1.26 | [1.17, 1.36] | <1e-4 |  | **sig** |
| orig | ADE | ×0.914 | [0.848, 0.986] | 0.0209 |  |  |
| orig vs recon | A | ×41.6 | [39.9, 43.3] | <1e-4 | ** | **sig** |
| orig vs recon | E | ×2.04 | [1.96, 2.13] | <1e-4 |  | **sig** |
| orig vs recon | AD | ×1.76 | [1.69, 1.84] | <1e-4 |  | **sig** |
| orig vs recon | ADX | ×1.68 | [1.61, 1.75] | <1e-4 |  | **sig** |
| orig vs recon | DX | ×1.54 | [1.48, 1.6] | <1e-4 |  | **sig** |
| orig vs recon | ADE | ×0.65 | [0.624, 0.677] | <1e-4 |  | **sig** |
| recon | A | ×50.6 | [48.9, 52.3] | <1e-4 |  | **sig** |
| recon | AD | ×2.97 | [2.87, 3.07] | <1e-4 |  | **sig** |
| recon | E | ×2.31 | [2.24, 2.39] | <1e-4 |  | **sig** |
| recon | ADE | ×0.462 | [0.447, 0.478] | <1e-4 |  | **sig** |
| recon | D | ×1.86 | [1.8, 1.93] | <1e-4 |  | **sig** |
| recon | DE | ×1.69 | [1.63, 1.74] | <1e-4 |  | **sig** |

### lat_p999 (ns)

| scope | effect | estimate | 95% CI | p | Lenth |  |
|---|---|---|---|---|---|---|
| orig | A | ×58.7 | [45.7, 75.5] | <1e-4 | * | **sig** |
| orig | ADE | ×0.613 | [0.477, 0.788] | 0.000382 |  | **sig** |
| orig | AD | ×1.47 | [1.15, 1.89] | 0.0036 |  | **sig** |
| orig | DE | ×0.687 | [0.535, 0.884] | 0.00471 |  | **sig** |
| orig | D | ×1.34 | [1.04, 1.72] | 0.0234 |  |  |
| orig | E | ×0.752 | [0.585, 0.967] | 0.0274 |  |  |
| orig vs recon | A | ×64.2 | [56.4, 73] | <1e-4 | ** | **sig** |
| orig vs recon | X | ×0.554 | [0.487, 0.63] | <1e-4 |  | **sig** |
| orig vs recon | DEX | ×1.72 | [1.51, 1.95] | <1e-4 |  | **sig** |
| orig vs recon | AD | ×1.58 | [1.39, 1.8] | <1e-4 |  | **sig** |
| orig vs recon | D | ×1.49 | [1.31, 1.69] | <1e-4 |  | **sig** |
| orig vs recon | ADE | ×0.674 | [0.593, 0.767] | <1e-4 |  | **sig** |
| recon | A | ×70.2 | [65.1, 75.7] | <1e-4 | * | **sig** |
| recon | DE | ×2.02 | [1.88, 2.18] | <1e-4 |  | **sig** |
| recon | AE | ×0.583 | [0.541, 0.629] | <1e-4 |  | **sig** |
| recon | AD | ×1.69 | [1.57, 1.83] | <1e-4 |  | **sig** |
| recon | D | ×1.64 | [1.53, 1.77] | <1e-4 |  | **sig** |
| recon | ADE | ×0.742 | [0.688, 0.8] | <1e-4 |  | **sig** |

### wait_p99 (ns)

| scope | effect | estimate | 95% CI | p | Lenth |  |
|---|---|---|---|---|---|---|
| orig | A | ×28.5 | [25.8, 31.5] | <1e-4 | * | **sig** |
| orig | E | ×2.51 | [2.27, 2.78] | <1e-4 |  | **sig** |
| orig | D | ×0.543 | [0.491, 0.6] | <1e-4 |  | **sig** |
| orig | DE | ×1.82 | [1.65, 2.01] | <1e-4 |  | **sig** |
| orig | AE | ×1.58 | [1.43, 1.75] | <1e-4 |  | **sig** |
| orig | AD | ×0.75 | [0.679, 0.829] | <1e-4 |  | **sig** |
| orig vs recon | A | ×37.2 | [35.4, 39.2] | <1e-4 | ** | **sig** |
| orig vs recon | E | ×2.48 | [2.35, 2.61] | <1e-4 |  | **sig** |
| orig vs recon | ADX | ×1.85 | [1.76, 1.95] | <1e-4 |  | **sig** |
| orig vs recon | DE | ×1.75 | [1.66, 1.84] | <1e-4 |  | **sig** |
| orig vs recon | ADEX | ×0.629 | [0.597, 0.662] | <1e-4 |  | **sig** |
| orig vs recon | DX | ×1.49 | [1.42, 1.57] | <1e-4 |  | **sig** |
| recon | A | ×48.6 | [47, 50.3] | <1e-4 | * | **sig** |
| recon | AD | ×2.57 | [2.49, 2.66] | <1e-4 |  | **sig** |
| recon | E | ×2.44 | [2.36, 2.52] | <1e-4 |  | **sig** |
| recon | ADE | ×0.507 | [0.491, 0.525] | <1e-4 |  | **sig** |
| recon | DE | ×1.68 | [1.62, 1.73] | <1e-4 |  | **sig** |
| recon | D | ×1.21 | [1.17, 1.25] | <1e-4 |  | **sig** |

### allocs_op (allocs)

| scope | effect | estimate | 95% CI | p | Lenth |  |
|---|---|---|---|---|---|---|
| orig | E | +1.56 | [+1.53, +1.59] | <1e-4 |  | **sig** |
| orig | AE | -1.43 | [-1.46, -1.4] | <1e-4 |  | **sig** |
| orig | A | +1.06 | [+1.03, +1.08] | <1e-4 |  | **sig** |
| orig | D | -0.667 | [-0.696, -0.638] | <1e-4 |  | **sig** |
| orig | DE | +0.657 | [+0.629, +0.686] | <1e-4 |  | **sig** |
| orig | AD | +0.0222 | [-0.0066, +0.051] | 0.126 |  |  |
| orig vs recon | X | -11.1 | [-11.1, -11.1] | <1e-4 | ** | **sig** |
| orig vs recon | EX | -0.985 | [-0.999, -0.97] | <1e-4 |  | **sig** |
| orig vs recon | AE | -0.873 | [-0.888, -0.859] | <1e-4 |  | **sig** |
| orig vs recon | A | +0.853 | [+0.838, +0.867] | <1e-4 |  | **sig** |
| orig vs recon | E | +0.573 | [+0.559, +0.588] | <1e-4 |  | **sig** |
| orig vs recon | D | -0.566 | [-0.58, -0.551] | <1e-4 |  | **sig** |
| recon | A | +0.649 | [+0.641, +0.657] | <1e-4 |  | **sig** |
| recon | D | -0.465 | [-0.472, -0.457] | <1e-4 |  | **sig** |
| recon | E | -0.411 | [-0.419, -0.404] | <1e-4 |  | **sig** |
| recon | DE | +0.392 | [+0.384, +0.4] | <1e-4 |  | **sig** |
| recon | AE | -0.32 | [-0.327, -0.312] | <1e-4 |  | **sig** |
| recon | ADE | +0.221 | [+0.213, +0.229] | <1e-4 |  | **sig** |

### bytes_op (B)

| scope | effect | estimate | 95% CI | p | Lenth |  |
|---|---|---|---|---|---|---|
| orig | E | +114 | [+113, +115] | <1e-4 |  | **sig** |
| orig | D | -98.8 | [-99.8, -97.9] | <1e-4 |  | **sig** |
| orig | DE | +98.3 | [+97.3, +99.3] | <1e-4 |  | **sig** |
| orig | AE | -54.1 | [-55, -53.1] | <1e-4 |  | **sig** |
| orig | A | +34.7 | [+33.8, +35.7] | <1e-4 |  | **sig** |
| orig | ADE | +27.2 | [+26.3, +28.2] | <1e-4 |  | **sig** |
| orig vs recon | X | -479 | [-480, -479] | <1e-4 | ** | **sig** |
| orig vs recon | D | -83.9 | [-84.5, -83.4] | <1e-4 |  | **sig** |
| orig vs recon | EX | -73.1 | [-73.6, -72.5] | <1e-4 |  | **sig** |
| orig vs recon | DE | +65.1 | [+64.5, +65.7] | <1e-4 |  | **sig** |
| orig vs recon | E | +40.7 | [+40.2, +41.3] | <1e-4 |  | **sig** |
| orig vs recon | AE | -38.4 | [-39, -37.9] | <1e-4 |  | **sig** |
| recon | D | -69 | [-69.6, -68.4] | <1e-4 |  | **sig** |
| recon | E | -32.4 | [-33, -31.8] | <1e-4 |  | **sig** |
| recon | A | +32.1 | [+31.5, +32.7] | <1e-4 |  | **sig** |
| recon | DE | +31.9 | [+31.3, +32.5] | <1e-4 |  | **sig** |
| recon | AE | -22.8 | [-23.4, -22.2] | <1e-4 |  | **sig** |
| recon | ADE | +20.5 | [+19.9, +21] | <1e-4 |  | **sig** |

### peak_goroutines (goroutines)

| scope | effect | estimate | 95% CI | p | Lenth |  |
|---|---|---|---|---|---|---|
| orig | E | -9.06e+03 | [-1.21e+04, -6.01e+03] | <1e-4 |  | **sig** |
| orig | D | +8.91e+03 | [+5.86e+03, +1.2e+04] | <1e-4 |  | **sig** |
| orig | DE | -8.77e+03 | [-1.18e+04, -5.72e+03] | <1e-4 |  | **sig** |
| orig | ADE | -8.03e+03 | [-1.11e+04, -4.98e+03] | <1e-4 |  | **sig** |
| orig | AD | +7.91e+03 | [+4.86e+03, +1.1e+04] | <1e-4 |  | **sig** |
| orig | A | +7.78e+03 | [+4.73e+03, +1.08e+04] | <1e-4 |  | **sig** |
| orig vs recon | X | -4.86e+03 | [-6.35e+03, -3.36e+03] | <1e-4 |  | **sig** |
| orig vs recon | E | -4.53e+03 | [-6.02e+03, -3.04e+03] | <1e-4 |  | **sig** |
| orig vs recon | EX | +4.53e+03 | [+3.04e+03, +6.02e+03] | <1e-4 |  | **sig** |
| orig vs recon | D | +4.45e+03 | [+2.96e+03, +5.95e+03] | <1e-4 |  | **sig** |
| orig vs recon | DX | -4.45e+03 | [-5.95e+03, -2.96e+03] | <1e-4 |  | **sig** |
| orig vs recon | DE | -4.39e+03 | [-5.88e+03, -2.89e+03] | <1e-4 |  | **sig** |
| recon | A | +0 | [+0, +0] | – |  |  |
| recon | D | +0 | [+0, +0] | – |  |  |
| recon | E | +0 | [+0, +0] | – |  |  |
| recon | AD | +0 | [+0, +0] | – |  |  |
| recon | AE | +0 | [+0, +0] | – |  |  |
| recon | DE | +0 | [+0, +0] | – |  |  |

### err_rate (fraction)

| scope | effect | estimate | 95% CI | p | Lenth |  |
|---|---|---|---|---|---|---|
| orig | D | +0.242 | [+0.235, +0.249] | <1e-4 |  | **sig** |
| orig | E | -0.242 | [-0.249, -0.235] | <1e-4 |  | **sig** |
| orig | DE | -0.242 | [-0.249, -0.235] | <1e-4 |  | **sig** |
| orig | A | +0.208 | [+0.201, +0.215] | <1e-4 |  | **sig** |
| orig | AD | +0.208 | [+0.201, +0.215] | <1e-4 |  | **sig** |
| orig | AE | -0.208 | [-0.215, -0.201] | <1e-4 |  | **sig** |
| orig vs recon | D | +0.405 | [+0.401, +0.409] | <1e-4 |  | **sig** |
| orig vs recon | A | +0.281 | [+0.277, +0.284] | <1e-4 |  | **sig** |
| orig vs recon | AD | +0.281 | [+0.277, +0.284] | <1e-4 |  | **sig** |
| orig vs recon | X | +0.163 | [+0.16, +0.167] | <1e-4 |  | **sig** |
| orig vs recon | DX | +0.163 | [+0.16, +0.167] | <1e-4 |  | **sig** |
| orig vs recon | AE | -0.129 | [-0.132, -0.125] | <1e-4 |  | **sig** |
| recon | D | +0.568 | [+0.566, +0.571] | <1e-4 | ** | **sig** |
| recon | A | +0.354 | [+0.351, +0.356] | <1e-4 | * | **sig** |
| recon | AD | +0.354 | [+0.351, +0.356] | <1e-4 | * | **sig** |
| recon | AE | -0.0496 | [-0.0519, -0.0474] | <1e-4 |  | **sig** |
| recon | ADE | -0.0496 | [-0.0519, -0.0474] | <1e-4 |  | **sig** |
| recon | E | -0.0103 | [-0.0126, -0.0081] | <1e-4 |  | **sig** |

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
| orig | A | ×24 | [16.4, 35.1] | <1e-4 |  | **sig** |
| orig | DE | ×0.36 | [0.246, 0.526] | <1e-4 |  | **sig** |
| orig | ADE | ×0.388 | [0.266, 0.568] | <1e-4 |  | **sig** |
| orig | E | ×0.438 | [0.299, 0.64] | 0.000105 |  | **sig** |
| orig | D | ×2.24 | [1.53, 3.27] | 0.000147 |  | **sig** |
| orig | AD | ×2.07 | [1.42, 3.03] | 0.000458 |  | **sig** |
| orig vs recon | A | ×19.8 | [16.1, 24.3] | <1e-4 | * | **sig** |
| orig vs recon | X | ×0.413 | [0.337, 0.507] | <1e-4 |  | **sig** |
| orig vs recon | D | ×2.07 | [1.68, 2.54] | <1e-4 |  | **sig** |
| orig vs recon | DEX | ×1.99 | [1.62, 2.44] | <1e-4 |  | **sig** |
| orig vs recon | AD | ×1.98 | [1.61, 2.43] | <1e-4 |  | **sig** |
| orig vs recon | EX | ×1.63 | [1.33, 2] | <1e-4 |  | **sig** |
| recon | A | ×16.4 | [13.7, 19.5] | <1e-4 | * | **sig** |
| recon | D | ×1.91 | [1.61, 2.28] | <1e-4 |  | **sig** |
| recon | AD | ×1.9 | [1.59, 2.26] | <1e-4 |  | **sig** |
| recon | DE | ×1.42 | [1.2, 1.7] | 0.000253 |  | **sig** |
| recon | E | ×1.16 | [0.977, 1.39] | 0.0866 |  |  |
| recon | AE | ×0.875 | [0.735, 1.04] | 0.13 |  |  |

<details><summary>Half-normal plot: lat_p99, orig vs recon (15 contrasts)</summary>

```
 rank  z(half)  |effect|
    1    2.13     5.378  ██████████████████████████████ A **
    2    1.64     1.030  █████▊                         E 
    3    1.38     0.819  ████▋                          AD 
    4    1.19     0.749  ████▏                          ADX 
    5    1.04     0.624  ███▌                           DX 
    6    0.90     0.621  ███▌                           ADE 
    7    0.78     0.543  ███                            DE 
    8    0.67     0.492  ██▊                            ADEX 
    9    0.57     0.329  █▉                             AEX 
   10    0.48     0.283  █▋                             AX 
   11    0.39     0.275  █▌                             D 
   12    0.30     0.212  █▏                             DEX 
   13    0.21     0.205  █▏                             X 
   14    0.13     0.180  █                              EX 
   15    0.04     0.012  ▏                              AE 
```

</details>

<details><summary>Half-normal plot: throughput, orig vs recon (15 contrasts)</summary>

```
 rank  z(half)  |effect|
    1    2.13     1.127  ██████████████████████████████ D **
    2    1.64     1.045  ███████████████████████████▊   X **
    3    1.38     0.598  ███████████████▉               E *
    4    1.19     0.597  ███████████████▉               DE *
    5    1.04     0.425  ███████████▎                   DX *
    6    0.90     0.330  ████████▊                      AD *
    7    0.78     0.205  █████▌                         AE 
    8    0.67     0.151  ████                           ADE 
    9    0.57     0.117  ███▏                           ADEX 
   10    0.48     0.080  ██▏                            DEX 
   11    0.39     0.075  ██                             A 
   12    0.30     0.034  ▉                              AEX 
   13    0.21     0.020  ▌                              ADX 
   14    0.13     0.006  ▏                              AX 
   15    0.04     0.003  ▏                              EX 
```

</details>

## Per-point medians

| pt | levels ADE | orig ops/s | orig p99 | recon ops/s | recon p99 |
|---|---|---|---|---|---|
| 0 | `---` | 823k | 50.4µs | 1.29M | 46.8µs |
| 1 | `+--` | 456k | 1.24ms | 784k | 477µs |
| 2 | `-+-` | 1.43M | 29.6µs | 4.14M | 8.26µs |
| 3 | `++-` | 1.68M | 954µs | 4.34M | 3.42ms |
| 4 | `--+` | 630k | 57.1µs | 998k | 38.1µs |
| 5 | `+-+` | 652k | 2.54ms | 901k | 1.12ms |
| 6 | `-++` | 626k | 59.1µs | 1.72M | 85.5µs |
| 7 | `+++` | 667k | 2.38ms | 2.32M | 4.88ms |

