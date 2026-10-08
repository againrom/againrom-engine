"""Bounded physical byte atlas for explicitly supplied SAV files.

Leaf spans cover physical bytes; nested object ranges are separate metadata.
Names identify wire components, never current/source/rule/constructor origins.
Decoded offsets and compressed file offsets are distinct coordinate spaces.
"""

import argparse
import collections
import hashlib
import json
from pathlib import Path
import struct
import sys

sys.dont_write_bytecode = True
import raw_reader


MAX_COUNT = 1000000


class Cursor:
    def __init__(self, data, space, base=0):
        self.data, self.space, self.base, self.pos = data, space, base, 0
        self.spans = []

    def take(self, n, label, kind="value", **meta):
        if not 0 <= n <= len(self.data)-self.pos:
            raise raw_reader.BadSAV(f"{self.space} offset {self.base+self.pos}: truncated {label} ({n} bytes)")
        start = self.pos
        self.pos += n
        if n:
            self.spans.append({"start": self.base+start, "end": self.base+self.pos,
                               "length": n, "label": label, "kind": kind,
                               "bytes_hex": self.data[start:self.pos].hex(), **meta})
        return self.data[start:self.pos]

    def number(self, fmt, label, kind="value"):
        value = struct.unpack("<"+fmt, self.take(struct.calcsize("<"+fmt), label, kind, wire=fmt))[0]
        self.spans[-1]["value"] = value
        return value

    def count(self, label):
        n = self.number("H", label+".Count16", "framing")
        if n == 0xffff:
            n = self.number("I", label+".Count32", "framing")
        if n > MAX_COUNT:
            raise raw_reader.BadSAV(f"{label}: count exceeds reader bound")
        return n

    def string(self, label):
        n = self.number("B", label+".Length8", "string-framing")
        if n == 0xff:
            n = self.number("H", label+".Length16", "string-framing")
            if n == 0xfffe:
                raise raw_reader.BadSAV(f"{label}: Unicode CString is outside this reader")
            if n == 0xffff:
                n = self.number("I", label+".Length32", "string-framing")
        return self.take(n, label+".Bytes", "string").decode("latin1")

    def array(self, label, width=2, plain=False, opaque=False):
        n = self.number("I", label+".Count", "framing") if plain else self.count(label)
        if n > MAX_COUNT:
            raise raw_reader.BadSAV(f"{label}: array exceeds reader bound")
        self.take(width*n, label+".Values", "opaque" if opaque else "array-values",
                  element_width=width, element_count=n)
        return n


