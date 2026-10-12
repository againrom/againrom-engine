package sav

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"sync"
)

const MaxNativeActions = 8 << 20

func NativeActions(state DocumentStateData) ([]byte, bool, error) {
	value, found, err := nativeActionsView(state)
	return bytes.Clone(value), found, err
}

func nativeActionsView(state DocumentStateData) ([]byte, bool, error) {
	return nativeActionsLeaf.lookupPayload(state)
}

func SetNativeActions(state *DocumentStateData, value []byte) error {
	return nativeActionsLeaf.storePayload(state, value)
}

// Registry continuation addresses are part of the document graph. Relocate
// only those addresses; action values and carrier-local labels are opaque.
func remapNativeActionObjects(state *DocumentStateData, permutation []uint16) error {
	return remapNativeActionObjectsMode(state, permutation, false)
}

// remapNativeActionObjectsForCity applies the same archive relocation rules to
// a detached city source. A city edit may deliberately retire an ordinary
// item (for example, a sale) while retaining the rest of the current-state
// supplement. Ownership rows for that retired item are then absent state, not
// a malformed continuation; all other retired-object rules remain strict.
func remapNativeActionObjectsForCity(state *DocumentStateData, permutation []uint16) error {
	return remapNativeActionObjectsMode(state, permutation, true)
}

func remapNativeActionObjectsMode(state *DocumentStateData, permutation []uint16, dropRetiredOwnership bool) error {
	data, present, err := nativeActionsView(*state)
	if err != nil || !present {
		return err
	}
	if len(permutation) == 0 || permutation[0] != 0 {
		return fmt.Errorf("sav: current action remap lacks the null slot")
	}
	if out, ok := nativeRemaps.lookup(data, permutation, dropRetiredOwnership); ok {
		return SetNativeActions(state, out)
	}
	in := data
	var root map[string]json.RawMessage
	if err := json.Unmarshal(data, &root); err != nil {
		return fmt.Errorf("sav: current action remap: %w", err)
	}
	var version uint32
	if err := json.Unmarshal(root["Version"], &version); err != nil || version != 1 {
		return fmt.Errorf("sav: invalid current action remap version")
	}
	var identityRoot map[string]any
	if !dropRetiredOwnership {
		identity := true
		for i, object := range permutation {
			if int(object) != i {
				identity = false
				break
			}
		}
		if identity {
			identityRoot = make(map[string]any, len(root))
			for name, raw := range root {
				identityRoot[name] = raw
			}
		}
	}
	for _, name := range []string{"Bindings", "StructureBindings", "Objects", "Groups", "PlayerSlots", "AbsentPlayers", "SpellCasters", "Ownership", "NativeAreas", "NativeDeliveries", "AbsentDiaries", "ArchiveCoordinates", "ActorGroups", "EffectWidths"} {
		raw, ok := root[name]
		if !ok {
			continue
		}
		var rows []map[string]json.RawMessage
		if err := json.Unmarshal(raw, &rows); err != nil {
			return fmt.Errorf("sav: invalid current action %s", name)
		}
		next := make([]map[string]json.RawMessage, 0, len(rows))
		for _, row := range rows {
			var object uint16
			if err := json.Unmarshal(row["Object"], &object); err != nil || int(object) >= len(permutation) {
				return fmt.Errorf("sav: current action object exceeds remap")
			}
			if name == "Bindings" {
				var missing bool
				if err := json.Unmarshal(row["Missing"], &missing); err != nil || missing != (object == 0) {
					return fmt.Errorf("sav: inconsistent current action missing object")
				}
			} else if object == 0 && name != "Ownership" {
				return fmt.Errorf("sav: null current reserved object")
			}
			current := permutation[object]
			if name == "ActorGroups" {
				if current == 0 {
					continue
				}
				var player uint16
				if err := json.Unmarshal(row["Player"], &player); err != nil || int(player) >= len(permutation) {
					return fmt.Errorf("sav: current actor Group exceeds remap")
				}
				if player != 0 && permutation[player] == 0 {
					continue
				}
				row["Player"], _ = json.Marshal(permutation[player])
			}
			if name == "Ownership" && object != 0 && current == 0 {
				if !dropRetiredOwnership {
					return fmt.Errorf("sav: current ownership node was retired")
				}
				// A source city sale removed the physical node. Its ownership
				// row names that node and has no remaining current state.
				continue
			}
			if name == "EffectWidths" {
				var actor uint16
				if err := json.Unmarshal(row["Actor"], &actor); err != nil || actor == 0 || int(actor) >= len(permutation) {
					return fmt.Errorf("sav: current effect width actor exceeds remap")
				}
				if current == 0 || permutation[actor] == 0 {
					continue
				}
				row["Actor"], _ = json.Marshal(permutation[actor])
			}
			if name == "SpellCasters" {
				if current == 0 {
					continue
				}
				var caster uint16
				if err := json.Unmarshal(row["Caster"], &caster); err != nil || int(caster) >= len(permutation) {
					return fmt.Errorf("sav: current spell caster exceeds remap")
				}
				row["Caster"], _ = json.Marshal(permutation[caster])
			}
			if current == 0 && (name == "Groups" || name == "StructureBindings" || name == "PlayerSlots" || name == "AbsentPlayers") {
				return fmt.Errorf("sav: current Group container was retired")
			}
			if current == 0 && (name == "Objects" || name == "NativeAreas" || name == "NativeDeliveries" || name == "AbsentDiaries" || name == "ArchiveCoordinates") {
				// Explicit retirement detaches the archive alias. Its complete
				// reserved value remains in Actions for private session ownership.
				continue
			}
			row["Object"], _ = json.Marshal(current)
			if name == "Bindings" && current == 0 {
				row["Missing"] = json.RawMessage("true")
			}
			next = append(next, row)
		}
		if rows != nil {
			if identityRoot != nil {
				identityRoot[name] = next
			} else {
				root[name], err = json.Marshal(next)
				if err != nil {
					return err
				}
			}
		}
	}
	if raw, ok := root["Pending"]; ok {
		relocated, err := remapPendingCommandObjects(raw, permutation)
		if err != nil {
			return err
		}
		if identityRoot != nil {
			identityRoot["Pending"] = relocated
		} else {
			root["Pending"] = relocated
		}
	}
	if identityRoot != nil {
		data, err = json.Marshal(identityRoot)
	} else {
		data, err = json.Marshal(root)
	}
	if err != nil {
		return err
	}
	if err := SetNativeActions(state, data); err != nil {
		return err
	}
	nativeRemaps.store(in, permutation, dropRetiredOwnership, data)
	return nil
}

