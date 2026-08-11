package contract

import "testing"

func TestValidateSystemTuningAction(t *testing.T) {
	valid := SystemTuningActionRequest{Action: "apply", Items: append([]string(nil), SystemTuningItemIDs...), ExpectedResourceVersion: string(make([]byte, 0))}
	valid.ExpectedResourceVersion = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	if field, detail := ValidateSystemTuningAction(&valid); field != "" {
		t.Fatalf("valid request rejected: %s %s", field, detail)
	}
	duplicate := valid
	duplicate.Items = []string{"bbr", "bbr"}
	if field, _ := ValidateSystemTuningAction(&duplicate); field != "items" {
		t.Fatalf("duplicate items field = %q", field)
	}
	unknown := valid
	unknown.Items = []string{"shell"}
	if field, _ := ValidateSystemTuningAction(&unknown); field != "items" {
		t.Fatalf("unknown items field = %q", field)
	}
}
