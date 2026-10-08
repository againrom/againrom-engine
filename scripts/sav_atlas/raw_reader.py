"""Bounded SAV byte/graph witness. Standard library only; no Document DTO.

Grammar: pinned knowledge/formats/sav/{encoding,document,objects,actors,
player,groups,world,items,token}.md at k194. Outputs are evidence, not a
claim that repeated RuntimeID or changed numeric Identity is a native fault.
"""

import argparse
import collections
import hashlib
import json
from pathlib import Path
import struct


ITEMS = {"Item", "Armor", "Shield", "Weapon"}
PLACEABLES = {"Unit", "Human", "Humanoid", "Building", "Outpost", "Tavern", "Shop", "Sack"}


def sha(b):
    return hashlib.sha256(b).hexdigest()


class BadSAV(ValueError):
    pass


def unpack_sav(raw):
    if len(raw) < 20 or raw[:4] != b"Asg&":
        raise BadSAV("not an Asg& envelope")
    end, version, size, words = struct.unpack_from("<4I", raw, 4)
    if end != 16 + size or not 20 <= end <= len(raw) or version != 0x0BAD0002:
        raise BadSAV(f"unsupported/invalid envelope end={end} size={size} version={version:x}")
    if words > 64 * 1024 * 1024:
        raise BadSAV("decoded allocation exceeds witness bound")
    src, body = 20, bytearray()
    while src < end:
        op = raw[src]
        src += 1
        n = op & 0x7f
        length = 2 if op & 0x80 else n * 2
        if src + length > end:
            raise BadSAV(f"truncated word packet at file offset {src-1}")
        packet = raw[src:src+length]
        src += length
        body.extend(packet * n if op & 0x80 else packet)
        if len(body) > 2 * words:
            raise BadSAV("word packets exceed declared output")
    if len(body) != 2 * words:
        raise BadSAV("word packets disagree with declared output")
    return bytes(body), end


