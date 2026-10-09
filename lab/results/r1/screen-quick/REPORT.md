# DoE screen: orig=orig vs recon=recon

`20261009-232533-screen-orig-recon` · 2026-10-09T23:25:33+05:30 · go1.27.1 windows/amd64 · 20 CPUs (GOMAXPROCS 20) · 192 trials (0 failed) in 91s · 3 reps · warm-up 100ms + window 300ms · seed 1

Command: `go run ./lab/cmd/doe -quick -q -out lab/results/r1/screen-quick`

## Verdict

X is the implementation factor (−1 baseline, +1 candidate), crossed with every design run, so its main effect is the candidate's change averaged over the whole design. Log-scale responses show the ratio candidate/baseline of geometric means; raw responses show the difference. Significance is Bonferroni-corrected over all contrasts of a response; ratios within 5% count as *same*.

### Key responses

| response | arms | baseline | candidate | change [95% CI] | p | verdict | strongest X×factor |
|---|---|---|---|---|---|---|---|
| throughput | orig→recon | 239k | 534k | ×2.23 [2.2, 2.26] | <1e-4 | **better** | DX (×1.83) |
| goodput | orig→recon | 119k | 162k | ×1.37 [1.32, 1.41] | <1e-4 | **better** | DX (×1.12) |
| lat_p50 | orig→recon | 48.2µs | 29.5µs | ×0.611 [0.598, 0.625] | <1e-4 | **better** | CX (×1.61) |
| lat_p99 | orig→recon | 992µs | 663µs | ×0.669 [0.629, 0.711] | <1e-4 | **better** | KX (×0.826) |
| lat_p999 | orig→recon | 2.85ms | 1.39ms | ×0.487 [0.453, 0.524] | <1e-4 | **better** | DX (×0.777) |
| wait_p99 | orig→recon | 482µs | 293µs | ×0.608 [0.567, 0.652] | <1e-4 | **better** | AX (×1.65) |
| allocs_op | orig→recon | 14.8 | 2.54 | -12.3 [-12.3, -12.2] | <1e-4 | **better** | EX (-1.49) |
| bytes_op | orig→recon | 723 | 170 | -553 [-555, -550] | <1e-4 | **better** | EX (-105) |
| peak_goroutines | orig→recon | 3769 | 2.08 | -3.77e+03 [-4.37e+03, -3.16e+03] | <1e-4 | **better** | EX (+3.6e+03) |
| err_rate | orig→recon | 21.67% | 35.11% | +0.134 [+0.133, +0.136] | <1e-4 | **worse** | DX (+0.136) |
| cap_violations | orig→recon | 0 | 0 | +0 [+0, +0] | – | same | – |
| key_violations | orig→recon | 0 | 0 | +0 [+0, +0] | – | same | – |
| drain | orig→recon | 201µs | 131µs | ×0.653 [0.57, 0.749] | <1e-4 | **better** | EX (×1.72) |

<details><summary>All other responses</summary>

| response | arms | baseline | candidate | change [95% CI] | p | verdict | strongest X×factor |
|---|---|---|---|---|---|---|---|
| lat_p90 | orig→recon | 309µs | 151µs | ×0.489 [0.477, 0.501] | <1e-4 | **better** | EX (×0.533) |
| lat_max | orig→recon | 8.78ms | 4.39ms | ×0.499 [0.451, 0.553] | <1e-4 | **better** | EX (×1.46) |
| wait_p50 | orig→recon | 7.79µs | 2.93µs | ×0.377 [0.367, 0.386] | <1e-4 | **better** | EX (×0.695) |
| exec_p50 | orig→recon | 13µs | 8.62µs | ×0.663 [0.652, 0.674] | <1e-4 | **better** | CX (×1.5) |
| exec_p99 | orig→recon | 126µs | 154µs | ×1.22 [1.07, 1.39] | 0.00378 | same | DX (×2.53) |
| peak_heap | orig→recon | 38.7 | 32 | ×0.826 [0.777, 0.879] | <1e-4 | **better** | – |
| gc_per_s | orig→recon | 15.7 | 5.66 | -10.1 [-11.1, -9.04] | <1e-4 | **better** | CX (+6.87) |
| retries_op | orig→recon | 0.0474 | 0.0376 | -0.0098 [-0.0102, -0.00939] | <1e-4 | differs | GX (-0.0098) |
| fairness | orig→recon | 0.948 | 0.99 | +0.0423 [+0.0351, +0.0495] | <1e-4 | **better** | DX (+0.0423) |
| stats_lag | orig→recon | 0.0104 | 0 | -0.0104 [-0.031, +0.0102] | 0.319 | same | – |
| residual_goroutines | orig→recon | 0 | 0 | +0 [+0, +0] | – | same | – |

</details>

## Noise

Replicate error of a single trial (pooled within-cell SD) as a coefficient of variation: the ±1σ spread between identical trials.

| response | orig | recon |
|---|---|---|
| throughput | 5.5% | 4.7% |
| goodput | 15.8% | 4.9% |
| lat_p50 | 9.9% | 5.6% |
| lat_p90 | 7.0% | 10.7% |
| lat_p99 | 33.9% | 8.8% |
| lat_p999 | 33.0% | 25.3% |
| lat_max | 34.4% | 51.0% |
| wait_p50 | 9.7% | 8.5% |
| wait_p99 | 37.3% | 14.5% |
| exec_p50 | 6.8% | 5.3% |
| exec_p99 | 92.2% | 8.5% |
| allocs_op | 2.0% | 0.6% |
| bytes_op | 1.6% | 0.7% |
| peak_goroutines | 79.3% | 68.8% |
| peak_heap | 26.2% | 21.9% |
| gc_per_s | 29.4% | 35.6% |
| err_rate | 3.2% | 0.6% |
| retries_op | 2.9% | 4.1% |
| fairness | 3.7% | 0.3% |
| cap_violations | – | – |
| key_violations | – | – |
| drain | 83.1% | 35.3% |
| stats_lag | 979.8% | – |
| residual_goroutines | – | – |

