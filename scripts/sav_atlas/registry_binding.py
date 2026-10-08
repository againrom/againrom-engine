"""Bounded registry literal checks over exact SAV and current capture bytes.

Typed continuation correspondence is a proposal, not independent identity
or ownership authority. Direct leaves require independently keyed current nodes.
"""

import collections
import functools
import json
import re
import struct
from pathlib import Path

import byte_layout
import raw_reader as raw
import typed_payload
from source_binding import BindingError

PRODUCER_PATH = "pkg/game/savitemobjects_project.go"
PRODUCER_SHA256 = "7d919ec0f7093d83adeb1bec497f1f3458070f5ea8067908da4c90f98a2f5a47"
MAX_OBJECTS = 32767
COUNT_PROJECTION_PINS = {
    "pkg/game/savcurrentitems.go": "a15f3a9d92e1f53d15958f1098a87ce4625eac105f50b26cdf14309e3188967f",
    "pkg/game/savitemobjects_scroll.go": "95a23ba32698d8af8715bc5f06ed7879eb964fd1abf5bb6b02355ae004f63650",
}
CLASS_PROJECTION_PINS = {
    "pkg/mapload/sourceconstructor.go": "abab21d3ce1bbf963ef0ca9bd7c37fd44e12547b3ac04d34c26bae225d57b801",
    "pkg/mapload/sourceactor.go": "d196ad6cd642770c0ba6029fbe2c59d92e35833105b5e6ac4f383abb47c9fe85",
    "pkg/data/itemcode.go": "bfbc6e050a3f0ac583d10735ef7c24c70e88191581a362569d169d1a366b044a",
    "pkg/sim/sourceequipment.go": "89e479b587e473b28ad650ee39cf2404eca911ab2a7145efb5a353cc669038db",
    "pkg/sim/weight.go": "2788bd35a84e4767ed645e45a6983c217be6aa8aff9fc5ac1d25d24cac8db134",
    "pkg/sim/item.go": "44ba45638e89d3dbef6a2152a1b603eff67d4f96c2b8b165475224b2fa0867d7",
    "pkg/sim/savedobjectroots.go": "9b53211432aaab10751c2b73584bc417fc3be14f5b4a7ce4d1eb248597a49776",
}
CODE_PROJECTION_PINS = {
    "pkg/game/currentsave.go": "07f8cbe4794b040a0f10b74df7ad5ed2b1556f5f786790bf28c0912570f3e8ef",
    "pkg/game/modmark.go": "a9c7ec58b66a233747e48f35459d1063dc0107d31194d0f30974a60f86c45736",
    "pkg/game/mods.go": "224a500a0fa1aa5919f2aaa9952506bbc906244c220cac475b701acbbb2c8c04",
    "pkg/mod/digest.go": "4b1713f336e4e92f11092a8d5a58d462bbc4edec144d9d9e8a038a5769d1d379",
    "pkg/mapload/spawn.go": "7afde47ae3d2c6e764862e5562df73d9bf1881ce0b456bb0b4f2dbf34829b5a7",
}
CODE_STAND_IN_CALLSITE = {
    "path": "pkg\\game\\modmark.go", "line": 370,
    "sha256": CODE_PROJECTION_PINS["pkg/game/modmark.go"],
    "signature": "func standInModItems(doc *sav.DocumentData, items []mapload.ModItem) ([]modMarkItem, error) {",
}
BOOK_PROJECTION_PINS = {
    "pkg/game/savbookcurrent.go": "4183d864396bb3f0abed84059c4d582bc5ea421e3e6421d9361bf392e56c3b56",
    "pkg/game/savitemobjects.go": "eb4d059754baa3e1e37d74070d50ce9e96ce26a740d3f28530ad5e8467052058",
}
DIRECT_FIELD_MASKS = {
    'Items': {'Code': 0, 'Count': 16384, 'F47': 2048, 'F48': 4096, 'Flags': 16, 'Kind': 0,
              'Material': 1024, 'Position': 1, 'Price': 64, 'PublicationMask': 32,
              'Shape': 512, 'Type': 8},
    'Effects': {'EffectID': 0, 'EffectKind': 0, 'EffectMode': 0, 'Flags': 16,
                'Operand': 0, 'Position': 1, 'Price': 64, 'PublicationMask': 32, 'Type': 8},
    'Spells': {'ManaCost': 8192, 'ID': 8192, 'Defensive': 8192},
}

LITERAL_FIELDS = (
    ('Effects', 'EffectID', 'E0C', 1, 'u8', 'E0C', ('Effect',),
     ((332, 'Effect', (0, 1)),)),
    ('Effects', 'EffectKind', 'Value/Kind', 1, 'u8', 'E3C', ('Effect',),
     ((329, 'Effect', (0, 1)),)),
    ('Effects', 'EffectMode', 'Value/Mode', 1, 'u8', 'E3D', ('Effect',),
     ((330, 'Effect', (0, 1)),)),
    ('Effects', 'Flags', 'Token/T08', 4, 'u32', 'T08', ('Effect',),
     ((192, 'Effect', (0, 2)), (193, 'Effect', (2, 4)))),
    ('Effects', 'Operand', 'Value/Operand', 4, 'u32', 'E40', ('Effect',),
     ((331, 'Effect', (0, 4)),)),
    ('Effects', 'Position', 'Token/Position', 12, 'byte-array', 'Block12', ('Effect',),
     ((184, 'Effect', (0, 2)), (185, 'Effect', (2, 4)), (186, 'Effect', (4, 6)), (187, 'Effect', (6, 8)), (188, 'Effect', (8, 12)))),
    ('Effects', 'Price', 'Token/T1C', 4, 'u32', 'T1C', ('Effect',),
     ((195, 'Effect', (0, 4)),)),
    ('Effects', 'PublicationMask', 'Token/T18', 2, 'u16', 'T18', ('Effect',),
     ((194, 'Effect', (0, 2)),)),
    ('Effects', 'Type', 'Token/T0E', 2, 'u16', 'T0E', ('Effect',),
     ((191, 'Effect', (0, 2)),)),
    ('Items', 'Code', 'Value/Code', 2, 'u16', 'F40', ('Item', 'Weapon', 'Armor', 'Shield'),
     ((284, 'Item', (0, 2)), (294, 'Shield', (0, 2)), (305, 'Armor', (0, 2)), (317, 'Weapon', (0, 2)))),
    ('Items', 'Count', 'Value/Count', 2, 'u16', 'F42', ('Item', 'Weapon', 'Armor', 'Shield'),
     ((285, 'Item', (0, 2)), (295, 'Shield', (0, 2)), (306, 'Armor', (0, 2)), (318, 'Weapon', (0, 2)))),
    ('Items', 'F47', 'F47', 1, 'u8', 'F47', ('Item', 'Weapon', 'Armor', 'Shield'),
     ((291, 'Item', (0, 1)), (301, 'Shield', (0, 1)), (312, 'Armor', (0, 1)), (324, 'Weapon', (0, 1)))),
    ('Items', 'F48', 'F48', 2, 'u16', 'F48', ('Item', 'Weapon', 'Armor', 'Shield'),
     ((289, 'Item', (0, 2)), (299, 'Shield', (0, 2)), (310, 'Armor', (0, 2)), (322, 'Weapon', (0, 2)))),
    ('Items', 'Flags', 'Token/T08', 4, 'u32', 'T08', ('Item', 'Weapon', 'Armor', 'Shield'),
     ((136, 'Item', (0, 2)), (137, 'Item', (2, 4)), (150, 'Shield', (0, 2)), (151, 'Shield', (2, 4)), (164, 'Armor', (0, 2)), (165, 'Armor', (2, 4)), (178, 'Weapon', (0, 2)), (179, 'Weapon', (2, 4)))),
    ('Items', 'Kind', 'Value/Kind', 1, 'u8', 'F44', ('Item', 'Weapon', 'Armor', 'Shield'),
     ((286, 'Item', (0, 1)), (296, 'Shield', (0, 1)), (307, 'Armor', (0, 1)), (319, 'Weapon', (0, 1)))),
    ('Items', 'Material', 'F46', 1, 'u8', 'F46', ('Item', 'Weapon', 'Armor', 'Shield'),
     ((288, 'Item', (0, 1)), (298, 'Shield', (0, 1)), (309, 'Armor', (0, 1)), (321, 'Weapon', (0, 1)))),
    ('Items', 'Position', 'Token/Position', 12, 'byte-array', 'Block12', ('Item', 'Weapon', 'Armor', 'Shield'),
     ((128, 'Item', (0, 2)), (129, 'Item', (2, 4)), (130, 'Item', (4, 6)), (131, 'Item', (6, 8)), (132, 'Item', (8, 12)), (142, 'Shield', (0, 2)), (143, 'Shield', (2, 4)), (144, 'Shield', (4, 6)), (145, 'Shield', (6, 8)), (146, 'Shield', (8, 12)), (156, 'Armor', (0, 2)), (157, 'Armor', (2, 4)), (158, 'Armor', (4, 6)), (159, 'Armor', (6, 8)), (160, 'Armor', (8, 12)), (170, 'Weapon', (0, 2)), (171, 'Weapon', (2, 4)), (172, 'Weapon', (4, 6)), (173, 'Weapon', (6, 8)), (174, 'Weapon', (8, 12)))),
    ('Items', 'Price', 'Token/T1C', 4, 'u32', 'T1C', ('Item', 'Weapon', 'Armor', 'Shield'),
     ((139, 'Item', (0, 4)), (153, 'Shield', (0, 4)), (167, 'Armor', (0, 4)), (181, 'Weapon', (0, 4)))),
    ('Items', 'PublicationMask', 'Token/T18', 2, 'u16', 'T18', ('Item', 'Weapon', 'Armor', 'Shield'),
     ((138, 'Item', (0, 2)), (152, 'Shield', (0, 2)), (166, 'Armor', (0, 2)), (180, 'Weapon', (0, 2)))),
    ('Items', 'Shape', 'F45', 1, 'u8', 'F45', ('Item', 'Weapon', 'Armor', 'Shield'),
     ((287, 'Item', (0, 1)), (297, 'Shield', (0, 1)), (308, 'Armor', (0, 1)), (320, 'Weapon', (0, 1)))),
    ('Items', 'Type', 'Token/T0E', 2, 'u16', 'T0E', ('Item', 'Weapon', 'Armor', 'Shield'),
     ((135, 'Item', (0, 2)), (149, 'Shield', (0, 2)), (163, 'Armor', (0, 2)), (177, 'Weapon', (0, 2)))),
    ('Spells', 'Defensive', 'Value/Defensive', 1, 'u8', 'S0A', ('Spell',),
     ((772, 'Spell', (0, 1)),)),
    ('Spells', 'ID', 'Value/ID', 1, 'u8', 'S08', ('Spell',),
     ((770, 'Spell', (0, 1)),)),
    ('Spells', 'ManaCost', 'Value/ManaCost', 2, 'u16', 'S0C', ('Spell',),
     ((773, 'Spell', (0, 2)),)),
)


