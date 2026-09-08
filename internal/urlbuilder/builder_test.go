package urlbuilder

import (
	"net/url"
	"testing"
)

func TestBuildAddsDynamicAndConfiguredParameters(t *testing.T) {
	builder := New("https://example.test/search?existing=yes", map[string]string{
		"morphology": "on",
		"empty":      "",
	})
	raw, err := builder.Build("6317024749", "631701001")
	if err != nil {
		t.Fatal(err)
	}
	u, _ := url.Parse(raw)
	if got := u.Query().Get("inn"); got != "6317024749" {
		t.Fatalf("inn = %q", got)
	}
	if got := u.Query().Get("kpp"); got != "631701001" {
		t.Fatalf("kpp = %q", got)
	}
	if got := u.Query().Get("searchString"); got != "6317024749" {
		t.Fatalf("searchString = %q", got)
	}
	if _, exists := u.Query()["empty"]; exists {
		t.Fatal("empty configured parameter must be omitted")
	}
}