## Error classes

Share of all calls, over all trials of each arm. A class is the implementation-neutral error class, the outcome what `OutcomeOf` reports.

| class/outcome | orig | recon |
|---|---|---|
| deadline/permanent | 0.221% | 0.078% |
| deadline/uncertain | 0.014% | 0.006% |
| other/transient | 0.028% | 0.016% |
| overloaded/throttled | 43.742% | 66.838% |

## Design

32 runs × 2 arms × 3 reps, randomised within each replicate block. 2^(10−5) fractional factorial, resolution IV; generators F=ABCD, G=ABCE, H=ABDE, J=ACDE, K=BCDE.

| symbol | factor | low (−1) | high (+1) |
|---|---|---|---|
| A | clients | 8 | 256 |
| B | capacity | 4 | 16 |
| C | service | 0 | 100µs |
| D | admission | wait | reject |
| E | keys | none | 64·zipf1.2 |
| F | timeout | none | 1ms |
| G | failures | 0%·1try | 10%·3tries |
| H | weights | all 1u | 25% cap/2 |
| J | resources | 1 | 3 |
| K | cancel | 0% | 5% |

Defining relation: I = DEFG = CEFH = CDGH = BEFJ = BDGJ = BCHJ = AEFK = ADGK = ACHK = ABJK (+ 21 longer words, listed in summary.json)

Aliased two-factor interactions (each chain is one estimate): AB=JK, AC=HK, BC=HJ, AD=GK, BD=GJ, CD=GH, AE=FK, BE=FJ, CE=FH, DE=FG, AF=EK, BF=EJ, CF=EH, DF=EG, EF=DG=CH=BJ=AK, AG=DK, BG=DJ, CG=DH, AH=CK, BH=CJ, AJ=BK. Interactions with X are clear of every two-factor interaction.

<details><summary>Design matrix</summary>

```
run A B C D E F G H J K
  0 - - - - - + + + + +
  1 + - - - - - - - - +
  2 - + - - - - - - + -
  3 + + - - - + + + - -
  4 - - + - - - - + - -
  5 + - + - - + + - + -
  6 - + + - - + + - - +
  7 + + + - - - - + + +
  8 - - - + - - + - - -
  9 + - - + - + - + + -
 10 - + - + - + - + - +
 11 + + - + - - + - + +
 12 - - + + - + - - + +
 13 + - + + - - + + - +
 14 - + + + - - + + + -
 15 + + + + - + - - - -
 16 - - - - + + - - - -
 17 + - - - + - + + + -
 18 - + - - + - + + - +
 19 + + - - + + - - + +
 20 - - + - + - + - + +
 21 + - + - + + - + - +
 22 - + + - + + - + + -
 23 + + + - + - + - - -
 24 - - - + + - - + + +
 25 + - - + + + + - - +
 26 - + - + + + + - + -
 27 + + - + + - - + - -
 28 - - + + + + + + - -
 29 + - + + + - - - + -
 30 - + + + + - - - - +
 31 + + + + + + + + + +
```

</details>

## Effects

Top effects per scope by |t|. An effect is the change from the low to the high level (a ratio for log-scale responses). `*`/`**`: beyond Lenth's ME/SME; **sig**: Bonferroni p < 0.05/m on replicate error. effects.csv has every contrast.

### throughput (ops/s)

| scope | effect | estimate | 95% CI | p | Lenth |  |
|---|---|---|---|---|---|---|
| orig | C | ×0.11 | [0.108, 0.113] | <1e-4 | ** | **sig** |
| orig | D | ×3.24 | [3.17, 3.31] | <1e-4 | ** | **sig** |
| orig | E | ×0.372 | [0.364, 0.38] | <1e-4 | * | **sig** |
| orig | DE=FG | ×0.448 | [0.438, 0.458] | <1e-4 | * | **sig** |
| orig | CD=GH | ×1.86 | [1.82, 1.9] | <1e-4 | * | **sig** |
| orig | CE=FH | ×0.601 | [0.588, 0.614] | <1e-4 |  | **sig** |
| orig vs recon | C | ×0.126 | [0.124, 0.128] | <1e-4 | ** | **sig** |
| orig vs recon | D | ×5.94 | [5.86, 6.03] | <1e-4 | ** | **sig** |
| orig vs recon | CD=GH | ×2.58 | [2.54, 2.61] | <1e-4 | ** | **sig** |
| orig vs recon | X | ×2.23 | [2.2, 2.26] | <1e-4 | ** | **sig** |
| orig vs recon | E | ×0.495 | [0.488, 0.502] | <1e-4 | ** | **sig** |
| orig vs recon | DX | ×1.83 | [1.81, 1.86] | <1e-4 | * | **sig** |
| recon | D | ×10.9 | [10.7, 11.1] | <1e-4 | ** | **sig** |
| recon | C | ×0.144 | [0.141, 0.147] | <1e-4 | ** | **sig** |
| recon | CD=GH | ×3.56 | [3.5, 3.63] | <1e-4 | * | **sig** |
| recon | BD=GJ | ×0.54 | [0.53, 0.55] | <1e-4 |  | **sig** |
| recon | CF=EH | ×1.83 | [1.8, 1.87] | <1e-4 |  | **sig** |
| recon | AD=GK | ×1.78 | [1.74, 1.81] | <1e-4 |  | **sig** |