def literal_specs():
    bindings = {}
    callsites = {
        'Items': (1, 95, 'func savedCurrentItemRecord(row sim.SavedItemObject, indices map[sim.SavedObjectID]uint16) (sav.DocumentRecordData, error) {'),
        'Effects': (2, 144, 'func savedCurrentEffectRecord(row sim.SavedEffectObject) sav.DocumentRecordData {'),
        'Spells': (3, 174, 'func savedCurrentSpellRecord(row sim.SavedSpellObject) sav.DocumentRecordData {'),
    }
    for collection, field, value_path, width, encoding, wire, classes, rows in LITERAL_FIELDS:
        kind, line, signature = callsites[collection]
        metadata = {str(index): {"class_name": cls, "intervals": [list(interval)],
            "selector": cls+r"\[archive=\d+\]\."+field+"$", "wire_member": wire}
            for index, cls, interval in rows}
        bindings["World.SavedObjects."+collection+"."+field] = {
            "collection": collection, "field": field, "value_path": value_path, "width": width,
            "encoding": encoding, "wire_member": wire, "classes": list(classes), "kind": kind,
            "catalog_rows": [index for index, _, _ in rows], "catalog_metadata": metadata,
            "producer_callsite": {"path": PRODUCER_PATH.replace("/", "\\"), "line": line,
                "sha256": PRODUCER_SHA256, "signature": signature}}
    return {"bindings": bindings, "family_bindings": {
        "keyed-"+spec["collection"]+"-"+spec["field"]: binding for binding, spec in bindings.items()}}


SPECS = literal_specs()


def digest(data):return raw.sha(data)
def at(root,path):
    for key in path.strip('/').split('/') if path else []:
        root=root[int(key)] if isinstance(root,list) else root[key.replace('~1','/').replace('~0','~')]
    return root
def need(test,message):
    if not test:raise BindingError(message)
def fact(path,value=None,op='eq'):return dict(capture='writer',path=path,op=op,**({'value':value} if op!='present' else {}))
def unknown(reason):return dict(verified=False,reason=reason,complete=False)

def canonical_records(w):
    by_index, visited, active = {}, {}, set()
    names = {"ActorEffects": "Effects", "HumanDiary": "Diary", "Extra68": "U68",
             "Pack": "Inventory", "Book": "Spells", "OwnedSpell": "WeaponSpell"}

    def visit(ref, depth=0):
        need(type(ref) is int and 0 <= ref < 32768, "ordinary reference width differs")
        if not ref:
            return
        need(ref not in active and depth <= 128, "ordinary graph is cyclic or exceeds depth bound")
        if ref in visited:
            return
        need(ref in w.records and len(visited) < MAX_OBJECTS, "ordinary reference is missing or exceeds object bound")
        record = w.records[ref]
        index = len(visited)+1
        visited[ref], by_index[index] = index, record
        active.add(ref)
        mapped = dict(names)
        if record["class"] == "AreaEffect":
            mapped["Effect"] = "AE44"
        elif record["class"] == "PointEffect":
            mapped["Effect"] = "PE48"
        elif record["class"] == "SpellTransport":
            mapped.update({"Child": "ST44", "Area": "ST48"})
        sites = sorted(record["refs"], key=(lambda k: int(k.split("[")[1].split("]")[0]))
                       if record["class"] == "Player" else lambda k: mapped.get(k, k))
        for site in sites:
            for child in record["refs"][site]:
                visit(child, depth+1)
        active.remove(ref)

    for root in ["Players", "Dead", "Buildings", "SpellEffects", "Sacks"]:
        for ref in w.roots.get(root, []):
            visit(ref)
    need(len(visited) == len(w.order), "canonical correspondence does not cover ordinary graph")
    return by_index


def incoming(w):
    owners = collections.defaultdict(list)
    for record in w.order:
        for field, refs in record["refs"].items():
            for slot, ref in enumerate(refs):
                if ref:
                    owners[ref].append(dict(owner_archive=record["archive_index"],
                        owner_ordinal=record["ordinal"], owner_class=record["class"],
                        owner_identity=record["values"].get("Identity"), field=field, slot=slot))
    return owners


def typed_correspondence(saved, body):
    cursor, _ = byte_layout.file_codec(saved, body)
    tail = byte_layout.tail_atlas(cursor)
    records, paths = tail["ya1_records"], {}

    def visit(index, parent, active):
        need(type(index) is int and 0 <= index < len(records) and index not in active
             and len(active) <= 64, "YA1 path graph is cyclic or outside bound")
        need(index not in paths, "YA1 record has alias owners")
        record = records[index]
        paths[index] = parent+"/"+record["name"]
        if record["kind"] & 1:
            for child in range(record["data"], record["data"]+record["size"]):
                visit(child, paths[index], active | {index})

    for index in range(tail["root_first"], tail["root_first"]+tail["root_count"]):
        visit(index, "", set())
    indices = [index for index, path in paths.items() if path == "/CurrentState/AgainromActions"]
    need(len(indices) <= 1, "typed action path is duplicated")
    if not indices:
        return None
    index = indices[0]
    label = f"YA1.Record[{index}].AgainromActions.PoolValue"
    need(not any(span.get("aliases") for span in cursor.spans
                 if label in [span["label"]]+span.get("aliases", [])), "typed action payload has physical aliases")
    value, _, _ = typed_payload.json_value({"tail": tail, "file_spans": cursor.spans}, index)
    return value


