import "./commands";

const slowCommandMs = Number(Cypress.env("slowCommandMs") ?? 0);

if (slowCommandMs > 0) {
  const delay = () =>
    new Cypress.Promise((resolve) => {
      setTimeout(resolve, slowCommandMs);
    });

  const delayCommands = [
    "visit",
    "request",
    "reload",
    "go",
    "click",
    "type",
    "select",
    "check",
    "uncheck",
    "trigger",
    "submit",
    "clear",
    "scrollIntoView"
  ];

  delayCommands.forEach((commandName) => {
    Cypress.Commands.overwrite(commandName as keyof Cypress.Chainable, (originalFn, ...args) =>
      delay().then(() => originalFn(...args))
    );
  });
}
