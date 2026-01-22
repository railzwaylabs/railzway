package pdf

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/johnfercher/maroto/v2"
	"github.com/johnfercher/maroto/v2/pkg/components/col"
	"github.com/johnfercher/maroto/v2/pkg/components/image"
	"github.com/johnfercher/maroto/v2/pkg/components/text"
	"github.com/johnfercher/maroto/v2/pkg/config"
	"github.com/johnfercher/maroto/v2/pkg/consts/align"
	"github.com/johnfercher/maroto/v2/pkg/consts/fontstyle"
	"github.com/johnfercher/maroto/v2/pkg/props"
)

type ReceiptData struct {
	InvoiceData // Embed InvoiceData as they share most fields
	DatePaid    string
}

func (p *PDFProvider) GenerateReceipt(ctx context.Context, data interface{}) (io.Reader, error) {
	receipt, ok := data.(ReceiptData)
	if !ok {
		// Fallback try casting to InvoiceData if that's what was passed
		if inv, ok := data.(InvoiceData); ok {
			receipt = ReceiptData{InvoiceData: inv}
		} else {
			return nil, fmt.Errorf("invalid data type for receipt PDF")
		}
	}

	cfg := config.NewBuilder().
		WithPageNumber(props.PageNumber{
			Pattern: "Page {current} of {total}",
			Place:   props.RightBottom,
		}).
		Build()

	m := maroto.New(cfg)

	m.AddRow(40,
		text.NewCol(6, "Receipt", props.Text{
			Size:  20,
			Style: fontstyle.Bold,
			Align: align.Left,
		}),
		image.NewFromFileCol(3, "assets/logo.png", props.Rect{
			Center:  false,
			Percent: 80,
			Left:    10, // Adjust alignment
		}),
	)

	// Receipt Meta
	m.AddRow(20,
		col.New(6).Add(
			text.New("Invoice number: "+receipt.InvoiceNumber, props.Text{Top: 0}),
			text.New("Date paid: "+receipt.DatePaid, props.Text{Top: 4}),
			text.New("Service period: "+receipt.ServicePeriod, props.Text{Top: 8}),
		),
		col.New(6),
	)

	// Addresses (Same as Invoice)
	m.AddRow(40,
		col.New(4).Add(
			text.New(receipt.OrgName, props.Text{Style: fontstyle.Bold}),
			text.New(receipt.OrgAddress, props.Text{Top: 5}),
			text.New(receipt.OrgEmail, props.Text{Top: 20}),
		),
		col.New(4).Add(
			text.New("Bill to", props.Text{Style: fontstyle.Bold}),
			text.New(receipt.BillToName, props.Text{Top: 5}),
			text.New(receipt.BillToAddress, props.Text{Top: 9}),
			text.New(receipt.BillToEmail, props.Text{Top: 25}),
		),
		col.New(4).Add(
			text.New("Ship to", props.Text{Style: fontstyle.Bold}),
			text.New(receipt.ShipToName, props.Text{Top: 5}),
			text.New(receipt.ShipToAddress, props.Text{Top: 9}),
		),
	)

	// Payment Confirmation Title
	m.AddRow(15,
		text.NewCol(12, receipt.Total+" paid on "+receipt.DatePaid, props.Text{
			Size:  14,
			Style: fontstyle.Bold,
			Top:   5,
		}),
	)

	// Bank Details
	m.AddRow(25,
		text.NewCol(12, receipt.BankDetails, props.Text{
			Size: 9,
			Top:  0,
		}),
	)

	// Table Header
	m.AddRow(10,
		text.NewCol(6, "Description", props.Text{Style: fontstyle.Bold, Size: 9}),
		text.NewCol(2, "Qty", props.Text{Style: fontstyle.Bold, Size: 9, Align: align.Right}),
		text.NewCol(2, "Unit price", props.Text{Style: fontstyle.Bold, Size: 9, Align: align.Right}),
		text.NewCol(2, "Amount", props.Text{Style: fontstyle.Bold, Size: 9, Align: align.Right}),
	)

	// Items
	for _, item := range receipt.Items {
		m.AddRow(15,
			text.NewCol(6, item.Description, props.Text{Size: 9}),
			text.NewCol(2, fmt.Sprintf("%d", item.Qty), props.Text{Size: 9, Align: align.Right}),
			text.NewCol(2, item.UnitPrice, props.Text{Size: 9, Align: align.Right}),
			text.NewCol(2, item.Amount, props.Text{Size: 9, Align: align.Right}),
		)
	}

	// Footer Totals
	m.AddRow(10,
		col.New(8),
		text.NewCol(2, "Subtotal", props.Text{Size: 9}),
		text.NewCol(2, receipt.Subtotal, props.Text{Size: 9, Align: align.Right}),
	)
	m.AddRow(10,
		col.New(8),
		text.NewCol(2, "Total", props.Text{Size: 9}),
		text.NewCol(2, receipt.Total, props.Text{Size: 9, Align: align.Right}),
	)

	doc, err := m.Generate()
	if err != nil {
		return nil, err
	}

	return bytes.NewReader(doc.GetBytes()), nil
}
