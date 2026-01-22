package render

import (
	"bytes"
	"fmt"
	"html/template"
	"regexp"
	"strings"
	"time"
)

const invoiceHTMLTemplate = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <title>Invoice {{.Invoice.Number}}</title>
  <style>
    :root {
      --primary: {{.Template.PrimaryColor}};
      --font: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
    }
    * { box-sizing: border-box; }
    body {
      margin: 0;
      padding: 40px;
      font-family: var(--font);
      color: #1a1f36;
      background: #f7f9fc;
      -webkit-font-smoothing: antialiased;
    }
    .invoice-card {
      background: #ffffff;
      max-width: 760px;
      margin: 0 auto;
      padding: 60px;
      box-shadow: 0 2px 5px rgba(0,0,0,0.04);
      border-radius: 4px;
    }
    .header {
      display: flex;
      justify-content: space-between;
      margin-bottom: 40px;
    }
    .header-left h1 {
      margin: 0;
      font-size: 24px;
      font-weight: 700;
      color: #1a1f36;
    }
    .header-right {
      text-align: right;
      font-weight: 600;
      color: #8792a2;
      font-size: 16px;
    }
    
    .meta-grid {
      display: flex;
      justify-content: space-between;
      margin-bottom: 40px;
    }
    .col {
      flex: 1;
    }
    .label {
      font-size: 11px;
      text-transform: uppercase;
      color: #8792a2;
      margin-bottom: 6px;
      font-weight: 600;
      letter-spacing: 0.3px;
    }
    .value {
      font-size: 14px;
      line-height: 1.5;
      color: #1a1f36;
    }
    
    .amount-section {
      margin-bottom: 40px;
    }
    .amount-large {
      font-size: 32px;
      font-weight: 700;
      color: #1a1f36;
      margin-bottom: 4px;
    }
    .pay-link {
      font-size: 13px;
      color: #006aff;
      text-decoration: none;
      font-weight: 500;
    }
    
    table {
      width: 100%;
      border-collapse: collapse;
      margin-bottom: 30px;
    }
    th {
      text-align: left;
      text-transform: uppercase;
      font-size: 11px;
      color: #8792a2;
      border-bottom: 1px solid #e3e8ee;
      padding: 10px 0;
      font-weight: 600;
      letter-spacing: 0.3px;
    }
    td {
      padding: 16px 0;
      border-bottom: 1px solid #e3e8ee;
      font-size: 14px;
      color: #1a1f36;
      vertical-align: top;
    }
    .td-right { text-align: right; }
    
    .item-title { font-weight: 600; margin-bottom: 2px; }
    .item-sub { font-size: 12px; color: #697386; }
    
    .totals {
      width: 100%;
      display: flex;
      flex-direction: column;
      align-items: flex-end;
    }
    .total-row {
      display: flex;
      justify-content: space-between;
      width: 250px;
      padding: 6px 0;
      font-size: 14px;
    }
    .total-label { color: #697386; }
    .total-value { color: #1a1f36; text-align: right; font-weight: 500; }
    .total-final {
      border-top: 1px solid #e3e8ee;
      margin-top: 10px;
      padding-top: 10px;
      font-weight: 700;
      font-size: 16px;
      color: #1a1f36;
    }
    
    .footer {
      margin-top: 60px;
      font-size: 12px;
      color: #8792a2;
      border-top: 1px solid #e3e8ee;
      padding-top: 20px;
    }
    
    /* Spacer utility */
    .mt-4 { margin-top: 4px; }
  </style>
</head>
<body>
  <div class="invoice-card">
    <!-- Header -->
    <div class="header">
      <div class="header-left">
        <h1>Invoice</h1>
        <div class="label mt-4" style="margin-top: 12px;">Invoice number</div>
        <div class="value">{{.Invoice.Number}}</div>
      </div>
      <div class="header-right">
        {{if .Template.LogoURL}}
          <img src="{{.Template.LogoURL}}" style="max-height: 40px;" alt="{{.Template.CompanyName}}">
        {{else}}
          {{.Template.CompanyName}}
        {{end}}
      </div>
    </div>

    <!-- Metadata Grid -->
    <div class="meta-grid">
      <div class="col">
        <div class="label">Bill to</div>
        <div class="value">
          <strong>{{.Customer.Name}}</strong><br>
          {{.Customer.Email}}<br>
          <!-- Address would go here -->
        </div>
      </div>
      <div class="col" style="flex: 0 0 200px;">
        <div class="label">Date due</div>
        <div class="value">{{formatDate .Invoice.DueAt}}</div>
        
        <div class="label" style="margin-top: 16px;">Date issued</div>
        <div class="value">{{formatDate .Invoice.IssuedAt}}</div>
      </div>
    </div>

    <!-- Amount Due -->
    <div class="amount-section">
      <div class="amount-large">{{formatMoney .Invoice.SubtotalAmount .Invoice.Currency}}</div>
      <div class="value" style="color: #697386; margin-bottom: 8px;">due {{formatDate .Invoice.DueAt}}</div>
      <a href="#" class="pay-link" onclick="return false;">Pay online &rarr;</a>
    </div>

    <!-- Line Items -->
    <table>
      <thead>
        <tr>
          <th style="width: 50%;">Description</th>
          <th class="td-right">Qty</th>
          <th class="td-right">Unit Price</th>
          <th class="td-right">Amount</th>
        </tr>
      </thead>
      <tbody>
        {{range .Items}}
        <tr>
          <td>
            <div class="item-title">{{.Title}}</div>
            {{if .SubTitle}}<div class="item-sub">{{.SubTitle}}</div>{{end}}
          </td>
          <td class="td-right">{{formatQuantity .Quantity}}</td>
          <td class="td-right">{{formatMoney .UnitPrice $.Invoice.Currency}}</td>
          <td class="td-right" style="font-weight: 500;">{{formatMoney .Amount $.Invoice.Currency}}</td>
        </tr>
        {{end}}
      </tbody>
    </table>

    <!-- Totals -->
    <div class="totals">
      <div class="total-row">
        <span class="total-label">Subtotal</span>
        <span class="total-value">{{formatMoney .Invoice.SubtotalAmount .Invoice.Currency}}</span>
      </div>
       <!-- Tax/Discounts would go here if available in view -->
      <div class="total-row total-final">
        <span class="total-label" style="color: #1a1f36;">Total</span>
        <span class="total-value">{{formatMoney .Invoice.SubtotalAmount .Invoice.Currency}}</span>
      </div>
      <div class="total-row">
        <span class="total-label">Amount due</span>
        <span class="total-value">{{formatMoney .Invoice.SubtotalAmount .Invoice.Currency}}</span>
      </div>
    </div>

    <!-- Footer -->
    {{if .Template.FooterNotes}}
    <div class="footer">
      {{.Template.FooterNotes}}
      {{if .Template.FooterLegal}}<br><br>{{.Template.FooterLegal}}{{end}}
    </div>
    {{end}}
    
  </div>
</body>
</html>
`

var (
	hexColorPattern  = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)
	fontFamilyFilter = regexp.MustCompile(`^[A-Za-z0-9 \-]+$`)
)

type HTMLRenderer struct {
	tpl *template.Template
}

func NewRenderer() Renderer {
	funcs := template.FuncMap{
		"formatMoney":    formatMoney,
		"formatDate":     formatDate,
		"formatQuantity": formatQuantity,
	}
	return &HTMLRenderer{
		tpl: template.Must(template.New("invoice").Funcs(funcs).Parse(invoiceHTMLTemplate)),
	}
}

func (r *HTMLRenderer) RenderHTML(input RenderInput) (string, error) {
	input.Template.PrimaryColor = sanitizeColor(input.Template.PrimaryColor)
	input.Template.FontFamily = sanitizeFont(input.Template.FontFamily)
	if input.Template.CompanyName == "" {
		input.Template.CompanyName = "Invoice"
	}

	var buf bytes.Buffer
	if err := r.tpl.Execute(&buf, input); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func formatMoney(amount int64, currency string) string {
	currency = strings.ToUpper(strings.TrimSpace(currency))
	if currency == "" {
		currency = "USD"
	}
	value := float64(amount) / 100.0
	return fmt.Sprintf("%s %.2f", currency, value)
}

func formatDate(value *time.Time) string {
	if value == nil || value.IsZero() {
		return "-"
	}
	return value.UTC().Format("2006-01-02")
}

func formatQuantity(value float64) string {
	return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.2f", value), "0"), ".")
}

func sanitizeColor(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "#111827"
	}
	if hexColorPattern.MatchString(trimmed) {
		return trimmed
	}
	return "#111827"
}

func sanitizeFont(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "Space Grotesk"
	}
	if fontFamilyFilter.MatchString(trimmed) {
		return trimmed
	}
	return "Space Grotesk"
}
