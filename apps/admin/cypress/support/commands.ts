// Custom commands can be added here.
declare global {
  namespace Cypress {
    interface Chainable {
      login(): Chainable<void>;
      ensureOrg(): Chainable<string>;
      createOrg(name?: string): Chainable<string>;
      orgVisit(path: string): Chainable<Window>;
      fillByLabel(label: string, value: string): Chainable<void>;
      selectByLabel(label: string, value: string): Chainable<void>;
      selectRadix(testId: string, option: string): Chainable<void>;
      selectAutocomplete(testId: string, query: string, optionText?: string): Chainable<void>;
      csrfRequest(
        method: string,
        url: string,
        body?: Cypress.RequestBody
      ): Chainable<Cypress.Response<any>>;
      csrfRequest(options: Partial<Cypress.RequestOptions>): Chainable<Cypress.Response<any>>;
    }
  }
}

const ORG_KEY = "railzway_admin_org_id";
const DEFAULT_CSRF_COOKIE = "rz_admin_csrf";

Cypress.Commands.add("login", () => {
  const email = Cypress.env("adminEmail") as string;
  const password = Cypress.env("adminPassword") as string;

  expect(email, "CYPRESS_ADMIN_EMAIL").to.be.a("string").and.not.be.empty;
  expect(password, "CYPRESS_ADMIN_PASSWORD").to.be.a("string").and.not.be.empty;

  cy.clearCookies();
  cy.clearLocalStorage();
  cy.window().then((win) => win.sessionStorage.clear());

  // First GET to get the CSRF cookie if it's set on landing page or health check
  cy.request("/healthz").then(() => {
    return cy.request("POST", "/admin/v1/auth/login", {
      email,
      password
    }).then(() => {
      return cy.csrfRequest({
        method: "POST",
        url: "/admin/v1/auth/skip-password-change",
        failOnStatusCode: false
      });
    });
  });
});

Cypress.Commands.add("ensureOrg", () => {
  const cached = Cypress.env("orgId") as string | undefined;
  if (cached) {
    return cy.csrfRequest({ method: "POST", url: `/admin/v1/auth/using/${cached}` }).then(() => {
      return cy.wrap(cached);
    });
  }
  return cy.request("/admin/v1/organizations").then((resp) => {
    const orgId = resp.body?.[0]?.id as string | undefined;
    if (orgId) {
      Cypress.env("orgId", orgId);
      return cy.csrfRequest({ method: "POST", url: `/admin/v1/auth/using/${orgId}` }).then(() => {
        return cy.wrap(orgId);
      });
    }
    return cy.createOrg();
  });
});

Cypress.Commands.add("createOrg", (name?: string) => {
  const orgName = name || `Example ${Date.now()}`;
  return cy.csrfRequest("POST", "/admin/v1/organizations", {
    name: orgName,
    country_code: "US",
    timezone_name: "UTC"
  }).then((resp) => {
    const orgId = resp.body?.id as string | undefined;
    expect(orgId, "created org id").to.be.a("string").and.not.be.empty;
    Cypress.env("orgId", orgId);
    return cy.csrfRequest({ method: "POST", url: `/admin/v1/auth/using/${orgId}` }).then(() => {
      return cy.wrap(orgId);
    });
  });
});

Cypress.Commands.add("orgVisit", (path: string) => {
  return cy.ensureOrg().then((orgId) => {
    const normalized = path.startsWith("/") ? path : `/${path}`;
    const url = `/organizations/${orgId}${normalized}`;
    return cy.visit(url, {
      onBeforeLoad(win) {
        win.localStorage.setItem(ORG_KEY, orgId);
      }
    });
  });
});

Cypress.Commands.add("fillByLabel", (label: string, value: string) => {
  cy.contains("label", label)
    .should("be.visible")
    .then(($label) => {
      const $container = $label.closest("div");
      const $input = $container.find("input, textarea");
      if ($input.length) {
        cy.wrap($input.first()).clear().type(value);
        return;
      }
      cy.wrap($label).parent().find("input, textarea").first().clear().type(value);
    });
});

Cypress.Commands.add("selectByLabel", (label: string, value: string) => {
  cy.contains("label", label)
    .should("be.visible")
    .then(($label) => {
      const $container = $label.closest("div");
      const $select = $container.find("select");
      if ($select.length) {
        cy.wrap($select.first()).select(value);
        return;
      }
      cy.wrap($label).parent().find("select").first().select(value);
    });
});

Cypress.Commands.add("selectRadix", (testId: string, option: string) => {
  cy.get(`[data-testid="${testId}"]`).click();
  cy.get("[role=\"option\"]").contains(option).click();
});

Cypress.Commands.add("selectAutocomplete", (testId: string, query: string, optionText?: string) => {
  cy.get(`[data-testid="${testId}"]`).click();
  if (query) {
    cy.get(`[data-testid="${testId}-search"]`).clear().type(query);
  }
  cy.contains("[cmdk-item]", optionText ?? query).click();
});

Cypress.Commands.add("csrfRequest", (...args: any[]) => {
  let options: Partial<Cypress.RequestOptions>;
  if (typeof args[0] === "string" && typeof args[1] === "string") {
    options = { method: args[0], url: args[1], body: args[2] };
  } else {
    options = args[0] ?? {};
  }
  const method = String(options.method ?? "GET").toUpperCase();
  const unsafe = method !== "GET" && method !== "HEAD" && method !== "OPTIONS";
  if (!unsafe) {
    return cy.request(options as Cypress.RequestOptions);
  }
  return cy.getCookies().then((cookies) => {
    const cookie = cookies.find((item) => item.name.endsWith("_csrf")) ?? cookies.find((item) => item.name === DEFAULT_CSRF_COOKIE);
    const headers = { ...(options.headers ?? {}) } as Record<string, string>;
    if (cookie?.value && !headers["X-CSRF-Token"]) {
      headers["X-CSRF-Token"] = decodeURIComponent(cookie.value);
    }
    return cy.request({ ...options, headers } as Cypress.RequestOptions);
  });
});

export {};
