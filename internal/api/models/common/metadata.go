package common

import (
	"time"

	"github.com/lloydmeta/tasques/internal/domain/metadata"
)

type Version struct {
	SeqNum      uint64 `json:"seq_num"`
	PrimaryTerm uint64 `json:"primary_term"`
}

type Metadata struct {
	CreatedAt  time.Time `json:"created_at" swaggertype:"string" format:"date-time"`
	ModifiedAt time.Time `json:"modified_at" swaggertype:"string" format:"date-time"`
	Version    Version   `json:"version"`
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
