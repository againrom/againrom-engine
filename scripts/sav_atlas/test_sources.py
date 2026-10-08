import copy
import base64
import json
import re
import struct
import subprocess
import sys
import tempfile
import unittest
from unittest import mock
from pathlib import Path

import byte_layout
import join_sources
import source_binding
import registry_binding
import typed_payload


def synthetic_save():
    word = lambda *values: struct.pack("<"+"I"*len(values), *values)
    body = word(0, 0)+b"\x00"+word(*([0]*14))+word(0, 0)+b"\x00"+word(0xBADFACE1, 0, 0, 0)+bytes(392)
    body += bytes(len(body) % 2)
    packets = word(len(body)//2)
    for at in range(0, len(body), 254):
        block = body[at:at+254]
        packets += bytes([len(block)//2])+block
    store = b"&YA1"+word(0, 0, 1, 0, 0, 0)
    campaign = word(*([0]*8))+word(*([0]*10))+word(*([0]*8))
    return b"Asg&"+word(16+len(packets), 0x0BAD0002, len(packets))+packets+bytes(256)+store+campaign


def synthetic_optional_books():
    word = lambda *values: struct.pack("<"+"I"*len(values), *values)
    def unit(book):
        body = bytes(37)+word(0)+bytes(4)+bytes(24+22+24+64+180+148)+bytes(2)
        body += bytes(19)+bytes(4)+b"\0"+bytes(28)+bytes(14+13)+bytes(2)
        body += b"\0"+bytes([int(book)])
        if book:
            body += word(0, 2)+bytes(2)
        return body+bytes(17)
    first = struct.pack("<HHH", 0xffff, 1, 4)+b"Unit"+unit(False)
    second = struct.pack("<H", 0x8001)+unit(True)
    body = word(0, 0)+b"\0"+word(*([0]*14))+word(0, 2)+first+second+b"\0"
    body += word(0xBADFACE1, 0, 0, 0)+bytes(392)
    body += bytes(len(body) % 2)
    packets = word(len(body)//2)
    for at in range(0, len(body), 254):
        block = body[at:at+254]
        packets += bytes([len(block)//2])+block
    store = b"&YA1"+word(0, 0, 1, 0, 0, 0)
    return b"Asg&"+word(16+len(packets), 0x0BAD0002, len(packets))+packets+bytes(256)+store+word(*([0]*26))


def synthetic_actions(payload):
    word = lambda *values: struct.pack("<"+"I"*len(values), *values)
    raw = payload.encode("utf-8")
    raw = word(1, len(raw))+raw+bytes(((len(raw)+11)&~3)-(len(raw)+8))
    def record(name, data, size, kind):
        return word(0, data, size, kind)+name.encode("ascii").ljust(16, b"\0")
    store = b"&YA1"+word(0, 1, 1, 2, 0)
    store += record("CurrentState", 1, 1, 1)+record("AgainromActions", 0, len(raw), 6)
    return synthetic_save().split(b"&YA1")[0]+store+word(len(raw))+raw+word(*([0]*26))


class SourceTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.path = Path(self.temp.name)/"synthetic.sav"
        self.path.write_bytes(synthetic_save())
        self.layout = byte_layout.atlas(self.path)
        self.assertEqual("parsed", self.layout["status"], self.layout.get("error"))
        self.rows = []
        for space in ("decoded", "file"):
            for span in self.layout[space+"_spans"]:
                self.rows.append({"catalog_row": len(self.rows)+2, "class": "Synthetic",
                    "member_span": "whole", "physical_width": str(span["length"]),
                    "current_producer_guard": "synthetic fixture field",
                    "selector": re.compile("^"+re.escape(span["label"])+"$"), "path_selector": None,
                    "atlas_surface": space, "classification": "known constructor fixture"})

    def joined(self, receipt, rows=None):
        captures = {"fixture": {"selected": True, "preceding": False, "no_records": True, "value": 0, "other_value": 17,
                                "source": "synthetic", "knowledge": "synthetic"}}
        return join_sources.join(self.layout, self.rows if rows is None else rows, receipt, "synthetic", "synthetic", captures)

    def assert_inventory(self, joined):
        self.assertEqual("incomplete", joined["status"])
        self.assertTrue(joined["source_inventory_joined"])
        self.assertFalse(joined["source_classification_joined"])

    def receipt(self):
        result = {"input_sha256": self.layout["file_sha256"], "source_pin": "synthetic", "knowledge_pin": "synthetic", "decisions": [],
                  "capture_pin_paths": {"fixture": {"source": "/source", "knowledge": "/knowledge"}}}
        index = 0
        for space in ("decoded", "file"):
            for span in self.layout[space+"_spans"]:
                result["decisions"].append({"space": space, "selector": "^"+re.escape(span["label"])+"$",
                    "catalog_row": index+2, "selected_arm": "constructor",
                    "expected_leaf_count": 1,
                    "carrier_branch": "base", "catalog_guard": "synthetic fixture field",
                    "guard_result": True, "source_present": True, "prior_sources_absent": ["current", "rule", "Document"],
                    "guard_assertions": [{"capture": "fixture", "path": "/selected", "op": "eq", "value": True}],
                    "source_assertions": [{"capture": "fixture", "path": "/selected", "op": "eq", "value": True}],
                    "prior_assertions": {source: [{"capture": "fixture", "path": "/preceding", "op": "eq", "value": True}] for source in ("current", "rule", "Document")},
                    "source": "constructor", "guard": "explicit synthetic constructor",
                    "producer_callsite": "test_sources.synthetic_save", "witness": "independent packed fields",
                    "source_pin": "synthetic", "knowledge_pin": "synthetic", "authority": "synthetic fixture only"})
                index += 1
        return result

    def test_declared_source_inventory_and_missing_join(self):
        receipt = self.receipt()
        joined = self.joined(receipt)
        self.assert_inventory(joined)
        receipt["decisions"].pop()
        missing = self.joined(receipt)
        self.assertEqual("incomplete", missing["status"])
        self.assertGreater(missing["missing_source_bytes"], 0)
        self.assertFalse(missing["source_classification_joined"])

    def test_declared_predicates_do_not_establish_semantic_binding(self):
        receipt = self.receipt()
        receipt["decisions"][0]["guard_assertions"][0]["path"] = "/no_records"
        joined = self.joined(receipt)
        self.assertEqual("incomplete", joined["status"])
        self.assertFalse(joined["source_classification_joined"])
        self.assertGreater(joined["unverified_source_binding_bytes"], 0)

    def test_typed_action_members_and_omissions_are_not_opaque_debt(self):
        payload = {"Version": 1, "Actions": {"Actors": [
            {"Entity": 0, "Current": {"AdmittedBookSpell": 28}},
            {"Entity": 4, "Current": {"RotationSpeed": 0}}]}}
        self.path.write_bytes(synthetic_actions(json.dumps(payload)))
        layout = byte_layout.atlas(self.path)
        self.assertEqual("parsed", layout["status"], layout.get("error"))
        result = typed_payload.obligations(layout, join_sources.ya1_paths(layout), {})
        self.assertEqual(1, len(result))
        action = result[0]
        self.assertEqual("CurrentAction", action["family"])
        self.assertFalse(action["verified"])
        rows = {row["path"]: row for row in action["obligations"]}
        self.assertEqual(28, rows["/Actions/Actors/0/Current/AdmittedBookSpell"]["value"])
        self.assertEqual("scoped JSON omission", rows["/Actions/Actors/1/Current/AdmittedBookSpell"]["obligation"])
        self.assertEqual(4, rows["/Actions/Actors/1/Current/AdmittedBookSpell"]["entity"])
        self.assertIn("/AbsentSessionHead", rows)
        payload["Held"] = [{"Entity": 7, "Current": {"AdmittedBookSpell": 0}}, {"Entity": 9}]
        self.path.write_bytes(synthetic_actions(json.dumps(payload)))
        held_layout = byte_layout.atlas(self.path)
        held = typed_payload.obligations(held_layout, join_sources.ya1_paths(held_layout), {})
        held_rows = {row["path"]: row for row in held[0]["obligations"]}
        self.assertEqual(7, held_rows["/Held/0/Current/NativeBasis"]["entity"])
        self.assertEqual(9, held_rows["/Held/1/Current/NativeBasis"]["entity"])
        self.assertEqual(9, held_rows["/Held/1/Current/AdmittedBookSpell"]["entity"])
        for changed in ('{"Version":1,"Version":1}', '{"Version":true}', '{"Version":1,"Actions":{"Actors":[{"Entity":true}]}}'):
            self.path.write_bytes(synthetic_actions(changed))
            changed_layout = byte_layout.atlas(self.path)
            with self.assertRaises(typed_payload.PayloadError):
                typed_payload.obligations(changed_layout, join_sources.ya1_paths(changed_layout), {})

    def test_participant_layout_is_a_separate_exact_identity_obligation(self):
        base = bytes([109])+bytes(63)
        rows = struct.pack("<IIIII", 2, 101, 0, 205, 0xf1234567)
        raw = bytes([110])+base[1:]+rows+struct.pack("<I", len(rows))+bytes([109])+b"PPT1"
        peeled, section = typed_payload.participant_section(raw)
        self.assertEqual(base, peeled)
        self.assertTrue(section["present"])
        self.assertEqual([101, 205], [row["player"] for row in section["records"]])
        self.assertEqual([0, 0xf1234567], [row["value"] for row in section["records"]])
        self.assertEqual([72, 76], section["records"][0]["value_span"])
        for at, value in ((64, 3), (68, 0), (76, 101), (len(raw)-9, 3)):
            wrong = bytearray(raw)
            struct.pack_into("<I", wrong, at, value)
            with self.assertRaises(typed_payload.PayloadError):
                typed_payload.participant_section(wrong)
        empty = bytes([110])+bytes(63)+bytes(4)+struct.pack("<I", 4)+bytes([95])+b"PPT1"
        _, section = typed_payload.participant_section(empty)
        self.assertTrue(section["present"])
        self.assertEqual(0, section["count"])
        self.assertFalse(typed_payload.participant_section(base)[1]["present"])

    def test_independent_player_identity_presence_and_coordinates(self):
        base = bytes([109])+bytes(63)
        rows = bytes([1])+struct.pack("<I", 2)+struct.pack("<IIIIII", 101, 0, 0, 205, 0, 0xf1234567)
        raw = bytes([111])+base[1:]+rows+struct.pack("<I", len(rows))+bytes([109])+b"CPP1"
        peeled, section = typed_payload.participant_section(raw)
        self.assertEqual(base, peeled)
        self.assertEqual("CPP1", section["family"])
        self.assertTrue(section["participants_present"])
        self.assertEqual(0, section["records"][0]["slot"])
        self.assertEqual([69, 73], section["records"][0]["identity_span"])
        self.assertEqual([73, 77], section["records"][0]["slot_span"])
        self.assertEqual([89, 93], section["records"][1]["value_span"])
        for position, value in ((64, 2), (64, 0), (69, 0), (81, 101)):
            wrong = bytearray(raw)
            wrong[position] = value
            with self.assertRaises(typed_payload.PayloadError):
                typed_payload.participant_section(wrong)
        empty = bytes([111])+bytes(63)+bytes([0])+struct.pack("<I", 0)+struct.pack("<I", 5)+bytes([95])+b"CPP1"
        _, section = typed_payload.participant_section(empty)
        self.assertFalse(section["participants_present"])
        self.assertEqual(0, section["count"])

    def test_bsl_layout_and_residue_are_separate_capture_obligations(self):
        base = bytes([108])+bytes(63)
        rows = struct.pack("<IIHIH", 2, 0, 28, 99, 4)
        raw = base+rows+struct.pack("<I", len(rows))+bytes([107])+b"BSL1"
        captures = {"observed": {"Capture": {"Snapshot": {"World": base64.b64encode(raw).decode("ascii"),
                    "Residue": {"GroupTag": 0, "FogVisible": None}}}}}
        result = typed_payload.capture_obligations(captures)
        self.assertEqual(["Residue", "BSL1"], [row["family"] for row in result])
        section = result[1]["layout"]
        self.assertEqual("canonical World capture", section["coordinate_space"])
        self.assertEqual(2, section["count"])
        self.assertEqual(25, section["end"]-section["start"])
        self.assertEqual([0, 99], [row["entity"] for row in section["selections"]])
        self.assertFalse(result[1]["verified"])

        native = struct.pack("<IIBI", 1, 0, 15, (1 << 24)-1)+bytes(24)+struct.pack("<Q", (1 << 64)-1)+bytes(64)+bytes(2)
        nested = bytes([109])+raw[1:]+native+struct.pack("<I", len(native))+bytes([108])+b"NAB1"
        captures["observed"]["Capture"]["Snapshot"]["World"] = base64.b64encode(nested).decode("ascii")
        result = typed_payload.capture_obligations(captures)
        self.assertEqual(["Residue", "NAB1", "BSL1"], [row["family"] for row in result])
        self.assertTrue(result[1]["layout"]["present"])
        self.assertEqual([0, 99], [row["entity"] for row in result[2]["layout"]["selections"]])
        participants = struct.pack("<III", 1, 205, 0)
        wrapped = bytes([110])+nested[1:]+participants+struct.pack("<I", len(participants))+bytes([109])+b"PPT1"
        captures["observed"]["Capture"]["Snapshot"]["World"] = base64.b64encode(wrapped).decode("ascii")
        result = typed_payload.capture_obligations(captures)
        self.assertEqual(["Residue", "PPT1", "NAB1", "BSL1"], [row["family"] for row in result])
        self.assertEqual(205, result[1]["layout"]["records"][0]["player"])
        self.assertEqual([0, 99], [row["entity"] for row in result[3]["layout"]["selections"]])
        for offset, byte in ((len(raw)+8, 16), (len(raw)+12, 1), (len(nested)-5, 109)):
            wrong = bytearray(nested)
            wrong[offset] = byte
            with self.assertRaises(typed_payload.PayloadError):
                typed_payload.native_basis_section(wrong)
        self.assertFalse(typed_payload.book_section(bytes([107])+bytes(63))["present"])
        self.assertFalse(typed_payload.book_section(b"")["present"])
        for offset, data in ((64, bytes(4)), (68, struct.pack("<I", 100)), (72, bytes(2)), (-9, bytes(4))):
            wrong = bytearray(raw)
            at = offset if offset >= 0 else len(raw)+offset
            wrong[at:at+len(data)] = data
            with self.assertRaises(typed_payload.PayloadError):
                typed_payload.book_section(bytes(wrong))

    def test_cli_records_every_instrument_module_hash(self):
        output = Path(self.temp.name)/"catalog-only.json"
        scripts = Path(__file__).parent
        command = [sys.executable, "-B", str(scripts/"join_sources.py"), "--input", str(self.path),
                   "--output", str(output), "--source-map", str(scripts/"source_catalog.json"), "--catalog-only"]
        result = subprocess.run(command, capture_output=True, text=True)
        self.assertEqual(0, result.returncode, result.stderr)
        joined = json.loads(output.read_text(encoding="utf-8"))
        self.assertEqual("incomplete", joined["status"])
        modules = joined["instrument_bundle_sha256"]
        self.assertEqual({"byte_layout.py", "raw_reader.py", "source_binding.py", "registry_binding.py", "typed_payload.py"}, set(modules))
        for name, digest in modules.items():
            self.assertEqual(join_sources.raw_reader.sha((scripts/name).read_bytes()), digest)
        self.assertEqual(join_sources.raw_reader.sha((scripts/"join_sources.py").read_bytes()), joined["instrument_sha256"])

    def test_current_session_binding_checks_exact_predicates_and_physical_bytes(self):
        for wire, carrier, width in (("RawA828", "RawSessionMid", 400), ("Raw08", "RawSessionHead", 48)):
            raw = bytes(width) if width == 400 else bytes(16)+(10000000).to_bytes(8, "little")+bytes(8)+(10000).to_bytes(4, "little")+bytes(12)
            name = "World."+carrier
            captures = {"actual": {"Capture": {"WorldPresent": True, carrier: list(raw), "unrelated_true": True}}}
            guard = [{"capture": "actual", "path": "/Capture/WorldPresent", "op": "eq", "value": True}]
            if width == 48:
                guard.append({"capture": "actual", "path": "/Capture/RawSessionHead", "op": "any_nonzero"})
            decision = {"space": "decoded", "binding_id": name, "source_capture": "actual", "source": "current",
                        "carrier": name, "carrier_branch": "base", "guard_assertions": guard,
                        "source_assertions": [{"capture": "actual", "path": "/Capture/"+carrier, "op": "present"}],
                        "prior_assertions": {}, "carrier_value": {"capture": "actual", "path": "/Capture/"+carrier, "encoding": "byte-array"}}
            row = {"wire_member": wire}
            span = {"label": "Session."+wire, "length": width, "bytes_hex": raw.hex()}
            result = source_binding.verify(decision, row, span, 0, width, captures)
            self.assertTrue(result["value_verified"])
            self.assertFalse(result["verified"])
            wrong = copy.deepcopy(decision)
            wrong["guard_assertions"][0]["path"] = "/Capture/unrelated_true"
            with self.assertRaises(source_binding.BindingError):
                source_binding.verify(wrong, row, span, 0, width, captures)
            mutated = bytearray(raw)
            mutated[0] = 99
            with self.assertRaises(source_binding.BindingError):
                source_binding.verify(decision, row, {**span, "bytes_hex": mutated.hex()}, 0, width, captures)
            with self.assertRaises(source_binding.BindingError):
                source_binding.verify({**decision, "source": "Document"}, row, span, 0, width, captures)
            captures["actual"]["Capture"][carrier] = list(bytes(width))
            if width == 48:
                with self.assertRaises(source_binding.BindingError):
                    source_binding.verify(decision, row, {**span, "bytes_hex": bytes(width).hex()}, 0, width, captures)
            else:
                self.assertTrue(source_binding.verify(decision, row, span, 0, width, captures)["value_verified"])

    def test_per_object_body_omission_is_distinct_from_global_population(self):
        path = Path(self.temp.name)/"optional-books.sav"
        path.write_bytes(synthetic_optional_books())
        layout = byte_layout.atlas(path)
        self.assertEqual("parsed", layout["status"], layout.get("error"))
        obligations = join_sources.optional_body_obligations(layout)
        self.assertIn({"class": "Unit", "archive_index": 2, "branch": "Book",
                       "presence_label": "Unit[archive=2].HasSpellbook", "emitted_leaf_count": 0}, obligations)
        self.assertTrue(any(span["label"] == "Unit[archive=3].Book.Count" for span in layout["decoded_spans"]))
        captures = {"fixture": {"actors": [{"has_book": False}, {"has_book": True}]}}
        receipt = {"scoped_absences": []}
        self.assertEqual(obligations, join_sources.scoped_absences(layout, receipt, captures, "synthetic", "synthetic")[1])
        receipt["scoped_absences"] = [{"scope": {"class": "Unit", "archive_index": 2}, "branch": "Book",
            "presence_label": "Unit[archive=2].HasSpellbook", "emitted_leaf_count": 0, "absence_kind": "body",
            "source_pin": "synthetic", "knowledge_pin": "synthetic", "guard": "synthetic omitted book",
            "witness": "constructed grammar only", "capture_assertions": [{"capture": "fixture", "path": "/actors/0/has_book", "op": "eq", "value": False}]}]
        _, missing, asserted = join_sources.scoped_absences(layout, receipt, captures, "synthetic", "synthetic")
        self.assertEqual([("Unit", 2, "Book")], asserted)
        self.assertEqual(2, len(missing))
        receipt["scoped_absences"][0]["scope"]["archive_index"] = 3
        with self.assertRaises(join_sources.SourceJoinError):
            join_sources.scoped_absences(layout, receipt, captures, "synthetic", "synthetic")

    def test_gap_overlap_and_outside_are_rejected(self):
        for mutation in ("gap", "overlap", "outside"):
            broken = copy.deepcopy(self.layout)
            span = broken["decoded_spans"][0]
            if mutation == "gap":
                span["start"] += 1
            elif mutation == "overlap":
                broken["decoded_spans"].append(copy.deepcopy(span))
            else:
                span["start"] = -1
            with self.assertRaises(join_sources.SourceJoinError):
                join_sources.join(broken, [])

    def test_bad_source_intervals_receipts_and_unwitnessed_branch(self):
        for field, value in (("source", "opaque"), ("end", 1<<30), ("guard", "")):
            receipt = self.receipt()
            receipt["decisions"][0][field] = value
            with self.assertRaises(join_sources.SourceJoinError):
                self.joined(receipt)
        receipt = self.receipt()
        receipt["input_sha256"] = "different"
        with self.assertRaises(join_sources.SourceJoinError):
            self.joined(receipt)
        receipt = self.receipt()
        receipt["decisions"].append({**receipt["decisions"][0], "selector": "^AbsentClass\\."})
        with self.assertRaises(join_sources.SourceJoinError):
            self.joined(receipt)
        receipt = self.receipt()
        receipt["decisions"].append(receipt["decisions"][0])
        with self.assertRaises(join_sources.SourceJoinError):
            self.joined(receipt)

    def test_unknown_Document_is_distinct_from_known_missing_basis(self):
        receipt = self.receipt()
        row = receipt["decisions"][0]
        row.update(source="Document", selected_arm="Document", prior_sources_absent=["current", "rule"], named_debt="missing modeled source")
        del row["prior_assertions"]["Document"]
        known = self.joined(receipt)
        self.assertEqual("incomplete", known["status"])
        self.assertTrue(known["source_inventory_joined"])
        self.assertFalse(known["unknown_only_Document_fallback"])
        row.update(meaning_unknown=True)
        with self.assertRaises(join_sources.SourceJoinError):
            self.joined(receipt)
        row.update(reason="meaning unnamed by this fixture", authority="synthetic only")
        with self.assertRaises(join_sources.SourceJoinError):
            self.joined(receipt)
        self.rows[0]["classification"] = "unknown-meaning synthetic span"
        self.assert_inventory(self.joined(receipt))

    def test_pins_catalog_and_broad_selector_cannot_claim_acceptance(self):
        for field, value in (("source_pin", "different"), ("knowledge_pin", "different"),
                             ("catalog_row", 100000), ("catalog_row", True), ("start", True), ("end", True),
                             ("selected_arm", "current"), ("selector", ".*"), ("guard_result", False), ("source_present", False),
                             ("catalog_guard", "other guard"), ("carrier_branch", "overlay")):
            receipt = self.receipt()
            receipt["decisions"][0][field] = value
            with self.assertRaises(join_sources.SourceJoinError):
                self.joined(receipt)
        receipt = self.receipt()
        receipt["decisions"][0]["expected_leaf_count"] = 2
        with self.assertRaises(join_sources.SourceJoinError):
            self.joined(receipt)
        receipt = self.receipt()
        receipt["source_pin"] = "different"
        with self.assertRaises(join_sources.SourceJoinError):
            self.joined(receipt)
        self.assertEqual("incomplete", self.joined(None, rows=[])["status"])
        with self.assertRaises(join_sources.SourceJoinError):
            self.joined(self.receipt(), rows=[])

    def test_absent_population_requires_an_explicit_witness(self):
        self.rows.append({**self.rows[0], "catalog_row": len(self.rows)+2,
                          "selector": re.compile("^Unpopulated\\.")})
        receipt = self.receipt()
        self.assertEqual("incomplete", self.joined(receipt)["status"])
        receipt["absences"] = [{"catalog_row": self.rows[-1]["catalog_row"], "emitted_leaf_count": 0,
            "scope": "entire synthetic fixture", "absence_kind": "class",
            "capture_assertions": [{"capture": "fixture", "path": "/no_records", "op": "eq", "value": True}],
            "source_pin": "synthetic", "knowledge_pin": "synthetic", "source_present": False,
            "guard_result": False, "guard": "no synthetic actor", "witness": "fixture contains no records"}]
        self.assert_inventory(self.joined(receipt))
        receipt["absences"][0]["catalog_row"] = 2
        with self.assertRaises(join_sources.SourceJoinError):
            self.joined(receipt)

    def test_bound_guard_and_precedence_predicates_are_evaluated(self):
        for key in ("guard_assertions", "source_assertions"):
            receipt = self.receipt()
            receipt["decisions"][0][key][0]["value"] = False
            with self.assertRaises(join_sources.SourceJoinError):
                self.joined(receipt)
        receipt = self.receipt()
        receipt["decisions"][0]["prior_assertions"]["current"][0]["path"] = "/selected"
        with self.assertRaises(join_sources.SourceJoinError):
            self.joined(receipt)
        receipt = self.receipt()
        receipt["capture_pin_paths"]["fixture"]["source"] = "/absent_revision"
        with self.assertRaises(join_sources.SourceJoinError):
            self.joined(receipt)
        receipt = self.receipt()
        receipt["decisions"][0]["source_assertions"][0]["path"] = "/absent"
        with self.assertRaises(join_sources.SourceJoinError):
            self.joined(receipt)

    def test_shared_physical_owners_need_compatible_exact_source_choices(self):
        span = self.layout["decoded_spans"][0]
        alias = "SharedOwner.ExactField"
        span["aliases"] = [alias]
        alias_row = {**self.rows[0], "catalog_row": len(self.rows)+2, "selector": re.compile("^"+re.escape(alias)+"$")}
        self.rows.append(alias_row)
        receipt = self.receipt()
        with self.assertRaises(join_sources.SourceJoinError):
            self.joined(receipt)
        choice = {**receipt["decisions"][0], "catalog_row": alias_row["catalog_row"], "owner_label": alias}
        receipt["decisions"][0]["alias_decisions"] = [choice]
        value = {"capture": "fixture", "path": "/value", "encoding": "u32"}
        choice["carrier_value"] = value
        receipt["decisions"][0]["carrier_value"] = value
        self.assert_inventory(self.joined(receipt))
        choice["carrier_value"] = {**value, "path": "/other_value"}
        with self.assertRaises(join_sources.SourceJoinError):
            self.joined(receipt)
        choice["carrier_value"] = value
        choice.update(source="current", selected_arm="current", prior_sources_absent=[], carrier="other live carrier")
        with self.assertRaises(join_sources.SourceJoinError):
            self.joined(receipt)

    def test_exact_member_source_outranks_transport_parent(self):
        span = self.layout["decoded_spans"][0]
        broad = {**self.rows[0], "class": "Player", "wire_member": "Diary", "catalog_row": len(self.rows)+2}
        self.rows.append(broad)
        receipt = self.receipt()
        self.assert_inventory(self.joined(receipt))
        receipt["decisions"][0]["catalog_row"] = broad["catalog_row"]
        with self.assertRaises(join_sources.SourceJoinError):
            self.joined(receipt)

    def test_YA1_source_choice_is_bound_to_owner_path_and_kind(self):
        span = self.layout["decoded_spans"][0]
        span["label"] = "YA1.Record[0].A.Data"
        self.layout["tail"].update(root_first=0, root_count=1, ya1_records=[{"name": "A", "kind": 2}])
        self.rows[0].update(selector=re.compile("^"+re.escape(span["label"])+"$"),
                            path_selector=re.compile("^/A$"), atlas_ya1_kind="2")
        receipt = self.receipt()
        with self.assertRaises(join_sources.SourceJoinError):
            self.joined(receipt)
        receipt["decisions"][0].update(ya1_path="/A", ya1_kind=2)
        self.assert_inventory(self.joined(receipt))
        receipt["decisions"][0]["ya1_kind"] = 6
        with self.assertRaises(join_sources.SourceJoinError):
            self.joined(receipt)

    def test_output_cannot_overwrite_or_alias_input(self):
        with self.assertRaises(join_sources.SourceJoinError):
            join_sources.safe_output(self.path, [self.path])

    def test_unsupported_archive_and_truncated_codec_are_rejected(self):
        raw = synthetic_save()
        with self.assertRaises(byte_layout.raw_reader.BadSAV):
            byte_layout.raw_reader.unpack_sav(raw[:25])
        walker = byte_layout.AtlasWalker(b"\xff\x7f")
        with self.assertRaises(byte_layout.raw_reader.BadSAV):
            walker.obj()

    def test_repeated_subspan_partition_and_exact_field_numbers(self):
        row = {"member_span": "2-3", "physical_width": "4", "wire_layout": "subspan repeats", "catalog_row": 2}
        self.assertEqual([(2, 4), (6, 8)], join_sources.offsets(row, 8))
        with self.assertRaises(join_sources.SourceJoinError):
            join_sources.offsets(row, 7)
        parts = list(join_sources.partition(8, [{"start": 2, "end": 4}, {"start": 6, "end": 8}], []))
        self.assertEqual([(0, 2, 0), (2, 4, 1), (4, 6, 0), (6, 8, 1)], [(a, b, len(c)) for a, b, c, _ in parts])


def synthetic_registry_save(actions=None, aliases=False, second_item=False, prefix_item=False,
                            shared_effect=False, repeated_effect=False, weapon=False, shared_spell=False,
                            item_class=None, item_code=0x0101, weapon_spell=True):
    word = lambda *values: struct.pack("<"+"I"*len(values), *values)
    token = lambda key: struct.pack("<12sIBHIHiII", bytes(range(12)), 9, 0, 7, 3, 2, 17, key, 0)
    definition = lambda name: struct.pack("<HHH", 0xffff, 1, len(name))+name.encode("ascii")
    effect = definition("Effect")+token(303)+struct.pack("<BBIB", 12, 0, 3, 0)
    children = word(2)+effect+struct.pack("<H", 6) if repeated_effect else word(1)+effect
    clazz = item_class or ("Weapon" if weapon else "Item")
    weapon = clazz == "Weapon"
    fields = struct.pack("<HHBBBHhB", item_code, 1, 7, 2, 3, 4, 5, 6)
    spell = definition("Spell")+struct.pack("<BBBHI", 13, 8, 0, 10, 505)
    suffix = bytes(47)+struct.pack("<H", 0) if weapon else (bytes(23) if clazz == "Armor" else
             (bytes(22) if clazz == "Shield" else b""))
    item = definition(clazz)+token(202)+children+fields
    if weapon:
        item += bytes(46)+b"\2"+spell if weapon_spell else suffix
    else:
        item += suffix
    refs = word(2)+item+struct.pack("<H", 4) if aliases else word(1)+item
    if second_item or shared_effect or shared_spell:
        children = word(1)+struct.pack("<H", 6) if shared_effect else word(0)
        other = struct.pack("<H", 0x8003)+token(404)+children+fields
        if weapon:
            other += bytes(46)+b"\2"+struct.pack("<H", 8 if shared_spell else 0)
        else:
            other += suffix
        refs = word(2)+item+other
    if prefix_item:
        other = definition(clazz)+token(404)+word(0)+fields+suffix
        refs = word(2)+other+struct.pack("<H", 0x8003)+item[len(definition(clazz)):]
    sack = definition("Sack")+token(101)+word(5)+refs+word(0, 5)
    body = word(0, 0)+b"\0"+word(*([0]*14))+word(0, 0)+b"\1"
    body += word(0, 0)+bytes(4)+word(1)+bytes(4374)+word(1)+sack
    body += word(0xBADFACE1, 0, 0, 0)+bytes(392)
    body += bytes(len(body) % 2)
    packets = word(len(body)//2)
    for at in range(0, len(body), 254):
        block = body[at:at+254]
        packets += bytes([len(block)//2])+block
    if actions is None:
        actions = {"Version": 1, "Ownership": [{"Kind": 1, "ID": 6, "Object": 2},
            {"Kind": 2, "ID": 10, "Object": 3}, {"Kind": 4, "ID": 9, "Object": 1}], "Bindings": []}
    encoded = actions.encode("utf-8") if isinstance(actions, str) else json.dumps(actions).encode("utf-8")
    pool = word(1, len(encoded))+encoded+bytes(((len(encoded)+11)&~3)-(len(encoded)+8))
    record = lambda name, data, size, kind: word(0, data, size, kind)+name.encode("ascii").ljust(16, b"\0")
    store = b"&YA1"+word(0, 1, 1, 2, 0)
    store += record("CurrentState", 1, 1, 1)+record("AgainromActions", 0, len(pool), 6)
    store += word(len(pool))+pool+word(*([0]*26))
    return b"Asg&"+word(16+len(packets), 0x0BAD0002, len(packets))+packets+bytes(256)+store


class RegistryBindingTests(unittest.TestCase):
    source_pin, knowledge_pin = "a"*40, "b"*40

    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.path = Path(self.temp.name)/"registry.sav"
        self.saved = synthetic_registry_save()
        self.path.write_bytes(self.saved)
        self.layout = byte_layout.atlas(self.path)
        self.assertEqual("parsed", self.layout["status"], self.layout.get("error"))
        self.rows = join_sources.catalog(Path(__file__).parent/"source_catalog.json")
        token = lambda key: {"Position": list(range(12)), "RuntimeID": 9, "T0C": 0, "T0E": 7,
            "T08": 3, "T18": 2, "T1C": 17, "Identity": key, "Reference": 0}
        owner = {"Kind": 3, "Entity": 0, "Object": 9, "Slot": 0, "SessionHandle": 0}
        objects = {"Version": 2, "NextID": 11, "Items": [{"ID": 6, "Origin": {"Kind": 2, "Parent": 0},
            "Retired": False, "InFlight": 0, "Value": {"Code": 0x0101, "Count": 1, "Kind": 7,
                "WeightPresent": True, "SourceEquipment": {"Class": 0},
                "Effects": [{"Kind": 12, "Mode": 0, "Operand": 3}]},
            "Token": token(202), "Coverage": {"Unknown": 0, "Unsupported": ""},
            "F45": 2, "F46": 3, "F47": 6, "F48": 4, "Effects": [10], "Spell": 0}],
            "Effects": [{"ID": 10, "Retired": False, "Token": token(303),
                "Value": {"Kind": 12, "Mode": 0, "Operand": 3}, "E0C": 0,
                "Coverage": {"Unknown": 0, "Unsupported": ""}, "ExternalReferences": 0}],
            "Spells": [], "Sacks": [{"ID": 9, "Retired": False, "Token": token(101)}],
            "Containers": [{"Owner": owner, "Present": True, "Items": [6]}], "ItemRoots": [], "BookRoots": []}
        self.writer = {"Revision": self.source_pin, "KnowledgePin": self.knowledge_pin,
            "Capture": {"WorldPresent": True, "Objects": objects, "ItemWeights": [],
                "Snapshot": {"SavedDocument": {"Objects": {"Version": 2}}}}}

    def inputs(self, writer=None, saved=None, expected_source=None, expected_knowledge=None):
        writer = self.writer if writer is None else writer
        raw = json.dumps(writer, separators=(",", ":")).encode("utf-8")
        captures = {"writer": copy.deepcopy(writer)}
        context = source_binding.binding_context(self.layout, captures,
            self.source_pin if expected_source is None else expected_source,
            self.knowledge_pin if expected_knowledge is None else expected_knowledge,
            {"writer": raw}, self.saved if saved is None else saved)
        return captures, context

    def prepared(self, binding="World.SavedObjects.Items.Code", captures=None, context=None, archive=None,
                 index=0, interval_slot=0):
        if captures is None:
            captures, context = self.inputs()
        spec = registry_binding.SPECS["bindings"][binding]
        span = next(s for s in self.layout["decoded_spans"] if re.fullmatch(
            "(?:"+"|".join(re.escape(clazz) for clazz in spec["classes"])+r")\[archive=\d+\]\."+re.escape(spec["field"]), s["label"])
            and (archive is None or f"[archive={archive}]" in s["label"]))
        clazz = span["label"].split("[")[0]
        row_index = [number for number in spec["catalog_rows"] if self.rows[number-2]["class"] == clazz][interval_slot]
        row = self.rows[row_index-2]
        start, end = spec["catalog_metadata"][str(row_index)]["intervals"][0]
        base = {"source_pin": self.source_pin, "knowledge_pin": self.knowledge_pin,
            "input_sha256": self.layout["file_sha256"], "layout_sha256": context["layout_sha256"],
            "capture_sha256": {"writer": join_sources.raw_reader.sha(context["capture_bytes"]["writer"])},
            "space": "decoded", "catalog_row": row_index, "selector": "^"+re.escape(span["label"])+"$",
            "owner_label": span["label"], "start": start, "end": end, "expected_leaf_count": 1,
            "catalog_guard": row["current_producer_guard"], "guard_result": True, "source_present": True,
            "guard": "current synthetic registry literal", "witness": "independent packed fixture"}
        decision, error = registry_binding.prepare(binding, index, span, captures, base, context)
        self.assertIsNone(error)
        return decision, row, span, captures, context

    def verify(self, prepared):
        decision, row, span, captures, context = prepared
        return source_binding.verify(decision, row, span, decision["start"], decision["end"], captures, context)

    def use_saved(self, saved):
        self.saved = saved
        self.path.write_bytes(saved)
        self.layout = byte_layout.atlas(self.path)
        self.assertEqual("parsed", self.layout["status"], self.layout.get("error"))

    def add_shared_parent(self, writer, spell=False):
        objects = writer["Capture"]["Objects"]
        other = copy.deepcopy(objects["Items"][0])
        other["ID"], other["Token"]["Identity"] = 7, 404
        if spell:
            other["Effects"], other["Value"]["Effects"] = [], []
        objects["Items"].append(other)
        objects["Containers"][0]["Items"] = [6, 7]

    def weapon_capture(self, shared=False):
        actions = {"Version": 1, "Ownership": [{"Kind": 1, "ID": 6, "Object": 2},
            {"Kind": 2, "ID": 10, "Object": 3}, {"Kind": 3, "ID": 8, "Object": 4},
            {"Kind": 4, "ID": 9, "Object": 1}], "Bindings": []}
        self.use_saved(synthetic_registry_save(actions, weapon=True, shared_spell=shared))
        writer = copy.deepcopy(self.writer)
        value = {"Present": True, "ID": 13, "Range": 8, "Defensive": 0, "ManaCost": 10}
        item = writer["Capture"]["Objects"]["Items"][0]
        item["Spell"], item["Value"]["SourceEquipment"] = 8, {"Class": 1, "Spell": value}
        writer["Capture"]["Objects"]["Spells"] = [{"ID": 8, "This": 505, "Retired": False,
            "Value": copy.deepcopy(value), "ExternalReferences": 0, "Coverage": {"Unknown": 0, "Unsupported": ""}}]
        writer["Capture"]["Spells"] = [{"ID": 13, "ManaCost": 999, "Defensive": True}]
        if shared:
            self.add_shared_parent(writer, spell=True)
        return writer

    def test_direct_fields_independent_current_mutation_raw_loss_masks_and_absence(self):
        fields = {
            "Items": {"F47": "F47", "F48": "F48", "Flags": "Token/T08", "Kind": "Value/Kind",
                "Material": "F46", "Position": "Token/Position", "Price": "Token/T1C",
                "PublicationMask": "Token/T18", "Shape": "F45", "Type": "Token/T0E"},
            "Effects": {"EffectID": "E0C", "EffectKind": "Value/Kind", "EffectMode": "Value/Mode",
                "Flags": "Token/T08", "Operand": "Value/Operand", "Position": "Token/Position",
                "Price": "Token/T1C", "PublicationMask": "Token/T18", "Type": "Token/T0E"}}
        self.assertEqual(set(fields["Items"]), set(registry_binding.DIRECT_FIELD_MASKS["Items"])-{"Count", "Code"})
        self.assertEqual(set(fields["Effects"]), set(registry_binding.DIRECT_FIELD_MASKS["Effects"]))
        original_saved = self.saved
        for collection, members in fields.items():
            for field, path in members.items():
                with self.subTest(collection=collection, field=field):
                    self.use_saved(original_saved)
                    binding = f"World.SavedObjects.{collection}.{field}"
                    prepared = self.prepared(binding)
                    self.assertTrue(self.verify(prepared)["verified"])
                    decision, row, span, captures, context = prepared
                    writer = copy.deepcopy(self.writer)
                    ptr = f"/Capture/Objects/{collection}/0/"+path
                    parent_path, name = ptr.rsplit("/", 1)
                    current_parent = registry_binding.at(writer, parent_path)
                    value = current_parent[name]
                    if isinstance(value, list):
                        value[0] ^= 1
                    else:
                        current_parent[name] ^= 1
                    if collection == "Effects" and path.startswith("Value/"):
                        writer["Capture"]["Objects"]["Items"][0]["Value"]["Effects"][0] = copy.deepcopy(
                            writer["Capture"]["Objects"]["Effects"][0]["Value"])
                    changed_captures, changed_context = self.inputs(writer=writer)
                    with self.assertRaises(source_binding.BindingError):
                        self.verify(self.prepared(binding, changed_captures, changed_context))
                    body, _ = byte_layout.raw_reader.unpack_sav(original_saved)
                    changed = bytearray(body)
                    changed[span["start"]+decision["start"]] ^= 1
                    self.use_saved(byte_layout.raw_reader.replace_body(original_saved, bytes(changed)))
                    with self.assertRaises(source_binding.BindingError):
                        self.verify(self.prepared(binding))
                    self.use_saved(original_saved)
                    writer = copy.deepcopy(self.writer)
                    writer["Capture"]["Objects"][collection][0]["Coverage"]["Unknown"] = (
                        registry_binding.DIRECT_FIELD_MASKS[collection][field] or 128)
                    changed_captures, changed_context = self.inputs(writer=writer)
                    direct, error = registry_binding.direct_literal(registry_binding.actual_context(changed_context),
                        writer["Capture"]["Objects"], 0, span, registry_binding.SPECS["bindings"][binding])
                    self.assertIsNone(direct)
                    writer = copy.deepcopy(self.writer)
                    registry_binding.at(writer, parent_path).pop(name)
                    changed_captures, changed_context = self.inputs(writer=writer)
                    direct, error = registry_binding.direct_literal(registry_binding.actual_context(changed_context),
                        writer["Capture"]["Objects"], 0, span, registry_binding.SPECS["bindings"][binding])
                    self.assertIsNone(direct)

    def test_direct_zero_position_parts_and_coherent_emitted_swap(self):
        writer = copy.deepcopy(self.writer)
        body, _ = byte_layout.raw_reader.unpack_sav(self.saved)
        changed = bytearray(body)
        parsed = byte_layout.raw_reader.Walker(body).parse()
        for record in parsed.order:
            if record["class"] in ("Item", "Effect"):
                offset = record["offsets"]["Position"]
                changed[offset:offset+12] = bytes(12)
        self.use_saved(byte_layout.raw_reader.replace_body(self.saved, bytes(changed)))
        for collection in ("Items", "Effects"):
            writer["Capture"]["Objects"][collection][0]["Token"]["Position"] = [0]*12
        captures, context = self.inputs(writer=writer)
        for collection in ("Items", "Effects"):
            for interval in range(5):
                self.assertTrue(self.verify(self.prepared(f"World.SavedObjects.{collection}.Position",
                    captures, context, interval_slot=interval))["verified"])
        self.use_saved(synthetic_registry_save({"Version": 1, "Ownership": [
            {"Kind": 1, "ID": 6, "Object": 4}, {"Kind": 2, "ID": 10, "Object": 2},
            {"Kind": 4, "ID": 9, "Object": 1}], "Bindings": []}, second_item=True))
        for collection in ("Items", "Effects"):
            for field in registry_binding.DIRECT_FIELD_MASKS[collection]:
                if field == "Code":
                    continue
                self.assertTrue(self.verify(self.prepared(f"World.SavedObjects.{collection}.{field}"))["verified"])

    def test_direct_effect_compatible_shared_and_repeated_edges(self):
        self.use_saved(synthetic_registry_save(shared_effect=True))
        writer = copy.deepcopy(self.writer)
        self.add_shared_parent(writer)
        captures, context = self.inputs(writer=writer)
        prepared = self.prepared("World.SavedObjects.Effects.Operand", captures, context)
        self.assertTrue(self.verify(prepared)["verified"])
        changed = copy.deepcopy(writer)
        changed["Capture"]["Objects"]["Items"][1]["Value"]["Effects"][0]["Operand"] = 77
        changed_captures, changed_context = self.inputs(writer=changed)
        decision, row, span, _, _ = prepared
        decision = {**decision, "capture_sha256": {"writer": join_sources.raw_reader.sha(changed_context["capture_bytes"]["writer"])}}
        self.assertFalse(self.verify((decision, row, span, changed_captures, changed_context))["verified"])
        changed = copy.deepcopy(self.writer)
        changed_captures, changed_context = self.inputs(writer=changed)
        decision = {**decision, "capture_sha256": {"writer": join_sources.raw_reader.sha(changed_context["capture_bytes"]["writer"])}}
        self.assertFalse(self.verify((decision, row, span, changed_captures, changed_context))["verified"])
        self.use_saved(synthetic_registry_save(repeated_effect=True))
        writer = copy.deepcopy(self.writer)
        writer["Capture"]["Objects"]["Items"][0]["Effects"] = [10, 10]
        writer["Capture"]["Objects"]["Items"][0]["Value"]["Effects"] *= 2
        captures, context = self.inputs(writer=writer)
        self.assertTrue(self.verify(self.prepared("World.SavedObjects.Effects.Operand", captures, context))["verified"])
        writer["Capture"]["Objects"]["Items"][0]["Effects"] = [10]
        writer["Capture"]["Objects"]["Items"][0]["Value"]["Effects"] = writer["Capture"]["Objects"]["Items"][0]["Value"]["Effects"][:1]
        captures, context = self.inputs(writer=writer)
        with self.assertRaises(source_binding.BindingError):
            self.prepared("World.SavedObjects.Effects.Operand", captures, context)

    def test_direct_child_pruning_and_unknown_key_remain_incomplete(self):
        prepared = self.prepared("World.SavedObjects.Effects.Operand")
        decision, row, span, _, _ = prepared
        for patch in ("pruned", "keyless", "external"):
            writer = copy.deepcopy(self.writer)
            objects = writer["Capture"]["Objects"]
            if patch == "pruned":
                objects["Items"][0]["Effects"] = [0, 10]
                objects["Items"][0]["Value"]["Effects"].insert(0, {"Kind": 0, "Mode": 0, "Operand": 0})
            elif patch == "keyless":
                objects["Effects"][0]["Token"]["Identity"] = 0
            else:
                objects["Effects"][0]["ExternalReferences"] = 1
            captures, context = self.inputs(writer=writer)
            direct, error = registry_binding.direct_literal(registry_binding.actual_context(context), objects,
                0, span, registry_binding.SPECS["bindings"][decision["binding_id"]])
            self.assertIsNone(direct)

    def test_protected_weapon_mana_priority_shared_values_and_late_code(self):
        writer = self.weapon_capture()
        captures, context = self.inputs(writer=writer)
        prepared = self.prepared("World.SavedObjects.Spells.ManaCost", captures, context)
        self.assertTrue(self.verify(prepared)["verified"])
        self.assertFalse(self.verify(prepared)["holder_topology_verified"])
        writer["Capture"]["Spells"][0]["ManaCost"] = 7777
        captures, context = self.inputs(writer=writer)
        self.assertTrue(self.verify(self.prepared("World.SavedObjects.Spells.ManaCost", captures, context))["verified"])
        for field in ("ID", "Defensive"):
            self.assertTrue(self.verify(self.prepared("World.SavedObjects.Spells."+field, captures, context))["verified"])
        with mock.patch.dict(registry_binding.BOOK_PROJECTION_PINS, {"pkg/game/savbookcurrent.go": "f"*64}):
            self.assertFalse(self.verify(self.prepared("World.SavedObjects.Spells.ManaCost", captures, context))["verified"])
        writer["Capture"]["Objects"]["Spells"][0]["Value"]["ManaCost"] = 11
        writer["Capture"]["Objects"]["Items"][0]["Value"]["SourceEquipment"]["Spell"]["ManaCost"] = 11
        captures, context = self.inputs(writer=writer)
        with self.assertRaises(source_binding.BindingError):
            self.verify(self.prepared("World.SavedObjects.Spells.ManaCost", captures, context))
        writer = self.weapon_capture(shared=True)
        captures, context = self.inputs(writer=writer)
        prepared = self.prepared("World.SavedObjects.Spells.ManaCost", captures, context)
        self.assertTrue(self.verify(prepared)["verified"])
        writer["Capture"]["Objects"]["Items"][1]["Value"]["SourceEquipment"]["Spell"]["Range"] = 9
        captures, context = self.inputs(writer=writer)
        decision, row, span, _, _ = prepared
        decision = {**decision, "capture_sha256": {"writer": join_sources.raw_reader.sha(context["capture_bytes"]["writer"])}}
        self.assertFalse(self.verify((decision, row, span, captures, context))["verified"])

    def test_protected_mana_raw_loss_unknown_absent_key_and_book_only(self):
        writer = self.weapon_capture()
        captures, context = self.inputs(writer=writer)
        prepared = self.prepared("World.SavedObjects.Spells.ManaCost", captures, context)
        decision, row, span, _, _ = prepared
        original_saved = self.saved
        body, _ = byte_layout.raw_reader.unpack_sav(original_saved)
        changed = bytearray(body)
        changed[span["start"]] ^= 1
        self.use_saved(byte_layout.raw_reader.replace_body(original_saved, bytes(changed)))
        captures, context = self.inputs(writer=writer)
        with self.assertRaises(source_binding.BindingError):
            self.verify(self.prepared("World.SavedObjects.Spells.ManaCost", captures, context))
        self.use_saved(original_saved)
        for patch in ("unknown_cost", "unknown_key", "keyless", "absent", "book_only", "external"):
            with self.subTest(patch=patch):
                changed = copy.deepcopy(writer)
                objects = changed["Capture"]["Objects"]
                child = objects["Spells"][0]
                if patch.startswith("unknown"):
                    child["Coverage"]["Unknown"] = 8192 if patch == "unknown_cost" else 128
                elif patch == "keyless":
                    child["This"] = 0
                elif patch == "absent":
                    child["Value"].pop("ManaCost")
                elif patch == "book_only":
                    objects["Items"][0]["Spell"] = 0
                else:
                    child["ExternalReferences"] = 1
                captures, context = self.inputs(writer=changed)
                direct, error = registry_binding.direct_literal(registry_binding.actual_context(context), objects,
                    0, span, registry_binding.SPECS["bindings"][decision["binding_id"]])
                self.assertIsNone(direct)
        writer = self.weapon_capture(shared=True)
        objects = writer["Capture"]["Objects"]
        objects["Items"].pop()
        objects["Containers"][0]["Items"] = [6]
        captures, context = self.inputs(writer=writer)
        span = next(s for s in self.layout["decoded_spans"] if s["label"] == "Spell[archive=8].ManaCost")
        direct, error = registry_binding.direct_literal(registry_binding.actual_context(context), objects,
            0, span, registry_binding.SPECS["bindings"][decision["binding_id"]])
        self.assertIsNone(direct)
        self.assertIn("unexplained incoming", error)

    def test_protected_spell_scalars_bounds_loss_unknown_and_absence(self):
        for field, legal, invalid in (("ID", (1, 28), (0, 29, True, 256)),
                                      ("Defensive", (0, 255), (-1, True, 256))):
            binding = "World.SavedObjects.Spells."+field
            writer = self.weapon_capture()
            original_saved = self.saved
            captures, context = self.inputs(writer=writer)
            _, _, span, _, _ = self.prepared(binding, captures, context)
            for value in legal:
                with self.subTest(field=field, legal=value):
                    body, _ = byte_layout.raw_reader.unpack_sav(original_saved)
                    changed = bytearray(body)
                    changed[span["start"]] = value
                    self.use_saved(byte_layout.raw_reader.replace_body(original_saved, bytes(changed)))
                    observed = copy.deepcopy(writer)
                    objects = observed["Capture"]["Objects"]
                    objects["Spells"][0]["Value"][field] = value
                    objects["Items"][0]["Value"]["SourceEquipment"]["Spell"][field] = value
                    captures, context = self.inputs(writer=observed)
                    result = self.verify(self.prepared(binding, captures, context))
                    self.assertTrue(result["verified"])
                    self.assertFalse(result["complete"])
            self.use_saved(original_saved)
            for value in invalid:
                with self.subTest(field=field, invalid=value):
                    observed = copy.deepcopy(writer)
                    objects = observed["Capture"]["Objects"]
                    objects["Spells"][0]["Value"][field] = value
                    objects["Items"][0]["Value"]["SourceEquipment"]["Spell"][field] = value
                    captures, context = self.inputs(writer=observed)
                    with self.assertRaises(source_binding.BindingError):
                        self.prepared(binding, captures, context)
            observed = copy.deepcopy(writer)
            objects = observed["Capture"]["Objects"]
            objects["Spells"][0]["Value"][field] ^= 1
            objects["Items"][0]["Value"]["SourceEquipment"]["Spell"][field] = objects["Spells"][0]["Value"][field]
            captures, context = self.inputs(writer=observed)
            with self.assertRaises(source_binding.BindingError):
                self.verify(self.prepared(binding, captures, context))
            body, _ = byte_layout.raw_reader.unpack_sav(original_saved)
            changed = bytearray(body)
            changed[span["start"]] ^= 1
            self.use_saved(byte_layout.raw_reader.replace_body(original_saved, bytes(changed)))
            captures, context = self.inputs(writer=writer)
            with self.assertRaises(source_binding.BindingError):
                self.verify(self.prepared(binding, captures, context))
            self.use_saved(original_saved)
            for patch in (128, 8192, "absent", "keyless", "book_only"):
                with self.subTest(field=field, unavailable=patch):
                    observed = copy.deepcopy(writer)
                    objects = observed["Capture"]["Objects"]
                    if type(patch) is int:
                        objects["Spells"][0]["Coverage"]["Unknown"] = patch
                    elif patch == "absent":
                        objects["Spells"][0]["Value"].pop(field)
                    elif patch == "keyless":
                        objects["Spells"][0]["This"] = 0
                    else:
                        objects["Items"][0]["Spell"] = 0
                    captures, context = self.inputs(writer=observed)
                    direct, error = registry_binding.direct_literal(registry_binding.actual_context(context), objects,
                        0, span, registry_binding.SPECS["bindings"][binding])
                    self.assertIsNone(direct)

    def test_protected_spell_scalars_full_shared_and_book_input_values(self):
        for field in ("ID", "Defensive"):
            binding = "World.SavedObjects.Spells."+field
            writer = self.weapon_capture(shared=True)
            captures, context = self.inputs(writer=writer)
            prepared = self.prepared(binding, captures, context)
            self.assertTrue(self.verify(prepared)["verified"])
            for other in ("ID", "Range", "Defensive", "ManaCost"):
                if other == field:
                    continue
                observed = copy.deepcopy(writer)
                observed["Capture"]["Objects"]["Items"][1]["Value"]["SourceEquipment"]["Spell"][other] ^= 1
                captures, context = self.inputs(writer=observed)
                decision, row, span, _, _ = prepared
                decision = {**decision, "capture_sha256": {"writer": join_sources.raw_reader.sha(context["capture_bytes"]["writer"])}}
                self.assertFalse(self.verify((decision, row, span, captures, context))["verified"])
            observed = copy.deepcopy(writer)
            observed["Capture"]["Objects"]["Items"].pop()
            observed["Capture"]["Objects"]["Containers"][0]["Items"] = [6]
            captures, context = self.inputs(writer=observed)
            direct, error = registry_binding.direct_literal(registry_binding.actual_context(context),
                observed["Capture"]["Objects"], 0, span, registry_binding.SPECS["bindings"][binding])
            self.assertIsNone(direct)
            self.assertIn("unexplained incoming", error)
            for other in ("ID", "Range", "Defensive", "ManaCost"):
                observed = copy.deepcopy(writer)
                request = copy.deepcopy(observed["Capture"]["Objects"]["Spells"][0]["Value"])
                request[other] ^= 1
                slots = [{"Range": 0, "Defensive": 0, "ManaCost": 0} for _ in range(28)]
                slots[request["ID"]-1] = {name: request[name] for name in ("Range", "Defensive", "ManaCost")}
                observed["Capture"]["Entities"] = [{"ID": 77, "KnownSpells": 1 << request["ID"],
                    "Book": {"State": 2, "Slots": slots}}]
                observed["Capture"]["Spells"] = [{"ID": request["ID"], "ManaCost": request["ManaCost"],
                    "Defensive": bool(request["Defensive"])}]
                captures, context = self.inputs(writer=observed)
                result = self.verify(self.prepared(binding, captures, context))
                self.assertTrue(result["verified"])
                self.assertFalse(result["holder_topology_verified"])
                self.assertFalse(result["complete"])
            captures, context = self.inputs(writer=writer)
            with mock.patch.dict(registry_binding.BOOK_PROJECTION_PINS, {"pkg/game/savbookcurrent.go": "f"*64}):
                with self.assertRaises(source_binding.BindingError):
                    self.prepared(binding, captures, context)

    def test_direct_widths_keys_aliases_and_child_raw_edge_loss(self):
        for binding, path, value in (
            ("World.SavedObjects.Items.Price", "/Capture/Objects/Items/0/Token/T1C", True),
            ("World.SavedObjects.Items.Position", "/Capture/Objects/Items/0/Token/Position", [0]*11),
            ("World.SavedObjects.Effects.Operand", "/Capture/Objects/Effects/0/Value/Operand", 1 << 32),
            ("World.SavedObjects.Effects.Type", "/Capture/Objects/Effects/0/Token/T0E", -1),
            ("World.SavedObjects.Effects.Type", "/Capture/Objects/Effects/0/Token/Identity", 101),
            ("World.SavedObjects.Effects.Type", "/Capture/Objects/Effects/0/Token/Identity", True)):
            with self.subTest(binding=binding, path=path, value=value):
                prepared = self.prepared(binding)
                writer = copy.deepcopy(self.writer)
                parent_path, name = path.rsplit("/", 1)
                registry_binding.at(writer, parent_path)[name] = value
                if path.endswith("Value/Operand"):
                    writer["Capture"]["Objects"]["Items"][0]["Value"]["Effects"][0]["Operand"] = value
                captures, context = self.inputs(writer=writer)
                with self.assertRaises(source_binding.BindingError):
                    self.prepared(binding, captures, context)
        decision, row, span, captures, context = self.prepared("World.SavedObjects.Effects.Operand")
        with self.assertRaises(source_binding.BindingError):
            registry_binding.direct_literal(registry_binding.actual_context(context),
                self.writer["Capture"]["Objects"], 0, {**span, "aliases": ["unexplained"]},
                registry_binding.SPECS["bindings"][decision["binding_id"]])
        self.use_saved(synthetic_registry_save(repeated_effect=True))
        writer = copy.deepcopy(self.writer)
        writer["Capture"]["Objects"]["Items"][0]["Effects"] = [10, 10]
        writer["Capture"]["Objects"]["Items"][0]["Value"]["Effects"] *= 2
        body, _ = byte_layout.raw_reader.unpack_sav(self.saved)
        parsed = byte_layout.raw_reader.Walker(body).parse()
        parent = next(record for record in parsed.order if record["class"] == "Item")
        changed = bytearray(body)
        offset = parent["ref_offsets"]["Effects"][1]
        changed[offset:offset+2] = struct.pack("<H", 0)
        self.use_saved(byte_layout.raw_reader.replace_body(self.saved, bytes(changed)))
        captures, context = self.inputs(writer=writer)
        with self.assertRaises(source_binding.BindingError):
            self.prepared("World.SavedObjects.Effects.Operand", captures, context)

    def test_main_join_calls_literal_adapter_without_promoting_typed_owners(self):
        decision, row, span, captures, context = self.prepared()
        result = self.verify((decision, row, span, captures, context))
        self.assertTrue(result["value_verified"])
        self.assertFalse(result["verified"])
        self.assertFalse(result["owner_relation_verified"])
        receipt = {"source_pin": self.source_pin, "knowledge_pin": self.knowledge_pin,
            "input_sha256": self.layout["file_sha256"], "decisions": [decision],
            "capture_pin_paths": {"writer": {"source": "/Revision", "knowledge": "/KnowledgePin"}}}
        joined = join_sources.join(self.layout, self.rows, receipt, self.source_pin, self.knowledge_pin, captures, context)
        leaf = next(s for s in joined["joined_spans"] if s["label"] == span["label"])
        self.assertTrue(leaf["source_bindings"][0]["value_verified"])
        self.assertFalse(joined["source_classification_joined"])
        self.assertEqual("incomplete", joined["status"])
        self.assertGreater(joined["unverified_source_binding_bytes"], 0)

    def test_count_leaf_binds_current_key_without_holder_or_document_maps(self):
        writer = copy.deepcopy(self.writer)
        writer["Capture"].pop("Snapshot")
        self.saved = synthetic_registry_save({"Version": 1, "Ownership": [], "Bindings": []})
        self.path.write_bytes(self.saved)
        self.layout = byte_layout.atlas(self.path)
        captures, context = self.inputs(writer=writer)
        prepared = self.prepared("World.SavedObjects.Items.Count", captures, context)
        result = self.verify(prepared)
        self.assertTrue(result["verified"])
        self.assertFalse(result["owner_relation_verified"])
        self.assertFalse(result["holder_topology_verified"])
        self.assertFalse(result["complete"])
        self.assertEqual("/Capture/Objects/Items/0/Value/Count", result["carrier_path"])
        decision, row, span, captures, context = prepared
        receipt = {"source_pin": self.source_pin, "knowledge_pin": self.knowledge_pin,
            "input_sha256": self.layout["file_sha256"], "decisions": [decision],
            "capture_pin_paths": {"writer": {"source": "/Revision", "knowledge": "/KnowledgePin"}}}
        joined = join_sources.join(self.layout, self.rows, receipt, self.source_pin, self.knowledge_pin, captures, context)
        leaf = next(s for s in joined["joined_spans"] if s["label"] == span["label"])
        self.assertTrue(leaf["source_bindings"][0]["verified"])
        self.assertFalse(joined["source_classification_joined"])
        self.assertEqual("incomplete", joined["status"])

    def test_count_ignores_coherently_swapped_emitted_same_class_map(self):
        self.saved = synthetic_registry_save({"Version": 1, "Ownership": [
            {"Kind": 1, "ID": 6, "Object": 4}, {"Kind": 2, "ID": 10, "Object": 3},
            {"Kind": 4, "ID": 9, "Object": 1}], "Bindings": []}, second_item=True)
        self.path.write_bytes(self.saved)
        self.layout = byte_layout.atlas(self.path)
        self.assertTrue(self.verify(self.prepared("World.SavedObjects.Items.Count", archive=4))["verified"])
        with self.assertRaises(source_binding.BindingError):
            self.prepared("World.SavedObjects.Items.Count", archive=7)

    def test_count_survives_reindex_and_native_id_namespace_change(self):
        self.saved = synthetic_registry_save(prefix_item=True)
        self.path.write_bytes(self.saved)
        self.layout = byte_layout.atlas(self.path)
        self.assertTrue(self.verify(self.prepared("World.SavedObjects.Items.Count", archive=5))["verified"])
        writer = copy.deepcopy(self.writer)
        writer["Capture"]["Objects"]["Items"][0]["ID"] = 77
        writer["Capture"]["Objects"]["NextID"] = 100
        writer["Capture"]["Objects"]["Containers"][0]["Items"] = [77]
        captures, context = self.inputs(writer=writer)
        self.assertTrue(self.verify(self.prepared("World.SavedObjects.Items.Count", captures, context, archive=5))["verified"])

    def test_count_current_alias_roots_select_one_scalar(self):
        self.saved = synthetic_registry_save(aliases=True)
        self.path.write_bytes(self.saved)
        self.layout = byte_layout.atlas(self.path)
        writer = copy.deepcopy(self.writer)
        writer["Capture"]["Objects"]["Containers"][0]["Items"] = [6, 6]
        captures, context = self.inputs(writer=writer)
        result = self.verify(self.prepared("World.SavedObjects.Items.Count", captures, context))
        self.assertTrue(result["verified"])
        self.assertEqual(2, len(result["directed_relation"]["current_locations"]))
        self.assertFalse(result["holder_topology_verified"])

    def test_count_missing_key_and_root_facts_remain_unverified(self):
        mutations = [lambda o: o["Items"][0]["Token"].__setitem__("Identity", 0),
            lambda o: o["Items"][0]["Coverage"].__setitem__("Unknown", 128),
            lambda o: o["Items"][0]["Coverage"].__setitem__("Unknown", 16384),
            lambda o: o.pop("NextID"), lambda o: o.__setitem__("Containers", [])]
        for change in mutations:
            writer = copy.deepcopy(self.writer)
            change(writer["Capture"]["Objects"])
            captures, context = self.inputs(writer=writer)
            prepared = self.prepared("World.SavedObjects.Items.Count", captures, context) if writer["Capture"]["Objects"].get("Containers") else None
            if prepared is None:
                span = next(s for s in self.layout["decoded_spans"] if s["label"] == "Item[archive=4].Count")
                result, error = registry_binding.direct_item_count(registry_binding.actual_context(context),
                    writer["Capture"]["Objects"], 0, span, registry_binding.SPECS["bindings"]["World.SavedObjects.Items.Count"])
                self.assertIsNone(result)
            else:
                self.assertFalse(self.verify(prepared)["verified"])
        writer = copy.deepcopy(self.writer)
        writer["Capture"]["Objects"]["Items"][0]["Value"]["WeightPresent"] = False
        writer["Capture"].pop("ItemWeights")
        captures, context = self.inputs(writer=writer)
        span = next(s for s in self.layout["decoded_spans"] if s["label"] == "Item[archive=4].Count")
        result, error = registry_binding.direct_item_count(registry_binding.actual_context(context),
            writer["Capture"]["Objects"], 0, span, registry_binding.SPECS["bindings"]["World.SavedObjects.Items.Count"])
        self.assertIsNone(result)
        self.assertIn("fallback", error)

    @staticmethod
    def weight_row(code=0x0101, clazz=0):
        return {"Code": code, "Weight": 20, "Constructor": {"Class": clazz, "DefinitionRow": 0,
            "OwnKind": 0, "Attack": [0]*24, "Defence": [0]*22, "EffectsUnsupported": False,
            "Definition": {"Present": False, "AttackType": 0, "Hands": 0, "Charge": 0, "Relax": 0, "Suitable": 0},
            "Spell": {"Present": False, "ID": 0, "Range": 0, "Defensive": 0, "ManaCost": 0}}}

    def fallback_capture(self, clazz="Weapon", code=0x0101, **saved_options):
        self.use_saved(synthetic_registry_save(item_class=clazz, item_code=code, weapon_spell=False, **saved_options))
        writer = copy.deepcopy(self.writer)
        writer["Capture"]["Objects"]["Items"][0]["Value"].update(Code=code, WeightPresent=False)
        return writer

    def test_class_fallback_code_arms_and_table_observations_are_class_only(self):
        for b in range(16):
            clazz = "Weapon" if b == 1 else ("Shield" if b == 2 else ("Armor" if 3 <= b <= 12 else "Item"))
            with self.subTest(b=b):
                writer = self.fallback_capture(clazz, (b << 8)+1)
                writer["Capture"]["ItemWeights"] = None
                for table in (False, True):
                    writer["Capture"]["ConstructionTablePresent"] = table
                    writer["Capture"]["ConstructionInputs"] = {"Weapons": {"Present": table, "Rows": []}}
                    captures, context = self.inputs(writer=writer)
                    for binding in ("World.SavedObjects.Items.Count", "World.SavedObjects.Items.Kind",
                                    "World.SavedObjects.Effects.Operand"):
                        result = self.verify(self.prepared(binding, captures, context))
                        self.assertTrue(result["verified"])
                        self.assertFalse(result["holder_topology_verified"])
                        self.assertFalse(result["complete"])

    def test_class_fallback_current_and_weight_constructor_precedence(self):
        writer = self.fallback_capture("Shield")
        writer["Capture"]["ItemWeights"] = [self.weight_row(clazz=3)]
        captures, context = self.inputs(writer=writer)
        self.assertTrue(self.verify(self.prepared("World.SavedObjects.Items.Kind", captures, context))["verified"])
        writer = self.fallback_capture("Armor")
        writer["Capture"]["Objects"]["Items"][0]["Value"]["SourceEquipment"]["Class"] = 2
        writer["Capture"]["ItemWeights"] = [self.weight_row(clazz=3)]
        captures, context = self.inputs(writer=writer)
        self.assertTrue(self.verify(self.prepared("World.SavedObjects.Items.Count", captures, context))["verified"])
        writer["Capture"].pop("ItemWeights")
        captures, context = self.inputs(writer=writer)
        self.assertTrue(self.verify(self.prepared("World.SavedObjects.Items.Count", captures, context))["verified"])
        self.use_saved(synthetic_registry_save())
        writer = copy.deepcopy(self.writer)
        writer["Capture"]["ItemWeights"] = [self.weight_row(clazz=1)]
        captures, context = self.inputs(writer=writer)
        self.assertTrue(self.verify(self.prepared("World.SavedObjects.Items.Count", captures, context))["verified"])

    def test_class_fallback_held_predicate_and_current_class_mutation(self):
        writer = self.fallback_capture()
        objects = writer["Capture"]["Objects"]
        objects["Items"][0]["Value"]["WeightPresent"] = True
        objects["ItemRoots"] = [{"ID": 6, "Owner": {"Kind": 2, "Entity": 1, "Object": 0, "Slot": 1, "SessionHandle": 0}}]
        captures, context = self.inputs(writer=writer)
        self.assertTrue(self.verify(self.prepared("World.SavedObjects.Items.Count", captures, context))["verified"])
        for change in (lambda o: o.__setitem__("ItemRoots", []),
                       lambda o: o["ItemRoots"][0]["Owner"].__setitem__("Kind", 4),
                       lambda o: o["Items"][0]["Value"]["SourceEquipment"].__setitem__("Class", 3),
                       lambda o: o["Items"][0]["Value"].__setitem__("Code", 0x0201)):
            observed = copy.deepcopy(writer)
            change(observed["Capture"]["Objects"])
            captures, context = self.inputs(writer=observed)
            with self.assertRaises(source_binding.BindingError):
                self.prepared("World.SavedObjects.Items.Count", captures, context)

    def test_class_fallback_nil_absent_and_partial_constructor_getters(self):
        writer = self.fallback_capture()
        for weights in (None, [], [self.weight_row()], [self.weight_row(code=0x0201)]):
            writer["Capture"]["ItemWeights"] = weights
            captures, context = self.inputs(writer=writer)
            self.assertTrue(self.verify(self.prepared("World.SavedObjects.Items.Count", captures, context))["verified"])
        for change in (lambda c: c.pop("ItemWeights"),
                       lambda c: c["ItemWeights"][0].pop("Constructor"),
                       lambda c: c["ItemWeights"][0]["Constructor"].pop("Class"),
                       lambda c: c["ItemWeights"][0]["Constructor"]["Definition"].pop("Present")):
            writer["Capture"]["ItemWeights"] = [self.weight_row()]
            observed = copy.deepcopy(writer)
            change(observed["Capture"])
            captures, context = self.inputs(writer=observed)
            span = next(s for s in self.layout["decoded_spans"] if s["label"].endswith("].Count"))
            direct, error = registry_binding.direct_item_count(registry_binding.actual_context(context),
                observed["Capture"]["Objects"], 0, span, registry_binding.SPECS["bindings"]["World.SavedObjects.Items.Count"])
            self.assertIsNone(direct)
            self.assertIsNotNone(error)

    def test_class_fallback_corrupt_weights_and_constructor_state_reject(self):
        mutations = (lambda w: w.append(copy.deepcopy(w[0])),
            lambda w: w.insert(0, self.weight_row(code=0x0201)),
            lambda w: w[0].__setitem__("Code", 65536), lambda w: w[0].__setitem__("Code", True),
            lambda w: w[0].__setitem__("Weight", 1 << 31),
            lambda w: w[0]["Constructor"].__setitem__("Class", 4),
            lambda w: w[0]["Constructor"].__setitem__("Class", True),
            lambda w: w[0]["Constructor"].__setitem__("OwnKind", 1),
            lambda w: w[0]["Constructor"].__setitem__("Attack", [0]*23),
            lambda w: w[0]["Constructor"].__setitem__("Defence", [False]*22),
            lambda w: w[0]["Constructor"].__setitem__("EffectsUnsupported", True),
            lambda w: w[0]["Constructor"]["Spell"].__setitem__("Present", True),
            lambda w: w[0]["Constructor"]["Definition"].__setitem__("Charge", 1))
        writer = self.fallback_capture()
        writer["Capture"]["ItemWeights"] = [self.weight_row()]
        for change in mutations:
            with self.subTest(change=change):
                observed = copy.deepcopy(writer)
                change(observed["Capture"]["ItemWeights"])
                captures, context = self.inputs(writer=observed)
                with self.assertRaises(source_binding.BindingError):
                    self.prepared("World.SavedObjects.Items.Count", captures, context)

    def test_class_fallback_weights_scalar_values_never_replace_literal_fields(self):
        writer = self.fallback_capture()
        weight = self.weight_row(clazz=1)
        weight["Weight"] = -123
        weight["Constructor"]["Attack"][14] = 99
        weight["Constructor"]["Definition"] = {"Present": True, "AttackType": 8, "Hands": 2, "Charge": 4, "Relax": 7, "Suitable": 1}
        writer["Capture"]["ItemWeights"] = [weight]
        captures, context = self.inputs(writer=writer)
        for field in registry_binding.DIRECT_FIELD_MASKS["Items"]:
            if field == "Code":
                continue
            self.assertTrue(self.verify(self.prepared("World.SavedObjects.Items."+field, captures, context))["verified"])
        observed = copy.deepcopy(writer)
        observed["Capture"]["ItemWeights"][0]["Constructor"]["Class"] = 2
        observed["Capture"]["ItemWeights"][0]["Constructor"]["Attack"] = [0]*24
        observed["Capture"]["ItemWeights"][0]["Constructor"]["Definition"] = self.weight_row()["Constructor"]["Definition"]
        captures, context = self.inputs(writer=observed)
        with self.assertRaises(source_binding.BindingError):
            self.prepared("World.SavedObjects.Items.Count", captures, context)

    def test_class_fallback_coherent_emitted_swap_reindex_and_alias_roots(self):
        actions = {"Version": 1, "Ownership": [{"Kind": 1, "ID": 6, "Object": 4},
            {"Kind": 2, "ID": 10, "Object": 2}, {"Kind": 4, "ID": 9, "Object": 1}], "Bindings": []}
        writer = self.fallback_capture("Armor", 0x0301, actions=actions, second_item=True)
        captures, context = self.inputs(writer=writer)
        self.assertTrue(self.verify(self.prepared("World.SavedObjects.Items.Count", captures, context, archive=4))["verified"])
        with self.assertRaises(source_binding.BindingError):
            self.prepared("World.SavedObjects.Items.Count", captures, context, archive=7)
        writer = self.fallback_capture("Armor", 0x0301, prefix_item=True)
        captures, context = self.inputs(writer=writer)
        self.assertTrue(self.verify(self.prepared("World.SavedObjects.Items.Count", captures, context, archive=5))["verified"])
        writer = self.fallback_capture("Shield", 0x0201, aliases=True)
        writer["Capture"]["Objects"]["Containers"][0]["Items"] = [6, 6]
        captures, context = self.inputs(writer=writer)
        result = self.verify(self.prepared("World.SavedObjects.Items.Count", captures, context))
        self.assertTrue(result["verified"])
        self.assertFalse(result["holder_topology_verified"])

    def test_class_fallback_shared_effect_full_values_and_pruning(self):
        writer = self.fallback_capture("Armor", 0x0301, shared_effect=True)
        self.add_shared_parent(writer)
        captures, context = self.inputs(writer=writer)
        binding = "World.SavedObjects.Effects.Operand"
        self.assertTrue(self.verify(self.prepared(binding, captures, context))["verified"])
        for change in (lambda o: o["Items"][1]["Value"]["Effects"][0].__setitem__("Mode", 1),
                       lambda o: o["Items"].pop(), lambda o: o["Items"][0].__setitem__("Effects", [0, 10])):
            observed = copy.deepcopy(writer)
            change(observed["Capture"]["Objects"])
            captures, context = self.inputs(writer=observed)
            span = next(s for s in self.layout["decoded_spans"] if s["label"].endswith("].Operand"))
            direct, error = registry_binding.direct_literal(registry_binding.actual_context(context),
                observed["Capture"]["Objects"], 0, span, registry_binding.SPECS["bindings"][binding])
            self.assertIsNone(direct)
            self.assertIsNotNone(error)

    def test_class_fallback_current_raw_masks_keys_and_code_pins(self):
        writer = self.fallback_capture("Armor", 0x0301)
        captures, context = self.inputs(writer=writer)
        for binding, path, name in (("World.SavedObjects.Items.Count", "Value", "Count"),
                                    ("World.SavedObjects.Items.Kind", "Value", "Kind"),
                                    ("World.SavedObjects.Effects.Operand", "Value", "Operand")):
            prepared = self.prepared(binding, captures, context)
            decision, row, span, _, _ = prepared
            collection = registry_binding.SPECS["bindings"][binding]["collection"]
            observed = copy.deepcopy(writer)
            observed["Capture"]["Objects"][collection][0][path][name] ^= 1
            if collection == "Effects":
                observed["Capture"]["Objects"]["Items"][0]["Value"]["Effects"][0] = copy.deepcopy(
                    observed["Capture"]["Objects"]["Effects"][0]["Value"])
            changed_captures, changed_context = self.inputs(writer=observed)
            with self.assertRaises(source_binding.BindingError):
                self.verify(self.prepared(binding, changed_captures, changed_context))
            body, _ = byte_layout.raw_reader.unpack_sav(self.saved)
            changed = bytearray(body)
            changed[span["start"]] ^= 1
            altered = byte_layout.raw_reader.replace_body(self.saved, bytes(changed))
            original = self.saved
            self.use_saved(altered)
            changed_captures, changed_context = self.inputs(writer=writer)
            with self.assertRaises(source_binding.BindingError):
                self.verify(self.prepared(binding, changed_captures, changed_context))
            self.use_saved(original)
        for mask in (128, 16384, 16):
            observed = copy.deepcopy(writer)
            observed["Capture"]["Objects"]["Items"][0]["Coverage"]["Unknown"] = mask
            changed_captures, changed_context = self.inputs(writer=observed)
            self.assertFalse(self.verify(self.prepared("World.SavedObjects.Items.Flags", changed_captures, changed_context))["verified"])
        observed = copy.deepcopy(writer)
        observed["Capture"]["Objects"]["Effects"][0]["Token"]["Identity"] = 202
        changed_captures, changed_context = self.inputs(writer=observed)
        with self.assertRaises(source_binding.BindingError):
            self.prepared("World.SavedObjects.Items.Count", changed_captures, changed_context)
        with mock.patch.dict(registry_binding.CLASS_PROJECTION_PINS, {"pkg/mapload/sourceconstructor.go": "f"*64}):
            self.assertFalse(self.verify(self.prepared("World.SavedObjects.Items.Count", captures, context))["verified"])

    def test_class_fallback_never_promotes_original_spell_before_constructor_copy(self):
        writer = self.weapon_capture()
        writer["Capture"]["Objects"]["Items"][0]["Value"]["SourceEquipment"]["Class"] = 0
        writer["Capture"]["Objects"]["Items"][0]["Value"]["WeightPresent"] = False
        writer["Capture"]["ItemWeights"] = [self.weight_row(clazz=1)]
        captures, context = self.inputs(writer=writer)
        for field in registry_binding.DIRECT_FIELD_MASKS["Spells"]:
            self.assertFalse(self.verify(self.prepared("World.SavedObjects.Spells."+field, captures, context))["verified"])

    def code_capture(self, code=0x0101, written=None, empty=True, stand_ins=None, clazz="Item", **saved_options):
        self.use_saved(synthetic_registry_save(item_class=clazz, item_code=code if written is None else written,
                                               weapon_spell=False, **saved_options))
        writer = copy.deepcopy(self.writer)
        writer["Capture"]["Objects"]["Items"][0]["Value"]["Code"] = code
        writer["Capture"]["ModSetEmpty"] = empty
        writer["Capture"]["ModItemStandIns"] = stand_ins
        return writer

    def test_code_actual_empty_no_match_and_captured_nil_inputs(self):
        for empty, stand_ins in ((True, None), (True, [{"Code": 0x0101, "StandInCode": 0, "StandInRow": 0}]),
                                  (False, None), (False, []),
                                  (False, [{"Code": 0x0102, "StandInCode": 123, "StandInRow": 7}])):
            with self.subTest(empty=empty, stand_ins=stand_ins):
                writer = self.code_capture(empty=empty, stand_ins=stand_ins)
                captures, context = self.inputs(writer=writer)
                prepared = self.prepared("World.SavedObjects.Items.Code", captures, context)
                result = self.verify(prepared)
                self.assertTrue(result["verified"])
                self.assertEqual("/Capture/Objects/Items/0/Value/Code", result["carrier_path"])
                self.assertEqual(registry_binding.SPECS["bindings"][prepared[0]["binding_id"]]["producer_callsite"],
                                 result["producer_callsite"])
                self.assertFalse(result["complete"])
                self.assertFalse(result["holder_topology_verified"])

    def test_code_late_last_matching_row_selects_real_source_and_callsite(self):
        rows = [{"Code": 0x0101, "StandInCode": 0x0105, "StandInRow": 5},
                {"Code": 0x0102, "StandInCode": 0x0109, "StandInRow": 9},
                {"Code": 0x0101, "StandInCode": 0x0114, "StandInRow": 20}]
        writer = self.code_capture(written=0x0114, empty=False, stand_ins=rows)
        captures, context = self.inputs(writer=writer)
        prepared = self.prepared("World.SavedObjects.Items.Code", captures, context)
        result = self.verify(prepared)
        self.assertTrue(result["verified"])
        self.assertEqual("/Capture/ModItemStandIns/2/StandInCode", result["carrier_path"])
        self.assertEqual(registry_binding.CODE_STAND_IN_CALLSITE, result["producer_callsite"])
        self.assertEqual(2, result["directed_relation"]["code_selection"]["selected_stand_in_index"])
        decision, row, span, _, _ = prepared
        spoofed = copy.deepcopy(decision)
        spoofed["producer_callsite"] = registry_binding.SPECS["bindings"][decision["binding_id"]]["producer_callsite"]
        with self.assertRaises(source_binding.BindingError):
            self.verify((spoofed, row, span, captures, context))
        for change in (lambda c: c["ModItemStandIns"].reverse(),
                       lambda c: c["ModItemStandIns"][2].__setitem__("StandInCode", 0x0105),
                       lambda c: c.__setitem__("ModSetEmpty", True),
                       lambda c: c["Objects"]["Items"][0]["Value"].__setitem__("Code", 0x0102)):
            observed = copy.deepcopy(writer)
            change(observed["Capture"])
            changed_captures, changed_context = self.inputs(writer=observed)
            with self.assertRaises(source_binding.BindingError):
                self.verify(self.prepared("World.SavedObjects.Items.Code", changed_captures, changed_context))

    def test_code_zero_high_substitution_and_class_before_written_code(self):
        for written in (0, 65535):
            writer = self.code_capture(written=written, empty=False,
                stand_ins=[{"Code": 0x0101, "StandInCode": written, "StandInRow": 255}])
            captures, context = self.inputs(writer=writer)
            self.assertTrue(self.verify(self.prepared("World.SavedObjects.Items.Code", captures, context))["verified"])
        writer = self.code_capture(code=0x0301, written=0x0201, empty=False, clazz="Armor",
            stand_ins=[{"Code": 0x0301, "StandInCode": 0x0201, "StandInRow": 0}])
        writer["Capture"]["Objects"]["Items"][0]["Value"]["WeightPresent"] = False
        captures, context = self.inputs(writer=writer)
        for field in ("Code", "Shape", "Material"):
            result = self.verify(self.prepared("World.SavedObjects.Items."+field, captures, context))
            self.assertTrue(result["verified"])
            if field == "Code":
                self.assertTrue(result["directed_relation"]["code_selection"]["class_selected_before_substitution"])

    def test_code_missing_effective_observer_operands_keep_old_captures_incomplete(self):
        self.assertFalse(self.verify(self.prepared("World.SavedObjects.Items.Code"))["verified"])
        for change in (lambda c: c.pop("ModSetEmpty"), lambda c: c.pop("ModItemStandIns"),
                       lambda c: c["ModItemStandIns"][0].pop("StandInRow")):
            writer = self.code_capture(empty=False,
                stand_ins=[{"Code": 0x0102, "StandInCode": 123, "StandInRow": 7}])
            change(writer["Capture"])
            writer["Capture"]["RawModConfig"] = {"Mods": [], "Items": []}
            captures, context = self.inputs(writer=writer)
            result = self.verify(self.prepared("World.SavedObjects.Items.Code", captures, context))
            self.assertFalse(result["verified"])
        writer = self.code_capture(empty=True)
        writer["Capture"].pop("ModItemStandIns")
        captures, context = self.inputs(writer=writer)
        self.assertTrue(self.verify(self.prepared("World.SavedObjects.Items.Code", captures, context))["verified"])

    def test_code_stand_in_widths_boolean_and_complete_ordered_population(self):
        mutations = (lambda c: c.__setitem__("ModSetEmpty", 1),
            lambda c: c.__setitem__("ModItemStandIns", {}),
            lambda c: c["ModItemStandIns"][0].__setitem__("Code", True),
            lambda c: c["ModItemStandIns"][0].__setitem__("Code", 65536),
            lambda c: c["ModItemStandIns"][0].__setitem__("StandInCode", -1),
            lambda c: c["ModItemStandIns"][0].__setitem__("StandInCode", 65536),
            lambda c: c["ModItemStandIns"][0].__setitem__("StandInRow", 256),
            lambda c: c["ModItemStandIns"][0].__setitem__("StandInRow", False),
            lambda c: c["Objects"]["Items"][0]["Value"].__setitem__("Code", 0))
        for change in mutations:
            writer = self.code_capture(empty=False,
                stand_ins=[{"Code": 0x0101, "StandInCode": 0x0101, "StandInRow": 0}])
            change(writer["Capture"])
            captures, context = self.inputs(writer=writer)
            with self.assertRaises(source_binding.BindingError):
                self.prepared("World.SavedObjects.Items.Code", captures, context)

    def test_code_independent_current_and_raw_byte_loss_no_match_addition(self):
        writer = self.code_capture(empty=False, stand_ins=[])
        captures, context = self.inputs(writer=writer)
        self.assertTrue(self.verify(self.prepared("World.SavedObjects.Items.Code", captures, context))["verified"])
        for change in (lambda c: c["Objects"]["Items"][0]["Value"].__setitem__("Code", 0x0102),
                       lambda c: c["ModItemStandIns"].append({"Code": 0x0101, "StandInCode": 99, "StandInRow": 0})):
            observed = copy.deepcopy(writer)
            change(observed["Capture"])
            changed_captures, changed_context = self.inputs(writer=observed)
            with self.assertRaises(source_binding.BindingError):
                self.verify(self.prepared("World.SavedObjects.Items.Code", changed_captures, changed_context))
        writer = self.code_capture(written=0x0105, empty=False,
            stand_ins=[{"Code": 0x0101, "StandInCode": 0x0105, "StandInRow": 5}])
        captures, context = self.inputs(writer=writer)
        prepared = self.prepared("World.SavedObjects.Items.Code", captures, context)
        body, _ = byte_layout.raw_reader.unpack_sav(self.saved)
        changed = bytearray(body)
        changed[prepared[2]["start"]] ^= 1
        self.use_saved(byte_layout.raw_reader.replace_body(self.saved, bytes(changed)))
        changed_captures, changed_context = self.inputs(writer=writer)
        with self.assertRaises(source_binding.BindingError):
            self.verify(self.prepared("World.SavedObjects.Items.Code", changed_captures, changed_context))

    def test_code_identity_masks_alias_reindex_and_coherent_emitted_swap(self):
        actions = {"Version": 1, "Ownership": [{"Kind": 1, "ID": 6, "Object": 4},
            {"Kind": 2, "ID": 10, "Object": 2}, {"Kind": 4, "ID": 9, "Object": 1}], "Bindings": []}
        writer = self.code_capture(actions=actions, second_item=True)
        captures, context = self.inputs(writer=writer)
        self.assertTrue(self.verify(self.prepared("World.SavedObjects.Items.Code", captures, context, archive=4))["verified"])
        with self.assertRaises(source_binding.BindingError):
            self.prepared("World.SavedObjects.Items.Code", captures, context, archive=7)
        for option, archive in (({"prefix_item": True}, 5), ({"aliases": True}, None)):
            writer = self.code_capture(**option)
            if "aliases" in option:
                writer["Capture"]["Objects"]["Containers"][0]["Items"] = [6, 6]
            captures, context = self.inputs(writer=writer)
            self.assertTrue(self.verify(self.prepared("World.SavedObjects.Items.Code", captures, context, archive=archive))["verified"])
        for change in (lambda o: o["Items"][0]["Coverage"].__setitem__("Unknown", 128),
                       lambda o: o["Items"][0]["Coverage"].__setitem__("Unknown", 16384),
                       lambda o: o["Items"][0]["Token"].__setitem__("Identity", 0),
                       lambda o: o.__setitem__("Containers", [])):
            writer = self.code_capture()
            change(writer["Capture"]["Objects"])
            captures, context = self.inputs(writer=writer)
            span = next(s for s in self.layout["decoded_spans"] if s["label"].endswith("].Code"))
            direct, error = registry_binding.direct_literal(registry_binding.actual_context(context),
                writer["Capture"]["Objects"], 0, span, registry_binding.SPECS["bindings"]["World.SavedObjects.Items.Code"])
            self.assertIsNone(direct)
            self.assertIsNotNone(error)

    def test_code_exact_late_dependencies_and_main_dispatch(self):
        writer = self.code_capture(empty=False, written=0x0105,
            stand_ins=[{"Code": 0x0101, "StandInCode": 0x0105, "StandInRow": 5}])
        captures, context = self.inputs(writer=writer)
        prepared = self.prepared("World.SavedObjects.Items.Code", captures, context)
        decision, row, span, _, _ = prepared
        result = self.verify(prepared)
        self.assertEqual(registry_binding.CODE_PROJECTION_PINS["pkg/game/mods.go"], result["producer_dependencies"]["pkg/game/mods.go"])
        receipt = {"source_pin": self.source_pin, "knowledge_pin": self.knowledge_pin,
            "input_sha256": self.layout["file_sha256"], "decisions": [decision],
            "capture_pin_paths": {"writer": {"source": "/Revision", "knowledge": "/KnowledgePin"}}}
        joined = join_sources.join(self.layout, self.rows, receipt, self.source_pin, self.knowledge_pin, captures, context)
        leaf = next(s for s in joined["joined_spans"] if s["label"] == span["label"])
        self.assertTrue(leaf["source_bindings"][0]["verified"])
        self.assertEqual("/Capture/ModItemStandIns/0/StandInCode", leaf["source_bindings"][0]["carrier_path"])
        self.assertEqual("incomplete", joined["status"])
        self.assertFalse(joined["source_classification_joined"])
        with mock.patch.dict(registry_binding.CODE_PROJECTION_PINS, {"pkg/game/modmark.go": "f"*64}):
            self.assertFalse(self.verify(prepared)["verified"])

    def test_count_exact_current_key_count_root_and_collision_controls(self):
        mutations = [lambda o: o["Items"][0]["Token"].__setitem__("Identity", 404),
            lambda o: o["Items"][0]["Value"].__setitem__("Count", 2),
            lambda o: o["Items"][0]["Value"].__setitem__("Count", True),
            lambda o: o["Items"][0]["Value"].__setitem__("Count", 65536),
            lambda o: o["Items"][0].__setitem__("InFlight", 1),
            lambda o: o["Containers"][0].__setitem__("Present", False),
            lambda o: o["Containers"][0]["Owner"].__setitem__("Kind", True),
            lambda o: o["Containers"][0].__setitem__("Items", [True]),
            lambda o: o["Effects"][0]["Token"].__setitem__("Identity", 202),
            lambda o: o["Spells"].append({"ID": 8, "This": 202}),
            lambda o: o["Sacks"][0].__setitem__("ID", 6)]
        for change in mutations:
            writer = copy.deepcopy(self.writer)
            change(writer["Capture"]["Objects"])
            captures, context = self.inputs(writer=writer)
            with self.assertRaises(source_binding.BindingError):
                self.verify(self.prepared("World.SavedObjects.Items.Count", captures, context))
        body, _ = byte_layout.raw_reader.unpack_sav(self.saved)
        walker = byte_layout.raw_reader.Walker(body).parse()
        effect = next(record for record in walker.order if record["class"] == "Effect")
        changed = bytearray(body)
        struct.pack_into("<I", changed, effect["offsets"]["Identity"], 202)
        self.saved = byte_layout.raw_reader.replace_body(self.saved, bytes(changed))
        self.path.write_bytes(self.saved)
        self.layout = byte_layout.atlas(self.path)
        with self.assertRaises(source_binding.BindingError):
            self.prepared("World.SavedObjects.Items.Count")

    def test_count_current_predicate_code_pin_is_required(self):
        with mock.patch.dict(registry_binding.COUNT_PROJECTION_PINS,
                             {"pkg/game/savitemobjects_scroll.go": "f"*64}):
            result = self.verify(self.prepared("World.SavedObjects.Items.Count"))
            self.assertTrue(result["value_verified"])
            self.assertFalse(result["verified"])

    def test_exact_effect_array_subspan_and_corruption_controls(self):
        for binding in ("World.SavedObjects.Items.Position", "World.SavedObjects.Effects.Operand"):
            with self.subTest(binding=binding):
                prepared = self.prepared(binding)
                self.assertTrue(self.verify(prepared)["value_verified"])
                decision, row, span, captures, context = prepared
                for patch in ({"guard_assertions": [{"capture": "writer", "path": "/Capture/WorldPresent", "op": "eq", "value": True}]},
                              {"tool_pin": "f"*64}, {"input_sha256": "0"*64}, {"layout_sha256": "0"*64}):
                    with self.assertRaises(source_binding.BindingError):
                        self.verify(({**decision, **patch}, row, span, captures, context))
                with self.assertRaises(source_binding.BindingError):
                    self.verify((decision, row, {**span, "length": True}, captures, context))

    def test_independent_expected_pins_and_actual_capture_bytes_are_required(self):
        for source, knowledge in (("c"*40, self.knowledge_pin), (self.source_pin, "c"*40)):
            captures, context = self.inputs(expected_source=source, expected_knowledge=knowledge)
            with self.assertRaises(source_binding.BindingError):
                self.prepared(captures=captures, context=context)
        prepared = self.prepared()
        decision, row, span, captures, context = prepared
        changed = copy.deepcopy(captures)
        changed["writer"]["Capture"]["WorldPresent"] = 1
        with self.assertRaises(source_binding.BindingError):
            self.verify((decision, row, span, changed, context))
        missing = {**context, "capture_bytes": {}}
        with self.assertRaises(source_binding.BindingError):
            self.verify((decision, row, span, captures, missing))
        self.assertFalse(source_binding.verify(decision, row, span, decision["start"], decision["end"], captures)["verified"])

    def test_cli_uses_actual_registry_bytes_and_names_concrete_modules(self):
        decision, _, span, _, context = self.prepared()
        scripts = Path(__file__).parent
        capture = Path(self.temp.name)/"writer.json"
        capture.write_bytes(context["capture_bytes"]["writer"])
        receipt = {"source_pin": self.source_pin, "knowledge_pin": self.knowledge_pin,
            "input_sha256": self.layout["file_sha256"], "decisions": [decision],
            "capture_pin_paths": {"writer": {"source": "/Revision", "knowledge": "/KnowledgePin"}},
            "capture_sha256": {"writer": join_sources.raw_reader.sha(capture.read_bytes())},
            "source_map_sha256": join_sources.raw_reader.sha((scripts/"source_catalog.json").read_bytes())}
        receipt_path, output = Path(self.temp.name)/"receipt.json", Path(self.temp.name)/"joined.json"
        receipt_path.write_text(json.dumps(receipt), encoding="utf-8")
        result = subprocess.run([sys.executable, "-B", str(scripts/"join_sources.py"),
            "--input", str(self.path), "--output", str(output), "--source-map", str(scripts/"source_catalog.json"),
            "--decisions", str(receipt_path), "--source-pin", self.source_pin,
            "--knowledge-pin", self.knowledge_pin, "--capture", "writer="+str(capture)],
            capture_output=True, text=True)
        self.assertEqual(2, result.returncode, result.stderr)
        joined = json.loads(output.read_bytes())
        leaf = next(row for row in joined["joined_spans"] if row["label"] == span["label"])
        self.assertTrue(leaf["source_bindings"][0]["value_verified"])
        self.assertFalse(joined["source_classification_joined"])
        self.assertEqual({"byte_layout.py", "raw_reader.py", "source_binding.py", "registry_binding.py", "typed_payload.py"},
            set(joined["instrument_bundle_sha256"]))
        self.assertEqual(join_sources.raw_reader.sha(self.saved), join_sources.raw_reader.sha(self.path.read_bytes()))

    def test_current_value_loss_cannot_pass_equal_retained_or_predicate_metadata(self):
        writer = copy.deepcopy(self.writer)
        writer["Capture"]["Objects"]["Items"][0]["Value"]["Code"] += 1
        captures, context = self.inputs(writer=writer)
        prepared = self.prepared(captures=captures, context=context)
        with self.assertRaises(source_binding.BindingError):
            self.verify(prepared)
        writer["Capture"]["Objects"]["Items"][0]["Token"]["Position"][0] = True
        captures, context = self.inputs(writer=writer)
        with self.assertRaises(source_binding.BindingError):
            self.prepared("World.SavedObjects.Items.Position", captures, context)

    def test_same_class_emitted_current_owner_swap_never_establishes_identity(self):
        actions = {"Version": 1, "Ownership": [{"Kind": 1, "ID": 6, "Object": 4},
            {"Kind": 1, "ID": 7, "Object": 2}, {"Kind": 2, "ID": 10, "Object": 3},
            {"Kind": 4, "ID": 9, "Object": 1}], "Bindings": []}
        self.saved = synthetic_registry_save(actions, second_item=True)
        self.path.write_bytes(self.saved)
        self.layout = byte_layout.atlas(self.path)
        self.assertEqual("parsed", self.layout["status"], self.layout.get("error"))
        writer = copy.deepcopy(self.writer)
        objects = writer["Capture"]["Objects"]
        other = copy.deepcopy(objects["Items"][0])
        other.update(ID=7, Effects=[])
        objects["Items"].append(other)
        objects["Containers"][0]["Items"] = [7, 6]
        captures, context = self.inputs(writer)
        span = next(s for s in self.layout["decoded_spans"] if s["label"] == "Item[archive=7].Code")
        row = self.rows[284-2]
        base = {"source_pin": self.source_pin, "knowledge_pin": self.knowledge_pin,
            "input_sha256": self.layout["file_sha256"], "layout_sha256": context["layout_sha256"],
            "capture_sha256": {"writer": join_sources.raw_reader.sha(context["capture_bytes"]["writer"])},
            "space": "decoded", "catalog_row": 284}
        decision, error = registry_binding.prepare("World.SavedObjects.Items.Code", 0, span, captures, base, context)
        self.assertIsNone(error)
        result = source_binding.verify(decision, row, span, 0, 2, captures, context)
        self.assertTrue(result["value_verified"])
        self.assertFalse(result["verified"])
        self.assertFalse(result["owner_relation_verified"])

    def test_framed_secondary_alias_cycle_and_width_controls(self):
        for saved in (synthetic_registry_save(aliases=True), synthetic_registry_save('{"Version":1,"Version":1}'),
                      synthetic_registry_save({"Version": 1, "Ownership": [{"Kind": 1, "ID": 6, "Object": True}]})):
            body, _ = join_sources.raw_reader.unpack_sav(saved)
            with self.assertRaises((source_binding.BindingError, typed_payload.PayloadError)):
                ctx = registry_binding.context(saved, self.inputs()[1]["capture_bytes"]["writer"], self.source_pin, self.knowledge_pin)
                if ctx:
                    record = ctx["canonical"][2]
                    registry_binding.item_relation(ctx, self.writer["Capture"]["Objects"], 6, record)
        saved = bytearray(self.saved)
        at = self.layout["tail"]["store_start"]+24
        struct.pack_into("<II", saved, at+4, 0, 1)
        body, _ = join_sources.raw_reader.unpack_sav(saved)
        with self.assertRaises(source_binding.BindingError):
            registry_binding.typed_correspondence(bytes(saved), body)
        walker = join_sources.raw_reader.Walker(b"")
        walker.records = {2: {"archive_index": 2, "class": "Sack", "refs": {"Contents": [2]}}}
        walker.order, walker.roots = [walker.records[2]], {"Sacks": [2]}
        with self.assertRaises(source_binding.BindingError):
            registry_binding.canonical_records(walker)


if __name__ == "__main__":
    unittest.main()
