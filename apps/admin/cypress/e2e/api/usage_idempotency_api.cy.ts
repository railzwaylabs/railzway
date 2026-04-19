import { seedBillingFixture, switchOrg } from "../../support/billing";

describe("Admin API Usage Idempotency", () => {
  beforeEach(() => {
    cy.login();
  });

  it("returns one effective usage event for duplicate idempotency keys", () => {
    const uid = Date.now();
    const duplicateKey = `usage-dup-${uid}`;
    let orgId = "";

    cy.createOrg(`API Usage Org ${uid}`).then((id) => {
      orgId = id;
      return switchOrg(orgId);
    });

    cy.then(() => {
      return seedBillingFixture({
        uid,
        prefix: `api_usage_${uid}`,
        includeUsage: true
      }).then((fixture) => {
        return cy.csrfRequest("POST", "/admin/v1/usage/events", {
          meter_code: fixture.meterCode,
          customer_id: fixture.customerId,
          value: 7,
          recorded_at: "2026-03-05T00:00:00Z",
          idempotency_key: duplicateKey
        }).then((firstResp) => {
          expect(firstResp.body.id).to.be.a("string").and.not.be.empty;
          return cy.csrfRequest("POST", "/admin/v1/usage/events", {
            meter_code: fixture.meterCode,
            customer_id: fixture.customerId,
            value: 7,
            recorded_at: "2026-03-05T00:00:00Z",
            idempotency_key: duplicateKey
          }).then((secondResp) => {
            expect(secondResp.body.id).to.equal(firstResp.body.id);
            return cy.csrfRequest("GET", `/admin/v1/usage/events?page_size=20&customer_id=${fixture.customerId}`);
          }).then((listResp) => {
            expect(listResp.body.events).to.have.length(1);
            expect(listResp.body.events[0].meter_code).to.equal(fixture.meterCode);
            expect(listResp.body.events[0].value).to.equal(7);
          });
        });
      });
    });
  });
});
