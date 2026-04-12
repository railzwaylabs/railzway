describe("Rating", () => {
  beforeEach(() => {
    cy.login();
  });

  it("filters rating results and aggregates", () => {
    cy.intercept("GET", "/admin/v1/rating/results*").as("ratingResults");
    cy.intercept("GET", "/admin/v1/rating/aggregates*").as("ratingAggregates");

    cy.orgVisit("/rating");

    cy.get("[data-testid=\"rating-results-usage-event\"]").type("evt_");
    cy.get("[data-testid=\"rating-results-apply\"]").click();
    cy.wait("@ratingResults");

    cy.get("[data-testid=\"rating-aggregates-period-from\"]").clear().type("2026-03-01T00:00:00Z");
    cy.get("[data-testid=\"rating-aggregates-apply\"]").click();
    cy.wait("@ratingAggregates");
  });

  it("resets rating filters", () => {
    cy.orgVisit("/rating");
    cy.get("[data-testid=\"rating-results-usage-event\"]").type("evt_reset");
    cy.get("[data-testid=\"rating-results-reset\"]").click();
    cy.get("[data-testid=\"rating-results-usage-event\"]").should("have.value", "");

    cy.get("[data-testid=\"rating-aggregates-period-from\"]").type("2026-03-01T00:00:00Z");
    cy.get("[data-testid=\"rating-aggregates-reset\"]").click();
    cy.get("[data-testid=\"rating-aggregates-period-from\"]").should("have.value", "");
  });
});
