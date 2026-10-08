"""Code-bound source/value checks for explicitly supported producer families.

An evaluated arbitrary predicate is not a producer binding. Unimplemented
families remain unverified even when their declared byte partition is complete.
"""


class BindingError(ValueError):
    pass


def bytes_value(value, length):
    if not isinstance(value, list) or len(value) != length or any(type(v) is not int or not 0 <= v <= 255 for v in value):
        raise BindingError("current raw carrier has no exact byte-array width")
    return bytes(value)


def current_session(decision, row, span, start, end, captures):
    binding = decision.get("binding_id")
    names = {"World.RawSessionMid": ("RawA828", "RawSessionMid", 400),
             "World.RawSessionHead": ("Raw08", "RawSessionHead", 48)}
    if binding not in names:
        return {"verified": False, "reason": "no code-bound producer/value adapter"}
    wire, carrier, width = names[binding]
    if decision.get("space") != "decoded" or row.get("wire_member") != wire or span["label"] != "Session."+wire or span["length"] != width:
        raise BindingError("source binding names a different physical field")
    name = decision.get("source_capture")
    if name not in captures or not isinstance(captures[name].get("Capture"), dict):
        raise BindingError("source binding has no actual producer capture")
    capture = captures[name]["Capture"]
    if capture.get("WorldPresent") is not True:
        raise BindingError("current session source has no captured World")
    value = bytes_value(capture.get(carrier), width)
    if binding == "World.RawSessionHead" and not any(value):
        raise BindingError("zero current head does not select the current writer arm")
    guard = [{"capture": name, "path": "/Capture/WorldPresent", "op": "eq", "value": True}]
    if binding == "World.RawSessionHead":
        guard.append({"capture": name, "path": "/Capture/RawSessionHead", "op": "any_nonzero"})
    source = [{"capture": name, "path": "/Capture/"+carrier, "op": "present"}]
    if decision.get("source") != "current" or decision.get("carrier") != binding or decision.get("carrier_branch") != "base":
        raise BindingError("source binding does not select its actual current producer arm")
    if decision.get("guard_assertions") != guard or decision.get("source_assertions") != source or decision.get("prior_assertions") != {}:
        raise BindingError("source predicates do not bind the producer's exact operands")
    if decision.get("carrier_value") != {"capture": name, "path": "/Capture/"+carrier, "encoding": "byte-array"}:
        raise BindingError("source value does not bind the exact current carrier")
    observed = bytes.fromhex(span.get("bytes_hex", ""))
    if len(observed) != width or observed[start:end] != value[start:end]:
        raise BindingError("written physical bytes differ from the current source carrier")
    return {"verified": True, "binding_id": binding, "source_capture": name,
            "carrier_path": "/Capture/"+carrier, "source": "current"}


def instrument_modules():
    import byte_layout
    import raw_reader
    import registry_binding
    import typed_payload
    import sys
    return (byte_layout, raw_reader, sys.modules[__name__], registry_binding, typed_payload)


def binding_context(layout, captures, source_pin, knowledge_pin, capture_bytes=None, saved_bytes=None):
    import hashlib
    import json
    if saved_bytes is None:
        spans = sorted(layout.get("file_spans", []), key=lambda span: span["start"])
        saved_bytes = b"".join(bytes.fromhex(span["bytes_hex"]) for span in spans)
    if hashlib.sha256(saved_bytes).hexdigest() != layout.get("file_sha256"):
        raise BindingError("binding context differs from actual layout input bytes")
    return {"saved_bytes": saved_bytes, "capture_bytes": capture_bytes or {},
            "source_pin": source_pin, "knowledge_pin": knowledge_pin,
            "layout_sha256": hashlib.sha256(json.dumps(layout, sort_keys=True, separators=(",", ":"),
                ensure_ascii=True).encode("utf-8")).hexdigest()}


def verify(decision, row, span, start, end, captures, binding_context=None):
    if str(decision.get("binding_id", "")).startswith("World.SavedObjects."):
        if binding_context is None:
            return {"verified": False, "complete": False,
                    "reason": "registry binding lacks independent actual byte/pin context"}
        import registry_binding
        try:
            return registry_binding.verify(decision, row, span, start, end, captures, binding_context)
        except BindingError:
            raise
        except (KeyError, IndexError, TypeError, ValueError, OverflowError, RecursionError) as exc:
            raise BindingError("registry binding input is malformed: "+str(exc)) from exc
    result = current_session(decision, row, span, start, end, captures)
    if result.get("verified") is not True:
        return result
    result.update(value_verified=True, complete=False)
    if binding_context is None:
        return {**result, "verified": False,
                "reason": "session value comparison lacks independent actual byte/pin context"}
    import registry_binding
    from pathlib import Path
    name = decision.get("source_capture")
    try:
        raw_capture = binding_context["capture_bytes"][name]
        context = registry_binding.context(binding_context["saved_bytes"], raw_capture,
            binding_context["source_pin"], binding_context["knowledge_pin"])
        need = registry_binding.need
        need(decision.get("input_sha256") == context["saved_sha256"] and
             decision.get("capture_sha256") == {name: context["capture_sha256"]},
             "session binding differs from actual input/capture hashes")
        need(decision.get("layout_sha256") == binding_context.get("layout_sha256"),
             "session binding differs from actual layout hash")
        need(decision.get("source_pin") == context["source_pin"] and
             decision.get("knowledge_pin") == context["knowledge_pin"],
             "session binding differs from independent expected pins")
        need(decision.get("tool_pin") == registry_binding.digest(Path(__file__).read_bytes()),
             "session binding differs from concrete module bytes")
        import json
        need(json.dumps(registry_binding.strict_json(raw_capture), sort_keys=True, allow_nan=False) ==
             json.dumps(captures[name], sort_keys=True, allow_nan=False),
             "session mirror differs from actual capture bytes")
        need(span == context["decoded_spans"].get(span["label"]),
             "session field differs from independently parsed raw bytes")
        need(start == 0 and end == span["length"] and
             decision.get("selected_arm") == "current" and decision.get("prior_sources_absent") == [],
             "session binding has a different interval or source precedence")
        callsite = {"path": "pkg\\game\\savdocument.go", "line": 330,
            "sha256": "35cc6666a770d9eba7c083babd404dc33ace2a03ea5d2bbbda28af15d992cb4e",
            "signature": "func snapshotSavedDocument(ms *Mission, fresh ...bool) (*SnapshotSAVDocument, error) {"}
        need(decision.get("producer_callsite") == callsite, "session literal producer callsite differs")
        actual = Path(__file__).resolve().parents[2]/"pkg/game/savdocument.go"
        if registry_binding.digest(actual.read_bytes()) != callsite["sha256"]:
            return {**result, "verified": False, "producer_bound": False,
                    "reason": "session producer file differs from the bounded callsite; value comparison only"}
        return {**result, "producer_bound": True}
    except BindingError:
        raise
    except (KeyError, IndexError, TypeError, ValueError, OverflowError, RecursionError) as exc:
        raise BindingError("session binding context is malformed: "+str(exc)) from exc