### goodput (ops/s)

| scope | effect | estimate | 95% CI | p | Lenth |  |
|---|---|---|---|---|---|---|
| orig | C | ×0.0608 | [0.0573, 0.0646] | <1e-4 | ** | **sig** |
| orig | AE=FK | ×1.8 | [1.69, 1.91] | <1e-4 |  | **sig** |
| orig | A | ×0.562 | [0.529, 0.596] | <1e-4 |  | **sig** |
| orig | BH=CJ | ×1.58 | [1.49, 1.68] | <1e-4 |  | **sig** |
| orig | AD=GK | ×0.667 | [0.628, 0.708] | <1e-4 |  | **sig** |
| orig | B | ×1.39 | [1.31, 1.47] | <1e-4 |  | **sig** |
| orig vs recon | C | ×0.0618 | [0.0599, 0.0637] | <1e-4 | ** | **sig** |
| orig vs recon | A | ×0.543 | [0.527, 0.56] | <1e-4 | ** | **sig** |
| orig vs recon | AE=FK | ×1.64 | [1.59, 1.69] | <1e-4 | ** | **sig** |
| orig vs recon | AD=GK | ×0.662 | [0.642, 0.683] | <1e-4 | ** | **sig** |
| orig vs recon | B | ×1.49 | [1.44, 1.54] | <1e-4 | * | **sig** |
| orig vs recon | AC=HK | ×1.48 | [1.43, 1.52] | <1e-4 | * | **sig** |
| recon | C | ×0.0627 | [0.0615, 0.0639] | <1e-4 | ** | **sig** |
| recon | A | ×0.525 | [0.515, 0.536] | <1e-4 | * | **sig** |
| recon | AC=HK | ×1.8 | [1.76, 1.83] | <1e-4 |  | **sig** |
| recon | BF=EJ | ×1.66 | [1.63, 1.69] | <1e-4 |  | **sig** |
| recon | B | ×1.6 | [1.56, 1.63] | <1e-4 |  | **sig** |
| recon | CD=GH | ×1.57 | [1.54, 1.6] | <1e-4 |  | **sig** |

### lat_p50 (ns)

| scope | effect | estimate | 95% CI | p | Lenth |  |
|---|---|---|---|---|---|---|
| orig | C | ×16 | [15.4, 16.7] | <1e-4 | ** | **sig** |
| orig | DE=FG | ×3.89 | [3.74, 4.04] | <1e-4 | ** | **sig** |
| orig | D | ×0.265 | [0.255, 0.276] | <1e-4 | ** | **sig** |
| orig | A | ×3.52 | [3.39, 3.66] | <1e-4 | ** | **sig** |
| orig | BH=CJ | ×2.67 | [2.57, 2.77] | <1e-4 | ** | **sig** |
| orig | AD=GK | ×0.38 | [0.366, 0.395] | <1e-4 | ** | **sig** |
| orig vs recon | C | ×25.8 | [25.3, 26.4] | <1e-4 | ** | **sig** |
| orig vs recon | DE=FG | ×4.1 | [4.01, 4.2] | <1e-4 | ** | **sig** |
| orig vs recon | D | ×0.284 | [0.278, 0.291] | <1e-4 | ** | **sig** |
| orig vs recon | A | ×3.4 | [3.33, 3.48] | <1e-4 | ** | **sig** |
| orig vs recon | E | ×0.36 | [0.352, 0.368] | <1e-4 | ** | **sig** |
| orig vs recon | AD=GK | ×0.383 | [0.374, 0.391] | <1e-4 | ** | **sig** |
| recon | C | ×41.6 | [40.7, 42.5] | <1e-4 | ** | **sig** |
| recon | DE=FG | ×4.33 | [4.24, 4.43] | <1e-4 | ** | **sig** |
| recon | E | ×0.257 | [0.252, 0.263] | <1e-4 | ** | **sig** |
| recon | A | ×3.29 | [3.21, 3.36] | <1e-4 | ** | **sig** |
| recon | D | ×0.305 | [0.298, 0.312] | <1e-4 | ** | **sig** |
| recon | AE=FK | ×0.372 | [0.364, 0.381] | <1e-4 | ** | **sig** |

### lat_p99 (ns)

| scope | effect | estimate | 95% CI | p | Lenth |  |
|---|---|---|---|---|---|---|
| orig | A | ×31.7 | [28.1, 35.7] | <1e-4 | ** | **sig** |
| orig | C | ×7.04 | [6.25, 7.93] | <1e-4 | ** | **sig** |
| orig | E | ×2.77 | [2.46, 3.12] | <1e-4 | ** | **sig** |
| orig | D | ×0.647 | [0.575, 0.729] | <1e-4 | * | **sig** |
| orig | CE=FH | ×1.43 | [1.27, 1.61] | <1e-4 |  | **sig** |
| orig | BC=HJ | ×0.741 | [0.658, 0.834] | <1e-4 |  | **sig** |
| orig vs recon | A | ×36.3 | [34.2, 38.6] | <1e-4 | ** | **sig** |
| orig vs recon | C | ×6.62 | [6.22, 7.03] | <1e-4 | ** | **sig** |
| orig vs recon | E | ×2.45 | [2.31, 2.61] | <1e-4 | ** | **sig** |
| orig vs recon | CDX | ×0.622 | [0.585, 0.661] | <1e-4 | * | **sig** |
| orig vs recon | CD=GH | ×0.633 | [0.595, 0.673] | <1e-4 | * | **sig** |
| orig vs recon | X | ×0.669 | [0.629, 0.711] | <1e-4 | * | **sig** |
| recon | A | ×41.7 | [40.3, 43.1] | <1e-4 | ** | **sig** |
| recon | C | ×6.22 | [6.01, 6.44] | <1e-4 | ** | **sig** |
| recon | CD=GH | ×0.394 | [0.38, 0.408] | <1e-4 | ** | **sig** |
| recon | E | ×2.17 | [2.1, 2.25] | <1e-4 | * | **sig** |
| recon | AC=HK | ×0.622 | [0.6, 0.643] | <1e-4 | * | **sig** |
| recon | BF=EJ | ×0.634 | [0.612, 0.656] | <1e-4 | * | **sig** |

