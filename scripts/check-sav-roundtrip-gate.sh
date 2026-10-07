#!/usr/bin/env bash
# The SAV round-trip gate as its own leg: the two TestSAVRoundTripGate tests
# per asset root, through check-milestone2-acceptance.sh's runner, so a chain
# can run the gate beside a milestone-2 leg that skips it
# (AGAINROM_M2_SAVGATE=skip). Same arguments and environment as that script.
AGAINROM_M2_SAVGATE=only exec bash "$(dirname "$0")/check-milestone2-acceptance.sh" "$@"
