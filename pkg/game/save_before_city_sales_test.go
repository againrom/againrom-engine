package game

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"io"
	"testing"
)

func beforeCitySalesEnvelope(t *testing.T, fixture string) []byte {
	t.Helper()
	compressed, err := base64.StdEncoding.DecodeString(fixture)
	if err != nil {
		t.Fatal(err)
	}
	z, err := gzip.NewReader(bytes.NewReader(compressed))
	if err != nil {
		t.Fatal(err)
	}
	defer z.Close()
	raw, err := io.ReadAll(z)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}
