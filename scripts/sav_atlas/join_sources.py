"""Join explicit writer source decisions to bounded physical SAV leaves.

The catalog describes guarded producer candidates. It never selects a branch.
Acceptance requires an input-hash-bound decision for every physical byte.
Outputs use separate decoded/file coordinates; compressed payloads retain their
packet relations. Parent object ranges are not coverage leaves.
"""

import argparse
import csv
import json
from pathlib import Path
import re
import sys
import struct

sys.dont_write_bytecode = True
import byte_layout
import raw_reader
import source_binding
import typed_payload


SOURCES = {"current", "rule", "Document", "constructor"}
RULE_KINDS = {"framing", "string-framing", "class-framing", "class-name",
              "compression-framing", "compressed-payload", "transport-padding"}


class SourceJoinError(ValueError):
    pass


def safe_output(output, inputs):
    target = Path(output).resolve()
    for path in inputs:
        source = Path(path).resolve()
        if target == source or (target.exists() and source.exists() and target.samefile(source)):
            raise SourceJoinError("output would overwrite an input")
    return target


def catalog(path):
    with Path(path).open(encoding="utf-8-sig", newline="") as stream:
        if Path(path).suffix == ".json":
            encoded = json.load(stream)
            rows = [{key: encoded["strings"][value] for key, value in zip(encoded["columns"], row)} for row in encoded["rows"]]
        else:
            rows = list(csv.DictReader(stream, delimiter="\t"))
    required = {"class", "wire_member", "member_span", "atlas_selector",
                "current_producer_guard", "actual_fallback", "named_debt"}
    if not rows or not required.issubset(rows[0]):
        raise SourceJoinError("catalog schema is incomplete")
    for index, row in enumerate(rows):
        row["catalog_row"] = index + 2
        row["selector"] = re.compile(row["atlas_selector"])
        row["path_selector"] = re.compile(row["atlas_path_selector"]) if row.get("atlas_path_selector") else None
    return rows


def offsets(row, length):
    span = row["member_span"]
    if not re.fullmatch(r"\d+(?:-\d+)?(?:,\d+(?:-\d+)?)*", span):
        return [(0, length)]
    result = []
    for part in span.split(","):
        ends = [int(value) for value in part.split("-")]
        start, end = ends[0], ends[-1] + 1
        if not 0 <= start < end <= length:
            raise SourceJoinError(f"catalog row {row['catalog_row']} source span outside field")
        result.append((start, end))
    if "subspan repeats" in row.get("wire_layout", ""):
        width = int(row["physical_width"])
        if length % width:
            raise SourceJoinError("repeating source span disagrees with physical array width")
        return [(base+start, base+end) for base in range(0, length, width) for start, end in result]
    return result


def ya1_paths(layout):
    tail = layout.get("tail", {})
    records = tail.get("ya1_records", [])
    paths = {}
    def visit(index, parent, stack):
        if index < 0 or index >= len(records) or index in stack or len(stack) > 64:
            raise SourceJoinError("YA1 path graph is out of bounds or cyclic")
        record = records[index]
        path = parent+"/"+record["name"]
        if index in paths and paths[index] != path:
            raise SourceJoinError("YA1 record has multiple owning paths")
        paths[index] = path
        if record["kind"] & 1:
            for child in range(record["data"], record["data"]+record["size"]):
                visit(child, path, stack | {index})
    for index in range(tail.get("root_first", 0), tail.get("root_first", 0)+tail.get("root_count", 0)):
        visit(index, "", set())
    return paths


def partition(length, candidates, selected):
    events = {0: [], length: []}
    for group, intervals in (("candidate", candidates), ("selected", selected)):
        for index, interval in enumerate(intervals):
            key = (group, index)
            events.setdefault(interval["start"], []).append((True, key, interval))
            events.setdefault(interval["end"], []).append((False, key, interval))
    active = {}
    points = sorted(events)
    for start, end in zip(points, points[1:]):
        for adding, key, interval in events[start]:
            if adding:
                active[key] = interval
            else:
                active.pop(key, None)
        yield start, end, [v for (group, _), v in active.items() if group == "candidate"], [v for (group, _), v in active.items() if group == "selected"]


def leaf_owners(span, layout, paths):
    result = []
    for label in [span["label"]]+span.get("aliases", []):
        record = re.search(r"YA1\.Record\[(\d+)\]", label)
        index = int(record.group(1)) if record else None
        result.append({"label": label, "ya1_path": paths.get(index),
                       "ya1_kind": layout["tail"]["ya1_records"][index]["kind"] if index is not None else None})
    return result


