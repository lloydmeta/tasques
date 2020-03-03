package common

import (
	"time"

	"github.com/lloydmeta/tasques/internal/domain/metadata"
)

// Version holds data that allows for optimistic locking of data
type Version struct {
	SeqNum      uint64 `json:"seq_num"`
	PrimaryTerm uint64 `json:"primary_term"`
}

// Metadata holds information about the data it's embedded in
type Metadata struct {
	// When the data was created
	CreatedAt time.Time `json:"created_at" swaggertype:"string" format:"date-time"`
	// When the data was last modified
	ModifiedAt time.Time `json:"modified_at" swaggertype:"string" format:"date-time"`
	// Data versioning information
	Version Version `json:"version"`
}

func FromDomainMetadata(m *metadata.Metadata) Metadata {
	return Metadata{
		CreatedAt:  time.Time(m.CreatedAt),
		ModifiedAt: time.Time(m.ModifiedAt),
		Version: Version{
			SeqNum:      uint64(m.Version.SeqNum),
			PrimaryTerm: uint64(m.Version.PrimaryTerm),
		},
	}
}