### lat_p999 (ns)

| scope | effect | estimate | 95% CI | p | Lenth |  |
|---|---|---|---|---|---|---|
| orig | A | ×32.1 | [28.6, 36.1] | <1e-4 | ** | **sig** |
| orig | C | ×2.67 | [2.38, 3] | <1e-4 | ** | **sig** |
| orig | DE=FG | ×0.635 | [0.565, 0.713] | <1e-4 |  | **sig** |
| orig | BH=CJ | ×0.653 | [0.581, 0.733] | <1e-4 |  | **sig** |
| orig | AE=FK | ×0.693 | [0.617, 0.778] | <1e-4 |  | **sig** |
| orig | CE=FH | ×1.43 | [1.27, 1.6] | <1e-4 |  | **sig** |
| orig vs recon | A | ×36.4 | [33.8, 39.1] | <1e-4 | ** | **sig** |
| orig vs recon | C | ×2.99 | [2.78, 3.22] | <1e-4 | ** | **sig** |
| orig vs recon | X | ×0.487 | [0.453, 0.524] | <1e-4 | ** | **sig** |
| orig vs recon | AJ=BK | ×0.702 | [0.652, 0.755] | <1e-4 | * | **sig** |
| orig vs recon | CDX | ×0.708 | [0.658, 0.762] | <1e-4 | * | **sig** |
| orig vs recon | E | ×1.4 | [1.3, 1.51] | <1e-4 | * | **sig** |
| recon | A | ×41.2 | [37.6, 45.1] | <1e-4 | ** | **sig** |
| recon | C | ×3.35 | [3.05, 3.67] | <1e-4 | ** | **sig** |
| recon | CD=GH | ×0.562 | [0.513, 0.616] | <1e-4 | * | **sig** |
| recon | AJ=BK | ×0.587 | [0.535, 0.643] | <1e-4 | * | **sig** |
| recon | AC=HK | ×0.595 | [0.542, 0.652] | <1e-4 | * | **sig** |
| recon | E | ×1.5 | [1.37, 1.65] | <1e-4 | * | **sig** |

### wait_p99 (ns)

| scope | effect | estimate | 95% CI | p | Lenth |  |
|---|---|---|---|---|---|---|
| orig | A | ×50.9 | [44.7, 57.9] | <1e-4 | ** | **sig** |
| orig | E | ×7.79 | [6.84, 8.86] | <1e-4 | ** | **sig** |
| orig | C | ×4.72 | [4.15, 5.38] | <1e-4 | ** | **sig** |
| orig | D | ×0.34 | [0.299, 0.387] | <1e-4 | * | **sig** |
| orig | CE=FH | ×2.79 | [2.45, 3.17] | <1e-4 | * | **sig** |
| orig | DE=FG | ×2.49 | [2.18, 2.83] | <1e-4 | * | **sig** |
| orig vs recon | A | ×83.8 | [78.1, 89.8] | <1e-4 | ** | **sig** |
| orig vs recon | E | ×5.18 | [4.83, 5.56] | <1e-4 | ** | **sig** |
| orig vs recon | D | ×0.298 | [0.278, 0.32] | <1e-4 | ** | **sig** |
| orig vs recon | C | ×3.17 | [2.96, 3.4] | <1e-4 | ** | **sig** |
| orig vs recon | CD=GH | ×0.401 | [0.374, 0.43] | <1e-4 | * | **sig** |
| orig vs recon | ADX | ×1.98 | [1.84, 2.12] | <1e-4 | * | **sig** |
| recon | A | ×138 | [131, 146] | <1e-4 | ** | **sig** |
| recon | CD=GH | ×0.208 | [0.197, 0.22] | <1e-4 | * | **sig** |
| recon | D | ×0.262 | [0.248, 0.277] | <1e-4 | * | **sig** |
| recon | E | ×3.45 | [3.26, 3.64] | <1e-4 | * | **sig** |
| recon | AD=GK | ×2.53 | [2.39, 2.67] | <1e-4 | * | **sig** |
| recon | C | ×2.13 | [2.01, 2.25] | <1e-4 |  | **sig** |

### allocs_op (allocs)

