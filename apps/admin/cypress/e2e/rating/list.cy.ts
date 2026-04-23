describe("Rating", () => {
  beforeEach(() => {
    cy.login();
  });

  it("switches between results and aggregates view", () => {
    cy.orgVisit("/rating");
    cy.get("[data-testid=\"page-header-title\"]").should("contain", "Rating");

    // Results is active by default
    cy.get("[data-testid=\"rating-tab-results\"]").should("have.class", "is-active");
    cy.get("[data-testid=\"rating-results-apply\"]").should("be.visible");

    // Switch to aggregates
    cy.get("[data-testid=\"rating-tab-aggregates\"]").click();
    cy.get("[data-testid=\"rating-tab-aggregates\"]").should("have.class", "is-active");
    cy.get("[data-testid=\"rating-aggregates-apply\"]").should("be.visible");
  });

  it("uses results filters (positive scenario)", () => {
    cy.orgVisit("/rating");
    cy.get("[data-testid=\"rating-tab-results\"]").click();
    cy.get("[data-testid=\"rating-results-plan-price\"]").type("price_123");
    cy.get("[data-testid=\"rating-results-apply\"]").click();
    cy.get(".data-table").should("exist");
  });

  it("resets result filters back to a broad billing view", () => {
    cy.orgVisit("/rating");
    cy.get("[data-testid=\"rating-tab-results\"]").click();
    cy.get("[data-testid=\"rating-results-plan-price\"]").type("price_123");
    cy.get("[data-testid=\"rating-results-usage-event\"]").type("evt_123");

    cy.get("[data-testid=\"rating-results-reset\"]").click();

    cy.get("[data-testid=\"rating-results-plan-price\"]").should("have.value", "");
    cy.get("[data-testid=\"rating-results-usage-event\"]").should("have.value", "");
    cy.get(".data-table").should("exist");
  });
});
