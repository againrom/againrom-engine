package sav

import "reflect"

// The validated archive retains the wire checks. Copy its already canonical
// DTO fields directly, remapping only local object references and normalizing
// empty slices, without constructing the record shapes a second time.
func copyValidatedDocumentData(data DocumentData, validated *saveDocument, permutation []uint16) (DocumentData, error) {
	if _, err := worldStateSerializedSize(validated.state); err != nil {
		return DocumentData{}, err
	}
	refs := func(ids []uint16) []uint16 {
		out := cityDataCopy(ids)
		if permutation != nil {
			for i, id := range out {
				out[i] = permutation[id]
			}
		}
		return out
	}
	var record func(DocumentRecordData) DocumentRecordData
	record = func(src DocumentRecordData) DocumentRecordData {
		out := DocumentRecordData{Class: src.Class, Values: cityDataCopy(src.Values), Texts: cityDataCopy(src.Texts), Counts: cityDataCopy(src.Counts)}
		for _, raw := range src.Raw {
			out.Raw = append(out.Raw, DocumentRawData{Name: raw.Name, Bytes: cityDataCopy(raw.Bytes)})
		}
		for _, slot := range src.RefSlots {
			out.RefSlots = append(out.RefSlots, DocumentRefsData{Name: slot.Name, Objects: refs(slot.Objects)})
		}
		for _, inline := range src.Inline {
			out.Inline = append(out.Inline, DocumentInlineData{Name: inline.Name, Record: record(inline.Record)})
		}
		for _, group := range src.Groups {
			out.Groups = append(out.Groups, record(group))
		}
		return out
	}
	out := DocumentData{
		Version: data.Version, FileVersion: data.FileVersion, Label: cityDataCopy(data.Label), Head: data.Head,
		Players: refs(data.Players), DeadActors: refs(data.DeadActors), Marker: data.Marker, GlobalDWord: data.GlobalDWord, Trailer: data.Trailer,
		State: DocumentStateData{RootKind: data.State.RootKind, DirectoryRecords: cityDataCopy(data.State.DirectoryRecords)},
	}
	if len(data.Objects) != 0 {
		out.Objects = make([]DocumentRecordData, len(data.Objects))
		for old, src := range data.Objects {
			index := old
			if permutation != nil {
				index = int(permutation[old+1]) - 1
			}
			out.Objects[index] = record(src)
		}
	}
	if w := data.World; w != nil {
		out.World = &DocumentWorldData{
			Buildings: refs(w.Buildings), Effects: refs(w.Effects), Sacks: refs(w.Sacks),
			Blocks: cityDataCopy(w.Blocks), Cells: cityDataCopy(w.Cells), TerrainIdentity: w.TerrainIdentity, Session: w.Session,
		}
	}
	for _, r := range data.State.ValueRecords {
		v := r.Value
		v.Bytes = cityDataCopy(v.Bytes)
		if v.Kind != 2 {
			v.Int32 = 0
		}
		out.State.ValueRecords = append(out.State.ValueRecords, CityStateRecordData{Path: r.Path, Value: v})
	}
	c := data.Campaign
	out.Campaign = CityCampaignData{
		Base:     CityCampaignBaseData{DWords: c.Base.DWords, Arrays: cityDataArrays2(c.Base.Arrays)},
		Parallel: cityDataArrays2(c.Parallel), DWords: cityDataCopy(c.DWords), Arrays: cityDataArrays6(c.Arrays),
		Documents: cityDataCopy(c.Documents), Carriers: cityDataCopy(c.Carriers), Scalars: c.Scalars,
	}
	for _, child := range c.Children {
		out.Campaign.Children = append(out.Campaign.Children, CityCampaignChildData{
			Base: CityCampaignBaseData{DWords: child.Base.DWords, Arrays: cityDataArrays2(child.Base.Arrays)}, Age: child.Age,
		})
	}
	for _, marker := range c.Markers {
		out.Campaign.Markers = append(out.Campaign.Markers, CityCampaignMarkerData{Value: marker.Value, Text: cityDataCopy(marker.Text), Tail: marker.Tail})
	}
	if err := (&documentDataBudget{}).check(reflect.ValueOf(out), 0); err != nil {
		return DocumentData{}, err
	}
	return out, nil
}
