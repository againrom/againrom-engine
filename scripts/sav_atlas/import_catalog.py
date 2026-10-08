"""Compact an explicit guarded TSV catalog without retaining sample offsets."""

import argparse
import csv
import json
from pathlib import Path
import sys

sys.dont_write_bytecode = True
import join_sources
import raw_reader


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--input", required=True)
    parser.add_argument("--output", required=True)
    args = parser.parse_args()
    output = join_sources.safe_output(args.output, [args.input])
    with Path(args.input).open(encoding="utf-8-sig", newline="") as stream:
        reader = csv.DictReader(stream, delimiter="\t")
        columns = [key for key in reader.fieldnames if key not in {"atlas_inputs", "atlas_spans"}]
        strings, index, rows = [], {}, []
        for row in reader:
            packed = []
            for column in columns:
                value = row[column]
                if value not in index:
                    index[value] = len(strings)
                    strings.append(value)
                packed.append(index[value])
            rows.append(packed)
    document = {"schema": 1, "source_catalog_sha256": raw_reader.sha(Path(args.input).read_bytes()),
                "columns": columns, "strings": strings, "rows": rows}
    output.parent.mkdir(parents=True, exist_ok=True)
    header = json.dumps({key: value for key, value in document.items() if key not in {"strings", "rows"}}, ensure_ascii=False)[:-1]
    packed_strings = ",\n".join("    "+json.dumps(value, ensure_ascii=False) for value in strings)
    packed_rows = ",\n".join("    "+json.dumps(row, separators=(",", ":")) for row in rows)
    output.write_text(header+',\n  "strings": [\n'+packed_strings+'\n  ],\n  "rows": [\n'+packed_rows+'\n  ]\n}\n', encoding="utf-8", newline="\n")
    join_sources.catalog(output)
    print(json.dumps({"rows": len(rows), "strings": len(strings), "output_bytes": output.stat().st_size}))


if __name__ == "__main__":
    main()
