package common

import "time"

type Version struct {
	SeqNum      uint64 `json:"seq_num"`
	PrimaryTerm uint64 `json:"primary_term"`
}

type Metadata struct {
	CreatedAt  time.Time `json:"created_at" swaggertype:"string" format:"date-time"`
	ModifiedAt time.Time `json:"modified_at" swaggertype:"string" format:"date-time"`
	Version    Version   `json:"version"`
}
