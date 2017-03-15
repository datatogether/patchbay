package main

import (
	"testing"
)

func metadataEqual(a, b *Metadata) bool {
	if a.Hash != "" && b.Hash != "" {
		return a.Hash == b.Hash
	}

	return a.Timestamp.Equal(b.Timestamp) && a.Subject == b.Subject && a.KeyId == b.KeyId && a.Prev == b.Prev
}

func TestWriteMetadata(t *testing.T) {
	defer resetTestData(appDB, "metadata")

	m, err := NextMetadata(appDB, "test_key_id", "test_subject")
	if err != nil {
		t.Error(err.Error())
		return
	}

	m.Meta = map[string]interface{}{
		"key": "value",
	}

	if err := m.Write(appDB); err != nil {
		t.Error(err.Error())
		return
	}

	b := &Metadata{
		Timestamp: m.Timestamp,
		Subject:   "test_subject",
		KeyId:     "test_key_id",
		Meta: map[string]interface{}{
			"key": "value",
		},
	}

	if !metadataEqual(m, b) {
		t.Errorf("metdata mismach: %s != %s", m, b)
	}
}