| scope | effect | estimate | 95% CI | p | Lenth |  |
|---|---|---|---|---|---|---|
| orig | F | +3.91 | [+3.79, +4.03] | <1e-4 | ** | **sig** |
| orig | E | +2.58 | [+2.46, +2.7] | <1e-4 | ** | **sig** |
| orig | D | -1.8 | [-1.92, -1.68] | <1e-4 | * | **sig** |
| orig | DE=FG | +1.54 | [+1.42, +1.66] | <1e-4 | * | **sig** |
| orig | A | +1.08 | [+0.96, +1.2] | <1e-4 |  | **sig** |
| orig | DF=EG | -0.961 | [-1.08, -0.839] | <1e-4 |  | **sig** |
| orig vs recon | X | -12.3 | [-12.3, -12.2] | <1e-4 | ** | **sig** |
| orig vs recon | F | +3.47 | [+3.4, +3.53] | <1e-4 | ** | **sig** |
| orig vs recon | D | -2.13 | [-2.19, -2.07] | <1e-4 | ** | **sig** |
| orig vs recon | EX | -1.49 | [-1.55, -1.43] | <1e-4 | ** | **sig** |
| orig vs recon | DF=EG | -1.42 | [-1.48, -1.36] | <1e-4 | ** | **sig** |
| orig vs recon | E | +1.09 | [+1.03, +1.15] | <1e-4 | * | **sig** |
| recon | F | +3.02 | [+3.01, +3.02] | <1e-4 | ** | **sig** |
| recon | D | -2.45 | [-2.46, -2.45] | <1e-4 | ** | **sig** |
| recon | DF=EG | -1.87 | [-1.88, -1.87] | <1e-4 | ** | **sig** |
| recon | CD=GH | -0.734 | [-0.74, -0.728] | <1e-4 |  | **sig** |
| recon | BD=GJ | +0.616 | [+0.61, +0.623] | <1e-4 |  | **sig** |
| recon | DE=FG | +0.497 | [+0.491, +0.504] | <1e-4 |  | **sig** |

### bytes_op (B)

| scope | effect | estimate | 95% CI | p | Lenth |  |
|---|---|---|---|---|---|---|
| orig | F | +264 | [+260, +269] | <1e-4 | ** | **sig** |
| orig | D | -205 | [-210, -200] | <1e-4 | ** | **sig** |
| orig | E | +178 | [+173, +182] | <1e-4 | * | **sig** |
| orig | DE=FG | +160 | [+156, +165] | <1e-4 | * | **sig** |
| orig | DF=EG | -87.4 | [-92, -82.8] | <1e-4 |  | **sig** |
| orig | G | +60.2 | [+55.6, +64.7] | <1e-4 |  | **sig** |
| orig vs recon | X | -553 | [-555, -550] | <1e-4 | ** | **sig** |
| orig vs recon | F | +240 | [+238, +242] | <1e-4 | ** | **sig** |
| orig vs recon | D | -214 | [-216, -212] | <1e-4 | ** | **sig** |
| orig vs recon | DF=EG | -115 | [-118, -113] | <1e-4 | ** | **sig** |
| orig vs recon | EX | -105 | [-108, -103] | <1e-4 | ** | **sig** |
| orig vs recon | DE=FG | +99.4 | [+97.1, +102] | <1e-4 | ** | **sig** |
| recon | D | -223 | [-223, -222] | <1e-4 | ** | **sig** |
| recon | F | +216 | [+215, +216] | <1e-4 | ** | **sig** |
| recon | DF=EG | -143 | [-144, -143] | <1e-4 | ** | **sig** |
| recon | CD=GH | -68.2 | [-68.7, -67.7] | <1e-4 | * | **sig** |
| recon | BD=GJ | +50.2 | [+49.8, +50.7] | <1e-4 |  | **sig** |
| recon | AB=JK | -42.9 | [-43.4, -42.4] | <1e-4 |  | **sig** |

### peak_goroutines (goroutines)

| scope | effect | estimate | 95% CI | p | Lenth |  |
|---|---|---|---|---|---|---|
| orig | E | -7.2e+03 | [-8.42e+03, -5.98e+03] | <1e-4 | ** | **sig** |
| orig | D | +7.12e+03 | [+5.9e+03, +8.34e+03] | <1e-4 | ** | **sig** |
| orig | DE=FG | -7.11e+03 | [-8.33e+03, -5.89e+03] | <1e-4 | ** | **sig** |
| orig | A | +7.07e+03 | [+5.85e+03, +8.29e+03] | <1e-4 | ** | **sig** |
| orig | AD=GK | +6.94e+03 | [+5.72e+03, +8.16e+03] | <1e-4 | ** | **sig** |
| orig | BH=CJ | -6.92e+03 | [-8.14e+03, -5.7e+03] | <1e-4 | ** | **sig** |
| orig vs recon | X | -3.77e+03 | [-4.37e+03, -3.16e+03] | <1e-4 | ** | **sig** |
| orig vs recon | E | -3.6e+03 | [-4.2e+03, -3e+03] | <1e-4 | ** | **sig** |
| orig vs recon | EX | +3.6e+03 | [+3e+03, +4.2e+03] | <1e-4 | ** | **sig** |
| orig vs recon | D | +3.56e+03 | [+2.96e+03, +4.16e+03] | <1e-4 | ** | **sig** |
| orig vs recon | DX | -3.56e+03 | [-4.16e+03, -2.96e+03] | <1e-4 | ** | **sig** |
| orig vs recon | DEX | +3.55e+03 | [+2.95e+03, +4.16e+03] | <1e-4 | ** | **sig** |
| recon | K | +2.92 | [+2.33, +3.5] | <1e-4 | ** | **sig** |
| recon | A | +1.96 | [+1.37, +2.54] | <1e-4 | * | **sig** |
| recon | EF=DG=CH=BJ=AK | +1.88 | [+1.29, +2.46] | <1e-4 | * | **sig** |
| recon | BF=EJ | +1.62 | [+1.04, +2.21] | <1e-4 | * | **sig** |
| recon | CD=GH | +1.5 | [+0.916, +2.08] | <1e-4 | * | **sig** |
| recon | CG=DH | +1.38 | [+0.791, +1.96] | <1e-4 | * | **sig** |

### err_rate (fraction)

