package sav

import "fmt"

// ReadDocumentDiary uses the ordinary Diary decoder on the existing DTO.
func ReadDocumentDiary(data DocumentRecordData) (Diary, error) {
	if data.Class != "Diary" {
		return Diary{}, fmt.Errorf("sav: expected a Diary document record")
	}
	r := newRecord(data.Class, 0, 0)
	if err := documentRecordFromData(data, r, nil, false, 0); err != nil {
		return Diary{}, err
	}
	return diaryFromRecord(r)
}

// ProjectDocumentDiary reuses the same paired-array writer as ordinary SAV.
// Length and both values are independent current state; Self stays in data.
func ProjectDocumentDiary(data DocumentRecordData, current Diary) (DocumentRecordData, error) {
	if _, err := ReadDocumentDiary(data); err != nil {
		return DocumentRecordData{}, err
	}
	if current.Length < 0 || current.Length > maxListElements {
		return DocumentRecordData{}, fmt.Errorf("sav: Diary length is outside the archive bound")
	}
	dwords, words, err := encodeDiaryArrays(current)
	if err != nil {
		return DocumentRecordData{}, err
	}
	out := data
	out.Raw = append([]DocumentRawData(nil), data.Raw...)
	out.Counts = append([]DocumentCountData(nil), data.Counts...)
	for i := range out.Raw {
		switch out.Raw[i].Name {
		case "Journal":
			out.Raw[i].Bytes = dwords
		case "JournalWords":
			out.Raw[i].Bytes = words
		}
	}
	for i := range out.Counts {
		if out.Counts[i].Name == "Journal" || out.Counts[i].Name == "JournalWords" {
			out.Counts[i].Count = uint32(current.Length)
		}
	}
	return out, nil
}
