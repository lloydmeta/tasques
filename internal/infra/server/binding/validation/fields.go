package validation

import (
	"github.com/gin-gonic/gin/binding"
	"github.com/robfig/cron/v3"

	"github.com/lloydmeta/tasques/internal/domain/queue"
	"github.com/lloydmeta/tasques/internal/domain/task/recurring"

	"github.com/rs/zerolog/log"
	"gopkg.in/go-playground/validator.v9"
)

func SetUpValidators(scheduleParser recurring.ScheduleParser) {
	log.Info().Msg("Setting up custom validators")
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		err := v.RegisterValidation(QueueNameValidatorTag, QueueNameValidator)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to set up Queue name validator")
		}
		err = v.RegisterValidation(ScheduleExpressionValidatorTag, makeScheduleExpressionValidator(scheduleParser))
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to set up ScheduledExpression validator")
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

var ScheduleExpressionValidatorTag = "scheduleExpression"

func makeScheduleExpressionValidator(parser recurring.ScheduleParser) validator.Func {
	return func(fl validator.FieldLevel) bool {
		scheduleExpression, ok := fl.Field().Interface().(recurring.ScheduleExpression)
		if ok {
			if _, err := parser.Parse(string(scheduleExpression)); err != nil {
				fl.Field()
				return false
			}
		}
		return true
	}
}

// For testing
type TestStandardParser struct{}

func (s TestStandardParser) Parse(spec string) (recurring.Schedule, error) {
	return cron.ParseStandard(spec)
}
