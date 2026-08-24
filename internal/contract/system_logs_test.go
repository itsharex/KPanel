package contract

import (
	"encoding/json"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestParseSystemLogQueryAcceptsOnlyDocumentedFilters(t *testing.T) {
	tests := []struct {
		name   string
		values url.Values
		want   SystemLogQuery
		field  string
	}{
		{
			name: "system defaults", values: url.Values{"source": {"system"}},
			want: SystemLogQuery{Source: "system", Limit: 100, Priority: "all"},
		},
		{
			name: "service filters", values: url.Values{
				"source": {"service"}, "limit": {"200"}, "priority": {"warning"},
			},
			want: SystemLogQuery{Source: "service", Limit: 200, Priority: "warning"},
		},
		{
			name: "security without journal priority", values: url.Values{"source": {"security"}, "limit": {"50"}},
			want: SystemLogQuery{Source: "security", Limit: 50, Priority: "all"},
		},
		{name: "unknown source", values: url.Values{"source": {"kernel"}}, field: "source"},
		{name: "unit is no longer accepted", values: url.Values{"source": {"service"}, "unit": {"ssh.service"}}, field: "unit"},
		{name: "invalid limit", values: url.Values{"source": {"system"}, "limit": {"51"}}, field: "limit"},
		{name: "zero-padded limit", values: url.Values{"source": {"system"}, "limit": {"050"}}, field: "limit"},
		{name: "signed limit", values: url.Values{"source": {"system"}, "limit": {"+50"}}, field: "limit"},
		{name: "priority on security", values: url.Values{"source": {"security"}, "priority": {"all"}}, field: "priority"},
		{name: "priority on login", values: url.Values{"source": {"login"}, "priority": {"error"}}, field: "priority"},
		{name: "unknown parameter", values: url.Values{"source": {"system"}, "path": {"/tmp/log"}}, field: "path"},
		{name: "duplicate source", values: url.Values{"source": {"system", "login"}}, field: "source"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, field, _ := ParseSystemLogQuery(test.values)
			if field != test.field {
				t.Fatalf("field = %q, want %q", field, test.field)
			}
			if field == "" && got != test.want {
				t.Fatalf("query = %#v, want %#v", got, test.want)
			}
		})
	}
}

func TestSystemActionResultCarriesMaintenanceTaskIdentity(t *testing.T) {
	want := SystemActionResult{
		Action: "log-cleanup", Status: "accepted", Changed: true,
		Message: "queued", TaskID: "20260824T090000.000000000Z",
		MaintenancePolicy: "retain-3d", AppliedAt: time.Date(2026, 8, 24, 9, 0, 0, 0, time.UTC),
	}
	encoded, err := json.Marshal(want)
	if err != nil {
		t.Fatal(err)
	}
	text := string(encoded)
	if !strings.Contains(text, `"taskId":"20260824T090000.000000000Z"`) ||
		!strings.Contains(text, `"maintenancePolicy":"retain-3d"`) {
		t.Fatalf("maintenance identity fields missing from JSON: %s", text)
	}
	var got SystemActionResult
	if err := json.Unmarshal(encoded, &got); err != nil {
		t.Fatal(err)
	}
	if got.TaskID != want.TaskID || got.Action != want.Action || got.MaintenancePolicy != want.MaintenancePolicy {
		t.Fatalf("maintenance identity did not round trip: %#v", got)
	}
}
