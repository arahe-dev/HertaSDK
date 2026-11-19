| scenario | plain Go SDK | typed Go SDK | typed Go SDK + caller annotation | Herta, no evidence clause | Herta plan, evidence dispatch |
|---|---|---|---|---|---|
| pre-send refusal | 1 commit(s), Success | 1 commit(s), Success | 1 commit(s), Success | 1 commit(s), Success | 1 commit(s), Success |
| lost ack, honest adapter | 1 commit(s), Uncertain | 1 commit(s), Uncertain | 1 commit(s), Uncertain | 1 commit(s), Uncertain | 1 commit(s), Uncertain |
| lost ack, adapter says not sent | 2 commit(s), Success | 1 commit(s), Uncertain | 1 commit(s), Uncertain | 2 commit(s), Success | 1 commit(s), Uncertain |
| nested: checkout -> publish, mislabel | 2 commit(s), Success | 2 commit(s), Success | 1 commit(s), Uncertain | 2 commit(s), Success | 1 commit(s), Uncertain |
| adapter sends before dispatch point | 2 commit(s), Success | 2 commit(s), Success | 2 commit(s), Success | 2 commit(s), Success | 2 commit(s), Success |