def strict_json(blob):
    def pairs(items):
        result = {}
        for key, value in items:
            need(key not in result, "capture has duplicate JSON members")
            result[key] = value
        return result
    def constant(value):
        raise BindingError("capture has nonfinite JSON number")
    value = json.loads(blob.decode("utf-8-sig"), object_pairs_hook=pairs, parse_constant=constant)
    list(typed_payload.leaves(value))
    return value


@functools.lru_cache(maxsize=4)
def context(saved, writer_bytes, source_pin, knowledge_pin):
    need(isinstance(saved, bytes) and isinstance(writer_bytes, bytes), "binding context has no exact input bytes")
    need(len(saved) < 16 << 20 and len(writer_bytes) <= 64 << 20, "binding input exceeds byte bound")
    need(re.fullmatch(r"[0-9a-f]{40}", source_pin or "") and
         re.fullmatch(r"[0-9a-f]{40}", knowledge_pin or ""), "binding lacks independent expected full pins")
    writer = strict_json(writer_bytes)
    need(isinstance(writer, dict) and writer.get("Revision") == source_pin and
         writer.get("KnowledgePin") == knowledge_pin, "actual capture differs from independent expected pins")
    producer = Path(__file__).resolve().parents[2]/PRODUCER_PATH
    need(digest(producer.read_bytes()) == PRODUCER_SHA256, "current literal producer code changed")
    body, _ = raw.unpack_sav(saved)
    native = raw.Walker(body).parse()
    canonical = canonical_records(native)
    secondary = typed_correspondence(saved, body)
    ownership, actors = {}, {}
    if secondary is not None:
        need(isinstance(secondary.get("Ownership", []), list) and
             isinstance(secondary.get("Bindings", []), list), "typed correspondence population is malformed")
        for row in secondary.get("Ownership", []):
            need(isinstance(row, dict) and type(row.get("ID")) is int and 0 <= row["ID"] < 1 << 64 and
                 type(row.get("Kind")) is int and 1 <= row["Kind"] <= 4 and
                 type(row.get("Object")) is int and 0 <= row["Object"] < 32768,
                 "typed ownership row has corrupt widths")
            if row["ID"]:
                key = row["Kind"], row["ID"]
                need(key not in ownership, "typed correspondence repeats object identity")
                ownership[key] = row["Object"]
        for row in secondary.get("Bindings", []):
            need(isinstance(row, dict) and type(row.get("ID")) is int and 0 <= row["ID"] < 1 << 32 and
                 type(row.get("Object")) is int and 0 <= row["Object"] < 32768 and
                 type(row.get("Structure")) is bool and type(row.get("Missing")) is bool,
                 "typed actor row has corrupt widths")
            if not row["Structure"] and not row["Missing"]:
                need(row["ID"] not in actors, "typed correspondence repeats actor")
                actors[row["ID"]] = row["Object"]
    annotated = byte_layout.AtlasWalker(body)
    annotated.parse()
    need([(r["archive_index"], r["class"], r["body_offset"], r["end_offset"]) for r in annotated.order] ==
         [(r["archive_index"], r["class"], r["body_offset"], r["end_offset"]) for r in native.order],
         "annotated grammar differs from raw object extents")
    return dict(saved_sha256=digest(saved), capture_sha256=digest(writer_bytes), writer=writer,
        body=body, native=native, canonical=canonical, ownership=ownership, actors=actors,
        incoming=incoming(native), secondary_present=secondary is not None,
        source_pin=source_pin, knowledge_pin=knowledge_pin,
        decoded_spans={span["label"]: span for span in annotated.cursor.spans})


def actual_context(binding_context):
    need(isinstance(binding_context, dict), "registry binding has no independent byte/pin context")
    capture_bytes = binding_context.get("capture_bytes", {})
    need(isinstance(capture_bytes, dict) and "writer" in capture_bytes,
         "registry binding has no actual raw writer capture bytes")
    return context(binding_context.get("saved_bytes"), capture_bytes["writer"],
        binding_context.get("source_pin"), binding_context.get("knowledge_pin"))


def row_for(objects,collection,identity):
    rows=[(n,row) for n,row in enumerate(objects.get(collection) or []) if row['ID']==identity]
    need(len(rows)==1,'current registry ID is missing or ambiguous');return rows[0]

def locations(objects,identity):
    result=[]
    for index,c in enumerate(objects.get('Containers') or []):
        for slot,value in enumerate(c['Items'] or []):
            if value==identity and identity!=0:
                result.append(dict(owner=c['Owner'],slot=slot,path=f'/Capture/Objects/Containers/{index}/Items/{slot}',container=index))
    for index,root in enumerate(objects.get('ItemRoots') or []):
        if root['ID']==identity:
            result.append(dict(owner=root['Owner'],slot=0,path=f'/Capture/Objects/ItemRoots/{index}/ID',root=index))
    return result

def mapped(ctx,kind,identity):
    object_index=ctx['ownership'].get((kind,identity),0)
    need(object_index in ctx['canonical'],'typed canonical relation has no ordinary record')
    return ctx['canonical'][object_index]

def actor(ctx,identity):
    index=ctx['actors'].get(identity,0);need(index in ctx['canonical'],'directed holder has no mapped actor')
    record=ctx['canonical'][index];need(record['class'] in {'Unit','Humanoid','Human'},'holder maps to a non-actor');return record

def item_relation(ctx,objects,identity,record):
    index,current=row_for(objects,'Items',identity);ptr=f'/Capture/Objects/Items/{index}'
    need(current['Retired'] is False and current['InFlight']==0,'retired/in-flight Item cannot select current ordinary arm')
    ordinary=[v for v in locations(objects,identity) if v['owner']['Kind']!=4]
    need(len(locations(objects,identity))<=1,'Item has unsupported alias owners')
    if len(ordinary)!=1:return None,'Item requires one supported ordinary owner; session-only branch is unresolved'
    loc=ordinary[0];owner=loc['owner'];guards=[fact(ptr+'/ID',identity),fact(ptr+'/Retired',False),fact(ptr+'/InFlight',0),fact(loc['path'],identity)]
    location_prefix=loc['path'].rsplit('/Items/',1)[0] if 'container' in loc else loc['path'].rsplit('/ID',1)[0]
    guards.append(fact(location_prefix+'/Owner',owner))
    if owner['Kind']==1:
        parent=actor(ctx,owner['Entity']);field,slot='Pack',loc['slot']
        need(objects['Containers'][loc['container']]['Present'] is True,'ordinary Pack container is absent')
        guards.append(fact(f"/Capture/Objects/Containers/{loc['container']}/Present",True))
    elif owner['Kind']==3:
        sack_index,sack=row_for(objects,'Sacks',owner['Object']);need(not sack['Retired'],'ordinary Sack holder is retired')
        parent=mapped(ctx,4,owner['Object']);field,slot='Contents',loc['slot']
        guards.extend([fact(f'/Capture/Objects/Sacks/{sack_index}/Retired',False),fact(f"/Capture/Objects/Containers/{loc['container']}/Present",True)])
    elif owner['Kind']==2:
        parent=actor(ctx,owner['Entity']);slot=owner['Slot']
        need(type(slot) is int and 1<=slot<=12,'current worn slot is unsupported')
        field,slot=('HeldWeapon',0) if slot==1 else (('HeldShield',0) if slot==2 else ('Worn',slot-1))
        guards.append(fact(f"/Capture/Objects/ItemRoots/{loc['root']}/Owner",owner))
    else:return None,'ordinary owner kind is not implemented'
    refs=parent['refs'].get(field,[]);need(slot<len(refs) and refs[slot]==record['archive_index'],'current ordinary holder/slot differs from raw edge')
    incoming=ctx['incoming'].get(record['archive_index'],[])
    need(len(incoming)==1 and incoming[0]['owner_archive']==parent['archive_index'] and incoming[0]['field']==field and incoming[0]['slot']==slot,'directed owner is ambiguous or inconsistent')
    return dict(guards=guards,owner_archive=parent['archive_index'],field=field,slot=slot,held=owner['Kind']==2,location=loc),None

