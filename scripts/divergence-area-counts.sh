#!/bin/sh
set -eu

cd "$(dirname "$0")/.."
export LC_ALL=C

printf 'file\trows\topen_or_accepted\tbytes\n'
files=0
total_rows=0
total_active=0
total_bytes=0
for file in docs/divergences/*.md; do
	if [ ! -f "$file" ]; then
		printf 'no divergence area files found\n' >&2
		exit 1
	fi
	rows=$(awk -F '|' '$2 ~ /^[[:space:]]*DIV-[0-9]+[[:space:]]*$/ { n++ } END { print n+0 }' "$file")
	active=$(awk -F '|' '$2 ~ /^[[:space:]]*DIV-[0-9]+[[:space:]]*$/ && $(NF-1) ~ /^[[:space:]]*(OPEN|ACCEPTED)[[:space:]]*$/ { n++ } END { print n+0 }' "$file")
	bytes=$(wc -c < "$file")
	printf '%s\t%s\t%s\t%s\n' "$file" "$rows" "$active" "$bytes"
	files=$((files + 1))
	total_rows=$((total_rows + rows))
	total_active=$((total_active + active))
	total_bytes=$((total_bytes + bytes))
done
printf 'TOTAL (%s files)\t%s\t%s\t%s\n' "$files" "$total_rows" "$total_active" "$total_bytes"
