package ai

import (
	"context"
	"errors"

	"github.com/Forking-Around/SolarSense/internal/domain"
)

var ErrUnavailable = errors.New("AI extraction unavailable")

type Document struct {
	MIMEType string
	Data     []byte
}
type Extractor interface {
	ExtractBill(context.Context, Document) (domain.BillExtraction, error)
	Name() string
}

type Fallback struct{ Primary, Secondary Extractor }

func (f Fallback) ExtractBill(ctx context.Context, d Document) (domain.BillExtraction, error) {
	if f.Primary != nil {
		for range 2 {
			if v, e := f.Primary.ExtractBill(ctx, d); e == nil {
				return v, nil
			}
		}
	}
	if f.Secondary != nil {
		return f.Secondary.ExtractBill(ctx, d)
	}
	return domain.BillExtraction{}, ErrUnavailable
}

func (f Fallback) ExtractRoof(ctx context.Context, d Document) (domain.RoofPhotoObservation, error) {
	if p, ok := f.Primary.(interface{ ExtractRoof(context.Context, Document) (domain.RoofPhotoObservation, error) }); ok {
		for range 2 { if v, err := p.ExtractRoof(ctx, d); err == nil { return v, nil } }
	}
	if p, ok := f.Secondary.(interface{ ExtractRoof(context.Context, Document) (domain.RoofPhotoObservation, error) }); ok { return p.ExtractRoof(ctx, d) }
	return domain.RoofPhotoObservation{}, ErrUnavailable
}
