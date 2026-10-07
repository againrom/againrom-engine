# Lane headless drive

The lane-built `missionrun -mission <10|20> -trace -ticks 1` reports zero
`UNSUPPORTED` lines for each mission. The runner reports an actual final tick 64
for this invocation. Receipts are the seat's `review/story1216-mission10.txt`
and `review/story1216-mission20.txt`.

The measured script populations remain 16 checks/27 instants/12 triggers for
mission 10 and 14/15/11 for mission 20, exactly the existing EN rows in
`pipeline/milestone-baseline.txt`. This is the bounded lane drive, not the full
landed-tip milestone census. The SAV continuation proof and all remaining
limits are in [story.md](story.md); the seat owns the final census.