| scope | effect | estimate | 95% CI | p | Lenth |  |
|---|---|---|---|---|---|---|
| orig | D | +0.407 | [+0.404, +0.409] | <1e-4 | ** | **sig** |
| orig | DE=FG | -0.22 | [-0.223, -0.217] | <1e-4 | * | **sig** |
| orig | E | -0.218 | [-0.221, -0.216] | <1e-4 | * | **sig** |
| orig | C | +0.172 | [+0.17, +0.175] | <1e-4 |  | **sig** |
| orig | A | +0.167 | [+0.164, +0.17] | <1e-4 |  | **sig** |
| orig | CD=GH | +0.161 | [+0.158, +0.163] | <1e-4 |  | **sig** |
| orig vs recon | D | +0.543 | [+0.541, +0.544] | <1e-4 | ** | **sig** |
| orig vs recon | A | +0.211 | [+0.209, +0.212] | <1e-4 | ** | **sig** |
| orig vs recon | AD=GK | +0.2 | [+0.199, +0.202] | <1e-4 | ** | **sig** |
| orig vs recon | C | +0.15 | [+0.148, +0.151] | <1e-4 | ** | **sig** |
| orig vs recon | CD=GH | +0.138 | [+0.137, +0.139] | <1e-4 | ** | **sig** |
| orig vs recon | DX | +0.136 | [+0.135, +0.137] | <1e-4 | ** | **sig** |
| recon | D | +0.679 | [+0.678, +0.679] | <1e-4 | ** | **sig** |
| recon | A | +0.255 | [+0.254, +0.256] | <1e-4 | ** | **sig** |
| recon | AD=GK | +0.244 | [+0.243, +0.245] | <1e-4 | ** | **sig** |
| recon | C | +0.127 | [+0.126, +0.127] | <1e-4 | * | **sig** |
| recon | CD=GH | +0.115 | [+0.115, +0.116] | <1e-4 |  | **sig** |
| recon | BD=GJ | -0.0974 | [-0.0983, -0.0966] | <1e-4 |  | **sig** |

### cap_violations (count)

| scope | effect | estimate | 95% CI | p | Lenth |  |
|---|---|---|---|---|---|---|
| orig | A | +0 | [+0, +0] | – |  |  |
| orig | B | +0 | [+0, +0] | – |  |  |
| orig | C | +0 | [+0, +0] | – |  |  |
| orig | D | +0 | [+0, +0] | – |  |  |
| orig | E | +0 | [+0, +0] | – |  |  |
| orig | F | +0 | [+0, +0] | – |  |  |
| orig vs recon | A | +0 | [+0, +0] | – |  |  |
| orig vs recon | B | +0 | [+0, +0] | – |  |  |
| orig vs recon | C | +0 | [+0, +0] | – |  |  |
| orig vs recon | D | +0 | [+0, +0] | – |  |  |
| orig vs recon | E | +0 | [+0, +0] | – |  |  |
| orig vs recon | F | +0 | [+0, +0] | – |  |  |
| recon | A | +0 | [+0, +0] | – |  |  |
| recon | B | +0 | [+0, +0] | – |  |  |
| recon | C | +0 | [+0, +0] | – |  |  |
| recon | D | +0 | [+0, +0] | – |  |  |
| recon | E | +0 | [+0, +0] | – |  |  |
| recon | F | +0 | [+0, +0] | – |  |  |

### key_violations (count)

| scope | effect | estimate | 95% CI | p | Lenth |  |
|---|---|---|---|---|---|---|
| orig | A | +0 | [+0, +0] | – |  |  |
| orig | B | +0 | [+0, +0] | – |  |  |
| orig | C | +0 | [+0, +0] | – |  |  |
| orig | D | +0 | [+0, +0] | – |  |  |
| orig | E | +0 | [+0, +0] | – |  |  |
| orig | F | +0 | [+0, +0] | – |  |  |
| orig vs recon | A | +0 | [+0, +0] | – |  |  |
| orig vs recon | B | +0 | [+0, +0] | – |  |  |
| orig vs recon | C | +0 | [+0, +0] | – |  |  |
| orig vs recon | D | +0 | [+0, +0] | – |  |  |
| orig vs recon | E | +0 | [+0, +0] | – |  |  |
| orig vs recon | F | +0 | [+0, +0] | – |  |  |
| recon | A | +0 | [+0, +0] | – |  |  |
| recon | B | +0 | [+0, +0] | – |  |  |
| recon | C | +0 | [+0, +0] | – |  |  |
| recon | D | +0 | [+0, +0] | – |  |  |
| recon | E | +0 | [+0, +0] | – |  |  |
| recon | F | +0 | [+0, +0] | – |  |  |

### drain (ms)

| scope | effect | estimate | 95% CI | p | Lenth |  |
|---|---|---|---|---|---|---|
| orig | A | ×11.2 | [8.76, 14.3] | <1e-4 | ** | **sig** |
| orig | E | ×0.337 | [0.264, 0.432] | <1e-4 | * | **sig** |
| orig | D | ×2.83 | [2.21, 3.63] | <1e-4 | * | **sig** |
| orig | C | ×2.08 | [1.63, 2.66] | <1e-4 | * | **sig** |
| orig | BH=CJ | ×0.501 | [0.391, 0.641] | <1e-4 | * | **sig** |
| orig | DE=FG | ×0.503 | [0.393, 0.643] | <1e-4 | * | **sig** |
| orig vs recon | A | ×9.46 | [8.26, 10.8] | <1e-4 | ** | **sig** |
| orig vs recon | C | ×2.64 | [2.3, 3.03] | <1e-4 | ** | **sig** |
| orig vs recon | D | ×2.39 | [2.08, 2.74] | <1e-4 | ** | **sig** |
| orig vs recon | E | ×0.581 | [0.507, 0.666] | <1e-4 | * | **sig** |
| orig vs recon | EX | ×1.72 | [1.5, 1.97] | <1e-4 | * | **sig** |
| orig vs recon | AC=HK | ×0.589 | [0.514, 0.675] | <1e-4 | * | **sig** |
| recon | A | ×7.99 | [7.06, 9.04] | <1e-4 | ** | **sig** |
| recon | C | ×3.35 | [2.96, 3.79] | <1e-4 | ** | **sig** |
| recon | AC=HK | ×0.421 | [0.372, 0.476] | <1e-4 | ** | **sig** |
| recon | D | ×2.01 | [1.78, 2.28] | <1e-4 | ** | **sig** |
| recon | AD=GK | ×1.8 | [1.59, 2.04] | <1e-4 | ** | **sig** |
| recon | DE=FG | ×1.27 | [1.12, 1.44] | 0.000238 | * | **sig** |

