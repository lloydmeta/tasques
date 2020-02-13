package persistence

import (
	"github.com/elastic/go-elasticsearch/v8"

	"github.com/lloydmeta/tasques/example/ciphers/common"
)

// The server only needs to create and read
type MessagesRepo interface {
	common.MessageCreate
	common.MessageReads
}

func NewMessagesRepo(client *elasticsearch.Client) MessagesRepo {
	return &common.EsMessageRepo{Client: client}
}
