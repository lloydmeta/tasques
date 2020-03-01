package recurring

import (
	"testing"

	"github.com/lloydmeta/tasques/internal/domain/task"
)

func TestAlreadyExists_Error(t *testing.T) {
	type fields struct {
		ID task.RecurringTaskId
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "error string",
			fields: fields{
				ID: "id",
			},
			want: "Id already exists [id]",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := AlreadyExists{
				ID: tt.fields.ID,
			}
			if got := e.Error(); got != tt.want {
				t.Errorf("Error() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAlreadyExists_Id(t *testing.T) {
	type fields struct {
		ID task.RecurringTaskId
	}
	tests := []struct {
		name   string
		fields fields
		want   task.RecurringTaskId
	}{
		{
			name: "id",
			fields: fields{
				ID: "id",
			},
			want: "id",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := AlreadyExists{
				ID: tt.fields.ID,
			}
			if got := e.Id(); got != tt.want {
				t.Errorf("Id() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNotFound_Error(t *testing.T) {
	type fields struct {
		ID task.RecurringTaskId
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "error string",
			fields: fields{
				ID: "id",
			},
			want: "Could not find [id]",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := NotFound{
				ID: tt.fields.ID,
			}
			if got := e.Error(); got != tt.want {
				t.Errorf("Error() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNotFound_Id(t *testing.T) {
	type fields struct {
		ID task.RecurringTaskId
	}
	tests := []struct {
		name   string
		fields fields
		want   task.RecurringTaskId
	}{
		{
			name: "id",
			fields: fields{
				ID: "id",
			},
			want: "id",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := NotFound{
				ID: tt.fields.ID,
			}
			if got := e.Id(); got != tt.want {
				t.Errorf("Id() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInvalidVersion_Error(t *testing.T) {
	type fields struct {
		ID task.RecurringTaskId
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "error string",
			fields: fields{
				ID: "id",
			},
			want: "Version provided did not match persisted version for [id]",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := InvalidVersion{
				ID: tt.fields.ID,
			}
			if got := e.Error(); got != tt.want {
				t.Errorf("Error() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInvalidVersion_Id(t *testing.T) {
	type fields struct {
		ID task.RecurringTaskId
	}
	tests := []struct {
		name   string
		fields fields
		want   task.RecurringTaskId
	}{
		{
			name: "id",
			fields: fields{
				ID: "id",
			},
			want: "id",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := InvalidVersion{
				ID: tt.fields.ID,
			}
			if got := e.Id(); got != tt.want {
				t.Errorf("Id() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInvalidPersistedData_Error(t *testing.T) {
	type fields struct {
		PersistedData interface{}
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "error string",
			fields: fields{
				PersistedData: 1,
			},
			want: "Invalid persisted data [1]",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := InvalidPersistedData{
				PersistedData: tt.fields.PersistedData,
			}
			if got := e.Error(); got != tt.want {
				t.Errorf("Error() = %v, want %v", got, tt.want)
			}
		})
	}
}
