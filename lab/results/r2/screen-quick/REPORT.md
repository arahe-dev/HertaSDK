# DoE screen: orig=orig vs recon=recon

`20261010-042053-screen-orig-recon` · 2026-10-10T04:20:53+05:30 · go1.27.1 windows/amd64 · 20 CPUs (GOMAXPROCS 20) · 192 trials (0 failed) in 88s · 3 reps · warm-up 100ms + window 300ms · seed 1

Command: `go run ./lab/cmd/doe -quick -q -out lab/results/r2/screen-quick`

## Verdict

X is the implementation factor (−1 baseline, +1 candidate), crossed with every design run, so its main effect is the candidate's change averaged over the whole design. Log-scale responses show the ratio candidate/baseline of geometric means; raw responses show the difference. Significance is Bonferroni-corrected over all contrasts of a response; ratios within 5% count as *same*.

### Key responses

| response | arms | baseline | candidate | change [95% CI] | p | verdict | strongest X×factor |
|---|---|---|---|---|---|---|---|
| throughput | orig→recon | 243k | 417k | ×1.72 [1.69, 1.74] | <1e-4 | **better** | DX (×1.43) |
| goodput | orig→recon | 155k | 198k | ×1.28 [1.27, 1.3] | <1e-4 | **better** | AX (×0.916) |
| lat_p50 | orig→recon | 46.8µs | 28.5µs | ×0.61 [0.6, 0.619] | <1e-4 | **better** | CX (×1.62) |
| lat_p99 | orig→recon | 889µs | 346µs | ×0.39 [0.373, 0.407] | <1e-4 | **better** | DX (×0.555) |
| lat_p999 | orig→recon | 1.92ms | 741µs | ×0.386 [0.359, 0.414] | <1e-4 | **better** | DX (×0.569) |
| wait_p99 | orig→recon | 582µs | 114µs | ×0.195 [0.185, 0.206] | <1e-4 | **better** | DX (×0.269) |
| allocs_op | orig→recon | 14.7 | 2.61 | -12.1 [-12.1, -12.1] | <1e-4 | **better** | EX (-1.59) |
| bytes_op | orig→recon | 723 | 177 | -545 [-546, -544] | <1e-4 | **better** | EX (-106) |
| peak_goroutines | orig→recon | 534 | 2.6 | -531 [-630, -433] | <1e-4 | **better** | CX (+398) |
| err_rate | orig→recon | 19.68% | 30.53% | +0.108 [+0.106, +0.111] | <1e-4 | **worse** | DX (+0.11) |
| cap_violations | orig→recon | 0 | 0 | +0 [+0, +0] | – | same | – |
| key_violations | orig→recon | 0 | 0 | +0 [+0, +0] | – | same | – |
| drain | orig→recon | 118µs | 75.4µs | ×0.639 [0.557, 0.733] | <1e-4 | **better** | – |

<details><summary>All other responses</summary>

| response | arms | baseline | candidate | change [95% CI] | p | verdict | strongest X×factor |
|---|---|---|---|---|---|---|---|
| lat_p90 | orig→recon | 321µs | 186µs | ×0.578 [0.569, 0.587] | <1e-4 | **better** | DX (×0.646) |
| lat_max | orig→recon | 4.8ms | 2.1ms | ×0.437 [0.376, 0.507] | <1e-4 | **better** | DX (×0.669) |
| wait_p50 | orig→recon | 6.42µs | 2.54µs | ×0.396 [0.388, 0.403] | <1e-4 | **better** | FX (×1.27) |
| exec_p50 | orig→recon | 12.5µs | 8.6µs | ×0.69 [0.681, 0.699] | <1e-4 | **better** | CX (×1.44) |
| exec_p99 | orig→recon | 132µs | 80.2µs | ×0.609 [0.581, 0.639] | <1e-4 | **better** | CX (×1.35) |
| peak_heap | orig→recon | 8.04 | 7.04 | ×0.876 [0.837, 0.917] | <1e-4 | **better** | FX (×1.11) |
| gc_per_s | orig→recon | 88.8 | 27.6 | -61.3 [-67.1, -55.5] | <1e-4 | **better** | CX (+37.3) |
| retries_op | orig→recon | 0.0492 | 0.0405 | -0.00868 [-0.00915, -0.00821] | <1e-4 | differs | GX (-0.00868) |
| fairness | orig→recon | 0.992 | 0.995 | +0.00257 [+0.0018, +0.00334] | <1e-4 | **better** | DX (+0.00248) |
| stats_lag | orig→recon | 0.0104 | 0 | -0.0104 [-0.031, +0.0102] | 0.319 | same | – |
| residual_goroutines | orig→recon | 0 | 0 | +0 [+0, +0] | – | same | – |

</details>

## Noise

Replicate error of a single trial (pooled within-cell SD) as a coefficient of variation: the ±1σ spread between identical trials.

