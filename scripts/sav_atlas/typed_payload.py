"""Typed seams are separate obligations from an opaque physical leaf.

These inventories do not select a source arm. Canonical World coordinates are
capture coordinates, never offsets inside the written SAV.
"""

import base64
import json
import struct


class PayloadError(ValueError):
    pass


ACTION_OPTIONALS = (
    "WorldMapReturn", "Program", "TerminalMotions", "Fog", "PendingMessages",
    "Pending", "Animation", "DeathAges", "Manifest", "GroupPlayers",
    "GroupFormations", "GroupParticipants", "PlayerIdentities", "DepartedCharacters", "AbsentPlayers", "ArchiveCoordinates",
    "ActorGroups", "PlayerSlots", "AbsentSessionHead", "AbsentCellTails",
    "AbsentCellRecords", "AbsentMotionCells", "AbsentBlocks", "AbsentDiaries",
    "AbsentCellSacks", "AbsentStructureCells", "CellPlaneResidue", "EffectWidths",
    "Values", "Policy", "Party", "Roster", "Inventory", "Ownership", "NativeAreas",
    "NativeDeliveries", "Session", "NativeBasisWires", "HeldNativeBasisWires",
)


def pointer(key):
    return str(key).replace("~", "~0").replace("/", "~1")


def leaves(value, path="", depth=0):
    if depth > 64:
        raise PayloadError("typed payload exceeds nesting bound")
    if isinstance(value, dict) and value:
        for key in sorted(value):
            yield from leaves(value[key], path+"/"+pointer(key), depth+1)
    elif isinstance(value, list) and value:
        for index, child in enumerate(value):
            yield from leaves(child, path+"/"+str(index), depth+1)
    else:
        yield {"path": path, "value": value, "obligation": "selected source/value"}


def json_value(layout, index):
    record = layout["tail"]["ya1_records"][index]
    if record["kind"] != 6:
        raise PayloadError("typed action payload is not an exact YA1 dword array")
    label = f"YA1.Record[{index}].{record['name']}.PoolValue"
    spans = [row for row in layout["file_spans"] if label in [row["label"]]+row.get("aliases", [])]
    raw = b"".join(bytes.fromhex(row.get("bytes_hex", "")) for row in sorted(spans, key=lambda row: row["start"]))
    if len(raw) != record["size"] or not 8 <= len(raw) <= (8 << 20)+12 or len(raw) % 4:
        raise PayloadError("typed action payload has missing physical bytes or exceeds bound")
    version, size = struct.unpack_from("<II", raw)
    if version != 1 or not 0 < size <= 8 << 20 or len(raw) != (size+11)&~3 or any(raw[8+size:]):
        raise PayloadError("typed action framing/count/padding is malformed")
    framing = {"coordinate_space": "owning YA1 leaf", "version_span": [0, 4],
               "count_span": [4, 8], "JSON_span": [8, 8+size], "padding_span": [8+size, len(raw)]}
    def pairs(items):
        result = {}
        for key, value in items:
            if key in result:
                raise PayloadError("typed action payload has duplicate JSON members")
            result[key] = value
        return result
    def constant(value):
        raise PayloadError("nonfinite typed JSON number")
    try:
        value = json.loads(raw[8:8+size].decode("utf-8"), object_pairs_hook=pairs, parse_constant=constant)
    except (RecursionError, UnicodeError) as exc:
        raise PayloadError("typed JSON nesting or UTF-8 is malformed") from exc
    if not isinstance(value, dict) or type(value.get("Version")) is not int or value["Version"] != 1:
        raise PayloadError("unsupported typed action payload version/shape")
    return value, [{"start": row["start"], "end": row["end"]} for row in spans], framing


