import { test, expect, type BrowserContext, type Page } from "@playwright/test";
import { readFileSync } from "node:fs";
import { createHmac } from "node:crypto";
const api = process.env.PLAYWRIGHT_API_URL || "http://localhost:8080";
const site = process.env.PLAYWRIGHT_BASE_URL || "http://localhost:3000";
const fixture = JSON.parse(
  readFileSync(
    process.env.E2E_FIXTURE_FILE || "/tmp/chloe-browser-fixture.json",
    "utf8",
  ),
) as Record<string, string>;
async function owner(context: BrowserContext) {
  await context.addCookies([
    { name: "chloe_api", value: fixture.ownerToken, url: api },
  ]);
}
async function mutation(
  context: BrowserContext,
  path: string,
  body: unknown,
  method = "POST",
) {
  const session = await context.request.get(`${api}/v1/session`);
  const { csrf } = await session.json();
  return context.request.fetch(`${api}${path}`, {
    method,
    headers: { Origin: site, "X-CSRF-Token": csrf },
    data: body,
  });
}
async function note(page: Page) {
  await page.goto("/admin");
  await page.getByRole("button", { name: "Create note", exact: true }).click();
  const title = `Browser note ${Date.now()}`;
  await page.getByLabel("Title", { exact: true }).fill(title);
  await page
    .getByLabel("Short description")
    .fill("A private integration test note.");
  await page
    .getByRole("textbox", { name: "Article content" })
    .fill("A story written in the custom editor.");
  return title;
}
function totp(secret: string) {
  const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZ234567";
  let bits = "";
  for (const c of secret)
    bits += alphabet.indexOf(c).toString(2).padStart(5, "0");
  const key = Buffer.from(bits.match(/.{8}/g)!.map((v) => parseInt(v, 2)));
  const counter = Buffer.alloc(8);
  counter.writeBigUInt64BE(BigInt(Math.floor(Date.now() / 30000)));
  const digest = createHmac("sha1", key).update(counter).digest();
  const offset = digest[19] & 15;
  return String((digest.readUInt32BE(offset) & 0x7fffffff) % 1000000).padStart(
    6,
    "0",
  );
}