| response | orig | recon |
|---|---|---|
| throughput | 4.9% | 4.4% |
| goodput | 5.0% | 3.8% |
| lat_p50 | 6.4% | 5.1% |
| lat_p90 | 6.5% | 4.3% |
| lat_p99 | 22.7% | 7.1% |
| lat_p999 | 28.6% | 28.0% |
| lat_max | 76.6% | 60.4% |
| wait_p50 | 7.9% | 5.6% |
| wait_p99 | 29.4% | 8.0% |
| exec_p50 | 5.3% | 3.8% |
| exec_p99 | 24.7% | 8.5% |
| allocs_op | 0.6% | 0.7% |
| bytes_op | 0.7% | 0.8% |
| peak_goroutines | 91.4% | 98.6% |
| peak_heap | 17.2% | 17.0% |
| gc_per_s | 31.3% | 26.7% |
| err_rate | 4.9% | 1.2% |
| retries_op | 4.2% | 2.7% |
| fairness | 0.3% | 0.3% |
| cap_violations | – | – |
| key_violations | – | – |
| drain | 67.6% | 56.0% |
| stats_lag | 979.8% | – |
| residual_goroutines | – | – |

## Error classes

Share of all calls, over all trials of each arm. A class is the implementation-neutral error class, the outcome what `OutcomeOf` reports.

| class/outcome | orig | recon |
|---|---|---|
| deadline/permanent | 0.275% | 0.134% |
| deadline/uncertain | 0.020% | 0.007% |
| other/transient | 0.035% | 0.030% |
| overloaded/throttled | 28.370% | 39.236% |

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
| orig | C | ×0.102 | [0.0996, 0.104] | <1e-4 | ** | **sig** |
| orig | D | ×2.89 | [2.83, 2.95] | <1e-4 | * | **sig** |
| orig | E | ×0.463 | [0.454, 0.472] | <1e-4 | * | **sig** |
| orig | CD=GH | ×1.92 | [1.88, 1.95] | <1e-4 | * | **sig** |
| orig | DE=FG | ×0.549 | [0.539, 0.56] | <1e-4 |  | **sig** |
| orig | BD=GJ | ×0.64 | [0.627, 0.652] | <1e-4 |  | **sig** |
| orig vs recon | C | ×0.108 | [0.107, 0.109] | <1e-4 | ** | **sig** |
| orig vs recon | D | ×4.12 | [4.07, 4.17] | <1e-4 | ** | **sig** |
| orig vs recon | CD=GH | ×2.45 | [2.42, 2.48] | <1e-4 | ** | **sig** |
| orig vs recon | X | ×1.72 | [1.69, 1.74] | <1e-4 | ** | **sig** |
| orig vs recon | E | ×0.594 | [0.586, 0.602] | <1e-4 | ** | **sig** |
| orig vs recon | BD=GJ | ×0.645 | [0.636, 0.653] | <1e-4 | * | **sig** |
| recon | C | ×0.115 | [0.113, 0.117] | <1e-4 | ** | **sig** |
| recon | D | ×5.87 | [5.77, 5.97] | <1e-4 | ** | **sig** |
| recon | CD=GH | ×3.13 | [3.07, 3.18] | <1e-4 | ** | **sig** |
| recon | CF=EH | ×1.59 | [1.56, 1.61] | <1e-4 |  | **sig** |
| recon | BD=GJ | ×0.649 | [0.638, 0.661] | <1e-4 |  | **sig** |
| recon | AC=HK | ×1.43 | [1.41, 1.46] | <1e-4 |  | **sig** |

### goodput (ops/s)

| scope | effect | estimate | 95% CI | p | Lenth |  |
|---|---|---|---|---|---|---|
| orig | C | ×0.0529 | [0.0518, 0.0539] | <1e-4 | ** | **sig** |
| orig | B | ×1.34 | [1.32, 1.37] | <1e-4 |  | **sig** |
| orig | E | ×0.75 | [0.735, 0.765] | <1e-4 |  | **sig** |
| orig | BE=FJ | ×0.791 | [0.776, 0.807] | <1e-4 |  | **sig** |
| orig | CE=FH | ×0.828 | [0.812, 0.845] | <1e-4 |  | **sig** |
| orig | D | ×1.2 | [1.18, 1.23] | <1e-4 |  | **sig** |
| orig vs recon | C | ×0.0512 | [0.0506, 0.0518] | <1e-4 | ** | **sig** |
| orig vs recon | E | ×0.71 | [0.701, 0.719] | <1e-4 | * | **sig** |
| orig vs recon | B | ×1.41 | [1.39, 1.42] | <1e-4 | * | **sig** |
| orig vs recon | X | ×1.28 | [1.27, 1.3] | <1e-4 | * | **sig** |
| orig vs recon | D | ×1.28 | [1.26, 1.29] | <1e-4 | * | **sig** |
| orig vs recon | AC=HK | ×1.27 | [1.26, 1.29] | <1e-4 | * | **sig** |
| recon | C | ×0.0496 | [0.0488, 0.0503] | <1e-4 | ** | **sig** |
| recon | E | ×0.673 | [0.662, 0.683] | <1e-4 |  | **sig** |
| recon | B | ×1.47 | [1.45, 1.5] | <1e-4 |  | **sig** |
| recon | AC=HK | ×1.44 | [1.42, 1.47] | <1e-4 |  | **sig** |
| recon | AJ=BK | ×1.39 | [1.37, 1.42] | <1e-4 |  | **sig** |
| recon | BF=EJ | ×1.39 | [1.37, 1.41] | <1e-4 |  | **sig** |

