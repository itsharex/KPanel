package sites

import "testing"

func TestNormalizeRecipeInputUsesFixedKejilionSelectors(t *testing.T) {
	domain, selector, err := normalizeRecipeInput(SiteInput{
		PrimaryDomain: "forum.example.com",
		Type:          "recipe",
		Recipe:        "discuz",
	})
	if err != nil || domain != "forum.example.com" || selector != "3" {
		t.Fatalf("normalizeRecipeInput() = %q, %q, %v", domain, selector, err)
	}
	for _, input := range []SiteInput{
		{PrimaryDomain: "forum.example.com", Type: "recipe", Recipe: "unknown"},
		{PrimaryDomain: "forum.example.com", Type: "recipe", Recipe: "discuz", Aliases: []string{"www.example.com"}},
		{PrimaryDomain: "forum.example.com", Type: "php", Recipe: "discuz"},
	} {
		if _, _, err := normalizeRecipeInput(input); err == nil {
			t.Fatalf("unsafe recipe input was accepted: %#v", input)
		}
	}
}