test("public pages, metadata, real images and responsive layout", async ({
  page,
}, info) => {
  for (const route of [
    "/",
    "/work",
    "/about",
    "/photography",
    "/notes",
    "/now",
    "/playground",
    "/guestbook",
    "/contact",
    "/privacy",
  ]) {
    const response = await page.goto(route);
    expect(response?.status()).toBe(200);
    await expect(page.locator("h1")).toBeVisible();
    expect(
      await page.evaluate(
        () => document.documentElement.scrollWidth <= window.innerWidth,
      ),
    ).toBeTruthy();
    expect(
      await page.locator('link[rel="canonical"]').getAttribute("href"),
    ).toBe(`${site}${route === "/" ? "" : route}`);
    for (const img of await page.locator("img").all()) {
      await img.scrollIntoViewIfNeeded();
      await expect(img).toBeVisible();
      await expect
        .poll(() =>
          img.evaluate(
            (v: HTMLImageElement) => v.complete && v.naturalWidth > 0,
          ),
        )
        .toBeTruthy();
    }
  }
  await page.goto("/");
  await expect(
    page.getByText("Some thoughts take a little longer"),
  ).toHaveCount(0);
  await page.screenshot({
    path: `test-results/dark-home-${info.project.name}.png`,
    fullPage: true,
  });
  const work = page.locator('a[href^="/work/"]').first();
  await work.click();
  await expect(page.locator("h1")).toBeVisible();
  expect((await page.request.get("/notes/nonexistent-note")).status()).toBe(
    404,
  );
  for (const path of [
    "/sitemap.xml",
    "/robots.txt",
    "/feed.xml",
    "/opengraph-image",
  ]) {
    expect((await page.request.get(path)).status()).toBe(200);
  }
  expect((await page.request.get("/api/session")).status()).toBe(404);
});
test("keyboard memory game and reduced motion", async ({ page }) => {
  await page.emulateMedia({ reducedMotion: "reduce" });
  await page.goto("/playground");
  await page.getByRole("button", { name: "Start a round" }).focus();
  await page.keyboard.press("Enter");
  const tile = page.getByRole("button", { name: "Reveal tile 1", exact: true });
  await tile.focus();
  await page.keyboard.press("Enter");
  await expect(page.locator(".memory-tile.revealed")).toHaveCount(1);
  expect(
    await page
      .locator(".memory-tile.revealed")
      .evaluate((el) => getComputedStyle(el).transitionDuration),
  ).toBe("0s");
  await page.getByRole("button", { name: "Restart game" }).click();
  await expect(page.locator(".memory-tile.revealed")).toHaveCount(0);
});
test("contact persists and failures never show success", async ({ page }) => {
  await page.goto("/contact");
  const fill = async () => {
    await page.getByLabel("Your name").fill("Browser visitor");
    await page.getByLabel("Email address").fill("visitor@example.test");
    await page.getByLabel("What’s on your mind?").fill("Browser integration");
    await page
      .getByLabel("Your message")
      .fill("A message sent through the Go API.");
  };
  await fill();
  await page.getByRole("button", { name: "Send your message" }).click();
  await expect(page.getByRole("status")).toContainText("in my inbox");
  await page.route("**/v1/contact", (route) =>
    route.fulfill({
      status: 200,
      contentType: "application/json",
      body: "not-json",
    }),
  );
  await fill();
  await page.getByRole("button", { name: "Send your message" }).click();
  await expect(page.locator("main [role=alert]")).toContainText(
    "invalid response",
  );
  await expect(page.getByLabel("Your message")).toHaveValue(
    "A message sent through the Go API.",
  );
  await expect(page.getByRole("status")).toHaveCount(0);
});
test("owner dashboard preserves typing during saves and browser recovery", async ({
  page,
  context,
}) => {
  await owner(context);
  await note(page);
  const newTitle = `Newer typing ${Date.now()}`;
  const recoveredTitle = `Recovered unsent work ${Date.now()}`;
  let delayed = false;
  await page.route("**/v1/admin/content/*", async (route) => {
    if (route.request().method() === "POST" && !delayed) {
      delayed = true;
      const response = await route.fetch();
      await new Promise((resolve) => setTimeout(resolve, 800));
      await route.fulfill({ response });
    } else await route.continue();
  });
  await page.getByRole("button", { name: "Save draft", exact: true }).click();
  await expect(page.getByRole("button", { name: "Saving…" })).toBeVisible();
  await page.getByLabel("Title", { exact: true }).fill(newTitle);
  await expect(page.getByLabel("Title", { exact: true })).toHaveValue(newTitle);
  await expect(
    page.getByText("Draft saved automatically.", { exact: true }),
  ).toBeVisible();
  await expect(page.getByLabel("Title", { exact: true })).toHaveValue(newTitle);
  await page.unroute("**/v1/admin/content/*");
  await page.route("**/v1/admin/content/*", (route) =>
    route.request().method() === "POST"
      ? route.fulfill({
          status: 500,
          contentType: "application/json",
          body: JSON.stringify({ error: "Test save failure" }),
        })
      : route.continue(),
  );
  await page.getByLabel("Title", { exact: true }).fill(recoveredTitle);
  await expect(page.locator("main [role=alert]")).toContainText(
    "Test save failure",
  );
  page.on("dialog", (d) => d.accept());
  await page.reload();
  await page.getByRole("button", { name: new RegExp(newTitle) }).click();
  await expect(page.getByText(/Browser edits were recovered/)).toBeVisible();
  await page.getByRole("button", { name: "Restore browser edits" }).click();
  await expect(page.getByLabel("Title", { exact: true })).toHaveValue(
    recoveredTitle,
  );
  await page.unroute("**/v1/admin/content/*");
  await page.getByRole("button", { name: "Save draft", exact: true }).click();
  await expect(
    page.getByText("Revision saved.", { exact: true }),
  ).toBeVisible();
});
test("draft publish, private preview, revision restore and scheduling", async ({
  page,
  context,
}) => {
  await owner(context);
  const title = await note(page);
  await page.getByRole("button", { name: "Save draft", exact: true }).click();
  await expect(
    page.getByText("Revision saved.", { exact: true }),
  ).toBeVisible();
  const preview = await page
    .getByRole("link", { name: /Preview saved/ })
    .getAttribute("href");
  const id = preview!.split("/").pop()!;
  const slug = await page.getByLabel("URL slug").inputValue();
  expect((await page.request.get(`/notes/${slug}`)).status()).toBe(404);
  const privatePage = await context.newPage();
  await privatePage.goto(preview!);
  await expect(
    privatePage.getByText("A story written in the custom editor."),
  ).toBeVisible();
  await privatePage.close();
  await page.getByRole("button", { name: "Publish", exact: true }).click();
  await expect(page.getByRole("status")).toContainText("Published.");
  const article = await page.request.get(`/notes/${slug}`);
  expect(article.status()).toBe(200);
  expect(await article.text()).toContain(title);
  await page.getByText("Revision history & scheduled publishing").click();
  await expect(
    page.getByRole("button", { name: "Restore draft" }).first(),
  ).toBeVisible();
  await page.getByRole("button", { name: "Restore draft" }).last().click();
  await expect(page.locator(".editor-heading .form-help")).toContainText(
    "Draft",
  );
  const date = new Date(Date.now() + 3600000);
  const parts = Object.fromEntries(
    new Intl.DateTimeFormat("en-CA", {
      timeZone: "Europe/Berlin",
      year: "numeric",
      month: "2-digit",
      day: "2-digit",
      hour: "2-digit",
      minute: "2-digit",
      hourCycle: "h23",
    })
      .formatToParts(date)
      .map((p) => [p.type, p.value]),
  );
  await page
    .getByLabel("Publish at · Europe/Berlin")
    .fill(
      `${parts.year}-${parts.month}-${parts.day}T${parts.hour}:${parts.minute}`,
    );
  await page
    .getByRole("button", { name: "Save a revision & schedule" })
    .click();
  await expect(page.getByText(/Berlin · pending/)).toBeVisible();
  await page.getByRole("button", { name: "Cancel", exact: true }).click();
  await expect(page.getByText(/Berlin · cancelled/)).toBeVisible();
  const doc = (
    await (await context.request.get(`${api}/v1/admin/content/${id}`)).json()
  ).document;
  expect(
    (
      await mutation(context, `/v1/admin/content/${id}`, {
        action: "unpublish",
        revision: doc._rev,
        document: doc,
      })
    ).ok(),
  ).toBeTruthy();
  expect((await page.request.get(`/notes/${slug}`)).status()).toBe(404);
});
test("identity round trip and browser passkey registration", async ({
  page,
  context,
}, info) => {
  test.skip(
    info.project.name !== "desktop",
    "One identity ceremony per run avoids reuse of a TOTP step.",
  );
  await page.goto("/admin");
  await page.getByRole("button", { name: "Sign in with Chloe ID" }).click();
  await expect(page).toHaveURL(
    (url) => url.origin === api && url.pathname === "/login",
  );
  await page.getByLabel("Email", { exact: true }).fill("owner@example.test");
  await page.getByLabel("Password", { exact: true }).fill(fixture.password);
  await expect(page.getByLabel(/Authenticator code/)).toHaveCount(0);
  await page.getByRole("button", { name: "Sign in", exact: true }).click();
  await expect(page).toHaveURL(
    (url) => url.origin === api && url.pathname === "/step-up",
  );
  await page
    .getByLabel("Authenticator code", { exact: true })
    .fill(totp(fixture.totpSecret));
  await page
    .getByRole("button", { name: /Verify & sign in|Finish setup/ })
    .click();
  await expect(
    page.getByRole("heading", { name: "Behind the scenes." }),
  ).toBeVisible();
  const auth = await context.newPage();
  await auth.goto(`${api}/account`);
  const cdp = await context.newCDPSession(auth);
  await cdp.send("WebAuthn.enable");
  await cdp.send("WebAuthn.addVirtualAuthenticator", {
    options: {
      protocol: "ctap2",
      transport: "internal",
      hasResidentKey: true,
      hasUserVerification: true,
      isUserVerified: true,
      automaticPresenceSimulation: true,
    },
  });
  await auth.getByRole("button", { name: "Add a passkey" }).click();
  await expect(
    auth.getByRole("button", { name: "Remove", exact: true }).first(),
  ).toBeVisible();
  await auth
    .getByRole("button", { name: "Remove", exact: true })
    .first()
    .click();
  await expect(auth.locator(".notice[role=status]")).toContainText(
    "Account updated",
  );
  await auth.close();

  // A failed logout must keep the session visible and allow a retry.
  await page.route(`${api}/v1/session`, async (route) => {
    if (route.request().method() === "DELETE")
      await route.fulfill({
        status: 503,
        contentType: "application/json",
        body: JSON.stringify({ error: "Sign-out is temporarily unavailable." }),
      });
    else await route.continue();
  });
  await page
    .getByRole("button", { name: "Sign out of website", exact: true })
    .click();
  await expect(page.locator(".auth-controls").getByRole("alert")).toContainText(
    "Sign-out is temporarily unavailable.",
  );
  await expect(
    page.getByRole("tablist", { name: "Dashboard sections" }),
  ).toBeVisible();
  await page.unroute(`${api}/v1/session`);

  await page
    .getByRole("button", { name: "Sign out of website", exact: true })
    .click();
  await expect(
    page.getByRole("button", { name: "Sign in with Chloe ID" }),
  ).toBeVisible();
  await expect(
    page.getByRole("tablist", { name: "Dashboard sections" }),
  ).toHaveCount(0);
  await expect(
    page.getByRole("button", { name: "Create note", exact: true }),
  ).toHaveCount(0);
  expect((await context.request.get(`${api}/v1/admin/content`)).status()).toBe(
    401,
  );
  expect(
    (await (await context.request.get(`${api}/v1/session`)).json()).user,
  ).toBeNull();
  await page.reload();
  await expect(
    page.getByRole("button", { name: "Sign in with Chloe ID" }),
  ).toBeVisible();
  await expect(
    page.getByRole("tablist", { name: "Dashboard sections" }),
  ).toHaveCount(0);
  // Website logout keeps SSO available until Chloe ID is explicitly signed out.
  await page.getByRole("button", { name: "Sign in with Chloe ID" }).click();
  await expect(
    page.getByRole("heading", { name: "Behind the scenes." }),
  ).toBeVisible();
  // Chloe ID logout must also revoke the still-active website session.
  await page
    .getByRole("link", { name: "Sign out of Chloe ID", exact: true })
    .click();
  await expect(page).toHaveURL(
    (url) => url.origin === api && url.pathname === "/logout",
  );
  await expect(page.getByRole("heading", { name: "Sign out?" })).toBeVisible();
  await page.getByRole("button", { name: "Sign out", exact: true }).click();
  await expect(page).toHaveURL(`${site}/`);
  await page.goto("/admin");
  await expect(
    page.getByRole("button", { name: "Sign in with Chloe ID" }),
  ).toBeVisible();
  await page.reload();
  await expect(
    page.getByRole("tablist", { name: "Dashboard sections" }),
  ).toHaveCount(0);
  expect((await context.request.get(`${api}/v1/admin/content`)).status()).toBe(
    401,
  );
  await page.screenshot({ path: "test-results/chloe-id-sign-out-desktop.png" });
  await page.getByRole("button", { name: "Sign in with Chloe ID" }).click();
  await expect(page).toHaveURL(
    (url) => url.origin === api && url.pathname === "/login",
  );
  await expect(page.getByLabel("Email", { exact: true })).toBeVisible();
  await expect(page.getByLabel("Password", { exact: true })).toBeVisible();
});