### lat_p50 (ns)

| scope | effect | estimate | 95% CI | p | Lenth |  |
|---|---|---|---|---|---|---|
| orig | C | ×16.5 | [16.1, 16.9] | <1e-4 | ** | **sig** |
| orig | A | ×3.73 | [3.63, 3.82] | <1e-4 | ** | **sig** |
| orig | DE=FG | ×3.67 | [3.58, 3.77] | <1e-4 | ** | **sig** |
| orig | D | ×0.282 | [0.275, 0.289] | <1e-4 | ** | **sig** |
| orig | BH=CJ | ×2.55 | [2.49, 2.62] | <1e-4 | ** | **sig** |
| orig | AD=GK | ×0.395 | [0.385, 0.405] | <1e-4 | ** | **sig** |
| orig vs recon | C | ×26.8 | [26.3, 27.2] | <1e-4 | ** | **sig** |
| orig vs recon | DE=FG | ×4.43 | [4.36, 4.5] | <1e-4 | ** | **sig** |
| orig vs recon | A | ×3.45 | [3.4, 3.51] | <1e-4 | ** | **sig** |
| orig vs recon | D | ×0.301 | [0.296, 0.305] | <1e-4 | ** | **sig** |
| orig vs recon | AD=GK | ×0.389 | [0.383, 0.396] | <1e-4 | ** | **sig** |
| orig vs recon | E | ×0.393 | [0.387, 0.399] | <1e-4 | ** | **sig** |
| recon | C | ×43.5 | [42.6, 44.3] | <1e-4 | ** | **sig** |
| recon | DE=FG | ×5.34 | [5.24, 5.45] | <1e-4 | ** | **sig** |
| recon | E | ×0.312 | [0.306, 0.318] | <1e-4 | ** | **sig** |
| recon | A | ×3.2 | [3.14, 3.27] | <1e-4 | ** | **sig** |
| recon | D | ×0.321 | [0.314, 0.327] | <1e-4 | ** | **sig** |
| recon | AD=GK | ×0.384 | [0.376, 0.391] | <1e-4 | ** | **sig** |

### lat_p99 (ns)

| scope | effect | estimate | 95% CI | p | Lenth |  |
|---|---|---|---|---|---|---|
| orig | A | ×21.8 | [20.1, 23.7] | <1e-4 | ** | **sig** |
| orig | C | ×6.71 | [6.17, 7.29] | <1e-4 | ** | **sig** |
| orig | E | ×2.68 | [2.47, 2.92] | <1e-4 | ** | **sig** |
| orig | CE=FH | ×1.58 | [1.46, 1.72] | <1e-4 | * | **sig** |
| orig | D | ×0.639 | [0.588, 0.695] | <1e-4 | * | **sig** |
| orig | BF=EJ | ×0.665 | [0.612, 0.723] | <1e-4 |  | **sig** |
| orig vs recon | A | ×17.3 | [16.6, 18.1] | <1e-4 | ** | **sig** |
| orig vs recon | C | ×6.34 | [6.07, 6.62] | <1e-4 | ** | **sig** |
| orig vs recon | D | ×0.355 | [0.339, 0.37] | <1e-4 | ** | **sig** |
| orig vs recon | E | ×2.72 | [2.61, 2.84] | <1e-4 | ** | **sig** |
| orig vs recon | X | ×0.39 | [0.373, 0.407] | <1e-4 | ** | **sig** |
| orig vs recon | DX | ×0.555 | [0.531, 0.579] | <1e-4 | ** | **sig** |
| recon | A | ×13.7 | [13.3, 14.1] | <1e-4 | ** | **sig** |
| recon | C | ×5.98 | [5.82, 6.16] | <1e-4 | ** | **sig** |
| recon | D | ×0.197 | [0.191, 0.202] | <1e-4 | ** | **sig** |
| recon | E | ×2.76 | [2.69, 2.84] | <1e-4 | ** | **sig** |
| recon | CD=GH | ×0.37 | [0.36, 0.38] | <1e-4 | ** | **sig** |
| recon | AD=GK | ×0.48 | [0.467, 0.494] | <1e-4 | * | **sig** |