def expected_item_class(capture,item,held):
    source=item['Value']['SourceEquipment']['Class']
    need(type(source) is int and 0<=source<=3,'current equipment class is invalid')
    if source==0 and (held or not item['Value']['WeightPresent']):
        need('ItemWeights' in capture,'current class fallback lacks captured ItemWeights presence; missing is not empty')
        weights=[(n,v) for n,v in enumerate(capture.get('ItemWeights') or []) if v['Code']==item['Value']['Code']]
        need(len(weights)<=1,'current effective weight constructor is ambiguous')
        if weights and weights[0][1]['Constructor']['Class']:
            source=weights[0][1]['Constructor']['Class']
            need(type(source) is int and 1<=source<=3,'current constructor class width differs')
            return {1:'Weapon',2:'Armor',3:'Shield'}.get(source),fact(f'/Capture/ItemWeights/{weights[0][0]}/Constructor/Class',source)
        # Constructor.class and the explicit fallback use the same code B arm.
        b=(item['Value']['Code']>>8)&15
        source=1 if b==1 else (3 if b==2 else (2 if 3<=b<=12 else 0))
    return {0:'Item',1:'Weapon',2:'Armor',3:'Shield'}.get(source),None

def child_relation(ctx,objects,collection,identity,record):
    candidates=[]
    for item_index,item in enumerate(objects.get('Items') or []):
        if item['Retired']:continue
        for slot,ref in enumerate(item['Effects'] or []) if collection=='Effects' else []:
            if ref==identity:candidates.append(('item',item_index,slot,'Effects'))
        if collection=='Spells' and item['Spell']==identity:candidates.append(('item',item_index,0,'OwnedSpell'))
    if collection=='Spells':
        for book_index,book in enumerate(objects.get('BookRoots') or []):
            for slot,ref in enumerate(book['Slots']):
                if ref==identity:candidates.append(('book',book_index,slot,'Book'))
    need(len(candidates)<=1,'needed child has unsupported alias parents')
    if len(candidates)!=1:return None,'needed child has no supported directed parent'
    kind,index,slot,field=candidates[0]
    if kind=='item':
        parent_item=objects['Items'][index];parent=mapped(ctx,1,parent_item['ID'])
        relation,error=item_relation(ctx,objects,parent_item['ID'],parent)
        if relation is None:return None,error
        guards=relation['guards']+[fact(f'/Capture/Objects/Items/{index}/'+('Effects/'+str(slot) if collection=='Effects' else 'Spell'),identity)]
    else:
        book=objects['BookRoots'][index];parent=actor(ctx,book['Entity'])
        guards=[fact(f'/Capture/Objects/BookRoots/{index}/Entity',book['Entity']),fact(f'/Capture/Objects/BookRoots/{index}/Slots/{slot}',identity)]
    refs=parent['refs'].get(field,[]);need(slot<len(refs) and refs[slot]==record['archive_index'],'needed child directed raw edge differs')
    incoming=ctx['incoming'].get(record['archive_index'],[])
    need(len(incoming)==1 and incoming[0]['owner_archive']==parent['archive_index'] and incoming[0]['field']==field and incoming[0]['slot']==slot,'child has unsupported directed alias owners')
    return dict(guards=guards,owner_archive=parent['archive_index'],field=field,slot=slot,needed=True),None


def weight_constructor_class(constructor):
    names = ('Class', 'DefinitionRow', 'OwnKind', 'Attack', 'Defence', 'Definition', 'Spell', 'EffectsUnsupported')
    if not isinstance(constructor, dict) or any(name not in constructor for name in names):
        return None
    for name in ('Class', 'DefinitionRow', 'OwnKind'):
        need(type(constructor[name]) is int and 0 <= constructor[name] < 256,
             'current weight constructor scalar width differs')
    clazz = constructor['Class']
    need(clazz <= 3, 'current weight constructor class differs')
    for name, width in (('Attack', 24), ('Defence', 22)):
        need(isinstance(constructor[name], list) and len(constructor[name]) == width and
             all(type(byte) is int and 0 <= byte < 256 for byte in constructor[name]),
             'current weight constructor block width differs')
    definition, spell = constructor['Definition'], constructor['Spell']
    definition_fields = ('AttackType', 'Hands', 'Charge', 'Relax', 'Suitable')
    spell_fields = ('ID', 'Range', 'Defensive', 'ManaCost')
    if not isinstance(definition, dict) or not isinstance(spell, dict) or \
       any(name not in definition for name in ('Present',)+definition_fields) or \
       any(name not in spell for name in ('Present',)+spell_fields):
        return None
    need(type(definition['Present']) is bool and type(spell['Present']) is bool and
         type(constructor['EffectsUnsupported']) is bool, 'current weight constructor presence width differs')
    need(all(type(definition[name]) is int and -(1 << 31) <= definition[name] < 1 << 31
             for name in definition_fields), 'current weight constructor definition width differs')
    need(all(type(spell[name]) is int and 0 <= spell[name] < (1 << (16 if name == 'ManaCost' else 8))
             for name in spell_fields), 'current weight constructor Spell width differs')
    definition_empty = not definition['Present'] and not any(definition[name] for name in definition_fields)
    spell_empty = not spell['Present'] and not any(spell[name] for name in spell_fields)
    need(spell_empty and constructor['EffectsUnsupported'] is False,
         'current weight constructor carries instance-only state')
    need((definition['Present'] or definition_empty) and (clazz == 1 or definition_empty),
         'current weight constructor definition residue differs')
    need((clazz == 1 or not any(constructor['Attack'])) and (clazz != 3 or constructor['OwnKind'] == 0),
         'current weight constructor carries another class members')
    need(clazz != 0 or not any((constructor['DefinitionRow'], constructor['OwnKind'],
         *constructor['Attack'], *constructor['Defence'])) and definition_empty,
         'current zero weight constructor has residue')
    return clazz


def direct_item_class(ctx, value, held, ptr):
    source = value['SourceEquipment']['Class']
    need(type(source) is int and 0 <= source <= 3 and type(value.get('WeightPresent')) is bool,
         'current Item class-selection operands differ')
    guards = []
    if source == 0 and (held or not value['WeightPresent']):
        if 'Code' not in value or 'ItemWeights' not in ctx['writer']['Capture']:
            return None, guards, 'current Item class fallback operands are not completely captured'
        code = value['Code']
        need(type(code) is int and 0 < code < 1 << 16, 'current Item class fallback Code width differs')
        if not all(digest((Path(__file__).resolve().parents[2]/path).read_bytes()) == expected
                   for path, expected in CLASS_PROJECTION_PINS.items()):
            return None, guards, 'current Item class fallback code dependencies changed'
        weights = ctx['writer']['Capture']['ItemWeights']
        need(weights is None or isinstance(weights, list), 'current ItemWeights getter is malformed')
        previous, selected = 0, 0
        for weight in weights or []:
            if not isinstance(weight, dict) or any(name not in weight for name in ('Code', 'Weight', 'Constructor')):
                return None, guards, 'current ItemWeights getter operands are not completely captured'
            need(type(weight['Code']) is int and previous < weight['Code'] < 1 << 16,
                 'current ItemWeights code population is unordered, duplicated or outside width')
            need(type(weight['Weight']) is int and -(1 << 31) <= weight['Weight'] < 1 << 31,
                 'current ItemWeights weight width differs')
            constructor = weight_constructor_class(weight['Constructor'])
            if constructor is None:
                return None, guards, 'current weight constructor class/schema is not completely captured'
            if weight['Code'] == code:
                selected = constructor
            previous = weight['Code']
        if selected == 0:
            # Table success and the later code fallback select the same class.
            b = (code >> 8) & 15
            selected = 1 if b == 1 else (3 if b == 2 else (2 if 3 <= b <= 12 else 0))
        source = selected
        guards.extend((fact(ptr+'/Value/Code', code), fact('/Capture/ItemWeights', weights)))
    return {0: 'Item', 1: 'Weapon', 2: 'Armor', 3: 'Shield'}[source], guards, None


