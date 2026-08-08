package systeminfo

import (
	"strings"
	"testing"
)

func TestNormalizeProcessQueryKeepsFixedBounds(t *testing.T) {
	query, err := NormalizeProcessQuery(ProcessQuery{Search: " nginx ", Sort: "MEMORY", Order: "asc"})
	if err != nil {
		t.Fatal(err)
	}
	if query.Search != "nginx" || query.Sort != "memory" || query.Order != "asc" || query.Limit != DefaultProcessResults {
		t.Fatalf("normalized query=%#v", query)
	}
	for _, input := range []ProcessQuery{
		{Search: strings.Repeat("x", MaxProcessSearchBytes+1)},
		{Sort: "command"},
		{Order: "sideways"},
		{Limit: MaxProcessResults + 1},
	} {
		if _, err := NormalizeProcessQuery(input); err == nil {
			t.Fatalf("invalid query accepted: %#v", input)
		}
	}
}