class AtlasWalker(raw_reader.Walker):
    def __init__(self, body):
        super().__init__(body)
        self.cursor = Cursor(body, "decoded")
        self.path, self.depth = "Document", 0
        self.object_sites = []

    def read(self, name, fmt, kind="value"):
        value = self.cursor.number(fmt, self.path+"."+name, kind)
        self.pos = self.cursor.pos
        return value

    def raw(self, name, size, kind="opaque", **meta):
        value = self.cursor.take(size, self.path+"."+name, kind, **meta)
        self.pos = self.cursor.pos
        return value

    def string_at(self, name):
        value = self.cursor.string(self.path+"."+name)
        self.pos = self.cursor.pos
        return value

    def array_at(self, name, width=2, plain=False, opaque=False):
        n = self.cursor.array(self.path+"."+name, width, plain, opaque)
        self.pos = self.cursor.pos
        return n

    def field(self, r, name, fmt=None, size=None, kind=None):
        r["offsets"][name] = self.pos
        if kind is None and name in {"Identity", "Reference"}:
            kind = "numeric-identity" if name == "Identity" else "numeric-reference"
        value = self.raw(name, size, kind or "opaque").hex() if size is not None else self.read(name, fmt, kind or "value")
        aliases = {"Position": "Block12", "Selector": "T0C", "Type": "T0E", "Flags": "T08",
                   "PublicationMask": "T18", "Price": "T1C", "Code": "F40", "Count": "F42",
                   "Kind": "F44", "Shape": "F45", "Material": "F46", "Weight": "F4A",
                   "EffectKind": "E3C", "EffectMode": "E3D", "Operand": "E40", "EffectID": "E0C",
                   "ActionClock": "U138", "Gold": "S3C"}
        if r["class"] == "Weapon":
            aliases.update({"Attack": "W52", "Defence": "W6A", "OwnKind": "W50"})
        elif r["class"] in {"Armor", "Shield"}:
            aliases.update({"Defence": "A52" if r["class"] == "Armor" else "S50", "OwnKind": "A50"})
        elif r["class"] == "Spell":
            aliases.update({"ID": "S08", "Range": "S09", "Defensive": "S0A", "ManaCost": "S0C", "Identity": "This"})
        self.cursor.spans[-1]["wire_member"] = aliases.get(name, name)
        r["values"][name] = value
        return value

    def run(self, r, fields):
        for name, width in fields:
            if isinstance(width, int):
                self.field(r, name, size=width)
            else:
                self.field(r, name, width)

    def refs(self, r, name, n):
        if n > MAX_COUNT:
            self.fail("reference count exceeds reader bound")
        values, offsets = [], []
        parent = self.path
        for slot in range(n):
            offsets.append(self.pos)
            self.path = parent+f".{name}[{slot}]"
            values.append(self.obj())
            self.path = parent
        r["refs"][name] = values
        r.setdefault("ref_offsets", {})[name] = offsets
        return values

    def list32(self, r, name):
        r.setdefault("list_count_offsets", {})[name] = self.pos
        return self.refs(r, name, self.read(name+".Count", "I", "framing"))

    def diary_at(self, name):
        self.array_at(name+".Journal", 4)
        self.array_at(name+".JournalWords", 2)
        self.read(name+".D2C", "I", "numeric-reference")

    def obj(self):
        site, tag_at = self.path, self.pos
        tag = self.read("ArchiveTag", "H", "reference-slot")
        if tag == 0:
            self.object_sites.append({"site": site, "tag_offset": tag_at, "target": 0, "arm": "null"})
            return 0
        if tag == 0x7fff:
            self.fail("32-bit CArchive tag is outside this reader")
        if tag == 0xffff:
            schema = self.read("Class.Schema", "H", "class-framing")
            n = self.read("Class.NameLength", "H", "class-framing")
            name = self.raw("Class.Name", n, "class-name").decode("ascii")
            if schema != 1:
                self.fail(f"unsupported schema {schema}")
            self.classes[self.next_index] = name
            self.next_index += 1
            arm = "new-class-object"
        elif tag & 0x8000:
            name = self.classes.get(tag & 0x7fff)
            if name is None:
                self.fail(f"unknown class index {tag & 0x7fff}")
            arm = "known-class-object"
        else:
            if tag not in self.records:
                self.fail(f"unallocated back-reference {tag}")
            self.object_sites.append({"site": site, "tag_offset": tag_at, "target": tag, "arm": "back-reference"})
            return tag
        index = self.next_index
        self.next_index += 1
        self.object_sites.append({"site": site, "tag_offset": tag_at, "target": index, "arm": arm})
        r = {"archive_index": index, "ordinal": len(self.order)+1, "class": name,
             "tag_offset": tag_at, "body_offset": self.pos, "values": {}, "offsets": {}, "refs": {}}
        self.records[index] = r
        self.order.append(r)
        self.depth += 1
        if self.depth > 64:
            self.fail("recursive object depth exceeds reader bound")
        self.path = f"{name}[archive={index}]"
        self.program(r)
        self.path, self.depth = site, self.depth-1
        r["end_offset"] = self.pos
        return index

    def program(self, r):
        cls = r["class"]
        if cls == "Player":
            r["values"]["Name"] = self.string_at("Name")
            self.run(r, [("Slot", "H"), ("SlotAgain", "I"), ("Raw10", 8), ("F44", "B"),
                         ("Participant", "I"), ("F2C", "H"), ("Money", "I"), ("Outcome", "B"),
                         ("F3D", "B"), ("F48", "I"), ("F50", "I"), ("F54", "H"), ("F4C", "H"),
                         ("F58", "I"), ("Hero", "I"), ("Identity", "I")])
            n = self.read("Groups.Count", "I", "framing")
            if n > MAX_COUNT:
                self.fail("group count exceeds reader bound")
            for g in range(n):
                self.array_at(f"Group[{g}].G20")
                self.raw(f"Group[{g}].G3C", 80)
                self.array_at(f"Group[{g}].G4C")
                self.list32(r, f"Group[{g}].Actors")
                for name in ["G1C", "G40", "G44"]:
                    self.read(f"Group[{g}].{name}", "I")
            self.raw("PRaw32", 32)
            self.diary_at("InlineDiary")
            return
        if cls == "Diary":
            self.diary_at("Diary")
            return
        if cls == "Spell":
            self.run(r, [("ID", "B"), ("Range", "B"), ("Defensive", "B"), ("ManaCost", "H"), ("Identity", "I")])
            return
        token_classes = raw_reader.ITEMS | raw_reader.PLACEABLES | {"Token", "Effect", "Effect_DirectDamage", "SpellEffect", "PointEffect", "AreaEffect", "SpellTransport", "VirtualCaster"}
        if cls not in token_classes:
            self.fail(f"unsupported class {cls}")
        self.token(r)
        if cls in raw_reader.ITEMS:
            self.list32(r, "Effects")
            self.run(r, [("Code", "H"), ("Count", "H"), ("Kind", "B"), ("Shape", "B"),
                         ("Material", "B"), ("F48", "H"), ("Weight", "h"), ("F47", "B")])
            if cls == "Weapon":
                self.run(r, [("Attack", 24), ("Defence", 22), ("OwnKind", "B")])
                self.refs(r, "OwnedSpell", 1)
            elif cls in {"Armor", "Shield"}:
                self.field(r, "Defence", size=22)
                if cls == "Armor":
                    self.field(r, "OwnKind", "B")
        elif cls == "Sack":
            self.field(r, "Gold", "I")
            self.container(r, "Contents")
        elif cls in {"Building", "Outpost", "Tavern", "Shop"}:
            self.run(r, [("B52", 22), ("B40", "B"), ("B42", "H"), ("B44", "H"),
                         ("B46", "H"), ("B48", "B"), ("B60", "B"), ("B61", "B"), ("B64", "I"), ("B68", "I")])
            if cls == "Outpost":
                self.run(r, [("O84", "I"), ("O88", "I"), ("O80", "I"), ("O8C", "I")])
                self.array_at("O6C", 8, opaque=True)
            elif cls in {"Tavern", "Shop"}:
                self.read("T9C" if cls == "Tavern" else "S70", "I")
        elif cls in {"Effect", "Effect_DirectDamage"}:
            self.run(r, [("EffectKind", "B"), ("EffectMode", "B"), ("Operand", "I"), ("EffectID", "B")])
            if cls == "Effect_DirectDamage":
                self.field(r, "Damage", size=24)
        elif cls in {"SpellEffect", "PointEffect", "AreaEffect", "SpellTransport"}:
            self.run(r, [("SE40", "B"), ("SE41", "B")])
            if cls == "PointEffect":
                self.refs(r, "Effect", 1)
                self.read("PE44", "I", "numeric-reference")
            elif cls == "AreaEffect":
                self.raw("AE48", 4)
                self.read("AE4C", "H")
                self.refs(r, "Effect", 1)
            elif cls == "SpellTransport":
                self.refs(r, "Child", 1)
                self.refs(r, "Area", 1)
                self.read("ST4C", "H")
        elif cls == "VirtualCaster":
            self.read("VC3C", "B")
            self.raw("VC40", 6)
        elif cls in {"Unit", "Humanoid", "Human"}:
            self.list32(r, "ActorEffects")
            self.array_at("U15C")
            self.array_at("U178")
            for name, n in [("UA6", 24), ("UBE", 22), ("U114", 24), ("UD4", 64), ("U154", 180), ("U158", 148)]:
                self.raw(name, n)
            self.array_at("U158_90")
            self.run(r, [("U49", "B"), ("U4A", "B"), ("U4B", "B"), ("U4C", "B"),
                         ("U50", 4), ("U54", 4), ("U58", 4), ("U60", "B"), ("U61", "B"), ("U6C", "B")])
            self.refs(r, "HeldWeapon", 1)
            self.refs(r, "HeldShield", 1)
            r["values"]["Name"] = self.string_at("Name")
            for name in ["Body", "Reaction", "Mind", "Spirit", "Speed", "U8E", "U90", "Capacity",
                         "Health", "HealthMax", "HealthRegen", "Mana", "ManaMax", "ManaRegen"]:
                self.read(name, "H")
            self.run(r, [("UA2", "B"), ("UA3", "B"), ("UA0", "H"), ("UA4", "H"), ("U12C", "B"),
                         ("U130", "I"), ("U134", "B"), ("U135", "B"), ("U136", "B"),
                         ("ActionClock", "I"), ("Stage", "B"), ("U148", "I"), ("U144", "I")])
            self.refs(r, "Extra68", 1)
            if self.read("HasInventory", "B", "presence-framing"):
                self.container(r, "Pack")
            if self.read("HasSpellbook", "B", "presence-framing"):
                self.read("Book.Header", "I")
                n = self.read("Book.Count", "I", "framing")
                self.refs(r, "Book", max(n-1, 0))
            self.run(r, [("U5C", "I"), ("U64", "I"), ("U44", "I"), ("U40", "I"), ("U48", "B")])
            if cls in {"Humanoid", "Human"}:
                self.raw("H1CC", 24)
                self.refs(r, "Worn", 12)
                self.refs(r, "HumanDiary", 1)

    def parse(self):
        self.path = "Header"
        self.read("SubTick", "I")
        self.read("FullClock", "I")
        self.map_name = self.string_at("MapName")
        for name in ["W11C", "W124", "W128", "W12C", "W130", "W134", "W138", "W13C", "W148", "W144", "W140"]:
            self.read(name, "I")
        self.mission = self.read("Mission", "I")
        self.read("Difficulty", "I")
        self.read("PlayerList20", "I")
        self.path = "Document"
        for name in ["Players", "Dead"]:
            holder = {"refs": {}}
            self.roots[name] = self.list32(holder, name)
        self.world = self.read("WorldPresent", "B", "presence-framing")
        if self.world:
            for name in ["Buildings", "SpellEffects"]:
                holder = {"refs": {}}
                self.roots[name] = self.list32(holder, name)
            self.path = "Terrain"
            n = self.read("Blocks.Count", "H", "framing")
            self.raw("Blocks.Values", 4*n, "array-values", element_width=4, element_count=n)
            n = self.cursor.count("Terrain.Cells")
            self.pos = self.cursor.pos
            for i in range(n):
                self.path = f"Terrain.Cell[{i}]"
                cell = self.read("Key", "H")
                for name in ["Cost", "Static", "LayerCount", "Residue03"]:
                    self.read(name, "B")
                refs = {name: self.read(name, "I", "numeric-reference") for name in ["Ground", "Air", "Building", "Sack"]}
                for layer in range(6):
                    self.read(f"Layers[{layer}]", "I", "numeric-reference")
                for name in ["Operation", "Power", "SourceX", "SourceY", "TargetX", "TargetY"]:
                    self.read(name, "B")
                self.raw("Residue32", 2)
                self.cells.append({"cell": [cell & 255, cell >> 8], **refs})
            self.path = "Terrain"
            self.terrain_identity = self.read("Identity", "I", "numeric-identity")
            self.path = "Session"
            self.raw("Results", 400, "array-values", element_width=4, element_count=100)
            self.raw("Latches", 1000, "array-values", element_width=1, element_count=1000)
            self.raw("Raw08", 48)
            self.raw("RawA828", 400)
            self.raw("DiplomacyHeader", 8)
            self.raw("Diplomacy", 2500, "array-values", element_width=1, element_shape=[50, 50])
            for name, fmt in [("FlagA48", "B"), ("FlagA49", "B"), ("ValueA4C", "I"), ("Won", "I"), ("ValueB3B0", "I"), ("Lost", "I")]:
                self.read(name, fmt)
            self.path = "Document"
            holder = {"refs": {}}
            self.roots["Sacks"] = self.list32(holder, "Sacks")
        self.path = "Trailer"
        self.marker = self.read("Discriminator", "I", "framing")
        if self.marker == 0xBADFACE1:
            self.read("Global", "I")
        self.read("TurnTracing", "I")
        self.read("ScriptTracing", "I")
        self.raw("Residual", 392)
        if len(self.body)-self.pos != self.pos % 2:
            self.fail("unexpected decoded tail length")
        self.raw("Alignment", len(self.body)-self.pos, "transport-padding")
        return self


