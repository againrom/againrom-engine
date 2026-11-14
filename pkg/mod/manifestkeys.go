package mod

// ManifestValue is the kind of value a manifest key takes: a string, a list
// of strings or an integer.
type ManifestValue int

const (
	ManifestString ManifestValue = iota
	ManifestList
	ManifestInt
)

// ManifestEdit is how the map editor edits a key: not at all (api, entry),
// by the package's identity fields, or as a field of the mod's page.
type ManifestEdit int

const (
	EditNone ManifestEdit = iota
	EditIdentity
	EditField
)

// ManifestKey is one top-level key of mod.toml. Lenient skips a malformed
// value in a lax read; Continued spans bracketed lines; Bases offers
// base.ModBases.
type ManifestKey struct {
	Name      string
	Value     ManifestValue
	Lenient   bool
	Continued bool
	Bases     bool
	Edit      ManifestEdit
	Get       func(m Manifest) any
	Set       func(m *Manifest, v any)
}

// ManifestKeys is the one table of mod.toml's keys, in documentation order,
// for the reader, the writer and the editor's fields.
var ManifestKeys = []ManifestKey{
	{Name: "id", Value: ManifestString, Edit: EditIdentity,
		Get: func(m Manifest) any { return m.ID }, Set: func(m *Manifest, v any) { m.ID = v.(string) }},
	{Name: "title", Value: ManifestString, Edit: EditIdentity,
		Get: func(m Manifest) any { return m.Title }, Set: func(m *Manifest, v any) { m.Title = v.(string) }},
	{Name: "version", Value: ManifestString, Edit: EditField,
		Get: func(m Manifest) any { return m.Version }, Set: func(m *Manifest, v any) { m.Version = v.(string) }},
	{Name: "author", Value: ManifestString, Lenient: true, Edit: EditField,
		Get: func(m Manifest) any { return m.Author }, Set: func(m *Manifest, v any) { m.Author = v.(string) }},
	{Name: "description", Value: ManifestString, Lenient: true, Edit: EditField,
		Get: func(m Manifest) any { return m.Description }, Set: func(m *Manifest, v any) { m.Description = v.(string) }},
	{Name: "api", Value: ManifestInt,
		Get: func(m Manifest) any { return m.API }, Set: func(m *Manifest, v any) { m.API = v.(int) }},
	{Name: "applies-to", Value: ManifestList, Bases: true, Edit: EditField,
		Get: func(m Manifest) any { return m.AppliesTo }, Set: func(m *Manifest, v any) { m.AppliesTo = v.([]string) }},
	{Name: "entry", Value: ManifestString,
		Get: func(m Manifest) any { return m.Entry }, Set: func(m *Manifest, v any) { m.Entry = v.(string) }},
	{Name: "requires", Value: ManifestList, Continued: true, Edit: EditField,
		Get: func(m Manifest) any { return m.Requires }, Set: func(m *Manifest, v any) { m.Requires = v.([]string) }},
	{Name: "conflicts", Value: ManifestList, Continued: true, Edit: EditField,
		Get: func(m Manifest) any { return m.Conflicts }, Set: func(m *Manifest, v any) { m.Conflicts = v.([]string) }},
	{Name: "load-after", Value: ManifestList, Continued: true, Edit: EditField,
		Get: func(m Manifest) any { return m.LoadAfter }, Set: func(m *Manifest, v any) { m.LoadAfter = v.([]string) }},
}

// ManifestKeyNamed is the row of ManifestKeys named key.
func ManifestKeyNamed(key string) (ManifestKey, bool) {
	for _, k := range ManifestKeys {
		if k.Name == key {
			return k, true
		}
	}
	return ManifestKey{}, false
}

// manifestKeyNames are the names of ManifestKeys in their order.
func manifestKeyNames() []string {
	names := make([]string, len(ManifestKeys))
	for i, k := range ManifestKeys {
		names[i] = k.Name
	}
	return names
}
