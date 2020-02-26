package validation

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/go-playground/validator.v9"

	"github.com/lloydmeta/tasques/internal/domain/queue"
	"github.com/lloydmeta/tasques/internal/domain/task/recurring"
)

func Test_QueueNameValidator(t *testing.T) {
	validate := validator.New()
	_ = validate.RegisterValidation(QueueNameValidatorTag, QueueNameValidator)
	type args struct {
		name queue.Name
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "must not have illegal chars",
			args: args{
				"index?name",
			},
			wantErr: true,
		},
		{
			name: "must not have '#'",
			args: args{
				"index#name",
			},
			wantErr: true,
		},
		{
			name: "must not start with _",
			args: args{
				"_indexname",
			},
			wantErr: true,
		},
		{
			name: "must not start with -",
			args: args{
				"-indexname",
			},
			wantErr: true,
		},
		{
			name: "must not start with +",
			args: args{
				"+indexname",
			},
			wantErr: true,
		},
		{
			name: "must be lower case",
			args: args{
				"INDEXNAME",
			},
			wantErr: true,
		},
		{
			name: "must not be '..'",
			args: args{
				"..",
			},
			wantErr: true,
		},
		{
			name: "must not be '.'",
			args: args{
				".",
			},
			wantErr: true,
		},
		{
			name: "should work",
			args: args{
				"my-little-queue",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validate.Var(tt.args.name, QueueNameValidatorTag)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func Test_ScheduleExpressionValidator(t *testing.T) {
	validate := validator.New()
	_ = validate.RegisterValidation(ScheduleExpressionValidatorTag, makeScheduleExpressionValidator(TestStandardParser{}))
	type args struct {
		expression recurring.ScheduleExpression
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "should not support random stuff",
			args: args{
				"whats up dog",
			},
			wantErr: true,
		},
		{
			name: "should support traditional cron 1",
			args: args{
				"* * * * *",
			},
			wantErr: false,
		},
		{
			name: "should support traditional cron 2",
			args: args{
				"30 3-6,20-23 * * *",
			},
			wantErr: false,
		},
		{
			name: "should support traditional cron 3",
			args: args{
				"0 */2 * * *",
			},
			wantErr: false,
		},
		{
			name: "should support extended cron 1",
			args: args{
				"@hourly",
			},
			wantErr: false,
		},
		{
			name: "should support extended cron 2",
			args: args{
				"@every 1h30m",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validate.Var(tt.args.expression, ScheduleExpressionValidatorTag)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