def direct_item_count(ctx, objects, index, span, spec):
    """Bind one scalar node; current locations select its ordinary producer."""
    current = objects['Items'][index]
    ptr = f'/Capture/Objects/Items/{index}'
    token, coverage = current.get('Token'), current.get('Coverage')
    if not isinstance(token, dict) or not isinstance(coverage, dict) or 'Identity' not in token or 'Unknown' not in coverage:
        return None, 'Item.Count lacks independently captured key/knowledge operands'
    key, mask = token['Identity'], coverage['Unknown']
    need(type(key) is int and 0 <= key < 1 << 32 and type(mask) is int and 0 <= mask < 1 << 64,
         'current Item identity/knowledge width differs')
    if key == 0 or mask & (128 | 16384):
        return None, 'Item.Count current wire identity/count width is absent or unknown'
    need(type(current.get('ID')) is int and 0 < current['ID'] < 1 << 64 and
         current.get('Retired') is False and type(current.get('InFlight')) is int and current['InFlight'] == 0,
         'Item.Count current node is retired, malformed or in flight')
    if 'Unsupported' not in coverage or coverage['Unsupported']:
        return None, 'Item.Count current branch carries unsupported coverage'
    if not all(name in objects for name in ('NextID', 'Items', 'Effects', 'Spells', 'Sacks', 'Containers', 'ItemRoots')):
        return None, 'Item.Count current registry populations are not completely captured'
    next_id = objects['NextID']
    need(type(next_id) is int and 0 < next_id < 1 << 64, 'current registry allocator floor differs')
    guards = [fact('/Capture/WorldPresent', True), fact('/Capture/Objects/Version', 2),
              fact('/Capture/Objects/NextID', next_id)]
    ids, keys = set(), {}
    for collection in ('Items', 'Effects', 'Spells', 'Sacks'):
        rows = objects[collection]
        need(rows is None or isinstance(rows, list), 'current registry collection is malformed')
        previous = 0
        for at_index, row in enumerate(rows or []):
            identity = row['ID']
            need(type(identity) is int and previous < identity < next_id and identity not in ids,
                 'current native ID is unordered, duplicated or crosses kinds')
            ids.add(identity); previous = identity
            root = f'/Capture/Objects/{collection}/{at_index}'
            key_path = root+'/This' if collection == 'Spells' else root+'/Token/Identity'
            try:
                wire_key = at(ctx['writer'], key_path)
            except (KeyError, TypeError):
                return None, 'Item.Count cross-object key population is not completely captured'
            need(type(wire_key) is int and 0 <= wire_key < 1 << 32, 'current cross-object key width differs')
            need(wire_key == 0 or wire_key not in keys, 'current wire key is shared across registry objects')
            if wire_key:
                keys[wire_key] = collection, identity
            guards.extend((fact(root+'/ID', identity), fact(key_path, wire_key)))
    ordinary = {}
    for record in ctx['native'].order:
        for name in ('Identity', 'This'):
            wire_key = record['values'].get(name, 0)
            if not wire_key:
                continue
            need(wire_key not in ordinary or ordinary[wire_key] is record,
                 'ordinary Identity/This collides across objects/classes')
            ordinary[wire_key] = record
    need(key in ordinary, 'known current Item key has no ordinary record')
    record = ordinary[key]
    for name in ('Containers', 'ItemRoots'):
        need(objects[name] is None or isinstance(objects[name], list), 'current root population is malformed')
        guards.append(fact('/Capture/Objects/'+name, objects[name]))
        for root in objects[name] or []:
            need(isinstance(root, dict) and isinstance(root.get('Owner'), dict), 'current root shape differs')
            owner_kind = root['Owner'].get('Kind')
            need(type(owner_kind) is int and owner_kind in ((1, 3) if name == 'Containers' else (2, 4)),
                 'current root owner kind differs')
            references = root.get('Items') if name == 'Containers' else [root.get('ID')]
            need(references is None or isinstance(references, list), 'current root references are malformed')
            need(all(type(identity) is int and 0 <= identity < 1 << 64 for identity in references or []),
                 'current root native ID width differs')
            if name == 'Containers':
                need(type(root.get('Present')) is bool, 'current container presence width differs')
    ordinary_locations = []
    held = False
    for location in locations(objects, current['ID']):
        owner = location['owner']; kind = owner['Kind']
        need(type(kind) is int and kind in ((1, 3) if 'container' in location else (2, 4)),
             'current location has an unsupported owner kind')
        if kind == 4:
            continue
        if 'container' in location:
            need(objects['Containers'][location['container']]['Present'] is True,
                 'current ordinary container occurrence is absent')
        if kind == 2:
            need(type(owner.get('Slot')) is int and 1 <= owner['Slot'] <= 12, 'current worn slot differs')
            held = True
        ordinary_locations.append(location)
    if not ordinary_locations:
        return None, 'Item.Count currentItemHasOrdinaryRoot is false; detached/session-only branch'
    value = current.get('Value')
    if not isinstance(value, dict) or not all(name in value for name in ('Count', 'WeightPresent', 'SourceEquipment')) or \
       not isinstance(value['SourceEquipment'], dict) or 'Class' not in value['SourceEquipment']:
        return None, 'Item.Count direct value/class-selection facts are not completely captured'
    source = value['SourceEquipment']['Class']
    clazz, class_guards, error = direct_item_class(ctx, value, held, ptr)
    if clazz is None:
        return None, error
    guards.extend(class_guards)
    need(record['class'] == clazz, 'direct current Item key joins a different wire class')
    count = value['Count']
    need(type(count) is int and 0 < count <= 65535, 'Item.Count direct word branch is outside bounds')
    if not all(digest((Path(__file__).resolve().parents[2]/path).read_bytes()) == expected
               for path, expected in COUNT_PROJECTION_PINS.items()):
        return None, 'Item.Count current projection/root predicate code changed'
    label = clazz+'[archive='+str(record['archive_index'])+'].Count'
    offset = record['offsets']['Count']
    need(all(type(span.get(name)) is int for name in ('start', 'end', 'length')) and
         span.get('label') == label and span['start'] == offset and span['end'] == offset+2 and span['length'] == 2,
         'Item.Count names a different current key/physical node')
    need(not span.get('aliases') and bytes.fromhex(span.get('bytes_hex', '')) == ctx['body'][offset:offset+2],
         'Item.Count span aliases or differs from actual input')
    guards.extend((fact(ptr+'/ID', current['ID']), fact(ptr+'/Retired', False), fact(ptr+'/InFlight', 0),
                   fact(ptr+'/Coverage/Unknown', mask), fact(ptr+'/Coverage/Unsupported', ''),
                   fact(ptr+'/Value/SourceEquipment/Class', source),
                   fact(ptr+'/Value/WeightPresent', value['WeightPresent'])))
    return dict(spec=spec, ctx=ctx, carrier=ptr+'/Value/Count', guards=guards,
                sources=[fact(ptr+'/Value/Count', op='present')], expected=struct.pack('<H', count),
                relation=dict(current_locations=ordinary_locations, ordinary_root=True, holder_topology_verified=False),
                current=current, record=record, leaf_verified=True), None


def keyed_record(ctx, key, clazz=None):
    records = []
    keys = {}
    for record in ctx['native'].order:
        for name in ('Identity', 'This'):
            value = record['values'].get(name, 0)
            if value:
                need(value not in keys or keys[value] is record, 'ordinary Identity/This collides across objects/classes')
                keys[value] = record
        if key and key in (record['values'].get('Identity'), record['values'].get('This')):
            records.append(record)
    need(len(records) == 1 and (clazz is None or records[0]['class'] == clazz),
         'known current key has no unique exact typed ordinary record')
    return records[0]


def record_span(ctx, record, field, width):
    offset = record['offsets'][field]
    return dict(label=record['class']+'[archive='+str(record['archive_index'])+'].'+field,
                start=offset, end=offset+width, length=width, bytes_hex=ctx['body'][offset:offset+width].hex())