def coverage(spans, size, base=0):
    end, gaps, overlaps, outside = base, [], [], []
    by_kind = collections.Counter()
    for row in sorted(spans, key=lambda s: (s["start"], s["end"])):
        if row["start"] < base or row["end"] > base+size or row["end"] <= row["start"]:
            outside.append([row["start"], row["end"]])
        if row["start"] > end:
            gaps.append([end, row["start"]])
        elif row["start"] < end:
            overlaps.append([row["start"], min(end, row["end"])])
        end = max(end, row["end"])
        by_kind[row["kind"]] += row["length"]
    if end < base+size:
        gaps.append([end, base+size])
    return {"bytes": size, "leaf_spans": len(spans), "gap_bytes": sum(b-a for a, b in gaps),
            "overlap_bytes": sum(b-a for a, b in overlaps), "gaps": gaps, "overlaps": overlaps,
            "outside_bytes": sum(max(1, b-a) for a, b in outside), "outside": outside,
            "bytes_by_kind": dict(by_kind), "opaque_bytes": by_kind["opaque"]}


def file_codec(raw, body):
    c = Cursor(raw, "file")
    c.take(4, "Envelope.Magic", "framing")
    end = c.number("I", "Envelope.BlobEnd", "framing")
    c.number("I", "Envelope.Version", "framing")
    c.number("I", "Envelope.BlobBytes", "framing")
    c.number("I", "WordCodec.OutWords", "framing")
    packets, decoded = [], 0
    while c.pos < end:
        i, at = len(packets), c.pos
        op = c.number("B", f"WordCodec.Packet[{i}].Opcode", "compression-framing")
        n = op & 0x7f
        payload_at = c.pos
        c.take(2 if op & 0x80 else 2*n, f"WordCodec.Packet[{i}].Payload", "compressed-payload")
        packets.append({"packet": i, "file_opcode": at, "file_payload_start": payload_at,
                        "file_payload_end": c.pos, "decoded_start": decoded, "decoded_end": decoded+2*n,
                        "relation": "repeat-word" if op & 0x80 else "literal-copy", "words": n})
        decoded += 2*n
    if decoded != len(body) or c.pos != end:
        raise raw_reader.BadSAV("codec relation disagrees with declared span")
    return c, packets


