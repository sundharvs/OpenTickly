import { expect, test } from "@playwright/test";

import { loginE2eUser, registerE2eUser } from "./fixtures/e2e-auth.ts";

async function dragByOffset(
  page: import("@playwright/test").Page,
  from: { x: number; y: number },
  to: { x: number; y: number },
) {
  await page.mouse.move(from.x, from.y);
  await page.waitForTimeout(50);
  await page.mouse.down();
  await page.waitForTimeout(50);

  await page.mouse.move(from.x + 1, from.y + 1, { steps: 2 });
  await page.waitForTimeout(50);

  for (let i = 1; i <= 20; i++) {
    const x = from.x + (to.x - from.x) * (i / 20);
    const y = from.y + (to.y - from.y) * (i / 20);
    await page.mouse.move(x, y, { steps: 5 });
  }

  await page.waitForTimeout(50);
  await page.mouse.up();
}

test("dragging to create a calendar entry works on the second attempt, not just the first", async ({
  page,
}) => {
  const email = `cal-drag-twice-${test.info().workerIndex}-${Date.now()}@example.com`;
  const password = "secret-pass";
  await registerE2eUser(page, test.info(), {
    email,
    fullName: "Calendar Drag Twice",
    password,
  });
  await page.context().clearCookies();
  await loginE2eUser(page, test.info(), { email, password });

  await page.goto(new URL("/timer", page.url()).toString());
  await expect(page.getByTestId("tracking-timer-page")).toBeVisible();
  await expect(page.getByTestId("timer-calendar-view")).toBeVisible();

  const calendarBox = await page.getByTestId("timer-calendar-view").boundingBox();
  if (!calendarBox) throw new Error("calendar view has no bounding box");

  const dialog = page.getByTestId("time-entry-editor-dialog");
  const closeButton = page.getByRole("button", { name: "Close editor" });

  const firstX = calendarBox.x + calendarBox.width * 0.4;
  const firstTop = calendarBox.y + calendarBox.height * 0.3;
  await dragByOffset(
    page,
    { x: firstX, y: firstTop },
    { x: firstX, y: firstTop + 80 },
  );
  await expect(dialog).toBeVisible();
  await closeButton.click();
  await expect(dialog).not.toBeVisible();

  const secondX = calendarBox.x + calendarBox.width * 0.7;
  const secondTop = calendarBox.y + calendarBox.height * 0.5;
  await dragByOffset(
    page,
    { x: secondX, y: secondTop },
    { x: secondX, y: secondTop + 80 },
  );
  await expect(dialog).toBeVisible();
});
