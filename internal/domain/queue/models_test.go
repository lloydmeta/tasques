package queue

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func namePtr(name Name) *Name {
	return &name
}

func TestQueueFromString(t *testing.T) {
	type args struct {
		s string
	}
	tests := []struct {
		name    string
		args    args
		want    *Name
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
			name: "must not have ':'",
			args: args{
				"index:name",
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
			want:    namePtr(Name("my-little-queue")),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NameFromString(tt.args.s)
			if (err != nil) != tt.wantErr {
				t.Errorf("NameFromString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.EqualValues(t, tt.want, got)
		})
	}
}