def direct_item_node(ctx, objects, index):
    current = objects['Items'][index]
    key = (current.get('Token') or {}).get('Identity', 0)
    coverage = current.get('Coverage') or {}
    if 'Unknown' not in coverage:
        return None, 'current Item knowledge is not captured'
    mask = coverage['Unknown']
    need(type(mask) is int and 0 <= mask < 1 << 64 and type(key) is int and 0 <= key < 1 << 32,
         'current Item key/knowledge width differs')
    if mask & (128 | 16384):
        return None, 'current Item identity/count width is unknown'
    if not key:
        return None, 'current Item has no independent known key'
    record = keyed_record(ctx, key)
    need(record['class'] in SPECS['bindings']['World.SavedObjects.Items.Count']['classes'],
         'current Item key joins a different wire class')
    count_spec = SPECS['bindings']['World.SavedObjects.Items.Count']
    return direct_item_count(ctx, objects, index, record_span(ctx, record, 'Count', 2), count_spec)


def direct_child_node(ctx, objects, index, spec):
    collection = spec['collection']; current = objects[collection][index]
    ptr = f'/Capture/Objects/{collection}/{index}'
    token, coverage = current.get('Token') or {}, current.get('Coverage')
    if not isinstance(coverage, dict) or not all(name in coverage for name in ('Unknown', 'Unsupported')):
        return None, 'independent child coverage is not captured'
    mask = coverage['Unknown']; key = current.get('This') if collection == 'Spells' else token.get('Identity')
    need(type(mask) is int and 0 <= mask < 1 << 64 and type(key) is int and 0 <= key < 1 << 32,
         'current child key/knowledge width differs')
    if not key or mask & (128 | DIRECT_FIELD_MASKS[collection][spec['field']]) or coverage['Unsupported']:
        return None, 'independent child key/field authority is absent or unknown'
    need(current.get('Retired') is False, 'current child is retired')
    if type(current.get('ExternalReferences')) is not int or current['ExternalReferences'] != 0:
        return None, 'independent external child holders are unbound'
    record = keyed_record(ctx, key, 'Spell' if collection == 'Spells' else 'Effect')
    child_value = current.get('Value')
    names = ('Present', 'ID', 'Range', 'Defensive', 'ManaCost') if collection == 'Spells' else ('Kind', 'Mode', 'Operand')
    if not isinstance(child_value, dict) or any(name not in child_value for name in names):
        return None, 'independent child value is not completely captured'
    if collection == 'Spells':
        need(child_value['Present'] is True and type(child_value['ID']) is int and 1 <= child_value['ID'] <= 28,
             'current Spell initialization/logical ID differs')
        for name, bits in (('Range', 8), ('Defensive', 8), ('ManaCost', 16)):
            need(type(child_value[name]) is int and 0 <= child_value[name] < 1 << bits, 'current Spell value width differs')
        if not all(digest((Path(__file__).resolve().parents[2]/path).read_bytes()) == expected
                   for path, expected in BOOK_PROJECTION_PINS.items()):
            return None, 'protected Weapon Spell late-book writer/incoming code changed'
    else:
        for name, bits in (('Kind', 8), ('Mode', 8), ('Operand', 32)):
            need(type(child_value[name]) is int and 0 <= child_value[name] < 1 << bits, 'current Effect value width differs')
    guards, explained, parents = [], [], []
    for item_index, item in enumerate(objects.get('Items') or []):
        if item['Retired']:
            continue
        if 'Effects' not in item or 'Spell' not in item:
            return None, 'current parent child populations are not completely captured'
        effects = item['Effects']
        need(effects is None or isinstance(effects, list), 'current Effects references are malformed')
        need(all(type(ref) is int and 0 <= ref < 1 << 64 for ref in effects or []) and
             type(item['Spell']) is int and 0 <= item['Spell'] < 1 << 64, 'current child reference width differs')
        slots = [slot for slot, ref in enumerate(effects or []) if type(ref) is int and ref == current['ID']] \
            if collection == 'Effects' else ([0] if type(item['Spell']) is int and item['Spell'] == current['ID'] else [])
        if not slots:
            continue
        parent, error = direct_item_node(ctx, objects, item_index)
        if parent is None:
            return None, error
        raw_parent = parent['record']; parent_ptr = f'/Capture/Objects/Items/{item_index}'
        guards.extend(parent['guards'])
        if collection == 'Effects':
            values = item['Value'].get('Effects')
            if not isinstance(values, list) or len(values) != len(effects):
                return None, 'current parent Effect values/ordinals are not completely captured'
            references = []
            for ref in effects:
                if type(ref) is not int or ref <= 0:
                    return None, 'independent child ordinal pruning is unsupported'
                siblings = [sibling for sibling in objects['Effects'] or [] if sibling['ID'] == ref]
                if len(siblings) != 1 or siblings[0]['Retired'] is not False:
                    return None, 'independent child ordinal pruning is unsupported'
                sibling = siblings[0]; sibling_key = (sibling.get('Token') or {}).get('Identity')
                sibling_coverage = sibling.get('Coverage') or {}
                if 'Unknown' not in sibling_coverage or 'Unsupported' not in sibling_coverage:
                    return None, 'independent child ordinal coverage is not captured'
                need(type(sibling_coverage['Unknown']) is int and 0 <= sibling_coverage['Unknown'] < 1 << 64,
                     'current child ordinal knowledge width differs')
                if not sibling_key or sibling_coverage.get('Unknown', 128) & 128 or sibling_coverage.get('Unsupported', 'missing'):
                    return None, 'independent child ordinal binding is absent or unknown'
                references.append(keyed_record(ctx, sibling_key, 'Effect')['archive_index'])
            need(raw_parent['refs'].get('Effects') == references, 'current ordered Item.Effects keys differ from raw edges')
            for slot in slots:
                if json.dumps(values[slot], sort_keys=True) != json.dumps(child_value, sort_keys=True):
                    return None, 'independent shared Effect parents request incompatible values'
                explained.append((raw_parent['archive_index'], 'Effects', slot))
            guards.extend((fact(parent_ptr+'/Effects', effects), fact(parent_ptr+'/Value/Effects', values)))
        else:
            if raw_parent['class'] != 'Weapon':
                return None, 'Spell source requires an independently keyed Weapon parent'
            if item['Value']['SourceEquipment']['Class'] != 1:
                return None, 'protected Weapon Spell requires explicit current equipment class'
            value = item['Value']['SourceEquipment'].get('Spell')
            if json.dumps(value, sort_keys=True) != json.dumps(child_value, sort_keys=True):
                return None, 'independent shared Weapon parents request incompatible Spell values'
            need(raw_parent['refs'].get('OwnedSpell') == [record['archive_index']], 'current Weapon.Spell key differs from raw edge')
            explained.append((raw_parent['archive_index'], 'OwnedSpell', 0))
            guards.extend((fact(parent_ptr+'/Spell', current['ID']), fact(parent_ptr+'/Value/SourceEquipment/Spell', value)))
        parents.append(dict(native_id=item['ID'], archive_index=raw_parent['archive_index'], slots=slots))
    if not parents:
        return None, 'current child needed predicate has no independently keyed ordinary Item parent'
    observed = [(edge['owner_archive'], edge['field'], edge['slot'])
                for edge in ctx['incoming'].get(record['archive_index'], [])]
    if collections.Counter(observed) != collections.Counter(explained):
        return None, 'independent child has unexplained incoming owners or multiplicity'
    guards.extend((fact(ptr+'/ID', current['ID']), fact(ptr+'/Retired', False),
                   fact(ptr+'/Coverage/Unknown', mask), fact(ptr+'/Coverage/Unsupported', ''),
                   fact(ptr+'/ExternalReferences', 0), fact(ptr+'/Value', child_value)))
    return dict(current=current, record=record, guards=guards,
                relation=dict(parents=parents, exact_current_child_edges=True, needed=True,
                              protected_weapon_spell=collection == 'Spells', holder_topology_verified=False)), None


