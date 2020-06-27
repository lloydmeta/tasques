package task

import (
	"fmt"
	"testing"
	"time"

	"github.com/lloydmeta/tasques/internal/domain/queue"
	"github.com/lloydmeta/tasques/internal/domain/worker"
)

func TestNotFound_Error(t *testing.T) {
	type fields struct {
		ID        Id
		QueueName queue.Name
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "error string",
			fields: fields{
				ID:        "some id",
				QueueName: "q",
			},
			want: "Could not find [some id] in queue [q]",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := NotFound{
				ID:        tt.fields.ID,
				QueueName: tt.fields.QueueName,
			}
			if got := e.Error(); got != tt.want {
				t.Errorf("Error() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNotFound_Id(t *testing.T) {
	type fields struct {
		ID        Id
		QueueName queue.Name
	}
	tests := []struct {
		name   string
		fields fields
		want   Id
	}{
		{
			name: "id",
			fields: fields{
				ID:        "hello",
				QueueName: "q",
			},
			want: "hello",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := NotFound{
				ID:        tt.fields.ID,
				QueueName: tt.fields.QueueName,
			}
			if got := e.Id(); got != tt.want {
				t.Errorf("Id() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInvalidVersion_Error(t *testing.T) {
	type fields struct {
		ID Id
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
		ID Id
	}
	tests := []struct {
		name   string
		fields fields
		want   Id
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
	var tests = []struct {
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

func TestNotOwnedByWorker_Error(t *testing.T) {
	type fields struct {
		ID                Id
		WantedOwnerWorker worker.Id
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "error string",
			fields: fields{
				ID:                "id",
				WantedOwnerWorker: "werker",
			},
			want: "The Task was not owned by the worker [werker]",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := NotOwnedByWorker{
				ID:                tt.fields.ID,
				WantedOwnerWorker: tt.fields.WantedOwnerWorker,
			}
			if got := n.Error(); got != tt.want {
				t.Errorf("Error() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNotOwnedByWorker_Id(t *testing.T) {
	type fields struct {
		ID                Id
		WantedOwnerWorker worker.Id
	}
	tests := []struct {
		name   string
		fields fields
		want   Id
	}{
		{
			name: "id",
			fields: fields{
				ID:                "id",
				WantedOwnerWorker: "werk",
			},
			want: "id",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := NotOwnedByWorker{
				ID:                tt.fields.ID,
				WantedOwnerWorker: tt.fields.WantedOwnerWorker,
			}
			if got := n.Id(); got != tt.want {
				t.Errorf("Id() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNotClaimed_Error(t *testing.T) {
	type fields struct {
		ID    Id
		State State
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "error string",
			fields: fields{
				ID:    "id",
				State: QUEUED,
			},
			want: "The Task is not currently Claimed so cannot be reported on: [id]",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := NotClaimed{
				ID:    tt.fields.ID,
				State: tt.fields.State,
			}
			if got := a.Error(); got != tt.want {
				t.Errorf("Error() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNotClaimed_Id(t *testing.T) {
	type fields struct {
		ID    Id
		State State
	}
	tests := []struct {
		name   string
		fields fields
		want   Id
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
			a := NotClaimed{
				ID:    tt.fields.ID,
				State: tt.fields.State,
			}
			if got := a.Id(); got != tt.want {
				t.Errorf("Id() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReportFromThePast_Error(t *testing.T) {
	now := time.Now().UTC()
	type fields struct {
		ID                  Id
		AttemptedReportTime time.Time
		ExistingReportTime  time.Time
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "error string",
			fields: fields{
				ID:                  "id",
				AttemptedReportTime: now,
				ExistingReportTime:  now.Add(1 * time.Hour),
			},
			want: fmt.Sprintf("The Task [id] was already reported on at [%v], which overrides the current one reported at [%v]", now.Add(1*time.Hour), now),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := ReportFromThePast{
				ID:                  tt.fields.ID,
				AttemptedReportTime: tt.fields.AttemptedReportTime,
				ExistingReportTime:  tt.fields.ExistingReportTime,
			}
			if got := a.Error(); got != tt.want {
				t.Errorf("Error() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReportFromThePast_Id(t *testing.T) {
	type fields struct {
		ID                  Id
		AttemptedReportTime time.Time
		ExistingReportTime  time.Time
	}
	tests := []struct {
		name   string
		fields fields
		want   Id
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
			a := ReportFromThePast{
				ID:                  tt.fields.ID,
				AttemptedReportTime: tt.fields.AttemptedReportTime,
				ExistingReportTime:  tt.fields.ExistingReportTime,
			}
			if got := a.Id(); got != tt.want {
				t.Errorf("Id() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUnclaimable_Error(t *testing.T) {
	type fields struct {
		ID           Id
		CurrentState State
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "error string",
			fields: fields{
				ID:           "id",
				CurrentState: DEAD,
			},
			want: "The Task [id] is in state [dead], which is not claimable",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := Unclaimable{
				ID:           tt.fields.ID,
				CurrentState: tt.fields.CurrentState,
			}
			if got := a.Error(); got != tt.want {
				t.Errorf("Error() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUnclaimable_Id(t *testing.T) {
	type fields struct {
		ID           Id
		CurrentState State
	}
	tests := []struct {
		name   string
		fields fields
		want   Id
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
			a := Unclaimable{
				ID:           tt.fields.ID,
				CurrentState: tt.fields.CurrentState,
			}
			if got := a.Id(); got != tt.want {
				t.Errorf("Id() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAlreadyExists_Error(t *testing.T) {
	type fields struct {
		ID Id
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
			want: fmt.Sprintf("Task with Id [%v] already exists ", "id"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := AlreadyExists{
				ID: tt.fields.ID,
			}
			if got := a.Error(); got != tt.want {
				t.Errorf("Error() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAlreadyExists_Id(t *testing.T) {
	type fields struct {
		ID Id
	}
	tests := []struct {
		name   string
		fields fields
		want   Id
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
			a := AlreadyExists{
				ID: tt.fields.ID,
			}
			if got := a.Id(); got != tt.want {
				t.Errorf("Id() = %v, want %v", got, tt.want)
			}
		})
	}
}