test("visitor can register, sign out, and sign back in with a passkey", async ({
  page,
  context,
}, info) => {
  test.skip(
    info.project.name !== "desktop",
    "WebAuthn is exercised in desktop Chromium.",
  );
  await page.goto(`${api}/login`);
  await page.getByLabel("Email", { exact: true }).fill("visitor@example.test");
  await page.getByLabel("Password", { exact: true }).fill(fixture.password);
  await page.getByRole("button", { name: "Sign in", exact: true }).click();
  await expect(
    page.getByRole("heading", { name: "Your account." }),
  ).toBeVisible();
  await page.goto(`${api}/account/authenticator`);
  await expect(
    page.getByText("You currently sign in without an authenticator."),
  ).toBeVisible();
  await page
    .getByLabel("Current password", { exact: true })
    .fill(fixture.password);
  await page.getByRole("button", { name: "Set up authenticator" }).click();
  await expect(
    page.getByRole("img", { name: "QR code to set up your authenticator" }),
  ).toBeVisible();
  await page.getByRole("button", { name: "Cancel setup" }).click();
  await expect(
    page.getByText("You currently sign in without an authenticator."),
  ).toBeVisible();
  await page
    .getByRole("link", { name: "Back to account", exact: true })
    .click();
  const cdp = await context.newCDPSession(page);
  await cdp.send("WebAuthn.enable");
  await cdp.send("WebAuthn.addVirtualAuthenticator", {
    options: {
      protocol: "ctap2",
      transport: "internal",
      hasResidentKey: true,
      hasUserVerification: true,
      isUserVerified: true,
      automaticPresenceSimulation: true,
    },
  });
  await page.getByRole("button", { name: "Add a passkey" }).click();
  await expect(
    page.getByRole("button", { name: "Remove", exact: true }).first(),
  ).toBeVisible();
  await page.getByRole("link", { name: "Sign out", exact: true }).click();
  await page.getByRole("button", { name: "Sign out", exact: true }).click();
  await page.goto(`${api}/passkey-login`);
  await page.getByRole("button", { name: "Continue with passkey" }).click();
  await expect(
    page.getByRole("heading", { name: "Your account." }),
  ).toBeVisible();
  await expect(
    page.getByText("visitor@example.test", { exact: true }),
  ).toBeVisible();
  await page
    .getByRole("button", { name: "Remove", exact: true })
    .first()
    .click();
  await expect(page.locator(".notice[role=status]")).toContainText(
    "Account updated",
  );
});

test("independent confidential web application completes sign-in and consent", async ({
  page,
}, info) => {
  const example = process.env.E2E_WEB_EXAMPLE_URL;
  test.skip(
    !example || info.project.name !== "desktop",
    "Start examples/web and set E2E_WEB_EXAMPLE_URL.",
  );
  await page.goto(example!);
  await page.getByRole("link", { name: "Sign in with Chloe ID" }).click();
  await page.getByLabel("Email", { exact: true }).fill("visitor@example.test");
  await page.getByLabel("Password", { exact: true }).fill(fixture.password);
  await page.getByRole("button", { name: "Sign in", exact: true }).click();
  await expect(
    page.getByText("Signed in as visitor@example.test.", { exact: false }),
  ).toBeVisible();
  await page.getByRole("button", { name: "Allow access" }).click();
  await expect(page).toHaveURL(example! + "/");
  await expect(
    page.getByText("Signed in as Test visitor (visitor@example.test).", {
      exact: true,
    }),
  ).toBeVisible();
});
