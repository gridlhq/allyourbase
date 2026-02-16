import { test, expect } from "@playwright/test";

/**
 * SMOKE TEST: Collections - Create Record
 *
 * Critical Path: Navigate to table → Create new record → Verify appears in list
 *
 * This test validates:
 * 1. Collections view loads
 * 2. Table navigation works
 * 3. Record creation form works
 * 4. New records appear in data view
 *
 * UI-ONLY: No direct API calls allowed
 */

test.describe("Smoke: Collections Create", () => {
  test("create test table and add record via UI", async ({ page }) => {
    const runId = Date.now();

    // Step 1: Navigate to admin dashboard
    await page.goto("/admin/");
    await expect(page.getByText("AYB Admin").first()).toBeVisible();

    // Step 2: Navigate to SQL Editor via sidebar
    const sidebar = page.locator("aside");
    await sidebar.getByRole("button", { name: /^SQL Editor$/i }).click();

    // Step 3: Create a simple test table via SQL
    const sqlInput = page.locator("textarea").first();
    await expect(sqlInput).toBeVisible({ timeout: 5000 });

    const createTableSQL = `CREATE TABLE IF NOT EXISTS smoke_test_records (
  id SERIAL PRIMARY KEY,
  name TEXT NOT NULL,
  email TEXT NOT NULL,
  created_at TIMESTAMPTZ DEFAULT NOW()
);`;

    await sqlInput.fill(createTableSQL);
    const runButton = page.getByRole("button", { name: /run|execute/i });
    await runButton.click();

    // Wait for query to complete
    await page.waitForTimeout(500);

    // Step 4: Reload to see new table in sidebar
    await page.reload();
    await expect(page.getByText("AYB Admin").first()).toBeVisible();

    // Step 5: Click on the new table in sidebar
    const tableLink = sidebar.getByText("smoke_test_records", { exact: true });
    await expect(tableLink).toBeVisible({ timeout: 5000 });
    await tableLink.click();

    // Step 6: Click "New Row" button to create record
    const newButton = page.getByRole("button", { name: "New Row" });
    await expect(newButton).toBeVisible({ timeout: 5000 });
    await newButton.click();

    // Step 7: Verify form opened
    await expect(page.getByText("New Record")).toBeVisible();

    // Step 8: Fill in form fields (unique per run to avoid constraint issues)
    const nameInput = page.locator('label:has-text("name")').locator("..").locator("input,textarea").first();
    await nameInput.fill(`Smoke User ${runId}`);

    const emailInput = page.locator('label:has-text("email")').locator("..").locator("input,textarea").first();
    await emailInput.fill(`smoke-${runId}@test.com`);

    // Step 9: Submit form
    const createButton = page.getByRole("button", { name: "Create" });
    await createButton.click();

    // Step 10: Verify record appears in table (use cell role to avoid matching form textarea)
    await expect(page.getByRole("cell", { name: `Smoke User ${runId}` })).toBeVisible({ timeout: 5000 });
    await expect(page.getByRole("cell", { name: `smoke-${runId}@test.com` })).toBeVisible();

    console.log("✅ Smoke test passed: Collections create record");
  });
});