func remapPendingCommandObjects(raw json.RawMessage, permutation []uint16) (json.RawMessage, error) {
	if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return raw, nil
	}
	var queue map[string]json.RawMessage
	if err := json.Unmarshal(raw, &queue); err != nil {
		return nil, fmt.Errorf("sav: invalid pending command queue")
	}
	remap := func(raw json.RawMessage) (json.RawMessage, error) {
		var binding map[string]json.RawMessage
		if err := json.Unmarshal(raw, &binding); err != nil {
			return nil, fmt.Errorf("sav: invalid pending endpoint")
		}
		var object uint16
		var missing bool
		if err := json.Unmarshal(binding["Object"], &object); err != nil || int(object) >= len(permutation) {
			return nil, fmt.Errorf("sav: pending endpoint exceeds remap")
		}
		if err := json.Unmarshal(binding["Missing"], &missing); err != nil || missing != (object == 0) {
			return nil, fmt.Errorf("sav: inconsistent pending missing endpoint")
		}
		binding["Object"], _ = json.Marshal(permutation[object])
		binding["Missing"], _ = json.Marshal(permutation[object] == 0)
		return json.Marshal(binding)
	}
	var commands []map[string]json.RawMessage
	if err := json.Unmarshal(queue["Commands"], &commands); err != nil || len(commands) > 65536 {
		return nil, fmt.Errorf("sav: invalid pending commands")
	}
	for _, command := range commands {
		for _, name := range []string{"Issuer", "Target"} {
			if endpoint, ok := command[name]; ok {
				value, err := remap(endpoint)
				if err != nil {
					return nil, err
				}
				command[name] = value
			}
		}
	}
	var commanded []json.RawMessage
	if err := json.Unmarshal(queue["Commanded"], &commanded); err != nil || len(commanded) > 65536 {
		return nil, fmt.Errorf("sav: invalid pending commanded actors")
	}
	for i, raw := range commanded {
		value, err := remap(raw)
		if err != nil {
			return nil, err
		}
		commanded[i] = value
	}
	queue["Commands"], _ = json.Marshal(commands)
	queue["Commanded"], _ = json.Marshal(commanded)
	return json.Marshal(queue)
}

// nativeRemaps keeps the last few remaps by their exact input. A remap is a
// pure function of the supplement bytes, the permutation and the mode, and a
// load or save path relocates the same supplement many times over.
var nativeRemaps nativeRemapMemo

type nativeRemapMemo struct {
	mu      sync.Mutex
	entries []nativeRemapEntry
}

type nativeRemapEntry struct {
	in, permutation string
	drop            bool
	out             []byte
}

const nativeRemapEntries = 8

func remapPermutationKey(permutation []uint16) string {
	b := make([]byte, 2*len(permutation))
	for i, v := range permutation {
		binary.LittleEndian.PutUint16(b[2*i:], v)
	}
	return string(b)
}

func (m *nativeRemapMemo) lookup(in []byte, permutation []uint16, drop bool) ([]byte, bool) {
	key := remapPermutationKey(permutation)
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, e := range m.entries {
		if e.drop == drop && e.permutation == key && e.in == string(in) {
			return e.out, true
		}
	}
	return nil, false
}

func (m *nativeRemapMemo) store(in []byte, permutation []uint16, drop bool, out []byte) {
	e := nativeRemapEntry{in: string(in), permutation: remapPermutationKey(permutation), drop: drop, out: bytes.Clone(out)}
	m.mu.Lock()
	defer m.mu.Unlock()
	entries := append([]nativeRemapEntry{e}, m.entries...)
	m.entries = entries[:min(len(entries), nativeRemapEntries)]
}
