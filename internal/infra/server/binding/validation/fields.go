package validation

import (
	"github.com/gin-gonic/gin/binding"
	"github.com/rs/zerolog/log"
	"gopkg.in/go-playground/validator.v9"
	"github.com/lloydmeta/tasques/internal/domain/queue"
)

func SetUpValidators() {
	log.Info().Msg("Setting up custom validators")
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		err := v.RegisterValidation(QueueNameValidatorTag, QueueNameValidator)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to set up validators")
		}
	}
}

var QueueNameValidatorTag = "queueName"
var QueueNameValidator validator.Func = func(fl validator.FieldLevel) bool {
	queueName, ok := fl.Field().Interface().(queue.Name)
	if ok {
		if _, err := queue.NameFromString(string(queueName)); err != nil {
			fl.Field()
			return false
		}
	}
	return true

}