def action_obligations(layout, paths):
    result = []
    for index, path in paths.items():
        if path != "/CurrentState/AgainromActions":
            continue
        value, spans, framing = json_value(layout, index)
        rows = list(leaves(value))
        rows += [{"path": "/"+field, "obligation": "scoped JSON omission"}
                 for field in ACTION_OPTIONALS if field not in value]
        actions = value.get("Actions", {})
        if not isinstance(actions, dict):
            raise PayloadError("typed Actions has a malformed shape")
        for prefix, actors in (("/Actions/Actors", actions.get("Actors")), ("/Held", value.get("Held"))):
            if actors is not None and not isinstance(actors, list):
                raise PayloadError("typed action actor population is malformed")
            for position, actor in enumerate(actors or []):
                if not isinstance(actor, dict) or type(actor.get("Entity")) is not int:
                    raise PayloadError("typed action actor has no exact Entity scope")
                current = actor.get("Current")
                if current is not None and not isinstance(current, dict):
                    raise PayloadError("typed actor Current has a malformed shape")
                if current is not None and "AdmittedBookSpell" in current and (type(current["AdmittedBookSpell"]) is not int or not 0 <= current["AdmittedBookSpell"] <= 28):
                    raise PayloadError("typed actor admitted book slot is malformed")
                if current is None or "AdmittedBookSpell" not in current:
                    rows.append({"path": f"{prefix}/{position}/Current/AdmittedBookSpell",
                                 "entity": actor["Entity"], "obligation": "scoped JSON omission"})
                if current is None or "NativeBasis" not in current:
                    rows.append({"path": f"{prefix}/{position}/Current/NativeBasis",
                                 "entity": actor["Entity"], "obligation": "scoped JSON omission"})
        result.append({"family": "CurrentAction", "ya1_path": path, "ya1_kind": 6,
                       "file_spans": spans, "framing": framing, "obligations": rows, "verified": False})
    return result


def participant_section(raw):
    result = {"present": False, "records": [], "coordinate_space": "canonical World capture"}
    if not raw or raw[0] not in (110, 111):
        return raw, result
    independent = raw[0] == 111
    tag, prefix, width = (b"CPP1", 5, 12) if independent else (b"PPT1", 4, 8)
    if len(raw) < 10+prefix or raw[-4:] != tag:
        raise PayloadError("Player form has no bounded identity footer")
    span = struct.unpack_from("<I", raw, len(raw)-9)[0]
    start = len(raw)-9-span
    if start < 1 or span < prefix or raw[-5] >= raw[0]:
        raise PayloadError("Player span or base version is malformed")
    present = True
    if independent:
        if raw[start] not in (0, 1):
            raise PayloadError("current Player Participant presence is malformed")
        present = raw[start] == 1
    count = struct.unpack_from("<I", raw, start+prefix-4)[0]
    if count > 65535 or span != prefix+width*count:
        raise PayloadError("Player count does not match canonical span")
    records, previous = [], 0
    for index in range(count):
        at = start+prefix+width*index
        player = struct.unpack_from("<I", raw, at)[0]
        value = struct.unpack_from("<I", raw, at+width-4)[0]
        if player <= previous:
            raise PayloadError("Player identities are not exact ordered nonzero IDs")
        if not present and value:
            raise PayloadError("absent Participant has a current value")
        previous = player
        record = {"player": player, "value": value, "start": at, "end": at+width,
                  "identity_span": [at, at+4], "value_span": [at+width-4, at+width]}
        if independent:
            record.update(slot=struct.unpack_from("<I", raw, at+4)[0], slot_span=[at+4, at+8])
        records.append(record)
    base_raw = bytes([raw[-5]])+raw[1:start]
    result.update(present=True, family=tag.decode(), participants_present=present,
                  count=count, count_span=[start+prefix-4, start+prefix], start=start,
                  end=len(raw), footer_span=[len(raw)-9, len(raw)], base_version=raw[-5], records=records)
    if independent:
        result["presence_span"] = [start, start+1]
    return base_raw, result


def native_basis_section(raw):
    raw, _ = participant_section(raw)
    result = {"present": False, "records": [], "coordinate_space": "canonical World capture"}
    if not raw or raw[0] != 109:
        return raw, result
    if len(raw) < 121 or raw[-4:] != b"NAB1":
        raise PayloadError("form109 has no bounded NAB1 footer")
    span = struct.unpack_from("<I", raw, len(raw)-9)[0]
    start = len(raw)-9-span
    if start < 1 or span < 111 or raw[-5] >= 109:
        raise PayloadError("NAB1 span or base version is malformed")
    count = struct.unpack_from("<I", raw, start)[0]
    if not 0 < count <= 65535 or span != 4+107*count:
        raise PayloadError("NAB1 count does not match canonical span")
    records, previous = [], None
    for index in range(count):
        at = start+4+107*index
        entity, flags, base_known = struct.unpack_from("<IBI", raw, at)
        base = raw[at+9:at+33]
        modifier_known = struct.unpack_from("<Q", raw, at+33)[0]
        modifier = raw[at+41:at+105]
        body = struct.unpack_from("<H", raw, at+105)[0]
        if previous is not None and entity <= previous or not 0 < flags <= 15 or base_known >> 24:
            raise PayloadError("NAB1 entity order, flags or knowledge mask is malformed")
        if not flags & 1 and (base_known or any(base)) or not flags & 2 and (modifier_known or any(modifier)) or not flags & 4 and (flags & 8 or body):
            raise PayloadError("NAB1 absent component has residue")
        previous = entity
        records.append({"entity": entity, "flags": flags, "base_known": base_known,
                        "modifier_known": modifier_known, "body": body, "start": at, "end": at+107,
                        "base_span": [at+9, at+33], "modifier_span": [at+41, at+105],
                        "body_span": [at+105, at+107], "mask_spans": [[at+5, at+9], [at+33, at+41]]})
    base_raw = bytes([raw[-5]])+raw[1:start]
    result.update(present=True, count=count, count_span=[start, start+4], start=start,
                  end=len(raw), footer_span=[len(raw)-9, len(raw)], base_version=raw[-5], records=records)
    return base_raw, result