### lat_p999 (ns)

| scope | effect | estimate | 95% CI | p | Lenth |  |
|---|---|---|---|---|---|---|
| orig | A | ×10.4 | [9.4, 11.5] | <1e-4 | ** | **sig** |
| orig | C | ×3.08 | [2.78, 3.41] | <1e-4 | ** | **sig** |
| orig | E | ×1.8 | [1.63, 2] | <1e-4 | * | **sig** |
| orig | CE=FH | ×1.58 | [1.42, 1.75] | <1e-4 | * | **sig** |
| orig | AC=HK | ×1.43 | [1.29, 1.59] | <1e-4 | * | **sig** |
| orig | AE=FK | ×1.26 | [1.14, 1.4] | <1e-4 |  | **sig** |
| orig vs recon | A | ×10.2 | [9.5, 11] | <1e-4 | ** | **sig** |
| orig vs recon | C | ×3.41 | [3.18, 3.66] | <1e-4 | ** | **sig** |
| orig vs recon | X | ×0.386 | [0.359, 0.414] | <1e-4 | ** | **sig** |
| orig vs recon | D | ×0.513 | [0.478, 0.551] | <1e-4 | ** | **sig** |
| orig vs recon | E | ×1.87 | [1.74, 2.01] | <1e-4 | ** | **sig** |
| orig vs recon | DX | ×0.569 | [0.53, 0.611] | <1e-4 | ** | **sig** |
| recon | A | ×9.99 | [9.03, 11] | <1e-4 | ** | **sig** |
| recon | C | ×3.78 | [3.42, 4.18] | <1e-4 | ** | **sig** |
| recon | D | ×0.292 | [0.264, 0.323] | <1e-4 | ** | **sig** |
| recon | CD=GH | ×0.462 | [0.418, 0.511] | <1e-4 | ** | **sig** |
| recon | E | ×1.95 | [1.76, 2.15] | <1e-4 | * | **sig** |
| recon | AD=GK | ×0.569 | [0.515, 0.63] | <1e-4 | * | **sig** |

### wait_p99 (ns)

| scope | effect | estimate | 95% CI | p | Lenth |  |
|---|---|---|---|---|---|---|
| orig | A | ×46.8 | [42.2, 52] | <1e-4 | ** | **sig** |
| orig | E | ×4.85 | [4.37, 5.39] | <1e-4 | ** | **sig** |
| orig | C | ×4.76 | [4.28, 5.28] | <1e-4 | ** | **sig** |
| orig | CE=FH | ×2.5 | [2.25, 2.78] | <1e-4 | * | **sig** |
| orig | BE=FJ | ×1.7 | [1.53, 1.88] | <1e-4 |  | **sig** |
| orig | D | ×0.601 | [0.541, 0.668] | <1e-4 |  | **sig** |
| orig vs recon | A | ×38.5 | [36.5, 40.7] | <1e-4 | ** | **sig** |
| orig vs recon | D | ×0.162 | [0.153, 0.171] | <1e-4 | ** | **sig** |
| orig vs recon | E | ×5.67 | [5.37, 5.99] | <1e-4 | ** | **sig** |
| orig vs recon | X | ×0.195 | [0.185, 0.206] | <1e-4 | ** | **sig** |
| orig vs recon | DX | ×0.269 | [0.255, 0.284] | <1e-4 | ** | **sig** |
| orig vs recon | CD=GH | ×0.277 | [0.263, 0.293] | <1e-4 | ** | **sig** |
| recon | A | ×31.7 | [30.7, 32.7] | <1e-4 | ** | **sig** |
| recon | D | ×0.0435 | [0.0422, 0.0449] | <1e-4 | ** | **sig** |
| recon | CD=GH | ×0.103 | [0.1, 0.107] | <1e-4 | ** | **sig** |
| recon | E | ×6.63 | [6.43, 6.84] | <1e-4 | * | **sig** |
| recon | AB=JK | ×2.37 | [2.3, 2.45] | <1e-4 |  | **sig** |
| recon | BH=CJ | ×2.05 | [1.98, 2.11] | <1e-4 |  | **sig** |

### allocs_op (allocs)