def captured_value(assertion, captures, allow_absent=False):
    if not isinstance(assertion, dict) or assertion.get("capture") not in captures or not isinstance(assertion.get("path"), str):
        raise SourceJoinError("source predicate has no bound capture/path")
    pointer = assertion["path"]
    if pointer and not pointer.startswith("/"):
        raise SourceJoinError("capture path is not a JSON pointer")
    value = captures[assertion["capture"]]
    present = True
    try:
        for key in pointer[1:].split("/") if pointer else []:
            key = key.replace("~1", "/").replace("~0", "~")
            value = value[int(key)] if isinstance(value, list) else value[key]
    except (KeyError, IndexError, TypeError, ValueError):
        present = False
    if not present and not allow_absent:
        raise SourceJoinError("source predicate reads an absent captured operand")
    return present, value


def assertion_value(assertion, captures):
    present, value = captured_value(assertion, captures, True)
    op = assertion.get("op")
    if op == "present":
        return present
    if not present:
        raise SourceJoinError("source predicate reads an absent captured operand")
    expected = assertion.get("value")
    if op in {"eq", "ne"}:
        equal = type(value) is type(expected) and value == expected
        return equal if op == "eq" else not equal
    if op == "nonzero" and type(value) in {int, float}:
        return value != 0
    if op == "nonempty" and isinstance(value, (list, dict, str)):
        return len(value) != 0
    if op == "any_nonzero" and isinstance(value, list) and all(type(v) is int for v in value):
        return any(v != 0 for v in value)
    raise SourceJoinError("unsupported source predicate operation/type")


def carrier_value(choice, captures):
    spec = choice.get("carrier_value")
    if not isinstance(spec, dict):
        raise SourceJoinError("shared physical owner has no captured value")
    _, value = captured_value(spec, captures)
    encoding = spec.get("encoding")
    if encoding == "byte-array":
        try:
            return source_binding.bytes_value(value, len(value))
        except (source_binding.BindingError, TypeError) as exc:
            raise SourceJoinError(str(exc)) from exc
    if encoding in {"u8", "u16", "u32"}:
        width, fmt = {"u8": (1, "B"), "u16": (2, "H"), "u32": (4, "I")}[encoding]
        if type(value) is not int or not 0 <= value < 1 << (8*width):
            raise SourceJoinError("shared owner current value exceeds its physical width")
        return struct.pack("<"+fmt, value)
    raise SourceJoinError("unsupported shared owner value encoding")


def source_facts(decision, captures):
    if not captures:
        raise SourceJoinError("source decision has no bound captures")
    for field in ("guard_assertions", "source_assertions"):
        assertions = decision.get(field, [])
        if not assertions or not all(assertion_value(row, captures) for row in assertions):
            raise SourceJoinError("selected guard/source disagrees with captured operands")
    prior = decision.get("prior_sources_absent", [])
    assertions = decision.get("prior_assertions", {})
    if set(assertions) != set(prior):
        raise SourceJoinError("prior source absence has no exact captured predicates")
    for source in prior:
        if not assertions[source] or all(assertion_value(row, captures) for row in assertions[source]):
            raise SourceJoinError("a preceding source is eligible or unwitnessed")


def optional_body_obligations(layout):
    result = []
    emitted = set()
    for span in layout["decoded_spans"]:
        match = re.match(r"((?:Unit|Humanoid|Human)\[archive=\d+\])\.(Pack|Book)(?:\.|\[)", span["label"])
        if match:
            emitted.add((match[1], match[2]))
    for span in layout["decoded_spans"]:
        match = re.fullmatch(r"(Unit|Humanoid|Human)\[archive=(\d+)\]\.(HasInventory|HasSpellbook)", span["label"])
        if not match or span.get("value") != 0:
            continue
        prefix = span["label"].rsplit(".", 1)[0]
        branch = "Pack" if match[3] == "HasInventory" else "Book"
        if (prefix, branch) in emitted:
            raise SourceJoinError("absent optional body still has emitted child leaves")
        result.append({"class": match[1], "archive_index": int(match[2]), "branch": branch,
                       "presence_label": span["label"], "emitted_leaf_count": 0})
    return result


