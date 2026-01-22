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

type InvoiceData struct {
	OrgName       string
	OrgAddress    string
	OrgEmail      string
	InvoiceNumber string
	IssueDate     string
	DueDate       string
	ServicePeriod string

	BillToName    string
	BillToAddress string
	BillToEmail   string

	ShipToName    string
	ShipToAddress string

	TotalDue    string
	BankDetails string

	Items []InvoiceItem

	Subtotal  string
	Total     string
	AmountDue string
}

type InvoiceItem struct {
	Description string
	Qty         int
	UnitPrice   string
	Amount      string
}

type PDFProvider struct{}

func New() Provider {
	return &PDFProvider{}
}

func (p *PDFProvider) GenerateInvoice(ctx context.Context, data interface{}) (io.Reader, error) {
	invoice, ok := data.(InvoiceData)
	if !ok {
		return nil, fmt.Errorf("invalid data type for invoice PDF")
	}

	cfg := config.NewBuilder().
		WithPageNumber(props.PageNumber{
			Pattern: "Page {current} of {total}",
			Place:   props.RightBottom,
		}).
		Build()

	m := maroto.New(cfg)

	m.AddRow(40,
		image.NewFromFileCol(3, "assets/logo.png", props.Rect{
			Center:  false,
			Percent: 80,
		}),
		col.New(9), // Spacer
	)

	m.AddRow(10,
		text.NewCol(12, "Invoice", props.Text{
			Size:  20,
			Style: fontstyle.Bold,
			Align: align.Left,
		}),
	)

	// Invoice Meta
	m.AddRow(20,
		col.New(6).Add(
			text.New("Invoice number: "+invoice.InvoiceNumber, props.Text{Top: 0}),
			text.New("Date of issue: "+invoice.IssueDate, props.Text{Top: 4}),
			text.New("Date due: "+invoice.DueDate, props.Text{Top: 8}),
			text.New("Service period: "+invoice.ServicePeriod, props.Text{Top: 12}),
		),
		col.New(6),
	)

	// Addresses
	m.AddRow(40,
		col.New(4).Add(
			text.New(invoice.OrgName, props.Text{Style: fontstyle.Bold}),
			text.New(invoice.OrgAddress, props.Text{Top: 5}),
			text.New(invoice.OrgEmail, props.Text{Top: 20}),
		),
		col.New(4).Add(
			text.New("Bill to", props.Text{Style: fontstyle.Bold}),
			text.New(invoice.BillToName, props.Text{Top: 5}),
			text.New(invoice.BillToAddress, props.Text{Top: 9}),
			text.New(invoice.BillToEmail, props.Text{Top: 25}),
		),
		col.New(4).Add(
			text.New("Ship to", props.Text{Style: fontstyle.Bold}),
			text.New(invoice.ShipToName, props.Text{Top: 5}),
			text.New(invoice.ShipToAddress, props.Text{Top: 9}),
		),
	)

	// Summary Title
	m.AddRow(15,
		text.NewCol(12, invoice.TotalDue, props.Text{
			Size:  14,
			Style: fontstyle.Bold,
			Top:   5,
		}),
	)

	// Bank Details
	m.AddRow(25,
		text.NewCol(12, invoice.BankDetails, props.Text{
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

	m.AddRow(1, col.New(12).Add(
	// Line
	))

	// Items
	for _, item := range invoice.Items {
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
		text.NewCol(2, invoice.Subtotal, props.Text{Size: 9, Align: align.Right}),
	)
	m.AddRow(10,
		col.New(8),
		text.NewCol(2, "Total", props.Text{Size: 9}),
		text.NewCol(2, invoice.Total, props.Text{Size: 9, Align: align.Right}),
	)
	m.AddRow(10,
		col.New(8),
		text.NewCol(2, "Amount due", props.Text{Style: fontstyle.Bold, Size: 9}),
		text.NewCol(2, invoice.AmountDue, props.Text{Style: fontstyle.Bold, Size: 9, Align: align.Right}),
	)

	doc, err := m.Generate()
	if err != nil {
		return nil, err
	}

	return bytes.NewReader(doc.GetBytes()), nil
}