| scope | effect | estimate | 95% CI | p | Lenth |  |
|---|---|---|---|---|---|---|
| orig | F | +3.94 | [+3.9, +3.98] | <1e-4 | ** | **sig** |
| orig | E | +2.72 | [+2.68, +2.75] | <1e-4 | ** | **sig** |
| orig | D | -1.91 | [-1.95, -1.87] | <1e-4 | * | **sig** |
| orig | DE=FG | +1.66 | [+1.62, +1.69] | <1e-4 | * | **sig** |
| orig | C | +0.975 | [+0.938, +1.01] | <1e-4 |  | **sig** |
| orig | A | +0.965 | [+0.928, +1] | <1e-4 |  | **sig** |
| orig vs recon | X | -12.1 | [-12.1, -12.1] | <1e-4 | ** | **sig** |
| orig vs recon | F | +3.59 | [+3.57, +3.61] | <1e-4 | ** | **sig** |
| orig vs recon | D | -2.12 | [-2.14, -2.1] | <1e-4 | ** | **sig** |
| orig vs recon | EX | -1.59 | [-1.61, -1.57] | <1e-4 | ** | **sig** |
| orig vs recon | DF=EG | -1.29 | [-1.31, -1.27] | <1e-4 | ** | **sig** |
| orig vs recon | E | +1.13 | [+1.11, +1.14] | <1e-4 | * | **sig** |
| recon | F | +3.24 | [+3.23, +3.24] | <1e-4 | ** | **sig** |
| recon | D | -2.33 | [-2.34, -2.32] | <1e-4 | ** | **sig** |
| recon | DF=EG | -1.67 | [-1.68, -1.66] | <1e-4 | ** | **sig** |
| recon | CD=GH | -0.766 | [-0.773, -0.759] | <1e-4 |  | **sig** |
| recon | AB=JK | -0.541 | [-0.548, -0.534] | <1e-4 |  | **sig** |
| recon | BD=GJ | +0.536 | [+0.529, +0.544] | <1e-4 |  | **sig** |

### bytes_op (B)

| scope | effect | estimate | 95% CI | p | Lenth |  |
|---|---|---|---|---|---|---|
| orig | F | +268 | [+266, +271] | <1e-4 | ** | **sig** |
| orig | D | -204 | [-206, -202] | <1e-4 | ** | **sig** |
| orig | E | +173 | [+171, +175] | <1e-4 | ** | **sig** |
| orig | DE=FG | +155 | [+153, +158] | <1e-4 | * | **sig** |
| orig | DF=EG | -80.9 | [-83.1, -78.7] | <1e-4 |  | **sig** |
| orig | CD=GH | -60.5 | [-62.7, -58.3] | <1e-4 |  | **sig** |
| orig vs recon | X | -545 | [-546, -544] | <1e-4 | ** | **sig** |
| orig vs recon | F | +250 | [+249, +251] | <1e-4 | ** | **sig** |
| orig vs recon | D | -207 | [-208, -206] | <1e-4 | ** | **sig** |
| orig vs recon | EX | -106 | [-107, -105] | <1e-4 | ** | **sig** |
| orig vs recon | DF=EG | -104 | [-106, -103] | <1e-4 | ** | **sig** |
| orig vs recon | DE=FG | +94 | [+92.9, +95.1] | <1e-4 | ** | **sig** |
| recon | F | +232 | [+231, +233] | <1e-4 | ** | **sig** |
| recon | D | -210 | [-210, -209] | <1e-4 | ** | **sig** |
| recon | DF=EG | -128 | [-129, -127] | <1e-4 | ** | **sig** |
| recon | CD=GH | -72.3 | [-72.9, -71.8] | <1e-4 | * | **sig** |
| recon | AB=JK | -48.5 | [-49.1, -47.9] | <1e-4 |  | **sig** |
| recon | BD=GJ | +45.4 | [+44.8, +46] | <1e-4 |  | **sig** |

### peak_goroutines (goroutines)

| scope | effect | estimate | 95% CI | p | Lenth |  |
|---|---|---|---|---|---|---|
| orig | C | -798 | [-997, -599] | <1e-4 | * | **sig** |
| orig | AC=HK | +650 | [+451, +849] | <1e-4 |  | **sig** |
| orig | A | -471 | [-670, -272] | <1e-4 |  | **sig** |
| orig | CE=FH | +409 | [+210, +608] | 0.000117 |  | **sig** |
| orig | E | -396 | [-595, -197] | 0.000183 |  | **sig** |
| orig | CF=EH | +366 | [+167, +565] | 0.000487 |  | **sig** |
| orig vs recon | X | -531 | [-630, -433] | <1e-4 | * | **sig** |
| orig vs recon | C | -400 | [-498, -301] | <1e-4 | * | **sig** |
| orig vs recon | CX | +398 | [+300, +497] | <1e-4 | * | **sig** |
| orig vs recon | ACX | -325 | [-424, -227] | <1e-4 |  | **sig** |
| orig vs recon | AC=HK | +325 | [+226, +423] | <1e-4 |  | **sig** |
| orig vs recon | AX | +237 | [+138, +335] | <1e-4 |  | **sig** |
| recon | K | +3.58 | [+2.54, +4.63] | <1e-4 | * | **sig** |
| recon | A | +2.38 | [+1.33, +3.42] | <1e-4 | * | **sig** |
| recon | EF=DG=CH=BJ=AK | +2.25 | [+1.2, +3.3] | <1e-4 | * | **sig** |
| recon | AH=CK | -1.75 | [-2.8, -0.703] | 0.00141 |  | **sig** |
| recon | BF=EJ | +1.67 | [+0.62, +2.71] | 0.00227 |  |  |
| recon | CG=DH | +1.38 | [+0.328, +2.42] | 0.0109 |  |  |

