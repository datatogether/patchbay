package main

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/multiformats/go-multihash"
	"time"
)

// A snapshot is a record of a GET request to a url
// There can be many metadata of a given url
type Metadata struct {
	// Hash is the the sha256 multihash of all other fields in metadata
	// as expressed by Metadata.HashableBytes()
	Hash string `json:"hash"`
	// Creation timestamp
	Timestamp time.Time `json:"timestamp"`
	// Sha256 multihash of the public key that signed this metadata
	KeyId string `json:"keyId"`
	// Sha256 multihash of the content this metadata is describing
	Subject string `json:"subject"`
	// Hash value of the metadata that came before this, if any
	Prev string `json:"prev"`
	// Acutal metadata, a valid json Object
	Meta map[string]interface{} `json:"meta"`
}

// String is metadata's abbreviated string representation
func (m Metadata) String() string {
	return fmt.Sprintf("%s : %s.%s", m.Hash, m.KeyId, m.Subject)
}

// NextMetadata returns the next metadata block for a given subject. If no metablock
// exists a new one is created
func NextMetadata(db sqlQueryable, keyId, subject string) (*Metadata, error) {
	m, err := LatestMetadata(db, keyId, subject)
	if err != nil {
		if err == ErrNotFound {
			return &Metadata{
				KeyId:   keyId,
				Subject: subject,
				Meta:    map[string]interface{}{},
			}, nil
		} else {
			return nil, err
		}
	}

	return &Metadata{
		KeyId:   m.KeyId,
		Subject: m.Subject,
		Prev:    m.Hash,
		Meta:    m.Meta,
	}, nil
}

// LatestMetadata gives the most recent metadata timestamp for a given keyId & subject
// combination if one exists
func LatestMetadata(db sqlQueryable, keyId, subject string) (m *Metadata, err error) {
	row := db.QueryRow(fmt.Sprintf("select %s from metadata where key_id = $1 and subject = $2 order by time_stamp desc", metadataCols()), keyId, subject)
	err = m.UnmarshalSQL(row)
	return
}

// HashableBytes returns the exact structure to be used for hash
func (m *Metadata) HashableBytes() ([]byte, error) {
	hash := struct {
		Timestamp time.Time              `json:"timestamp"`
		KeyId     string                 `json:"keyId"`
		Subject   string                 `json:"subject"`
		Prev      string                 `json:"prev"`
		Meta      map[string]interface{} `json:"meta"`
	}{
		Timestamp: m.Timestamp,
		KeyId:     m.KeyId,
		Subject:   m.Subject,
		Prev:      m.Prev,
		Meta:      m.Meta,
	}
	return json.Marshal(&hash)
}

func (m *Metadata) calcHash() error {
	data, err := m.HashableBytes()
	if err != nil {
		return err
	}

	h := sha256.New()
	h.Write(data)

	mhBuf, err := multihash.EncodeName(h.Sum(nil), "sha2-256")
	if err != nil {
		return err
	}

	m.Hash = hex.EncodeToString(mhBuf)
	return nil
}

// WriteMetadata creates a snapshot record in the DB from a given Url struct
func (m *Metadata) Write(db sqlQueryExecable) error {

	m.Timestamp = time.Now().Round(time.Second)
	if err := m.calcHash(); err != nil {
		return err
	}
	metaBytes, err := json.Marshal(m.Meta)
	if err != nil {
		return err
	}

	_, err = db.Exec("insert into metadata values ($1, $2, $3, $4, $5, $6, false)", m.Hash, m.Timestamp.In(time.UTC).Round(time.Second), m.KeyId, m.Subject, m.Prev, metaBytes)
	return err
}

// MetadatasForUrl returns all metadata for a given url string
// func MetadatasForUrl(db sqlQueryable, url string) ([]*Metadata, error) {
// 	res, err := db.Query("select url, created, status, duration, hash, headers from metadata where url = $1", url)
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer res.Close()

// 	metadata := make([]*Metadata, 0)
// 	for res.Next() {
// 		c := &Metadata{}
// 		if err := c.UnmarshalSQL(res); err != nil {
// 			return nil, err
// 		}
// 		metadata = append(metadata, c)
// 	}

// 	return metadata, nil
// }

func metadataCols() string {
	return "hash, time_stamp, key_id, subject, prev, meta"
}

// UnmarshalSQL reads an SQL result into the snapshot receiver
func (m *Metadata) UnmarshalSQL(row sqlScannable) error {
	var (
		hash, keyId, subject, prev string
		timestamp                  time.Time
		metaBytes                  []byte
	)

	if err := row.Scan(&hash, &timestamp, &keyId, &subject, &prev, &metaBytes); err != nil {
		if err == sql.ErrNoRows {
			return ErrNotFound
		}
		return err
	}

	var meta map[string]interface{}
	if metaBytes != nil {
		if err := json.Unmarshal(metaBytes, &meta); err != nil {
			return err
		}
	}

	*m = Metadata{
		Hash:      hash,
		Timestamp: timestamp,
		KeyId:     keyId,
		Subject:   subject,
		Prev:      prev,
		Meta:      meta,
	}

	return nil
}
