package contract

import "testing"

func TestValidateTrafficShutdownAction(t *testing.T) {
	version := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	rx, tx, day := uint64(100), uint64(200), 5
	tests := []struct {
		name  string
		input TrafficShutdownActionRequest
		field string
	}{
		{name: "enable", input: TrafficShutdownActionRequest{Action: "enable", ExpectedResourceVersion: version, RXThresholdGiB: &rx, TXThresholdGiB: &tx, ResetDay: &day}},
		{name: "disable", input: TrafficShutdownActionRequest{Action: "disable", ExpectedResourceVersion: version}},
		{name: "missing rx", input: TrafficShutdownActionRequest{Action: "enable", ExpectedResourceVersion: version, TXThresholdGiB: &tx, ResetDay: &day}, field: "rxThresholdGiB"},
		{name: "disable fields", input: TrafficShutdownActionRequest{Action: "disable", ExpectedResourceVersion: version, RXThresholdGiB: &rx}, field: "action"},
		{name: "bad day", input: TrafficShutdownActionRequest{Action: "enable", ExpectedResourceVersion: version, RXThresholdGiB: &rx, TXThresholdGiB: &tx, ResetDay: intPointer(32)}, field: "resetDay"},
		{name: "bad version", input: TrafficShutdownActionRequest{Action: "disable", ExpectedResourceVersion: "stale"}, field: "expectedResourceVersion"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			field, _ := ValidateTrafficShutdownAction(&test.input)
			if field != test.field {
				t.Fatalf("field=%q want=%q", field, test.field)
			}
		})
	}
}

func intPointer(value int) *int { return &value }