### err_rate (fraction)

| scope | effect | estimate | 95% CI | p | Lenth |  |
|---|---|---|---|---|---|---|
| orig | D | +0.365 | [+0.361, +0.369] | <1e-4 | ** | **sig** |
| orig | C | +0.223 | [+0.219, +0.227] | <1e-4 | * | **sig** |
| orig | CD=GH | +0.209 | [+0.205, +0.213] | <1e-4 | * | **sig** |
| orig | BD=GJ | -0.154 | [-0.158, -0.15] | <1e-4 | * | **sig** |
| orig | B | -0.152 | [-0.156, -0.148] | <1e-4 | * | **sig** |
| orig | DE=FG | -0.151 | [-0.155, -0.147] | <1e-4 | * | **sig** |
| orig vs recon | D | +0.475 | [+0.473, +0.477] | <1e-4 | ** | **sig** |
| orig vs recon | C | +0.205 | [+0.203, +0.207] | <1e-4 | ** | **sig** |
| orig vs recon | CD=GH | +0.191 | [+0.189, +0.193] | <1e-4 | ** | **sig** |
| orig vs recon | A | +0.156 | [+0.154, +0.158] | <1e-4 | ** | **sig** |
| orig vs recon | AD=GK | +0.146 | [+0.144, +0.148] | <1e-4 | ** | **sig** |
| orig vs recon | BD=GJ | -0.128 | [-0.13, -0.126] | <1e-4 | ** | **sig** |
| recon | D | +0.585 | [+0.584, +0.587] | <1e-4 | ** | **sig** |
| recon | C | +0.187 | [+0.185, +0.188] | <1e-4 | * | **sig** |
| recon | A | +0.181 | [+0.18, +0.183] | <1e-4 | * | **sig** |
| recon | CD=GH | +0.173 | [+0.171, +0.174] | <1e-4 | * | **sig** |
| recon | AD=GK | +0.171 | [+0.17, +0.173] | <1e-4 | * | **sig** |
| recon | BD=GJ | -0.102 | [-0.104, -0.101] | <1e-4 |  | **sig** |

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
| orig | A | ×4.32 | [3.5, 5.34] | <1e-4 | ** | **sig** |
| orig | C | ×2.38 | [1.93, 2.93] | <1e-4 | ** | **sig** |
| orig | AC=HK | ×0.494 | [0.4, 0.609] | <1e-4 | ** | **sig** |
| orig | H | ×0.764 | [0.619, 0.944] | 0.0132 |  |  |
| orig | AE=FK | ×1.26 | [1.02, 1.56] | 0.0296 |  |  |
| orig | EF=DG=CH=BJ=AK | ×1.21 | [0.977, 1.49] | 0.0803 |  |  |
| orig vs recon | A | ×4.41 | [3.85, 5.07] | <1e-4 | ** | **sig** |
| orig vs recon | C | ×2.59 | [2.26, 2.98] | <1e-4 | ** | **sig** |
| orig vs recon | AC=HK | ×0.388 | [0.338, 0.445] | <1e-4 | ** | **sig** |
| orig vs recon | X | ×0.639 | [0.557, 0.733] | <1e-4 | ** | **sig** |
| orig vs recon | ACX | ×0.785 | [0.684, 0.901] | 0.000703 | * | **sig** |
| orig vs recon | CFX | ×0.815 | [0.71, 0.935] | 0.00381 | * |  |
| recon | A | ×4.51 | [3.76, 5.4] | <1e-4 | ** | **sig** |
| recon | AC=HK | ×0.305 | [0.254, 0.365] | <1e-4 | ** | **sig** |
| recon | C | ×2.83 | [2.36, 3.4] | <1e-4 | ** | **sig** |
| recon | CF=EH | ×0.798 | [0.665, 0.957] | 0.0155 |  |  |
| recon | AH=CK | ×1.24 | [1.04, 1.49] | 0.0192 |  |  |
| recon | AJ=BK | ×0.808 | [0.674, 0.969] | 0.0222 |  |  |

<details><summary>Half-normal plot: lat_p99, orig vs recon (63 contrasts)</summary>