def book_section(raw):
    raw, outer = native_basis_section(raw)
    if not raw:
        return {"present": False, "selections": [], "coordinate_space": "canonical World capture"}
    if raw[0] != 108:
        return {"present": False, "selections": [], "base_version": raw[0],
                "coordinate_space": "canonical World capture"}
    if len(raw) < 19 or raw[-4:] != b"BSL1":
        raise PayloadError("form108 has no bounded BSL1 footer")
    span = struct.unpack_from("<I", raw, len(raw)-9)[0]
    start = len(raw)-9-span
    if span < 10 or start < 1 or raw[-5] >= 108:
        raise PayloadError("BSL1 span or base version is malformed")
    count = struct.unpack_from("<I", raw, start)[0]
    if not 0 < count <= 65535 or span != 4+6*count:
        raise PayloadError("BSL1 count does not match canonical span")
    rows, previous = [], None
    for index in range(count):
        at = start+4+6*index
        entity, slot = struct.unpack_from("<IH", raw, at)
        if previous is not None and entity <= previous or not 1 <= slot <= 28:
            raise PayloadError("BSL1 entity order or sparse book slot is malformed")
        previous = entity
        rows.append({"entity": entity, "admitted_book_spell": slot, "start": at, "end": at+6})
    return {"present": True, "base_version": raw[-5], "start": start, "end": len(raw),
            "count": count, "count_span": [start, start+4], "selections": rows,
            "footer_span": [len(raw)-9, len(raw)], "coordinate_space": "canonical World capture"}


def capture_obligations(captures):
    result = []
    for name, wrapper in captures.items():
        capture = wrapper.get("Capture") if isinstance(wrapper, dict) else None
        if not isinstance(capture, dict) or not isinstance(capture.get("Snapshot"), dict):
            continue
        snapshot = capture["Snapshot"]
        if "Residue" in snapshot:
            result.append({"family": "Residue", "capture": name, "capture_path": "/Capture/Snapshot/Residue",
                           "obligations": list(leaves(snapshot["Residue"])), "verified": False})
        if "World" not in snapshot:
            continue
        encoded = snapshot["World"]
        if encoded is None:
            raw = b""
        elif isinstance(encoded, str) and len(encoded) <= (64 << 20)*4//3+4:
            try:
                raw = base64.b64decode(encoded, validate=True)
            except ValueError as exc:
                raise PayloadError("canonical World capture is not base64") from exc
        else:
            raise PayloadError("canonical World capture has unsupported shape/size")
        raw_base, participants = participant_section(raw)
        if participants["present"]:
            result.append({"family": participants["family"], "capture": name, "capture_path": "/Capture/Snapshot/World",
                           "layout": participants, "obligation": "exact Player identity, presence/source and ordinary Participant correspondence",
                           "verified": False})
        raw_base, basis = native_basis_section(raw_base)
        if basis["present"]:
            result.append({"family": "NAB1", "capture": name, "capture_path": "/Capture/Snapshot/World",
                           "layout": basis, "obligation": "typed per-byte knowledge/source and ordinary-wire correspondence",
                           "verified": False})
        section = book_section(raw_base)
        result.append({"family": "BSL1", "capture": name, "capture_path": "/Capture/Snapshot/World",
                       "layout": section, "obligation": "typed presence/source and action/raw U44 correspondence",
                       "verified": False})
    return result


def obligations(layout, paths, captures):
    return action_obligations(layout, paths)+capture_obligations(captures or {})
