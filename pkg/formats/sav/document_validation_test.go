package sav

import (
	"bytes"
	"reflect"
	"testing"
)

func documentCopyRawFields(data *DocumentData) []*DocumentRawData {
	var fields []*DocumentRawData
	var record func(*DocumentRecordData)
	record = func(r *DocumentRecordData) {
		for i := range r.Raw {
			fields = append(fields, &r.Raw[i])
		}
		for i := range r.Inline {
			record(&r.Inline[i].Record)
		}
		for i := range r.Groups {
			record(&r.Groups[i])
		}
	}
	for i := range data.Objects {
		record(&data.Objects[i])
	}
	return fields
}

func aliasDocumentCopyRaw(data *DocumentData) {
	shared := make(map[int][]byte)
	for _, field := range documentCopyRawFields(data) {
		n := len(field.Bytes)
		if _, ok := shared[n]; !ok {
			shared[n] = bytes.Repeat([]byte{0xa5}, n+1)[:n]
		}
		field.Bytes = shared[n]
	}
}

func documentCopyOperation(data DocumentData, reindex bool) (DocumentData, []uint16, error) {
	if reindex {
		return ReindexDocumentData(data)
	}
	out, err := CloneDocumentData(data)
	return out, nil, err
}

func TestDocumentCopyRawOccurrencesStayIndependent(t *testing.T) {
	for _, reindex := range []bool{false, true} {
		name := "clone"
		if reindex {
			name = "reindex"
		}
		t.Run(name, func(t *testing.T) {
			data := documentDataFixture(t)
			aliasDocumentCopyRaw(&data)
			before, _ := documentDataGobCopy(t, data)
			want, _, err := documentCopyOracle(data, reindex)
			if err != nil {
				t.Fatal(err)
			}
			got, _, err := documentCopyOperation(data, reindex)
			if err != nil || !reflect.DeepEqual(got, want) {
				t.Fatal("aliased input differs from owning record oracle", err)
			}
			fields, expected := documentCopyRawFields(&got), documentCopyRawFields(&want)
			for i, field := range fields {
				if len(field.Bytes) == 0 {
					continue
				}
				field.Bytes[0] ^= 0xff
				for j, other := range fields {
					if i != j && !bytes.Equal(other.Bytes, expected[j].Bytes) {
						t.Fatalf("mutating raw occurrence %d changed occurrence %d", i, j)
					}
				}
				after, _ := documentDataGobCopy(t, data)
				if !bytes.Equal(before, after) {
					t.Fatalf("raw occurrence %d aliases the input", i)
				}
				field.Bytes[0] ^= 0xff
			}
			outputBefore, _ := documentDataGobCopy(t, got)
			poisonDocumentCopy(reflect.ValueOf(&data).Elem())
			outputAfter, _ := documentDataGobCopy(t, got)
			if !bytes.Equal(outputBefore, outputAfter) {
				t.Fatal("returned copy retained input storage")
			}
		})
	}
}

func TestDocumentCopyRawValidationFailureKeepsInput(t *testing.T) {
	for _, reindex := range []bool{false, true} {
		data := documentDataFixture(t)
		aliasDocumentCopyRaw(&data)
		mutated := false
		for _, field := range documentCopyRawFields(&data) {
			if len(field.Bytes) != 0 {
				field.Bytes = field.Bytes[:len(field.Bytes)-1]
				mutated = true
				break
			}
		}
		if !mutated {
			t.Fatal("missing raw extent control")
		}
		before, _ := documentDataGobCopy(t, data)
		_, _, wantErr := documentCopyOracle(data, reindex)
		got, permutation, err := documentCopyOperation(data, reindex)
		if wantErr == nil || err == nil || err.Error() != wantErr.Error() || !reflect.DeepEqual(got, DocumentData{}) || permutation != nil {
			t.Fatal("invalid raw extent was accepted, changed error or returned partial state", err, wantErr)
		}
		after, _ := documentDataGobCopy(t, data)
		if !bytes.Equal(before, after) {
			t.Fatal("failed validation changed aliased input")
		}
	}
}

// This is the owning validation route before transient raw borrowing. Its
// output uses the same detached copier; the separate Record oracle owns values.
func documentCopyOwnedValidation(data DocumentData, reindex bool) (DocumentData, []uint16, error) {
	var permutation []uint16
	if reindex {
		permutation = make([]uint16, len(data.Objects)+1)
	}
	d, err := saveDocumentFromDataIndexed(data, permutation)
	if err != nil {
		return DocumentData{}, nil, err
	}
	out, err := copyValidatedDocumentData(data, d, permutation)
	if err != nil {
		return DocumentData{}, nil, err
	}
	if reindex {
		if err := remapNativeActionObjects(&out.State, permutation); err != nil {
			return DocumentData{}, nil, err
		}
	}
	return out, permutation, nil
}

var documentValidationSink DocumentData
var documentValidationPermutationSink []uint16

func TestDocumentCopyAvoidsRepeatedRawAllocation(t *testing.T) {
	data := documentDataFixture(t)
	aliasDocumentCopyRaw(&data)
	var rawCopies, rawBytes int
	for _, field := range documentCopyRawFields(&data) {
		if len(field.Bytes) != 0 {
			rawCopies++
			rawBytes += len(field.Bytes)
		}
	}
	if rawCopies < 8 {
		t.Fatal("allocation witness has too few independent raw occurrences")
	}
	for _, reindex := range []bool{false, true} {
		name := "clone"
		if reindex {
			name = "reindex"
		}
		t.Run(name, func(t *testing.T) {
			before, _ := documentDataGobCopy(t, data)
			want, wantPermutation, err := documentCopyOracle(data, reindex)
			if err != nil {
				t.Fatal(err)
			}
			for _, operation := range []func(DocumentData, bool) (DocumentData, []uint16, error){documentCopyOwnedValidation, documentCopyOperation} {
				got, permutation, err := operation(data, reindex)
				if err != nil || !reflect.DeepEqual(got, want) || !reflect.DeepEqual(permutation, wantPermutation) {
					t.Fatal("allocation routes differ from the whole Record oracle", err)
				}
				gotWire, gotErr := EncodeDocumentData(got)
				wantWire, wantErr := EncodeDocumentData(want)
				if gotErr != nil || wantErr != nil || !bytes.Equal(gotWire, wantWire) {
					t.Fatal("allocation routes differ in full SAV bytes", gotErr, wantErr)
				}
			}
			measure := func(operation func(DocumentData, bool) (DocumentData, []uint16, error)) float64 {
				return testing.AllocsPerRun(20, func() {
					var err error
					documentValidationSink, documentValidationPermutationSink, err = operation(data, reindex)
					if err != nil {
						t.Fatal(err)
					}
				})
			}
			owned, current := measure(documentCopyOwnedValidation), measure(documentCopyOperation)
			t.Logf("owning %.1f allocations, current %.1f; raw occurrences %d, bytes %d", owned, current, rawCopies, rawBytes)
			if owned-current < float64(rawCopies)/2 {
				t.Fatalf("repeated raw allocation remains: saving %.1f, require at least %.1f", owned-current, float64(rawCopies)/2)
			}
			after, _ := documentDataGobCopy(t, data)
			if !bytes.Equal(before, after) {
				t.Fatal("validation or repeated copying changed the source")
			}
		})
	}
	documentValidationSink, documentValidationPermutationSink = DocumentData{}, nil
}
