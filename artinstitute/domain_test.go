package artinstitute

import (
	"testing"
)

func TestDomainInfo(t *testing.T) {
	info := Domain{}.Info()
	if info.Scheme != "artinstitute" {
		t.Errorf("Scheme = %q, want artinstitute", info.Scheme)
	}
	if len(info.Hosts) == 0 {
		t.Error("Hosts is empty")
	}
	if info.Identity.Binary != "artinstitute" {
		t.Errorf("Identity.Binary = %q, want artinstitute", info.Identity.Binary)
	}
}

func TestClassify(t *testing.T) {
	typ, id, err := Domain{}.Classify("27992")
	if err != nil {
		t.Fatalf("Classify: %v", err)
	}
	if typ != "artwork" {
		t.Errorf("type = %q, want artwork", typ)
	}
	if id != "27992" {
		t.Errorf("id = %q, want 27992", id)
	}
}

func TestLocate(t *testing.T) {
	got, err := Domain{}.Locate("artwork", "27992")
	if err != nil {
		t.Fatalf("Locate: %v", err)
	}
	if got == "" {
		t.Error("Locate returned empty URL")
	}
}
