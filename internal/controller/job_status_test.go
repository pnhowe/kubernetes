package controller

import (
	"testing"
	"time"

	cclient "github.com/t3kton/contractor_goclient"
	contractorv1 "t3kton.com/api/v1"
)

func TestUpdateJobStatus(t *testing.T) {
	str := func(s string) *string { return &s }
	now := time.Now()

	cases := []struct {
		name          string
		status        *[]map[string]any
		wantProgress  string
		wantRemaining string
	}{
		{"nil status", nil, "0", ""},
		{"empty status", &[]map[string]any{}, "0", ""},
		{"no params", &[]map[string]any{{"percent": 0.0, "operation": "Scope", "parameters": nil}}, "0.0", ""},
		{"in progress", &[]map[string]any{
			{"percent": 12.5, "operation": "Scope", "parameters": map[string]any{"time_remaining": "00:07"}},
			{"percent": 0.0, "operation": "Function", "parameters": map[string]any{"name": "delay"}},
		}, "12.5", "00:07"},
		{"overrun", &[]map[string]any{{"percent": 50.0, "operation": "Scope", "parameters": map[string]any{"time_remaining": "-00:02"}}}, "50.0", "-00:02"},
		{"done", &[]map[string]any{{"percent": 100.0, "operation": "Scope", "parameters": nil}}, "100.0", "00:00"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			job := &cclient.ForemanStructureJob{
				State: str("running"), ScriptName: str("create"), Message: str("msg"),
				CanStart: str("true"), Created: &now, Updated: &now, Status: tc.status,
			}
			status := &contractorv1.StructureStatus{}
			updateJobStatus(job, status)
			if status.Job.Progress != tc.wantProgress {
				t.Errorf("progress = %q, want %q", status.Job.Progress, tc.wantProgress)
			}
			if status.Job.MaxTimeRemaining != tc.wantRemaining {
				t.Errorf("remaining = %q, want %q", status.Job.MaxTimeRemaining, tc.wantRemaining)
			}
		})
	}
}