def tail_atlas(c):
    label_at = c.pos
    label = c.data[c.pos:c.pos+256]
    if len(label) != 256:
        raise raw_reader.BadSAV("missing 256-byte Label")
    nul = label.find(b"\0")
    if nul < 0:
        c.take(256, "Label.BytesWithoutTerminator", "string")
    else:
        c.take(nul, "Label.LiveBytes", "string")
        c.take(1, "Label.Terminator", "string-framing")
        c.take(255-nul, "Label.Residue", "opaque")
    store_at = c.pos
    if c.take(4, "YA1.Magic", "framing") != b"&YA1":
        raise raw_reader.BadSAV("unsupported physical tail: no framed YA1 store")
    root_first = c.number("I", "YA1.Root.First", "framing")
    root_count = c.number("I", "YA1.Root.Count", "framing")
    c.number("I", "YA1.Root.Kind", "framing")
    n = c.number("I", "YA1.RecordCount", "framing")
    c.take(4, "YA1.Reserved", "opaque")
    if n > MAX_COUNT or root_first+root_count > n:
        raise raw_reader.BadSAV("YA1 root/count outside reader bound")
    records = []
    for i in range(n):
        prefix = f"YA1.Record[{i}]"
        c.take(4, prefix+".Reserved", "opaque")
        data_at = c.pos
        value = c.number("I", prefix+".Data")
        size = c.number("I", prefix+".Size")
        kind = c.number("I", prefix+".Kind", "framing")
        name = c.take(16, prefix+".NameBuffer", "string").split(b"\0", 1)[0].decode("latin1")
        if kind & 1 and (kind & 14 or value+size > n):
            raise raw_reader.BadSAV("YA1 directory range/type malformed")
        if not kind & 1 and kind & 14 not in {0, 2, 4, 6}:
            raise raw_reader.BadSAV(f"unsupported YA1 kind {kind & 14}")
        records.append({"index": i, "name": name, "kind": kind, "data": value, "size": size, "data_field": data_at})
    heap_size = c.number("I", "YA1.PoolLength", "framing")
    heap_at = c.pos
    if heap_size > len(c.data)-c.pos:
        raise raw_reader.BadSAV("YA1 pool exceeds actual file")
    heap_owners = []
    for r in records:
        kind = r["kind"] & 14
        if not r["kind"] & 1 and kind in {0, 6}:
            if r["data"]+r["size"] > heap_size or kind == 6 and r["size"] % 4:
                raise raw_reader.BadSAV("YA1 pool reference malformed")
            heap_owners.append({"start": heap_at+r["data"], "end": heap_at+r["data"]+r["size"],
                                "label": f"YA1.Record[{r['index']}].{r['name']}.PoolValue", "kind": "string" if kind == 0 else "array-values"})
    points = sorted({heap_at, heap_at+heap_size} | {p for r in heap_owners for p in [r["start"], r["end"]]})
    overlaps = []
    for a, b in zip(points, points[1:]):
        owners = [r for r in heap_owners if r["start"] <= a and b <= r["end"]]
        labels = [r["label"] for r in owners]
        if len(owners) > 1:
            overlaps.append({"start": a, "end": b, "owners": labels})
        c.take(b-a, "YA1.Pool" if not owners else labels[0], "opaque" if not owners else owners[0]["kind"], aliases=labels[1:])
    campaign_at = c.pos
    def array(name):
        return c.array("Campaign."+name, 2, plain=True)
    def base(name):
        for field in ["Mission", "MapObject", "Payment", "ShopLow", "ShopHigh", "Announced"]:
            c.number("I", "Campaign."+name+"."+field)
        array(name+".AddHero")
        array(name+".EnableMercenary")
    base("Base")
    n = c.number("I", "Campaign.Children.Count", "framing")
    if n > MAX_COUNT:
        raise raw_reader.BadSAV("campaign child count exceeds reader bound")
    for i in range(n):
        base(f"Child[{i}]")
        c.number("I", f"Campaign.Child[{i}].Age")
    n = c.number("I", "Campaign.Parallel.Count", "framing")
    c.take(2*n, "Campaign.Parallel.Working", "array-values", element_width=2, element_count=n)
    c.take(2*n, "Campaign.Parallel.Pristine", "array-values", element_width=2, element_count=n)
    c.array("Campaign.HireFlags", 4, plain=True)
    for name in ["Mercenaries", "PermanentUnlocks", "InnNPC", "InnMission", "TCMission", "ShopMission"]:
        array(name)
    n = c.number("I", "Campaign.Documents.Count", "framing")
    if n > MAX_COUNT:
        raise raw_reader.BadSAV("campaign document count exceeds reader bound")
    for i in range(n):
        c.number("I", f"Campaign.Document[{i}].Value")
        c.number("I", f"Campaign.Document[{i}].Kind")
    for name in ["SelectedMission", "Raw114", "AutoGetMission", "LastMission", "FirstMapPoint", "Raw124", "Raw128"]:
        c.number("I", "Campaign."+name)
    n = c.number("I", "Campaign.Markers.Count", "framing")
    if n > MAX_COUNT:
        raise raw_reader.BadSAV("campaign marker count exceeds reader bound")
    for i in range(n):
        c.number("I", f"Campaign.Marker[{i}].Value")
        size = c.number("I", f"Campaign.Marker[{i}].TextLength", "string-framing")
        text = c.take(size, f"Campaign.Marker[{i}].Text", "string")
        if not text or text[-1] != 0:
            raise raw_reader.BadSAV("campaign marker is not NUL-terminated")
        c.number("I", f"Campaign.Marker[{i}].Field0")
        c.number("I", f"Campaign.Marker[{i}].Field1")
    if c.pos != len(c.data):
        raise raw_reader.BadSAV(f"unsupported unframed file tail: {len(c.data)-c.pos} bytes")
    return {"label_start": label_at, "store_start": store_at, "store_end": campaign_at,
            "root_first": root_first, "root_count": root_count,
            "campaign_start": campaign_at, "campaign_end": c.pos, "ya1_records": records,
            "pool_reference_alias_ranges": overlaps,
            "limits": "YA1 names/indexes and campaign wire values only. Campaign document carrier payload is not decoded into current continuation/source obligations."}


