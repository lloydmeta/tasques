package common

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
	"github.com/google/uuid"
)

type NewMessage struct {
	Plain string `json:"plain"`
}

type Message struct {
	ID    string `json:"id"`
	Plain string `json:"plain"`

	Rot1  *string `json:"rot_1,omitempty"`
	Rot13 *string `json:"rot_13,omitempty"`

	Base64 *string `json:"base_64,omitempty"`
	Md5    *string `json:"md_5,omitempty"`

	CreatedAt  time.Time `json:"created_at"`
	ModifiedAt time.Time `json:"modified_at"`
}

var (
	Rot1       = "Rot1"
	Rot13      = "Rot13"
	Base64     = "Base64"
	Md5        = "Md5"
	AllCiphers = []string{Rot1, Rot13, Base64, Md5}
)

type MessageArgs struct {
	MessageId string `json:"message_id"`
}

func (j *MessageArgs) ToMap() map[string]interface{} {
	var m map[string]interface{}
	inrec, _ := json.Marshal(j)
	_ = json.Unmarshal(inrec, &m)
	return m
}

func FromInterface(i interface{}) (*MessageArgs, error) {
	var j MessageArgs
	inrec, err := json.Marshal(i)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(inrec, &j)
	if err != nil {
		return nil, err
	}
	return &j, nil
}

type MessageCreate interface {
	Create(ctx context.Context, message NewMessage) (*Message, error)
}

type MessageReads interface {
	Get(ctx context.Context, id string) (*Message, error)
	List(ctx context.Context) ([]Message, error)
}

type MessageUpdate interface {
	SetRot1(ctx context.Context, id string, data string) error
	SetRot13(ctx context.Context, id string, data string) error
	SetBase64(ctx context.Context, id string, data string) error
	SetMd5(ctx context.Context, id string, data string) error
}

var MessagesIndexName = "messages_and_their_ciphers"

func genId() string {
	return strings.ReplaceAll(uuid.New().String(), "-", "")
}

type EsMessageRepo struct {
	Client *elasticsearch.Client
}

func (r *EsMessageRepo) Get(ctx context.Context, id string) (*Message, error) {
	getReq := esapi.GetRequest{
		Index:      MessagesIndexName,
		DocumentID: id,
	}
	rawResp, err := getReq.Do(ctx, r.Client)
	if err != nil {
		return nil, err
	}
	defer rawResp.Body.Close()
	switch rawResp.StatusCode {
	case 200:
		var hit esHit
		if err := json.NewDecoder(rawResp.Body).Decode(&hit); err != nil {
			return nil, err
		}
		return &hit.Source, nil
	default:
		return nil, fmt.Errorf("unexpected status [%s]", rawResp.Status())
	}
}

func (r *EsMessageRepo) Create(ctx context.Context, message NewMessage) (*Message, error) {
	now := time.Now().UTC()
	id := genId()
	toPersist := Message{
		ID:         id,
		Plain:      message.Plain,
		Rot1:       nil,
		Rot13:      nil,
		Base64:     nil,
		Md5:        nil,
		CreatedAt:  now,
		ModifiedAt: now,
	}
	toPersistBytes, err := json.Marshal(toPersist)
	if err != nil {
		return nil, err
	}

	createReq := esapi.CreateRequest{
		DocumentID: id,
		Index:      MessagesIndexName,
		Body:       bytes.NewReader(toPersistBytes),
	}

	rawResp, err := createReq.Do(ctx, r.Client)
	if err != nil {
		return nil, err
	}
	defer rawResp.Body.Close()

	statusCode := rawResp.StatusCode
	switch {
	case 200 <= statusCode && statusCode <= 299:
		return &toPersist, nil
	default:
		return nil, fmt.Errorf("unexpected status [%s]", rawResp.Status())
	}
}

func (r *EsMessageRepo) List(ctx context.Context) ([]Message, error) {
	b, err := json.Marshal(listQuery)
	if err != nil {
		return nil, err
	}
	req := esapi.SearchRequest{
		Index:          []string{MessagesIndexName},
		AllowNoIndices: esapi.BoolPtr(true),
		Body:           bytes.NewReader(b),
	}
	resp, err := req.Do(ctx, r.Client)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var searchResp esSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return nil, err
	} else {
		items := make([]Message, 0, len(searchResp.Hits.Hits))
		for _, i := range searchResp.Hits.Hits {
			items = append(items, i.Source)
		}
		return items, nil
	}
}

type jsonObjMap = map[string]interface{}

var listQuery = jsonObjMap{
	"from": 0,
	"size": 10000,
	"sort": []jsonObjMap{
		{
			"created_at": jsonObjMap{
				"order": "desc",
			},
		},
	},
	"query": jsonObjMap{
		"match_all": jsonObjMap{},
	},
}

func (r *EsMessageRepo) SetRot1(ctx context.Context, id string, data string) error {
	return r.updateField(ctx, id, "rot_1", data)
}

func (r *EsMessageRepo) SetRot13(ctx context.Context, id string, data string) error {
	return r.updateField(ctx, id, "rot_13", data)
}

func (r *EsMessageRepo) SetBase64(ctx context.Context, id string, data string) error {
	return r.updateField(ctx, id, "base_64", data)
}

func (r *EsMessageRepo) SetMd5(ctx context.Context, id string, data string) error {
	return r.updateField(ctx, id, "md_5", data)
}

func (r *EsMessageRepo) updateField(ctx context.Context, id, field, data string) error {
	now := time.Now().UTC()
	partialDoc := jsonObjMap{
		field:         data,
		"modified_at": now,
	}
	updateBody := jsonObjMap{
		"doc": partialDoc,
	}
	bodyBytes, err := json.Marshal(updateBody)
	if err != nil {
		return err
	}

	updateReq := esapi.UpdateRequest{
		Index:           MessagesIndexName,
		DocumentID:      id,
		Body:            bytes.NewReader(bodyBytes),
		RetryOnConflict: esapi.IntPtr(1000),
	}

	resp, err := updateReq.Do(ctx, r.Client)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	statusCode := resp.StatusCode
	switch {
	case 200 <= statusCode && statusCode <= 299:
		return nil
	case statusCode == 404:
		return fmt.Errorf("no such id [%s]", id)
	default:
		return fmt.Errorf("unexpected [%s]", resp.Status())
	}
}

type esSearchResponse struct {
	Hits struct {
		Hits []esItem `json:"hits"`
	} `json:"hits"`
}

type esItem struct {
	Source Message `json:"_source"`
}

type esHit struct {
	Source Message `json:"_source"`
}