```
 rank  z(half)  |effect|
    1    2.65     4.112  ██████████████████████████████ A **
    2    2.26     2.664  ███████████████████▍           C **
    3    2.06     1.496  ██████████▉                    D **
    4    1.91     1.445  ██████████▌                    E **
    5    1.80     1.359  █████████▉                     X **
    6    1.71     0.851  ██████▎                        DX **
    7    1.63     0.846  ██████▏                        CD=GH **
    8    1.56     0.756  █████▌                         BF=EJ *
    9    1.49     0.663  ████▉                          AC=HK *
   10    1.44     0.624  ████▌                          AD=GK *
   11    1.38     0.589  ████▎                          CDX *
   12    1.33     0.445  ███▎                           CEX *
   13    1.29     0.435  ███▏                           ADX *
   14    1.24     0.428  ███▏                           AJX *
   15    1.20     0.411  ███                            AJ=BK *
   16    1.16     0.352  ██▋                            AE=FK 
   17    1.12     0.337  ██▌                            AX 
   18    1.09     0.320  ██▍                            EF=DG=CH=BJ=AK 
   19    1.05     0.312  ██▎                            DE=FG 
   20    1.02     0.295  ██▏                            G 
   21    0.98     0.266  ██                             ACX 
   22    0.95     0.237  █▊                             AB=JK 
   23    0.92     0.223  █▋                             BG=DJ 
   24    0.89     0.217  █▋                             CE=FH 
   25    0.86     0.216  █▋                             J 
   26    0.83     0.202  █▌                             DFX 
   27    0.81     0.192  █▍                             BGX 
   28    0.78     0.191  █▍                             BC=HJ 
   29    0.75     0.186  █▍                             BCX 
   30    0.73     0.185  █▍                             BEX 
   31    0.70     0.180  █▎                             DF=EG 
   32    0.67     0.173  █▎                             JX 
   33    0.65     0.168  █▎                             BFX 
   34    0.63     0.151  █▏                             AHX 
   35    0.60     0.134  █                              FX 
   36    0.58     0.131  █                              EFX 
   37    0.55     0.128  ▉                              AH=CK 
   38    0.53     0.107  ▊                              AF=EK 
   39    0.51     0.107  ▊                              BD=GJ 
   40    0.49     0.103  ▊                              GX 
   41    0.46     0.095  ▊                              K 
   42    0.44     0.083  ▋                              AG=DK 
   43    0.42     0.082  ▋                              CX 
   44    0.40     0.075  ▌                              ABX 
   45    0.38     0.071  ▌                              BDX 
   46    0.36     0.071  ▌                              F 
   47    0.33     0.061  ▌                              AFX 
   48    0.31     0.058  ▍                              AEX 
   49    0.29     0.055  ▍                              BX 
   50    0.27     0.045  ▍                              BH=CJ 
   51    0.25     0.045  ▍                              CG=DH 
   52    0.23     0.043  ▍                              H 
   53    0.21     0.042  ▎                              B 
   54    0.19     0.038  ▎                              BHX 
   55    0.17     0.026  ▏                              CF=EH 
   56    0.15     0.022  ▏                              DEX 
   57    0.13     0.022  ▏                              KX 
   58    0.11     0.021  ▏                              EX 
   59    0.09     0.018  ▏                              CFX 
   60    0.07     0.011  ▏                              CGX 
   61    0.05     0.011  ▏                              AGX 
   62    0.03     0.007                                 BE=FJ 
   63    0.01     0.006                                 HX 
```

</details>

<details><summary>Half-normal plot: throughput, orig vs recon (63 contrasts)</summary>

```
 rank  z(half)  |effect|
    1    2.65     3.211  ██████████████████████████████ C **
    2    2.26     2.042  ███████████████████▏           D **
    3    2.06     1.291  ████████████                   CD=GH **
    4    1.91     0.779  ███████▎                       X **
    5    1.80     0.751  ███████                        E **
    6    1.71     0.634  █████▉                         BD=GJ *
    7    1.63     0.511  ████▊                          DX *
    8    1.56     0.469  ████▍                          CF=EH *
    9    1.49     0.450  ████▎                          AF=EK *
   10    1.44     0.446  ████▏                          DE=FG *
   11    1.38     0.445  ████▏                          AD=GK *
   12    1.33     0.419  ███▉                           DEX *
   13    1.29     0.406  ███▊                           AC=HK *
   14    1.24     0.379  ███▌                           CEX *
   15    1.20     0.377  ███▌                           BF=EJ *
   16    1.16     0.372  ███▌                           BC=HJ *
   17    1.12     0.361  ███▍                           EX *
   18    1.09     0.353  ███▎                           CDX *
   19    1.05     0.339  ███▏                           AJX 
   20    1.02     0.288  ██▊                            DF=EG 
   21    0.98     0.277  ██▋                            B 
   22    0.95     0.247  ██▎                            A 
   23    0.92     0.245  ██▎                            CE=FH 
   24    0.89     0.225  ██▏                            AH=CK 
   25    0.86     0.213  ██                             H 
   26    0.83     0.196  █▉                             CFX 
   27    0.81     0.191  █▊                             BG=DJ 
   28    0.78     0.179  █▋                             DFX 
   29    0.75     0.167  █▌                             BH=CJ 
   30    0.73     0.164  █▌                             J 
   31    0.70     0.154  █▍                             F 
   32    0.67     0.149  █▍                             AB=JK 
   33    0.65     0.128  █▎                             EF=DG=CH=BJ=AK 
   34    0.63     0.117  █▏                             ABX 
   35    0.60     0.111  █                              ACX 
   36    0.58     0.108  █                              FX 
   37    0.55     0.108  █                              BHX 
   38    0.53     0.100  ▉                              BE=FJ 
   39    0.51     0.099  ▉                              G 
   40    0.49     0.088  ▉                              CX 
   41    0.46     0.088  ▉                              AEX 
   42    0.44     0.086  ▊                              GX 
   43    0.42     0.080  ▊                              BFX 
   44    0.40     0.068  ▋                              EFX 
   45    0.38     0.051  ▌                              BGX 
   46    0.36     0.050  ▌                              AX 
   47    0.33     0.043  ▍                              CGX 
   48    0.31     0.042  ▍                              HX 
   49    0.29     0.035  ▍                              AE=FK 
   50    0.27     0.030  ▎                              BX 
   51    0.25     0.028  ▎                              AGX 
   52    0.23     0.027  ▎                              BEX 
   53    0.21     0.023  ▎                              AFX 
   54    0.19     0.023  ▎                              AJ=BK 
   55    0.17     0.022  ▎                              K 
   56    0.15     0.020  ▏                              AG=DK 
   57    0.13     0.017  ▏                              JX 
   58    0.11     0.017  ▏                              KX 
   59    0.09     0.016  ▏                              CG=DH 
   60    0.07     0.015  ▏                              AHX 
   61    0.05     0.011  ▏                              BDX 
   62    0.03     0.011  ▏                              ADX 
   63    0.01     0.005                                 BCX 
```

