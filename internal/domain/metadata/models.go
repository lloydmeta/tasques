// metadata contains models that hold data about data. Since ES will be the one and only
// data store option for this project, we don't even try to abstract over things like seq
// number and primary term
package metadata

import "time"

type CreatedAt time.Time
type ModifiedAt time.Time

type SeqNum uint64
type PrimaryTerm uint64

type Version struct {
	SeqNum      SeqNum
	PrimaryTerm PrimaryTerm
}

type Metadata struct {
	CreatedAt  CreatedAt
	ModifiedAt ModifiedAt
	Version    Version
}