def atlas(path):
    raw = Path(path).read_bytes()
    body, end = raw_reader.unpack_sav(raw)
    w = AtlasWalker(body)
    file_cursor, packets = file_codec(raw, body)
    try:
        w.parse()
        tail = tail_atlas(file_cursor)
        native = raw_reader.Walker(body).parse()
        a = [(r["archive_index"], r["class"], r["tag_offset"], r["body_offset"], r["end_offset"]) for r in w.order]
        b = [(r["archive_index"], r["class"], r["tag_offset"], r["body_offset"], r["end_offset"]) for r in native.order]
        if a != b or w.roots != native.roots:
            raise raw_reader.BadSAV("annotated grammar disagrees with existing reader's object extents/roots")
        dc, fc = coverage(w.cursor.spans, len(body)), coverage(file_cursor.spans, len(raw))
        if any(c[k] for c in [dc, fc] for k in ["gap_bytes", "overlap_bytes", "outside_bytes"]):
            raise raw_reader.BadSAV("leaf atlas has a physical byte gap or overlap")
        return {"status": "parsed", "input_path": str(Path(path).resolve()), "file_bytes": len(raw),
                "file_sha256": raw_reader.sha(raw), "decoded_bytes": len(body), "decoded_sha256": raw_reader.sha(body),
                "blob_end": end, "world_present": bool(w.world), "mission": w.mission,
                "observed_classes": dict(collections.Counter(r["class"] for r in w.order)),
                "decoded_coverage": dc, "file_coverage": fc, "decoded_spans": w.cursor.spans,
                "file_spans": file_cursor.spans, "compression_packets": packets,
                "objects": [{k: r[k] for k in ["archive_index", "ordinal", "class", "tag_offset", "body_offset", "end_offset"]} for r in w.order],
                "object_reference_sites": w.object_sites, "tail": tail,
                "existing_reader_object_extent_match": True,
                "limits": "Observed input grammar only. Named arrays carry exact element widths/counts; opaque regions are explicit. Raw names/values prove no producer source, rule, constructor, hashed-state authority, native admission or universal class coverage. Parent object ranges overlap nested objects by design and are excluded from leaf coverage. Repeat packets have no one-to-one encoded/decoded byte relation."}
    except (raw_reader.BadSAV, UnicodeError, KeyError, struct.error) as exc:
        return {"status": "refused", "input_path": str(Path(path).resolve()), "file_sha256": raw_reader.sha(raw),
                "file_bytes": len(raw), "decoded_bytes": len(body), "error": str(exc),
                "decoded_coverage": coverage(w.cursor.spans, len(body)), "file_coverage": coverage(file_cursor.spans, len(raw)),
                "decoded_spans": w.cursor.spans, "file_spans": file_cursor.spans,
                "limits": "Partial attempt retained. Unsupported grammar is not filled by guessed object/tail boundaries."}


def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument("--input", required=True)
    p.add_argument("--output", required=True)
    args = p.parse_args()
    target = Path(args.output).resolve()
    source = Path(args.input).resolve()
    if target == source or (target.exists() and source.exists() and target.samefile(source)):
        p.exit(2, "byte atlas: output would overwrite input\n")
    try:
        result = atlas(args.input)
    except (raw_reader.BadSAV, OSError, UnicodeError, KeyError, struct.error) as exc:
        result = {"status": "refused", "input_path": args.input, "error": str(exc)}
        if Path(args.input).is_file():
            result["file_sha256"] = raw_reader.sha(Path(args.input).read_bytes())
    result["reader_sha256"] = raw_reader.sha(Path(__file__).read_bytes())
    result["grammar_reader_sha256"] = raw_reader.sha(Path(raw_reader.__file__).read_bytes())
    raw_reader.write_json(args.output, result)
    print(json.dumps({k: v for k, v in result.items() if k in {"status", "file_sha256", "observed_classes", "decoded_coverage", "file_coverage", "error", "world_present", "mission"}}, sort_keys=True))
    if result["status"] != "parsed":
        p.exit(2, "byte_atlas: refused; partial artifact retained\n")


if __name__ == "__main__":
    main()