def direct_item_code(ctx, node, ptr, spec):
    capture = ctx['writer']['Capture']
    if 'ModSetEmpty' not in capture:
        return None, 'Item.Code actual effective ModSet.Empty observation is absent'
    empty = capture['ModSetEmpty']
    need(type(empty) is bool, 'Item.Code effective ModSet.Empty is not boolean')
    if not all(digest((Path(__file__).resolve().parents[2]/path).read_bytes()) == expected
               for path, expected in CODE_PROJECTION_PINS.items()):
        return None, 'Item.Code current late projector/getter code dependencies changed'
    value = node['current']['Value']
    if 'Code' not in value:
        return None, 'Item.Code current selector operand is absent'
    code = value['Code']
    need(type(code) is int and 0 < code < 1 << 16, 'Item.Code current code width differs')
    carrier = ptr+'/Value/Code'
    guards = [fact('/Capture/ModSetEmpty', empty), fact(carrier, code)]
    callsite = spec['producer_callsite']
    arm, selected = 'current-code', None
    if not empty:
        if 'ModItemStandIns' not in capture:
            return None, 'Item.Code actual ordered stand-in getter is absent'
        stand_ins = capture['ModItemStandIns']
        need(stand_ins is None or isinstance(stand_ins, list), 'Item.Code ordered stand-in getter is malformed')
        for at_index, item in enumerate(stand_ins or []):
            if not isinstance(item, dict) or any(name not in item for name in ('Code', 'StandInCode', 'StandInRow')):
                return None, 'Item.Code ordered stand-in operands are not completely captured'
            for name, bits in (('Code', 16), ('StandInCode', 16), ('StandInRow', 8)):
                need(type(item[name]) is int and 0 <= item[name] < 1 << bits,
                     'Item.Code stand-in operand width differs')
            if item['Code'] == code:
                selected = at_index
        guards.append(fact('/Capture/ModItemStandIns', stand_ins))
        if selected is not None:
            carrier = f'/Capture/ModItemStandIns/{selected}/StandInCode'
            code = stand_ins[selected]['StandInCode']
            callsite = CODE_STAND_IN_CALLSITE
            arm = 'late-stand-in'
        else:
            arm = 'current-code-no-match'
    return dict(carrier=carrier, guards=guards, sources=[fact(carrier, op='present')],
                expected=struct.pack('<H', code), producer_callsite=callsite,
                code_selection=dict(arm=arm, selected_stand_in_index=selected,
                                    class_selected_before_substitution=True)), None


def direct_literal(ctx, objects, index, span, spec):
    collection = spec['collection']; current = objects[collection][index]
    if collection == 'Items' and spec['field'] == 'Code' and 'ModSetEmpty' not in ctx['writer']['Capture']:
        return None, 'Item.Code actual effective ModSet.Empty observation is absent'
    field_mask = DIRECT_FIELD_MASKS[collection][spec['field']]
    coverage = current.get('Coverage') or {}
    if 'Unknown' not in coverage:
        return None, 'current literal field knowledge is absent'
    mask = coverage['Unknown']
    need(type(mask) is int and 0 <= mask < 1 << 64, 'current literal knowledge width differs')
    if mask & field_mask:
        return None, 'current literal field is explicitly unknown'
    node, error = direct_item_node(ctx, objects, index) if collection == 'Items' else direct_child_node(ctx, objects, index, spec)
    if node is None:
        return None, error
    ptr = f'/Capture/Objects/{collection}/{index}'; carrier = ptr+'/'+spec['value_path']
    code_projection = None
    if collection == 'Items' and spec['field'] == 'Code':
        code_projection, error = direct_item_code(ctx, node, ptr, spec)
        if code_projection is None:
            return None, error
        carrier = code_projection['carrier']
    try:
        value = at(ctx['writer'], carrier)
    except (KeyError, IndexError, TypeError):
        return None, 'current literal carrier is not captured; no initializer is inferred'
    width = spec['width']
    if spec['encoding'] == 'byte-array':
        need(isinstance(value, list) and len(value) == width and all(type(byte) is int and 0 <= byte < 256 for byte in value),
             'current literal array width differs')
        expected = bytes(value)
    else:
        need(type(value) is int and 0 <= value < 1 << (width*8), 'current literal scalar width differs')
        expected = struct.pack({1: '<B', 2: '<H', 4: '<I'}[width], value)
    exact = record_span(ctx, node['record'], spec['field'], width)
    need(all(type(span.get(name)) is int for name in ('start', 'end', 'length')) and
         all(span.get(name) == exact[name] for name in ('label', 'start', 'end', 'length', 'bytes_hex')) and not span.get('aliases'),
         'current literal names a different typed key/field/physical interval')
    guards = []
    for guard in node['guards']:
        if guard not in guards:
            guards.append(guard)
    if code_projection is not None:
        guards.extend(code_projection['guards'])
        return dict(spec=spec, ctx=ctx, current=current, record=node['record'], guards=guards,
                    **{key: code_projection[key] for key in ('carrier', 'sources', 'expected', 'producer_callsite')},
                    relation={**node['relation'], 'code_selection': code_projection['code_selection']}, leaf_verified=True), None
    return dict(spec=spec, ctx=ctx, current=current, record=node['record'], guards=guards,
                carrier=carrier, sources=[fact(carrier, op='present')], expected=expected,
                relation=node['relation'], leaf_verified=True), None

