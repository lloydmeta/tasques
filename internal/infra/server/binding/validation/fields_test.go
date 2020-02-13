package validation

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/go-playground/validator.v9"

	"github.com/lloydmeta/tasques/internal/domain/queue"
)

func TestQueueNameValidator(t *testing.T) {
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