def replace_body(raw, body):
    """Literal-only legal packets; preserves the opaque post-blob tail."""
    _, end = unpack_sav(raw)
    if len(body) % 2:
        body += b"\0"
    packets = bytearray(struct.pack("<I", len(body) // 2))
    for pos in range(0, len(body), 254):
        block = body[pos:pos+254]
        packets.append(len(block)//2)
        packets.extend(block)
    return b"Asg&" + struct.pack("<III", 16+len(packets), 0x0BAD0002, len(packets)) + packets + raw[end:]


class Walker:
    def __init__(self, body):
        self.body, self.pos, self.next_index = body, 0, 1
        self.classes, self.records, self.order = {}, {}, []
        self.roots, self.cells = {}, []

    def fail(self, s):
        raise BadSAV(f"decoded offset {self.pos}: {s}")

    def take(self, n):
        if not 0 <= n <= len(self.body)-self.pos:
            self.fail(f"truncated span {n}")
        value = self.body[self.pos:self.pos+n]
        self.pos += n
        return value

    def number(self, fmt):
        return struct.unpack("<"+fmt, self.take(struct.calcsize("<"+fmt)))[0]

    def count(self):
        n = self.number("H")
        n = self.number("I") if n == 0xffff else n
        if n > 1000000:
            self.fail(f"count exceeds witness bound: {n}")
        return n

    def string(self):
        n = self.number("B")
        if n == 0xff:
            n = self.number("H")
            if n == 0xfffe:
                self.fail("Unicode CString outside this witness's supported grammar")
            if n == 0xffff:
                n = self.number("I")
        return self.take(n).decode("latin1")

    def wordlist(self):
        return self.take(2*self.count()).hex()

    def field(self, r, name, fmt=None, size=None):
        at = self.pos
        value = self.take(size).hex() if size is not None else self.number(fmt)
        r["values"][name] = value
        r["offsets"][name] = at
        return value

    def refs(self, r, name, n):
        if n > 1000000:
            self.fail(f"reference count exceeds witness bound: {n}")
        values, offsets = [], []
        for _ in range(n):
            offsets.append(self.pos)
            values.append(self.obj())
        r["refs"][name] = values
        r.setdefault("ref_offsets", {})[name] = offsets
        return values

    def list32(self, r, name):
        r.setdefault("list_count_offsets", {})[name] = self.pos
        return self.refs(r, name, self.number("I"))

    def container(self, r, name):
        self.list32(r, name)
        self.field(r, name+".InsertIndex", "I")
        self.field(r, name+".Load", "I")

    def diary(self):
        self.take(4*self.count())
        self.take(2*self.count())
        self.take(4)

    def token(self, r):
        self.field(r, "Position", size=12)
        for name, fmt in [("RuntimeID", "I"), ("Selector", "B"), ("Type", "H"),
                          ("Flags", "I"), ("PublicationMask", "H"), ("Price", "i"),
                          ("Identity", "I"), ("Reference", "I")]:
            self.field(r, name, fmt)

    def obj(self):
        tag_at, tag = self.pos, self.number("H")
        if tag == 0:
            return 0
        if tag == 0x7fff:
            self.fail("32-bit CArchive tag outside this witness's supported grammar")
        if tag == 0xffff:
            schema, length = self.number("H"), self.number("H")
            name = self.take(length).decode("ascii")
            if schema != 1:
                self.fail(f"unsupported schema {schema} for {name}")
            self.classes[self.next_index] = name
            self.next_index += 1
        elif tag & 0x8000:
            name = self.classes.get(tag & 0x7fff)
            if name is None:
                self.fail(f"unknown class index {tag & 0x7fff}")
        else:
            if tag not in self.records:
                self.fail(f"unallocated back-reference {tag}")
            return tag
        index = self.next_index
        self.next_index += 1
        r = {"archive_index": index, "ordinal": len(self.order)+1, "class": name,
             "tag_offset": tag_at, "body_offset": self.pos, "values": {}, "offsets": {}, "refs": {}}
        self.records[index] = r
        self.order.append(r)
        self.program(r)
        r["end_offset"] = self.pos
        return index

    def program(self, r):
        cls = r["class"]
        if cls == "Player":
            r["values"]["Name"] = self.string()
            self.field(r, "Prefix", size=51)
            r["values"]["Identity"] = struct.unpack_from("<I", bytes.fromhex(r["values"]["Prefix"]), 47)[0]
            n = self.number("I")
            for g in range(n):
                self.wordlist()
                self.take(80)
                self.wordlist()
                self.list32(r, f"Group[{g}].Actors")
                self.take(12)
            self.take(32)
            self.diary()
            return
        if cls == "Diary":
            self.diary()
            return
        if cls == "Spell":
            for name, fmt in [("ID", "B"), ("Range", "B"), ("Defensive", "B"), ("ManaCost", "H"), ("Identity", "I")]:
                self.field(r, name, fmt)
            return
        token_classes = ITEMS | PLACEABLES | {"Token", "Effect", "Effect_DirectDamage", "SpellEffect", "PointEffect", "AreaEffect", "SpellTransport", "VirtualCaster"}
        if cls not in token_classes:
            self.fail(f"unsupported class {cls}")
        self.token(r)
        if cls in ITEMS:
            self.list32(r, "Effects")
            for name, fmt in [("Code", "H"), ("Count", "H"), ("Kind", "B"), ("Shape", "B"),
                              ("Material", "B"), ("F48", "H"), ("Weight", "h"), ("F47", "B")]:
                self.field(r, name, fmt)
            if cls == "Weapon":
                self.field(r, "Attack", size=24)
                self.field(r, "Defence", size=22)
                self.field(r, "OwnKind", "B")
                self.refs(r, "OwnedSpell", 1)
            elif cls in {"Armor", "Shield"}:
                self.field(r, "Defence", size=22)
                if cls == "Armor":
                    self.field(r, "OwnKind", "B")
        elif cls == "Sack":
            self.field(r, "Gold", "I")
            self.container(r, "Contents")
        elif cls in {"Building", "Outpost", "Tavern", "Shop"}:
            self.take(40)
            if cls == "Outpost":
                self.take(16)
                self.take(8*self.count())
            elif cls in {"Tavern", "Shop"}:
                self.take(4)
        elif cls in {"Effect", "Effect_DirectDamage"}:
            for name, fmt in [("EffectKind", "B"), ("EffectMode", "B"), ("Operand", "I"), ("EffectID", "B")]:
                self.field(r, name, fmt)
            if cls == "Effect_DirectDamage":
                self.field(r, "Damage", size=24)
        elif cls in {"SpellEffect", "PointEffect", "AreaEffect", "SpellTransport"}:
            self.take(2)
            if cls == "PointEffect":
                self.refs(r, "Effect", 1)
                self.take(4)
            elif cls == "AreaEffect":
                self.take(6)
                self.refs(r, "Effect", 1)
            elif cls == "SpellTransport":
                self.refs(r, "Child", 1)
                self.refs(r, "Area", 1)
                self.take(2)
        elif cls == "VirtualCaster":
            self.take(7)
        elif cls in {"Unit", "Humanoid", "Human"}:
            self.list32(r, "ActorEffects")
            self.wordlist()
            self.wordlist()
            self.take(24+22+24+64+180+148)
            self.wordlist()
            self.take(4+12+3)
            self.refs(r, "HeldWeapon", 1)
            self.refs(r, "HeldShield", 1)
            r["values"]["Name"] = self.string()
            self.field(r, "Pools", size=28)
            self.take(2+4+8)
            self.field(r, "ActionClock", "I")
            self.field(r, "Stage", "B")
            self.take(8)
            self.refs(r, "Extra68", 1)
            if self.number("B"):
                self.container(r, "Pack")
            if self.number("B"):
                self.take(4)
                n = self.number("I")
                if n > 1000000:
                    self.fail("Spellbook exceeds witness bound")
                self.refs(r, "Book", max(n-1, 0))
            self.take(17)
            if cls in {"Humanoid", "Human"}:
                self.take(24)
                self.refs(r, "Worn", 12)
                self.refs(r, "HumanDiary", 1)

    def parse(self):
        self.take(8)
        self.map_name = self.string()
        self.take(44)
        self.mission = self.number("I")
        self.take(8)
        for name in ["Players", "Dead"]:
            self.roots[name] = [self.obj() for _ in range(self.number("I"))]
        self.world = self.number("B")
        if self.world:
            for name in ["Buildings", "SpellEffects"]:
                self.roots[name] = [self.obj() for _ in range(self.number("I"))]
            self.take(4*self.number("H"))
            for _ in range(self.count()):
                cell = self.number("H")
                payload = self.take(52)
                self.cells.append({"cell": [cell & 255, cell >> 8], "payload": payload.hex(),
                                   "Ground": struct.unpack_from("<I", payload, 4)[0],
                                   "Air": struct.unpack_from("<I", payload, 8)[0],
                                   "Building": struct.unpack_from("<I", payload, 12)[0],
                                   "Sack": struct.unpack_from("<I", payload, 16)[0]})
            self.terrain_identity = self.number("I")
            self.take(4374)
            self.roots["Sacks"] = [self.obj() for _ in range(self.number("I"))]
        self.marker = self.number("I")
        if self.marker == 0xBADFACE1:
            self.take(4)
        self.take(400)
        remaining = len(self.body)-self.pos
        if remaining != (self.pos % 2):
            self.fail(f"unexpected decoded tail length {remaining}")
        return self



def write_json(path, data):
    target = Path(path)
    target.parent.mkdir(parents=True, exist_ok=True)
    target.write_text(json.dumps(data, indent=2, sort_keys=True)+'\n', encoding='utf8')
