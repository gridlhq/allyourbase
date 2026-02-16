import { test, expect } from "@playwright/test";

/**
 * SMOKE TEST: Admin Dashboard - SQL Query Execution
 *
 * Critical Path: Admin logs in → Executes SQL query → Views results
 *
 * This test validates:
 * 1. Admin dashboard loads
 * 2. SQL query interface is accessible
 * 3. Query execution works
 * 4. Results are displayed correctly
 *
 * UI-ONLY: No direct API calls allowed
 */

test.describe("Smoke: Admin SQL Query", () => {
  test("admin can execute SQL query and view results", async ({ page }) => {
    // Step 1: Navigate to admin dashboard
    await page.goto("/admin/");
    await expect(page.getByText("AYB Admin").first()).toBeVisible();

    // Step 2: Navigate to SQL Editor via sidebar
    await page.locator("aside").getByRole("button", { name: /^SQL Editor$/i }).click();

    // Step 3: Find SQL input (textarea or code editor)
    const sqlInput = page.locator("textarea").or(
      page.locator('[contenteditable="true"]'),
    ).or(
      page.getByPlaceholder(/sql|query|select/i),
    ).first();

    await expect(sqlInput).toBeVisible({ timeout: 5000 });

    // Step 4: Execute a simple query to verify functionality
    const testQuery = "SELECT 1 AS test_column;";
    await sqlInput.fill(testQuery);

    // Step 5: Click Run/Execute button
    const runButton = page.getByRole("button", { name: /run|execute/i });
    await expect(runButton).toBeVisible();
    await runButton.click();

    // Step 6: Verify results appear
    // Look for "test_column" in results table header (use columnheader role to avoid strict mode violation)
    await expect(page.getByRole("columnheader", { name: /test_column/i })).toBeVisible({ timeout: 5000 });

    // Look for the value "1" in results
    const resultCell = page.locator("td").filter({ hasText: "1" });
    await expect(resultCell.first()).toBeVisible();

    console.log("✅ Smoke test passed: Admin SQL query execution");
  });
});
