// common contains models that are common to ES operations
package common

import (
	"bytes"
	"fmt"
	"time"

	"github.com/elastic/go-elasticsearch/v8/esapi"

	"github.com/lloydmeta/tasques/internal/domain/metadata"
)

type IndexName string
type DocumentID string

type ElasticsearchErr struct {
	Underlying error
}

func (e ElasticsearchErr) Error() string {
	return fmt.Sprintf("Error from Elasticsearch: %v", e.Underlying)
}

func (e ElasticsearchErr) Unwrap() error {
	return e.Underlying
}

type JsonSerdesErr struct {
	Underlying []error
}

func (e JsonSerdesErr) Error() string {
	return fmt.Sprintf("Error working with JSON: %v", e.Underlying)
}

func (e JsonSerdesErr) Unwrap() error {
	if len(e.Underlying) == 1 {
		return e.Underlying[0]
	} else {
		return fmt.Errorf("Multiple JSON serdes errors: [%v]", e.Underlying)
	}
}

func UnexpectedEsStatusError(rawResp *esapi.Response) ElasticsearchErr {
	var buf bytes.Buffer
	var body string
	if _, err := buf.ReadFrom(rawResp.Body); err != nil {
		body = buf.String()
	}
	return ElasticsearchErr{Underlying: fmt.Errorf("Unexpected status from ES: [%d], body: [%s]", rawResp.StatusCode, body)}
}

type EsCreateResponse struct {
	ID          string `json:"_id"`
	SeqNum      uint64 `json:"_seq_no"`
	PrimaryTerm uint64 `json:"_primary_term"`
}

func (r *EsCreateResponse) Version() metadata.Version {
	return metadata.Version{
		SeqNum:      metadata.SeqNum(r.SeqNum),
		PrimaryTerm: metadata.PrimaryTerm(r.PrimaryTerm),
	}
}

type EsUpdateResponse struct {
	Index       string `json:"_index"`
	ID          string `json:"_id"`
	SeqNum      uint64 `json:"_seq_no"`
	PrimaryTerm uint64 `json:"_primary_term"`
	Result      string `json:"result"`
}

func (r *EsUpdateResponse) Version() metadata.Version {
	return metadata.Version{
		SeqNum:      metadata.SeqNum(r.SeqNum),
		PrimaryTerm: metadata.PrimaryTerm(r.PrimaryTerm),
	}
}

type PersistedMetadata struct {
	CreatedAt  time.Time `json:"created_at"`
	ModifiedAt time.Time `json:"modified_at"`
}

type EsBulkResponse struct {
	Took   uint                 `json:"took"`
	Errors bool                 `json:"errors"`
	Items  []EsBulkResponseItem `json:"items"`
}

type EsBulkResponseItemInfo struct {
	Index       string `json:"_index"`
	ID          string `json:"_id"`
	SeqNum      uint64 `json:"_seq_no"`
	PrimaryTerm uint64 `json:"_primary_term"`
	Result      string `json:"result"`
	Status      uint   `json:"status"`
}

type EsBulkResponseItem struct {
	Index  *EsBulkResponseItemInfo `json:"index"`
	Delete *EsBulkResponseItemInfo `json:"delete"`
	Create *EsBulkResponseItemInfo `json:"create"`
	Update *EsBulkResponseItemInfo `json:"update"`
}

func (i *EsBulkResponseItem) Info() EsBulkResponseItemInfo {
	// It must be one of these.
	if i.Index != nil {
		return *i.Index
	} else if i.Delete != nil {
		return *i.Delete
	} else if i.Create != nil {
		return *i.Create
	} else {
		return *i.Update
	}
}

func (i *EsBulkResponseItemInfo) IsOk() bool {
	return 200 <= i.Status && i.Status <= 299
}

func (i *EsBulkResponseItemInfo) Version() metadata.Version {
	return metadata.Version{
		SeqNum:      metadata.SeqNum(i.SeqNum),
		PrimaryTerm: metadata.PrimaryTerm(i.PrimaryTerm),
	}
}