</details>

## Per-point medians

| pt | levels ABCDEFGHJK | orig ops/s | orig p99 | recon ops/s | recon p99 |
|---|---|---|---|---|---|
| 0 | `-----+++++` | 505k | 109µs | 646k | 40.2µs |
| 1 | `+--------+` | 522k | 1.47ms | 846k | 586µs |
| 2 | `-+------+-` | 993k | 49.9µs | 1.75M | 28.3µs |
| 3 | `++---+++--` | 520k | 1.42ms | 793k | 1.03ms |
| 4 | `--+----+--` | 28.5k | 424µs | 29k | 416µs |
| 5 | `+-+--++-+-` | 32.1k | 8.72ms | 32.8k | 8.09ms |
| 6 | `-++--++--+` | 67.3k | 305µs | 69.6k | 301µs |
| 7 | `+++----+++` | 41.6k | 7.63ms | 40.7k | 7.57ms |
| 8 | `---+--+---` | 1.47M | 29.3µs | 3.39M | 9.92µs |
| 9 | `+--+-+-++-` | 1.56M | 1.45ms | 1.78M | 62.7µs |
| 10 | `-+-+-+-+-+` | 1.1M | 26µs | 2.62M | 8.03µs |
| 11 | `++-+--+-++` | 1.33M | 1.94ms | 1.96M | 444µs |
| 12 | `--++-+--++` | 662k | 379µs | 1.08M | 119µs |
| 13 | `+-++--++-+` | 1.38M | 1.65ms | 1.12M | 301µs |
| 14 | `-+++--+++-` | 82.6k | 305µs | 86.2k | 210µs |
| 15 | `++++-+----` | 794k | 1.52ms | 1.11M | 126µs |
| 16 | `----++----` | 618k | 88.6µs | 777k | 44.8µs |
| 17 | `+---+-+++-` | 669k | 2.21ms | 914k | 1.06ms |
| 18 | `-+--+-++-+` | 634k | 78.3µs | 1.05M | 35.1µs |
| 19 | `++--++--++` | 625k | 2.44ms | 722k | 1.94ms |
| 20 | `--+-+-+-++` | 27.8k | 979µs | 27.7k | 979µs |
| 21 | `+-+-++-+-+` | 25.2k | 37ms | 25.2k | 38ms |
| 22 | `-++-++-++-` | 27.9k | 954µs | 27.3k | 963µs |
| 23 | `+++-+-+---` | 27.6k | 31.9ms | 27.8k | 31.6ms |
| 24 | `---++--+++` | 654k | 69.1µs | 1.31M | 49.9µs |
| 25 | `+--++++--+` | 620k | 2.54ms | 1.14M | 1.65ms |
| 26 | `-+-++++-+-` | 586k | 102µs | 1.17M | 50.4µs |
| 27 | `++-++--+--` | 783k | 1.96ms | 1.23M | 1.48ms |
| 28 | `--++++++--` | 92.9k | 946µs | 1.11M | 297µs |
| 29 | `+-+++---+-` | 152k | 12.3ms | 961k | 485µs |
| 30 | `-++++----+` | 33.1k | 774µs | 139k | 105µs |
| 31 | `++++++++++` | 41.6k | 26.1ms | 945k | 465µs |

