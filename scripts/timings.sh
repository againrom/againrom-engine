# timings.sh — the reader of the test timings file, sourced by check-fast.sh
# and check-touched.sh. The file holds one line per test of the last full run:
# package, test and seconds, separated by tabs.

# timings_file [path] prints the timings file: the argument, else
# AGAINROM_TEST_TIMINGS, else "$(go env GOCACHE).test-timings.txt".
timings_file() {
	printf '%s' "${1:-${AGAINROM_TEST_TIMINGS:-$(go env GOCACHE).test-timings.txt}}"
}

# timings_sorted <file> prints the file by package, then seconds, then test.
timings_sorted() {
	LC_ALL=C sort -t "$(printf '\t')" -k1,1 -k3,3n -k2,2 "$1"
}
