package ai

import (
	"context"
	"errors"
	"testing"

	"github.com/Forking-Around/SolarSense/internal/domain"
)

type fake struct {
	name     string
	failures *int
	result   domain.BillExtraction
}

func (f fake) Name() string { return f.name }
func (f fake) ExtractBill(context.Context, Document) (domain.BillExtraction, error) {
	if *f.failures > 0 {
		*f.failures--
		return domain.BillExtraction{}, errors.New("temporary")
	}
	return f.result, nil
}
func TestFallbackAfterTwoPrimaryFailures(t *testing.T) {
	n, m := 2, 0
	want := domain.BillExtraction{Utility: "test utility"}
	v, err := (Fallback{Primary: fake{"primary", &n, domain.BillExtraction{}}, Secondary: fake{"secondary", &m, want}}).ExtractBill(context.Background(), Document{})
	if err != nil || v.Utility != want.Utility {
		t.Fatalf("fallback failed: %#v %v", v, err)
	}
}
func TestDecodeRejectsInvalidConfidence(t *testing.T) {
	if _, err := decodeBill(`{"amount":100,"confidence":{"amount":2}}`); err == nil {
		t.Fatal("expected validation error")
	}
}