<details><summary>Half-normal plot: lat_p99, orig vs recon (63 contrasts)</summary>

```
 rank  z(half)  |effect|
    1    2.65     5.183  ██████████████████████████████ A **
    2    2.26     2.726  ███████████████▊               C **
    3    2.06     1.295  ███████▌                       E **
    4    1.91     0.686  ████                           CDX *
    5    1.80     0.659  ███▉                           CD=GH *
    6    1.71     0.581  ███▍                           X *
    7    1.63     0.564  ███▎                           D *
    8    1.56     0.526  ███                            AC=HK *
    9    1.49     0.368  ██▏                            AJ=BK 
   10    1.44     0.367  ██▏                            BF=EJ 
   11    1.38     0.356  ██                             ADX 
   12    1.33     0.317  █▉                             CEX 
   13    1.29     0.303  █▊                             BG=DJ 
   14    1.24     0.291  █▋                             DEX 
   15    1.20     0.290  █▋                             BFX 
   16    1.16     0.276  █▋                             KX 
   17    1.12     0.265  █▌                             EF=DG=CH=BJ=AK 
   18    1.09     0.262  █▌                             G 
   19    1.05     0.253  █▌                             AGX 
   20    1.02     0.251  █▌                             BHX 
   21    0.98     0.245  █▍                             AB=JK 
   22    0.95     0.230  █▍                             BC=HJ 
   23    0.92     0.228  █▍                             AJX 
   24    0.89     0.203  █▏                             DF=EG 
   25    0.86     0.203  █▏                             BCX 
   26    0.83     0.198  █▏                             AX 
   27    0.81     0.196  █▏                             JX 
   28    0.78     0.195  █▏                             CE=FH 
   29    0.75     0.193  █▏                             AEX 
   30    0.73     0.176  █                              EX 
   31    0.70     0.167  █                              FX 
   32    0.67     0.160  ▉                              ACX 
   33    0.65     0.160  ▉                              EFX 
   34    0.63     0.148  ▉                              H 
   35    0.60     0.147  ▉                              BH=CJ 
   36    0.58     0.146  ▉                              AD=GK 
   37    0.55     0.143  ▉                              AH=CK 
   38    0.53     0.134  ▊                              HX 
   39    0.51     0.129  ▊                              DFX 
   40    0.49     0.128  ▊                              J 
   41    0.46     0.127  ▊                              BGX 
   42    0.44     0.125  ▊                              CF=EH 
   43    0.42     0.124  ▊                              AG=DK 
   44    0.40     0.117  ▋                              BDX 
   45    0.38     0.106  ▋                              BD=GJ 
   46    0.36     0.105  ▋                              AF=EK 
   47    0.33     0.103  ▋                              F 
   48    0.31     0.099  ▋                              K 
   49    0.29     0.089  ▌                              CX 
   50    0.27     0.073  ▍                              GX 
   51    0.25     0.072  ▍                              BX 
   52    0.23     0.065  ▍                              BE=FJ 
   53    0.21     0.064  ▍                              DX 
   54    0.19     0.054  ▍                              DE=FG 
   55    0.17     0.049  ▎                              ABX 
   56    0.15     0.046  ▎                              BEX 
   57    0.13     0.030  ▏                              CGX 
   58    0.11     0.027  ▏                              CFX 
   59    0.09     0.026  ▏                              AHX 
   60    0.07     0.014  ▏                              B 
   61    0.05     0.008                                 AFX 
   62    0.03     0.005                                 CG=DH 
   63    0.01     0.005                                 AE=FK 
```

</details>

<details><summary>Half-normal plot: throughput, orig vs recon (63 contrasts)</summary>

