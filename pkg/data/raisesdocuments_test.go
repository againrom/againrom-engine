package data

import "testing"

func TestRaisesDocumentsAdmitsTheEightRowsAndNothingElse(t *testing.T) {
	rows := []int{28, 60, 92, 124, 156, 188, 220, 252}
	want := make(map[ItemCode]bool, len(rows)*16)
	for material := 0; material < 16; material++ {
		for _, row := range rows {
			want[ItemCode(material<<12|14<<8|row)] = true
		}
	}
	if len(want) != 128 {
		t.Fatalf("the admitted set is %d codes, want 8 rows over 16 materials", len(want))
	}
	if !want[QuestDocumentCode] {
		t.Errorf("QuestDocumentCode 0x%04x is not in the enumerated set", uint16(QuestDocumentCode))
	}

	bad := 0
	for c := 0; c < 65536; c++ {
		code := ItemCode(c)
		got := RaisesDocuments(code)
		if got != want[code] {
			bad++
			if bad <= 5 {
				t.Errorf("RaisesDocuments(0x%04x) = %v, want %v", c, got, want[code])
			}
		}
	}
	if bad > 5 {
		t.Errorf("and %d more codes disagree", bad-5)
	}
}