def scoped_absences(layout, receipt, captures, source_pin, knowledge_pin):
    obligations = optional_body_obligations(layout)
    by_scope = {(row["class"], row["archive_index"], row["branch"]): row for row in obligations}
    asserted = set()
    for absence in receipt.get("scoped_absences", []) if receipt else []:
        scope = absence.get("scope", {})
        if not isinstance(scope, dict) or type(scope.get("archive_index")) is not int:
            raise SourceJoinError("optional-body absence has no exact object scope")
        key = scope.get("class"), scope["archive_index"], absence.get("branch")
        if key not in by_scope or key in asserted or absence.get("presence_label") != by_scope[key]["presence_label"]:
            raise SourceJoinError("optional-body absence names an emitted, missing or duplicate branch")
        if type(absence.get("emitted_leaf_count")) is not int or absence["emitted_leaf_count"] != 0 or absence.get("absence_kind") != "body":
            raise SourceJoinError("optional-body absence has no zero physical child population")
        if absence.get("source_pin") != source_pin or absence.get("knowledge_pin") != knowledge_pin or not absence.get("guard") or not absence.get("witness"):
            raise SourceJoinError("optional-body absence has no matching producer witness")
        facts = absence.get("capture_assertions", [])
        if not facts or not all(assertion_value(fact, captures) for fact in facts):
            raise SourceJoinError("optional-body absence has no captured guard facts")
        asserted.add(key)
    return obligations, [row for key, row in by_scope.items() if key not in asserted], sorted(asserted)


def transport_candidate(row):
    return row.get("class") == "ArchiveFraming" and row.get("wire_member") == "NestedReference" or row.get("class") == "Player" and row.get("wire_member") == "Diary"


def decision_intervals(decision, span, source_pin, knowledge_pin):
    start, end = decision.get("start", 0), decision.get("end", span["length"])
    if type(start) is not int or type(end) is not int or not 0 <= start < end <= span["length"]:
        raise SourceJoinError("decision source span outside field")
    source = decision.get("source")
    if source not in SOURCES:
        raise SourceJoinError("decision has no legal source class")
    for key in ("guard", "producer_callsite", "witness", "source_pin", "knowledge_pin"):
        if not decision.get(key):
            raise SourceJoinError(f"decision has no explicit {key}")
    if decision["source_pin"] != source_pin or decision["knowledge_pin"] != knowledge_pin:
        raise SourceJoinError("decision pins differ from the expected producer pins")
    if type(decision.get("catalog_row")) is not int or decision.get("selected_arm") != source:
        raise SourceJoinError("decision does not select an exact catalog row and source arm")
    if type(decision.get("expected_leaf_count")) is not int or decision["expected_leaf_count"] <= 0:
        raise SourceJoinError("decision has no positive expected selector population")
    if decision.get("guard_result") is not True or decision.get("source_present") is not True:
        raise SourceJoinError("decision lacks actual guard result and selected source presence")
    if source in {"rule", "constructor"} and not decision.get("authority"):
        raise SourceJoinError("rule/constructor decision has no authority")
    if source == "current" and not decision.get("carrier"):
        raise SourceJoinError("current decision has no captured carrier")
    if source == "Document":
        if not decision.get("named_debt"):
            raise SourceJoinError("Document decision has no named debt")
        if decision.get("meaning_unknown") is True and (not decision.get("authority") or not decision.get("reason")):
            raise SourceJoinError("unknown Document fallback has no reason/authority")
    prior = {"current": [], "rule": ["current"], "Document": ["current", "rule"],
             "constructor": ["current", "rule", "Document"]}[source]
    if decision.get("prior_sources_absent", []) != prior:
        raise SourceJoinError("decision does not assert absence of preceding sources")
    return start, end


