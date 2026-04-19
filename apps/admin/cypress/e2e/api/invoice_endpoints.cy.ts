import { generateInvoice, seedBillingFixture, switchOrg } from "../../support/billing";

describe("Admin API Invoice Endpoints", () => {
  beforeEach(() => {
    cy.login();
  });

  it("generates invoice totals correctly via /admin/v1/invoices/generate", () => {
    const uid = Date.now();
    const expectedSubtotal = 2500;
    const expectedTax = 250;
    const expectedTotal = 2750;
    let orgId = "";

    cy.createOrg(`API Invoice Org ${uid}`).then((id) => {
      orgId = id;
      return switchOrg(orgId);
    });

    cy.then(() => {
      return seedBillingFixture({
        uid,
        prefix: `api_invoice_${uid}`,
        flatAmountCents: expectedSubtotal,
        taxPercentage: 10
      }).then((fixture) => {
        return generateInvoice(
          fixture.subscriptionId,
          fixture.periodStart,
          fixture.periodEnd,
          "2026-03-02T09:00:00Z",
          "2026-03-10T09:00:00Z"
        );
      }).then((invoice) => {
        expect(invoice.status).to.equal("draft");
        expect(invoice.subtotal_cents).to.equal(expectedSubtotal);
        expect(invoice.tax_cents).to.equal(expectedTax);
        expect(invoice.total_cents).to.equal(expectedTotal);
        expect(invoice.amount_due_cents).to.equal(expectedTotal);
        expect(invoice.amount_paid_cents).to.equal(0);

        return cy.csrfRequest("GET", `/admin/v1/invoices/${invoice.id}`).then((resp) => {
          expect(resp.body.invoice.subtotal_cents).to.equal(expectedSubtotal);
          expect(resp.body.invoice.tax_cents).to.equal(expectedTax);
          expect(resp.body.invoice.total_cents).to.equal(expectedTotal);
          expect(resp.body.items.map((item: { amount_cents: number }) => item.amount_cents).sort((a: number, b: number) => a - b)).to.deep.equal([expectedTax, expectedSubtotal]);
        });
      });
    });
  });

  it("opens and pays invoice via /admin/v1/invoices/:id/* and writes ledger side effects", () => {
    const uid = Date.now();
    const expectedTotal = 3300;
    let orgId = "";

    cy.createOrg(`API Lifecycle Org ${uid}`).then((id) => {
      orgId = id;
      return switchOrg(orgId);
    });

    cy.then(() => {
      return seedBillingFixture({
        uid,
        prefix: `api_lifecycle_${uid}`,
        flatAmountCents: 3000,
        taxPercentage: 10
      }).then((fixture) => {
        return generateInvoice(
          fixture.subscriptionId,
          fixture.periodStart,
          fixture.periodEnd,
          "2026-03-02T09:00:00Z",
          "2026-03-10T09:00:00Z"
        );
      }).then((invoice) => {
        const invoiceId = invoice.id;

        return cy.csrfRequest("POST", `/admin/v1/invoices/${invoiceId}/open`).then((openResp) => {
          expect(openResp.body.invoice.status).to.equal("open");
          return cy.csrfRequest("POST", `/admin/v1/invoices/${invoiceId}/pay`, {
            reason: "API payment"
          });
        }).then((payResp) => {
          expect(payResp.body.invoice.status).to.equal("paid");
          expect(payResp.body.invoice.amount_due_cents).to.equal(0);
          expect(payResp.body.invoice.amount_paid_cents).to.equal(expectedTotal);

          return cy.csrfRequest("GET", `/admin/v1/ledger/transactions?page_size=50&source_id=${invoiceId}`).then((ledgerResp) => {
            const transactions = ledgerResp.body.transactions as Array<{ source_type: string; source_id: string }>;
            expect(transactions.length).to.be.greaterThan(1);
            expect(transactions.map((tx) => tx.source_type)).to.include.members(["billing_cycle", "payment"]);
            expect(transactions.every((tx) => tx.source_id === invoiceId)).to.equal(true);
          });
        });
      });
    });
  });
});
