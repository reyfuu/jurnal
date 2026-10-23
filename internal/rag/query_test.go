package rag

import (
	"strings"
	"testing"
)

func TestNaturalQuestionQuery(t *testing.T) {
	if got := searchQuery("Bagaimana pajak untuk UMKM?"); got != "pajak | umkm" {
		t.Fatalf("question terms: %q", got)
	}
	if got := searchQuery("apa itu dan yang"); got != "" {
		t.Fatalf("stop words queried: %q", got)
	}
	got := searchQuery(`PP 55/2022 '); DROP TABLE x; --`)
	if strings.ContainsAny(got, "';)(") {
		t.Fatalf("tsquery operators injected: %q", got)
	}
}
