describe("Usage Ingest", () => {
  beforeEach(() => {
    cy.login();
  });

  it("uploads a CSV file", () => {
    const uid = Date.now();
    const meterCode = `mtr_usage_${uid}`;
    const customerEmail = `usage_${uid}@example.com`;

    cy.ensureOrg().then(() => {
      return cy.csrfRequest("POST", "/admin/v1/meters", {
        code: meterCode,
        name: `Usage Meter ${uid}`,
        aggregation: "sum",
        unit: "requests",
        active: true
      });
    }).then(() => {
      return cy.csrfRequest("POST", "/admin/v1/customers", {
        name: `Usage Customer ${uid}`,
        email: customerEmail,
        currency: "USD"
      });
    }).then((resp) => {
      const customerId = resp.body.id as string;
      const csv = [
        "meter_code,customer_id,value,recorded_at",
        `${meterCode},${customerId},5,2026-03-01T00:00:00Z`
      ].join("\n");

      cy.intercept("POST", "/admin/v1/usage/events").as("ingestUsage");
      cy.orgVisit("/usage/new");
      cy.get("[data-testid=\"usage-upload-file\"]").selectFile({
        contents: Cypress.Buffer.from(csv),
        fileName: "usage.csv",
        mimeType: "text/csv"
      });
      cy.get("[data-testid=\"usage-upload-submit\"]").should("not.be.disabled");
      cy.get("[data-testid=\"usage-upload-submit\"]").click();

      cy.wait("@ingestUsage");
      cy.url().should("include", "/usage");
    });
  });

  it("requires a CSV file before upload", () => {
    cy.orgVisit("/usage/new");
    cy.get("[data-testid=\"usage-upload-submit\"]").should("be.disabled");
  });
});
