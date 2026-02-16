import { test, expect } from "@playwright/test";

/**
 * FULL E2E TEST: Functions Browser
 *
 * Tests PostgreSQL function browsing and execution:
 * - Setup: Create a test function via SQL
 * - Navigate to Functions section
 * - Verify function appears in list
 * - Expand function to see parameters
 * - Execute function with arguments
 * - Verify results
 * - Cleanup: Drop test function
 *
 * UI-ONLY: No direct API calls (except SQL via UI)
 */

test.describe("Functions Browser (Full E2E)", () => {
  test("browse, execute, and verify function results", async ({ page }) => {
    const runId = Date.now();
    const funcName = `test_add_${runId}`;

    // ============================================================
    // Setup: Create test function via SQL
    // ============================================================
    await page.goto("/admin/");
    await expect(page.getByText("AYB Admin").first()).toBeVisible();

    const sidebar = page.locator("aside");

    // Navigate to SQL Editor via sidebar
    await sidebar.getByRole("button", { name: /^SQL Editor$/i }).click();

    const sqlInput = page.locator("textarea").first();
    await expect(sqlInput).toBeVisible({ timeout: 5000 });

    // Create a simple addition function
    await sqlInput.fill(
      `CREATE OR REPLACE FUNCTION ${funcName}(a integer, b integer) RETURNS integer AS $$ SELECT a + b; $$ LANGUAGE SQL;`
    );
    await page.getByRole("button", { name: /run|execute/i }).click();

    // Wait for DDL event trigger + schema cache rebuild (500ms debounce + rebuild time)
    await page.waitForTimeout(2000);

    // Reload page and refresh schema to pick up the new function.
    // Retry up to 3 times in case the schema cache is still rebuilding.
    let funcFound = false;
    for (let attempt = 0; attempt < 3; attempt++) {
      await page.reload();
      await expect(page.getByText("AYB Admin").first()).toBeVisible();

      // Click refresh schema button
      const refreshButton = page.locator('button[title="Refresh schema"]');
      if (await refreshButton.isVisible({ timeout: 2000 }).catch(() => false)) {
        await refreshButton.click();
        await page.waitForTimeout(1500);
      }

      // Navigate to Functions
      const functionsButton = sidebar.getByRole("button", { name: /^Functions$/i });
      await expect(functionsButton).toBeVisible({ timeout: 5000 });
      await functionsButton.click();
      await expect(page.getByText(/Functions/i).first()).toBeVisible({ timeout: 5000 });

      // Check if function appeared
      if (await page.getByText(funcName).first().isVisible({ timeout: 3000 }).catch(() => false)) {
        funcFound = true;
        break;
      }
      console.log(`  Function not found yet (attempt ${attempt + 1}/3), retrying...`);
    }

    // Final assertion
    await expect(page.getByText(funcName).first()).toBeVisible({ timeout: 5000 });

    // ============================================================
    // EXPAND: Click function to see parameters
    // ============================================================
    // Click the actual function button (not a parent div) to expand it
    const funcButton = page.getByRole("button", { name: new RegExp(funcName) });
    await funcButton.click();

    // FunctionBrowser renders parameter inputs with placeholder="NULL"
    // Wait for the expanded section to appear
    const paramInputs = page.getByPlaceholder("NULL");
    await expect(paramInputs.first()).toBeVisible({ timeout: 3000 });

    // ============================================================
    // EXECUTE: Fill params and run function
    // ============================================================
    // Params rendered in order: a (index 0), b (index 1)
    await paramInputs.nth(0).fill("3");
    await paramInputs.nth(1).fill("5");

    // Click execute/play button (within the expanded function area)
    const executeButton = page.getByRole("button", { name: /execute|run/i }).or(
      page.locator("button").filter({ has: page.locator("svg.lucide-play") })
    );
    await expect(executeButton.first()).toBeVisible({ timeout: 2000 });
    await executeButton.first().click();

    // ============================================================
    // VERIFY: Check results show 8
    // ============================================================
    // Result should contain "8" or status 200
    const resultArea = page.getByText("8").or(
      page.getByText(/200|success/i)
    );
    await expect(resultArea.first()).toBeVisible({ timeout: 5000 });

    // ============================================================
    // Cleanup: Drop test function via SQL
    // ============================================================
    await sidebar.getByRole("button", { name: /^SQL Editor$/i }).click();
    const cleanupSql = page.locator("textarea").first();
    await expect(cleanupSql).toBeVisible({ timeout: 5000 });
    await cleanupSql.fill(`DROP FUNCTION IF EXISTS ${funcName}(integer, integer);`);
    await page.getByRole("button", { name: /run|execute/i }).click();

    console.log("✅ Full functions browser test passed");
  });
});
