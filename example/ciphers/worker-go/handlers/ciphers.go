package handlers

import (
	"context"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"math/rand"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/lloydmeta/tasques/example/ciphers/common"
	"github.com/lloydmeta/tasques/example/ciphers/worker-go/persistence"
	"github.com/lloydmeta/tasques/worker"
)

type CipherWorker struct {
	Repo           persistence.MessagesRepo
	RandomFailures bool
}

func (w *CipherWorker) Handle(handle worker.TaskHandle) (*worker.Success, *worker.Failure) {
	// Simulate some processing time...
	pause := int64(rand.Intn(10))
	log.Log().Int64("pause_in_seconds", pause).Msg("Simulating a lot of processing..")
	time.Sleep(time.Duration(pause * int64(time.Second)))
	if w.RandomFailures && rand.Int()%2 == 0 {
		return nil, &worker.Failure{
			"error": "Destined to fail, I guess",
		}
	} else {
		ctx := context.Background()
		task := handle.Task
		if args, err := common.FromInterface(task.Args); err != nil {
			f := worker.Failure{
				"error": err.Error(),
			}
			return nil, &f
		} else {
			_ = handle.ReportIn(&map[string]interface{}{
				"starting": handle.Task.ID,
			})
			if message, err := w.Repo.Get(ctx, args.MessageId); err != nil {
				f := worker.Failure{
					"error": err.Error(),
				}
				return nil, &f
			} else {
				var ciphered string
				switch *task.Kind {
				case common.Rot1:
					ciphered = Rotate(message.Plain, 1)
					err = w.Repo.SetRot1(ctx, args.MessageId, ciphered)
				case common.Rot13:
					ciphered = Rotate(message.Plain, 13)
					err = w.Repo.SetRot13(ctx, args.MessageId, ciphered)
				case common.Base64:
					ciphered = Base64(message.Plain)
					err = w.Repo.SetBase64(ctx, args.MessageId, ciphered)
				case common.Md5:
					ciphered = Md5(message.Plain)
					err = w.Repo.SetMd5(ctx, args.MessageId, ciphered)
				default:
					err = fmt.Errorf("Unsupported rotation [%v]", task.Kind)
				}
				if err != nil {
					f := worker.Failure{
						"error": err.Error(),
					}
					return nil, &f
				} else {
					success := worker.Success{
						"ciphered": ciphered,
					}
					return &success, nil
				}
			}

		}
	}

}

var lower = []rune("abcdefghijklmnopqrstuvwxyz")
var upper = []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZ")

func Rotate(text string, shift uint) string {
	runes := []rune(text)
	for i, char := range runes {
		var lookup []rune
		if char >= upper[0] && char <= upper[len(upper)-1] {
			lookup = upper
		} else if char >= lower[0] && char <= lower[len(lower)-1] {
			lookup = lower
		}
		if lookup != nil {
			if char >= lookup[0] && char <= lookup[len(lookup)-1] {
				offset := uint(char) - uint(lookup[0])
				shiftedIdx := offset + shift
				if shiftedIdx > uint(len(lookup)-1) {
					shiftedIdx = shiftedIdx % uint(len(lookup))
				}
				runes[i] = lookup[shiftedIdx]
			}
		}
	}
	return string(runes)
}

func Base64(s string) string {
	return base64.StdEncoding.EncodeToString([]byte(s))
}

func Md5(s string) string {
	hasher := md5.New()
	hasher.Write([]byte(s))
	return hex.EncodeToString(hasher.Sum(nil))
}