```
 rank  z(half)  |effect|
    1    2.65     2.989  ██████████████████████████████ C **
    2    2.26     2.571  █████████████████████████▊     D **
    3    2.06     1.365  █████████████▊                 CD=GH **
    4    1.91     1.156  ███████████▋                   X **
    5    1.80     1.016  ██████████▎                    E **
    6    1.71     0.874  ████████▊                      DX *
    7    1.63     0.733  ███████▍                       BD=GJ *
    8    1.56     0.669  ██████▊                        DE=FG *
    9    1.49     0.649  ██████▌                        CF=EH *
   10    1.44     0.640  ██████▍                        AD=GK *
   11    1.38     0.540  █████▍                         AF=EK *
   12    1.33     0.490  ████▉                          DEX 
   13    1.29     0.476  ████▊                          BC=HJ 
   14    1.24     0.467  ████▊                          CDX 
   15    1.20     0.454  ████▌                          DF=EG 
   16    1.16     0.411  ████▏                          EX 
   17    1.12     0.370  ███▊                           A 
   18    1.09     0.370  ███▊                           B 
   19    1.05     0.370  ███▊                           CEX 
   20    1.02     0.365  ███▋                           CE=FH 
   21    0.98     0.343  ███▌                           AC=HK 
   22    0.95     0.318  ███▎                           AH=CK 
   23    0.92     0.307  ███▏                           AJX 
   24    0.89     0.298  ███                            H 
   25    0.86     0.289  ██▉                            AB=JK 
   26    0.83     0.288  ██▉                            F 
   27    0.81     0.268  ██▊                            BF=EJ 
   28    0.78     0.264  ██▋                            BH=CJ 
   29    0.75     0.241  ██▍                            BG=DJ 
   30    0.73     0.226  ██▎                            CFX 
   31    0.70     0.203  ██                             DFX 
   32    0.67     0.193  ██                             CX 
   33    0.65     0.190  █▉                             G 
   34    0.63     0.189  █▉                             EF=DG=CH=BJ=AK 
   35    0.60     0.188  █▉                             ADX 
   36    0.58     0.157  █▋                             AEX 
   37    0.55     0.156  █▋                             BDX 
   38    0.53     0.156  █▋                             J 
   39    0.51     0.150  █▌                             AX 
   40    0.49     0.144  █▌                             BHX 
   41    0.46     0.129  █▎                             BCX 
   42    0.44     0.111  █▏                             AJ=BK 
   43    0.42     0.110  █▏                             ABX 
   44    0.40     0.108  █▏                             FX 
   45    0.38     0.106  █                              ACX 
   46    0.36     0.104  █                              BX 
   47    0.33     0.078  ▊                              AE=FK 
   48    0.31     0.076  ▊                              BGX 
   49    0.29     0.073  ▊                              JX 
   50    0.27     0.071  ▊                              K 
   51    0.25     0.065  ▋                              BFX 
   52    0.23     0.059  ▋                              AFX 
   53    0.21     0.058  ▋                              EFX 
   54    0.19     0.053  ▌                              GX 
   55    0.17     0.050  ▌                              AG=DK 
   56    0.15     0.039  ▍                              BE=FJ 
   57    0.13     0.038  ▍                              AHX 
   58    0.11     0.038  ▍                              CGX 
   59    0.09     0.031  ▎                              AGX 
   60    0.07     0.025  ▎                              CG=DH 
   61    0.05     0.024  ▎                              HX 
   62    0.03     0.007  ▏                              BEX 
   63    0.01     0.007  ▏                              KX 
```

</details>

## Per-point medians

| pt | levels ABCDEFGHJK | orig ops/s | orig p99 | recon ops/s | recon p99 |
|---|---|---|---|---|---|
| 0 | `-----+++++` | 464k | 53µs | 576k | 40.2µs |
| 1 | `+--------+` | 456k | 1.48ms | 761k | 506µs |
| 2 | `-+------+-` | 908k | 78.3µs | 2.15M | 18.8µs |
| 3 | `++---+++--` | 429k | 4.49ms | 631k | 1.2ms |
| 4 | `--+----+--` | 28.3k | 428µs | 28.8k | 420µs |
| 5 | `+-+--++-+-` | 31.5k | 8.45ms | 32.6k | 8.09ms |
| 6 | `-++--++--+` | 67.1k | 301µs | 70.3k | 293µs |
| 7 | `+++----+++` | 40.9k | 7.5ms | 40.2k | 8.03ms |
| 8 | `---+--+---` | 1.54M | 34.6µs | 3.72M | 10.4µs |
| 9 | `+--+-+-++-` | 1.82M | 358µs | 4.03M | 4.36ms |
| 10 | `-+-+-+-+-+` | 1.25M | 34.6µs | 2.61M | 9.79µs |
| 11 | `++-+--+-++` | 1.66M | 2.47ms | 3.78M | 3.75ms |
| 12 | `--++-+--++` | 1.25M | 264µs | 5.15M | 136µs |
| 13 | `+-++--++-+` | 1.67M | 6.13ms | 3.82M | 3.52ms |
| 14 | `-+++--+++-` | 83.3k | 264µs | 86.2k | 222µs |
| 15 | `++++-+----` | 1.55M | 1.81ms | 3.47M | 4.29ms |
| 16 | `----++----` | 558k | 87.6µs | 689k | 51.5µs |
| 17 | `+---+-+++-` | 593k | 2.04ms | 807k | 1.29ms |
| 18 | `-+--+-++-+` | 588k | 104µs | 929k | 41.7µs |
| 19 | `++--++--++` | 522k | 5.93ms | 604k | 1.84ms |
| 20 | `--+-+-+-++` | 27.8k | 971µs | 27.3k | 995µs |
| 21 | `+-+-++-+-+` | 24.3k | 40.6ms | 24.8k | 38ms |
| 22 | `-++-++-++-` | 27.2k | 971µs | 26.7k | 987µs |
| 23 | `+++-+-+---` | 26.6k | 32.6ms | 26.7k | 33.4ms |
| 24 | `---++--+++` | 575k | 109µs | 1.49M | 81.4µs |
| 25 | `+--++++--+` | 527k | 5.21ms | 2.17M | 4.95ms |
| 26 | `-+-++++-+-` | 523k | 66µs | 1.23M | 80.4µs |
| 27 | `++-++--+--` | 719k | 2.34ms | 2.52M | 4.55ms |
| 28 | `--++++++--` | 91k | 971µs | 2.09M | 305µs |
| 29 | `+-+++---+-` | 78k | 19ms | 2.16M | 4.23ms |
| 30 | `-++++----+` | 32.8k | 799µs | 142k | 107µs |
| 31 | `++++++++++` | 35.1k | 28.4ms | 1.75M | 5.34ms |