def join(layout, rows, receipt=None, source_pin=None, knowledge_pin=None, captures=None, binding_context=None):
    if layout["status"] != "parsed":
        raise SourceJoinError("refused physical layout cannot receive source joins")
    for space in ("decoded", "file"):
        observed = byte_layout.coverage(layout[space+"_spans"], layout[space+"_bytes"])
        if observed["gap_bytes"] or observed["overlap_bytes"] or observed.get("outside_bytes", 0):
            raise SourceJoinError("physical layout has a gap, overlap or outside span")
    decisions = []
    if receipt is not None:
        if not source_pin or not knowledge_pin or receipt.get("source_pin") != source_pin or receipt.get("knowledge_pin") != knowledge_pin:
            raise SourceJoinError("source receipt has no matching expected producer pins")
        if receipt.get("input_sha256") != layout["file_sha256"]:
            raise SourceJoinError("source receipt names a different input hash")
        provenance = receipt.get("capture_pin_paths", {})
        if not captures or set(provenance) != set(captures):
            raise SourceJoinError("capture provenance paths do not name every bound capture")
        for name, paths in provenance.items():
            for key, expected in (("source", source_pin), ("knowledge", knowledge_pin)):
                if not assertion_value({"capture": name, "path": paths.get(key), "op": "eq", "value": expected}, captures):
                    raise SourceJoinError("captured producer/knowledge revision differs from expected pins")
        decisions = receipt.get("decisions", [])
    compiled = [(d, re.compile(d["selector"])) for d in decisions]
    joined, missing, known_debt, unmapped, unverified = [], [], [], [], []
    paths = ya1_paths(layout)
    selector_cache = {}
    used = set()
    selected_population = [0 for _ in decisions]
    population = {row["catalog_row"]: 0 for row in rows if row["class"] != "YA1.CurrentState.NativeActions"}
    for space in ("decoded", "file"):
        for span in layout[space+"_spans"]:
            candidates = []
            owners = leaf_owners(span, layout, paths)
            path, record_kind = owners[0]["ya1_path"], owners[0]["ya1_kind"]
            cache_key = space, tuple(owner["label"] for owner in owners)
            if cache_key not in selector_cache:
                selector_cache[cache_key] = [row for row in rows if row.get("atlas_surface", space) in {space, ""} and any(row["selector"].search(owner["label"]) for owner in owners)]
            for row in selector_cache[cache_key]:
                if row["class"] == "YA1.CurrentState.NativeActions":
                    continue
                owner_matches = [owner for owner in owners if row["selector"].search(owner["label"])
                    and (not row.get("atlas_ya1_kind") or str(owner["ya1_kind"]) in row["atlas_ya1_kind"].split("|"))
                    and (row["path_selector"] is None or owner["ya1_path"] is not None and row["path_selector"].search(owner["ya1_path"]))]
                if owner_matches:
                    for start, end in offsets(row, span["length"]):
                        for owner in owner_matches:
                            candidates.append({"start": start, "end": end, "catalog_row": row["catalog_row"], "owner": owner["label"]})
                    population[row["catalog_row"]] += 1
            selected = []
            for index, (decision, selector) in enumerate(compiled):
                if decision.get("space") == space and selector.search(span["label"]):
                    start, end = decision_intervals(decision, span, source_pin, knowledge_pin)
                    selected.append({**decision, "start": start, "end": end})
                    used.add(index)
                    selected_population[index] += 1
            for start, end, matches, chosen in partition(span["length"], candidates, selected):
                if len(chosen) > 1:
                    raise SourceJoinError("source decisions overlap")
                row = {"space": space, "label": span["label"], "wire_member": span.get("wire_member"),
                       "start": span["start"]+start, "end": span["start"]+end, "length": end-start,
                       "ya1_path": path, "ya1_kind": record_kind,
                       "owners": owners,
                       "kind": span["kind"], "candidates": matches}
                if not row["candidates"]:
                    unmapped.append({"space": space, "start": row["start"], "end": row["end"], "label": span["label"]})
                if chosen:
                    decision = chosen[0]
                    choices = [decision]+decision.get("alias_decisions", [])
                    if len(choices) != len(owners):
                        raise SourceJoinError("shared physical leaf needs every owner source choice")
                    choice_owners = set()
                    bindings, owner_values = [], []
                    for choice in choices:
                        owner_label = choice.get("owner_label", span["label"])
                        owner = next((o for o in owners if o["label"] == owner_label), None)
                        if owner is None or owner_label in choice_owners:
                            raise SourceJoinError("source choice names an absent or duplicate physical owner")
                        choice_owners.add(owner_label)
                        choice_start, choice_end = decision_intervals(choice, span, source_pin, knowledge_pin)
                        source_facts(choice, captures)
                        if choice_start > start or choice_end < end or choice["source"] != decision["source"]:
                            raise SourceJoinError("shared physical owner has incompatible source/span choice")
                        if owner["ya1_path"] is not None and (choice.get("ya1_path") != owner["ya1_path"] or choice.get("ya1_kind") != owner["ya1_kind"]):
                            raise SourceJoinError("YA1 source choice lacks its exact owner path/kind")
                        if not any(c["catalog_row"] == choice["catalog_row"] and c["owner"] == owner_label for c in matches):
                            raise SourceJoinError("decision is outside its exact catalog field span/path/kind")
                        catalog_row = rows[choice["catalog_row"]-2]
                        if transport_candidate(catalog_row) and any(c["owner"] == owner_label and not transport_candidate(rows[c["catalog_row"]-2]) for c in matches):
                            raise SourceJoinError("transport parent cannot replace an exact member source choice")
                        route = choice.get("carrier_branch")
                        guard_field = {"base": "current_producer_guard", "overlay": "current_overlay_guard", "dead": "dead_route_guard", "terminal": "dead_route_guard"}.get(route)
                        if guard_field is None or not catalog_row.get(guard_field) or choice.get("catalog_guard") != catalog_row[guard_field]:
                            raise SourceJoinError("decision lacks its exact catalog producer route/guard")
                        if choice["source"] == "Document" and choice.get("meaning_unknown") is True and "unknown-meaning" not in catalog_row["classification"]:
                            raise SourceJoinError("known catalog field cannot be relabeled unknown meaning")
                        try:
                            bindings.append(source_binding.verify(choice, catalog_row, span, start, end, captures, binding_context))
                        except source_binding.BindingError as exc:
                            raise SourceJoinError(str(exc)) from exc
                        if len(owners) > 1:
                            value = carrier_value(choice, captures)
                            if len(value) != span["length"]:
                                raise SourceJoinError("shared owner value differs from its physical leaf width")
                            owner_values.append(value[start:end])
                    if owner_values and any(value != owner_values[0] for value in owner_values):
                        raise SourceJoinError("shared physical owners disagree in captured current values")
                    row["source_bindings"] = bindings
                    if any(binding["verified"] is not True for binding in bindings):
                        unverified.append({"space": space, "start": row["start"], "end": row["end"],
                                           "label": span["label"], "bindings": bindings})
                    row["source"] = decision["source"]
                    row["decision"] = decision
                    if decision["source"] == "Document" and any(choice.get("meaning_unknown") is not True for choice in choices):
                        known_debt.append({"space": space, "start": row["start"], "end": row["end"],
                                           "named_debt": decision["named_debt"]})
                else:
                    row["source"] = None
                    missing.append({"space": space, "start": row["start"], "end": row["end"], "label": span["label"]})
                joined.append(row)
    if len(used) != len(decisions):
        raise SourceJoinError("source receipt contains an unwitnessed selector")
    if any(count != decision["expected_leaf_count"] for count, decision in zip(selected_population, decisions)):
        raise SourceJoinError("source selector population differs from its declared scope")
    absent_rows = {index for index, count in population.items() if count == 0}
    asserted = set()
    for absence in receipt.get("absences", []) if receipt else []:
        index = absence.get("catalog_row")
        if type(index) is not int or index not in absent_rows or index in asserted or type(absence.get("emitted_leaf_count")) is not int or absence.get("emitted_leaf_count") != 0:
            raise SourceJoinError("absence is duplicated or disagrees with emitted catalog population")
        if absence.get("source_pin") != source_pin or absence.get("knowledge_pin") != knowledge_pin:
            raise SourceJoinError("absence names different producer pins")
        if not isinstance(absence.get("guard_result"), bool) or not isinstance(absence.get("source_present"), bool) or not absence.get("guard") or not absence.get("witness") or not absence.get("scope") or absence.get("absence_kind") not in {"class", "body", "list", "JSON omission", "instrument-not-decoded"}:
            raise SourceJoinError("absence has no actual guard/source presence witness")
        assertions = absence.get("capture_assertions", [])
        if not captures or not assertions or not all(assertion_value(row, captures) for row in assertions):
            raise SourceJoinError("absence lacks agreeing captured operands")
        asserted.add(index)
    missing_absences = sorted(absent_rows - asserted)
    scopes, missing_scopes, asserted_scopes = scoped_absences(layout, receipt, captures, source_pin, knowledge_pin)
    try:
        typed = typed_payload.obligations(layout, paths, captures)
    except typed_payload.PayloadError as exc:
        raise SourceJoinError(str(exc)) from exc
    unverified_absences = receipt.get("absences", []) if receipt else []
    return {"status": "complete" if not missing and not known_debt and not unmapped and not missing_absences and not unverified and not unverified_absences and not scopes and not typed else "incomplete",
            "source_classification_joined": not missing and not unverified and not unverified_absences and not scopes and not typed,
            "source_inventory_joined": not missing,
            "unverified_source_bindings": unverified,
            "unverified_source_binding_bytes": sum(row["end"]-row["start"] for row in unverified),
            "optional_body_obligations": scopes,
            "missing_scoped_absence_assertions": missing_scopes,
            "scoped_absence_assertions": asserted_scopes,
            "unverified_scoped_absence_bindings": scopes,
            "unverified_absence_bindings": unverified_absences,
            "typed_payload_obligations": typed,
            "unknown_only_Document_fallback": not known_debt,
            "missing_source_joins": missing, "known_Document_debt": known_debt,
            "missing_source_bytes": sum(row["end"]-row["start"] for row in missing),
            "known_Document_debt_bytes": sum(row["end"]-row["start"] for row in known_debt),
            "unmapped_catalog_bytes": sum(row["end"]-row["start"] for row in unmapped),
            "unmapped_catalog_spans": unmapped,
            "catalog_population": population, "missing_absence_assertions": missing_absences,
            "absence_assertions": receipt.get("absences", []) if receipt else [],
            "catalog_rows": [{key: value for key, value in row.items() if key not in {"selector", "path_selector"}} for row in rows],
            "joined_spans": joined}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--input", required=True)
    parser.add_argument("--output", required=True)
    parser.add_argument("--source-map", required=True)
    parser.add_argument("--decisions")
    parser.add_argument("--source-pin", help="exact producer revision expected by the receipt")
    parser.add_argument("--knowledge-pin", help="exact knowledge revision expected by the receipt")
    parser.add_argument("--capture", action="append", default=[], help="named JSON capture as name=explicit-path")
    parser.add_argument("--catalog-only", action="store_true", help="write guarded candidates; never claim acceptance")
    args = parser.parse_args()
    inputs = [args.input, args.source_map] + ([args.decisions] if args.decisions else [])
    try:
        captures, capture_hashes, capture_bytes = {}, {}, {}
        for item in args.capture:
            name, separator, path = item.partition("=")
            if not separator or not name or name in captures or Path(path).stat().st_size > 64 << 20:
                raise SourceJoinError("capture name/path is duplicated, absent or outside size bound")
            raw = Path(path).read_bytes()
            captures[name], capture_hashes[name] = json.loads(raw.decode("utf-8-sig")), raw_reader.sha(raw)
            capture_bytes[name] = raw
            inputs.append(path)
        target = safe_output(args.output, inputs)
        layout = byte_layout.atlas(args.input)
        receipt = json.loads(Path(args.decisions).read_text(encoding="utf-8-sig")) if args.decisions else None
        if receipt is not None and (not re.fullmatch(r"[0-9a-f]{40}", args.source_pin or "") or not re.fullmatch(r"[0-9a-f]{40}", args.knowledge_pin or "")):
            raise SourceJoinError("acceptance needs explicit full producer and knowledge revisions")
        if receipt is not None and receipt.get("source_map_sha256") != raw_reader.sha(Path(args.source_map).read_bytes()):
            raise SourceJoinError("source receipt names a different source catalog hash")
        if receipt is not None and (not captures or receipt.get("capture_sha256") != capture_hashes):
            raise SourceJoinError("source receipt has no matching exact capture hashes")
        binding_context = source_binding.binding_context(layout, captures, args.source_pin, args.knowledge_pin,
            capture_bytes, Path(args.input).read_bytes())
        result = join(layout, catalog(args.source_map), receipt, args.source_pin, args.knowledge_pin, captures, binding_context)
        result.update({"input_sha256": layout["file_sha256"], "layout": layout,
                       "source_map_sha256": raw_reader.sha(Path(args.source_map).read_bytes()),
                       "instrument_sha256": raw_reader.sha(Path(__file__).read_bytes()),
                       "instrument_bundle_sha256": {Path(module.__file__).name: raw_reader.sha(Path(module.__file__).read_bytes())
                           for module in source_binding.instrument_modules()},
                       "capture_sha256": capture_hashes,
                       "producer_revision": args.source_pin, "knowledge_revision": args.knowledge_pin,
                       "catalog_only": args.catalog_only})
        raw_reader.write_json(target, result)
        print(json.dumps({key: result[key] for key in ("status", "input_sha256", "missing_source_bytes", "known_Document_debt_bytes", "catalog_only")}))
        if result["status"] != "complete" and not args.catalog_only:
            parser.exit(2, "source atlas: incomplete source bindings, absences, typed seams or Document debt; output retained\n")
    except (SourceJoinError, raw_reader.BadSAV, OSError, ValueError, KeyError) as exc:
        parser.exit(2, f"source atlas: {exc}\n")


if __name__ == "__main__":
    main()
