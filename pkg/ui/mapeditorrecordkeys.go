package ui

func (e *MapEditor) barRecordKey(ref RecordRef) uint64 {
	if n := e.linkedRecord(ref); n >= 0 {
		return e.doc.Records[n].Key
	}
	return 0
}