def derive(decision,span,captures,binding_context):
    binding=decision.get('binding_id');spec=SPECS['bindings'].get(binding)
    if spec is None:return None,'no implemented registry literal binding'
    ctx=actual_context(binding_context);writer=ctx['writer'];capture=writer['Capture']
    need(decision.get('input_sha256')==ctx['saved_sha256'] and decision.get('capture_sha256')=={'writer':ctx['capture_sha256']},'binding input/capture hash differs from actual bytes')
    need(decision.get('layout_sha256')==binding_context.get('layout_sha256'),'binding layout hash differs from actual layout')
    need(decision.get('source_pin')==ctx['source_pin'] and decision.get('knowledge_pin')==ctx['knowledge_pin'],'binding differs from independent expected pins')
    need(decision.get('tool_pin')==digest(Path(__file__).read_bytes()),'binding tool identity differs from concrete module bytes')
    supplied=captures.get('writer');need(isinstance(supplied,dict),'actual writer capture is absent')
    need(json.dumps(supplied,sort_keys=True,allow_nan=False)==json.dumps(writer,sort_keys=True,allow_nan=False),'writer mirror differs from actual raw capture bytes')
    if binding == 'World.SavedObjects.Items.Count':
        need(capture.get('WorldPresent') is True and isinstance(capture.get('Objects'), dict) and
             capture['Objects'].get('Version') == 2, 'Item.Count current registry/World is absent')
        index = decision.get('registry_index'); objects = capture['Objects']
        need(type(index) is int and 0 <= index < len(objects.get('Items') or []), 'Item.Count current row is absent')
        direct, error = direct_item_count(ctx, objects, index, span, spec)
        if direct is not None:
            return direct, None
        if error.startswith(('current Item class fallback operands', 'current ItemWeights getter operands',
                             'current weight constructor class/schema')):
            return None, error
    elif spec['field'] in DIRECT_FIELD_MASKS.get(spec['collection'], {}):
        need(capture.get('WorldPresent') is True and isinstance(capture.get('Objects'), dict) and
             capture['Objects'].get('Version') == 2, 'current registry/World is absent')
        index = decision.get('registry_index'); objects = capture['Objects']
        need(type(index) is int and 0 <= index < len(objects.get(spec['collection']) or []), 'current literal row is absent')
        direct, error = direct_literal(ctx, objects, index, span, spec)
        if direct is not None:
            return direct, None
        if error.startswith(('independent shared', 'independent child has unexplained', 'independent child ordinal',
                             'current Item class fallback operands', 'current ItemWeights getter operands',
                             'current weight constructor class/schema', 'Item.Code current late projector/getter')):
            return None, error
    if not ctx['secondary_present']:return None,'typed ordinary correspondence is absent; independent registry bridge unavailable'
    need(capture.get('WorldPresent') is True and isinstance(capture.get('Objects'),dict),'current registry presence is absent')
    objects=capture['Objects'];need(objects['Version']==2,'registry version unsupported')
    need(all(row['InFlight']==0 for row in objects.get('Items') or []),'current registry has in-flight transaction')
    snapshot_document=(capture.get('Snapshot') or {}).get('SavedDocument')
    if not isinstance(snapshot_document,dict) or not isinstance(snapshot_document.get('Objects'),dict):
        return None,'generated-entry materialization has no captured paired document Version2; route adapter is unimplemented'
    if snapshot_document['Objects'].get('Version')!=2:
        return None,'captured paired document selects a different projection version; adapter is unimplemented'
    collection=spec['collection'];index=decision.get('registry_index');need(type(index) is int and 0<=index<len(objects.get(collection) or []),'registry index is absent')
    current=objects[collection][index];need(type(current['ID']) is int and 0<current['ID']<1<<64 and current['Retired'] is False,'retired/unkeyed row has no current ordinary binding')
    if collection=='Items':
        value=current.get('Value')
        if not isinstance(value,dict) or any(key not in value for key in ['Code','Count','WeightPresent','SourceEquipment']):
            return None,'required current Item source-selection carrier is absent; no initializer is inferred'
        if not isinstance(value['SourceEquipment'],dict) or 'Class' not in value['SourceEquipment']:
            return None,'required current equipment class carrier is absent; no initializer is inferred'
        need(type(value['Code']) is int and 0<=value['Code']<1<<16 and
             type(value['Count']) is int and 0<value['Count']<1<<32 and
             type(value['WeightPresent']) is bool,'current Item source-selection widths differ')
    record=mapped(ctx,spec['kind'],current['ID']);need(record['class'] in spec['classes'],'registry class is outside binding family')
    if collection=='Items':
        relation_fact,error=item_relation(ctx,objects,current['ID'],record)
        if relation_fact is None:return None,error
        clazz,class_fact=expected_item_class(capture,current,relation_fact['held']);need(clazz==record['class'],'current selected equipment class differs from raw record')
        relation_fact['guards'].extend([fact(f'/Capture/Objects/Items/{index}/Value/SourceEquipment/Class',current['Value']['SourceEquipment']['Class']),
            fact(f'/Capture/Objects/Items/{index}/Value/WeightPresent',current['Value']['WeightPresent']),fact(f'/Capture/Objects/Items/{index}/Value/Code',current['Value']['Code'])])
        if class_fact:relation_fact['guards'].append(class_fact)
        need(0<current['Value']['Count']<=65535,'wide/empty Item literal branch is unsupported')
    else:
        relation_fact,error=child_relation(ctx,objects,collection,current['ID'],record)
        if relation_fact is None:return None,error
    ptr=f'/Capture/Objects/{collection}/{index}';carrier=ptr+'/'+spec['value_path']
    try:value=at(writer,carrier)
    except (KeyError,IndexError,TypeError):return None,'required current carrier is absent; no initializer is inferred'
    width=spec['width']
    try:
        if spec['encoding']=='byte-array':
            need(isinstance(value,list) and len(value)==width and all(type(v) is int and 0<=v<256 for v in value),'current raw array width differs');expected=bytes(value)
        else:
            need(type(value) is int and 0<=value<1<<(width*8),'current scalar exceeds exact width');expected=struct.pack({1:'<B',2:'<H',4:'<I'}[width],value)
    except (KeyError,ValueError,TypeError,struct.error) as exc:raise BindingError('literal encoding is unsupported') from exc
    label=record['class']+'[archive='+str(record['archive_index'])+'].'+spec['field']
    offset=record['offsets'].get(spec['field']);need(all(type(span.get(k)) is int for k in ('start','end','length')) and span.get('label')==label and span.get('start')==offset and span.get('length')==width and span.get('end')==offset+width,'binding names a different raw object/class/field/width')
    need(not span.get('aliases'),'physical aliases require all independent owners')
    need(bytes.fromhex(span.get('bytes_hex',''))==ctx['body'][offset:offset+width],'span bytes differ from actual pinned raw input')
    guards=[fact('/Capture/WorldPresent',True),fact('/Capture/Objects/Version',2),fact('/Capture/Snapshot/SavedDocument/Objects/Version',2),fact(ptr+'/ID',current['ID']),fact(ptr+'/Retired',False)]+relation_fact['guards']
    # Remove identical facts while retaining literal evaluated order.
    unique=[]
    for g in guards:
        if g not in unique:unique.append(g)
    return dict(spec=spec,ctx=ctx,carrier=carrier,guards=unique,sources=[fact(carrier,op='present')],expected=expected,
        relation=relation_fact,current=current,record=record),None

def prepare(binding,index,span,captures,base,binding_context):
    decision={**base,'binding_id':binding,'registry_index':index,'source_capture':'writer','source':'current','selected_arm':'current','carrier_branch':'base',
              'carrier':binding,'prior_sources_absent':[],'prior_assertions':{},'tool_pin':digest(Path(__file__).read_bytes())}
    derived,error=derive(decision,span,captures,binding_context)
    if derived is None:return None,error
    decision.update(guard_assertions=derived['guards'],source_assertions=derived['sources'],
        carrier_value=dict(capture='writer',path=derived['carrier'],encoding=derived['spec']['encoding']),
        producer_callsite=derived.get('producer_callsite', derived['spec']['producer_callsite']))
    return decision,None

def verify(decision,row,span,start,end,captures,binding_context):
    need(type(start) is int and type(end) is int,'physical interval has corrupt widths')
    derived,error=derive(decision,span,captures,binding_context)
    if derived is None:return unknown(error)
    spec=derived['spec'];need(row['catalog_row'] in spec['catalog_rows'] and row['wire_member']==spec['wire_member'],'binding names a different catalog field')
    meta=spec['catalog_metadata'][str(row['catalog_row'])]
    need(row['class']==meta['class_name'] and row['atlas_selector']==meta['selector'] and re.search(meta['selector'],span['label']) is not None,'catalog class/selector differs from pinned member')
    need([start,end] in meta['intervals'],'catalog submember interval differs')
    need(decision.get('space')=='decoded' and 0<=start<end<=spec['width'],'physical interval exceeds literal width')
    need(decision.get('source')==decision.get('selected_arm')=='current' and decision.get('carrier')==decision['binding_id'] and decision.get('carrier_branch')=='base','source precedence/current arm differs')
    need(decision.get('source_capture')=='writer' and decision.get('prior_sources_absent')==[] and decision.get('prior_assertions')=={},'current arm has unrelated prior-source assertions')
    need(decision.get('producer_callsite')==derived.get('producer_callsite', spec['producer_callsite']),
         'pinned actual selected projector callsite differs')
    need(decision.get('guard_assertions')==derived['guards'] and decision.get('source_assertions')==derived['sources'],'predicates are not the exact source-selection operands')
    need(decision.get('carrier_value')==dict(capture='writer',path=derived['carrier'],encoding=spec['encoding']),'current pointer/field encoding differs')
    for g in derived['guards']:
        need(at(derived['ctx']['writer'],g['path'])==g['value'],'a current producer predicate is false')
    observed=bytes.fromhex(span['bytes_hex']);need(observed[start:end]==derived['expected'][start:end],'written bytes differ from current literal value')
    if derived.get('leaf_verified'):
        return dict(verified=True, value_verified=True, owner_relation_verified=False, holder_topology_verified=False,
            complete=False, binding_id=decision['binding_id'], source_capture='writer', carrier_path=derived['carrier'],
            source='current', archive_index=derived['record']['archive_index'], directed_relation=derived['relation'],
            producer_callsite=derived.get('producer_callsite', spec['producer_callsite']),
            producer_dependencies={**COUNT_PROJECTION_PINS,
                **(CLASS_PROJECTION_PINS if any(g['path'] == '/Capture/ItemWeights' for g in derived['guards']) else {}),
                **(CODE_PROJECTION_PINS if spec['collection'] == 'Items' and spec['field'] == 'Code' else {}),
                **(BOOK_PROJECTION_PINS if spec['collection'] == 'Spells' else {})},
            limit='Bounded literal source only: independently keyed current node/root or compatible Item child edges and exact direct producer arm; holder topology, other priority branches and whole-save completeness remain unverified.')
    return dict(verified=False,value_verified=True,owner_relation_verified=False,complete=False,
        reason="independent current registry ID/owner bridge unavailable; typed correspondence is not classification authority",binding_id=decision['binding_id'],source_capture='writer',carrier_path=derived['carrier'],
        source='current',archive_index=derived['record']['archive_index'],directed_relation=derived['relation'],
        limit='Exact typed-correspondence guard/value comparison only; identity, owner classification, generated allocation, constructors, omission and whole-save completeness remain unverified.')
