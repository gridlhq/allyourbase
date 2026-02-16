import { test, expect } from "@playwright/test";

/**
 * FULL E2E TEST: Collections CRUD Operations
 *
 * Tests all CRUD operations via admin UI:
 * - Create record
 * - Read (view) records
 * - Update record
 * - Delete record
 *
 * UI-ONLY: No direct API calls
 */

test.describe("Collections CRUD (Full E2E)", () => {
  test("complete CRUD lifecycle via UI", async ({ page }) => {
    // ============================================================
    // Setup: Create test table
    // ============================================================
    await page.goto("/admin/");
    await expect(page.getByText("AYB Admin").first()).toBeVisible();

    // Navigate to SQL Editor via sidebar
    const sidebar = page.locator("aside");
    await sidebar.getByRole("button", { name: /^SQL Editor$/i }).click();

    const sqlInput = page.locator("textarea").first();

    const createTableSQL = `
      CREATE TABLE IF NOT EXISTS crud_test_products (
        id SERIAL PRIMARY KEY,
        name TEXT NOT NULL,
        price DECIMAL(10,2) NOT NULL,
        stock INTEGER DEFAULT 0,
        description TEXT,
        created_at TIMESTAMPTZ DEFAULT NOW()
      );
    `;

    await sqlInput.fill(createTableSQL);
    await page.getByRole("button", { name: /run|execute/i }).click();
    await page.waitForTimeout(500);

    // ============================================================
    // CREATE: Add a new product
    // ============================================================
    await page.reload();
    await expect(page.getByText("AYB Admin").first()).toBeVisible();

    // Wait for sidebar to fully load after reload
    await expect(sidebar).toBeVisible();

    // Wait for the test table to appear in sidebar (with longer timeout for table creation)
    await expect(sidebar.getByText("crud_test_products", { exact: true })).toBeVisible({ timeout: 10000 });
    await sidebar.getByText("crud_test_products", { exact: true }).click();

    await page.getByRole("button", { name: "New Row" }).click();
    await expect(page.getByText("New Record")).toBeVisible();

    // RecordForm uses label-based layout (no name attributes on inputs)
    await page.locator('label:has-text("name")').locator('..').locator('input, textarea').first().fill("Laptop");
    await page.locator('label:has-text("price")').locator('..').locator('input, textarea').first().fill("999.99");
    await page.locator('label:has-text("stock")').locator('..').locator('input, textarea').first().fill("10");
    await page.locator('label:has-text("description")').locator('..').locator('input, textarea').first()
      .fill("High-performance laptop");

    await page.getByRole("button", { name: "Create" }).click();

    // ============================================================
    // READ: Verify product appears in list
    // ============================================================
    await expect(page.getByRole("cell", { name: "Laptop", exact: true })).toBeVisible();
    await expect(page.getByRole("cell", { name: "999.99" })).toBeVisible();

    // ============================================================
    // UPDATE: Edit the product
    // ============================================================
    // Click the Edit button directly in the row. Do NOT click the row body first,
    // because that opens a Row Detail drawer whose overlay blocks the row buttons.
    const laptopRow = page.locator("tr").filter({ hasText: "Laptop" });
    await laptopRow.getByRole("button", { name: /edit/i }).click();

    // Update price and stock
    const priceInput = page.locator('label:has-text("price")').locator('..').locator('input, textarea').first();
    await priceInput.clear();
    await priceInput.fill("899.99");

    const stockInput = page.locator('label:has-text("stock")').locator('..').locator('input, textarea').first();
    await stockInput.clear();
    await stockInput.fill("15");

    // Save changes — button text is "Save Changes"
    const saveButton = page.getByRole("button", { name: /save changes|^save$|^update$/i }).first();
    await saveButton.click();

    // Verify updated values appear
    await expect(page.getByText("899.99")).toBeVisible({ timeout: 5000 });

    // Dismiss any open drawer/overlay before proceeding
    await page.keyboard.press("Escape");
    await page.waitForTimeout(300);

    // ============================================================
    // DELETE: Remove the product
    // ============================================================
    // Click Delete button directly in the row (don't click row body to avoid drawer overlay)
    const updatedRow = page.locator("tr").filter({ hasText: "Laptop" });
    await updatedRow.getByRole("button", { name: /delete/i }).first().click();

    // Confirm deletion — dialog heading is "Delete record?". Scope to dialog container
    // to avoid matching the row's Delete button (strict mode violation).
    const dialogHeading = page.getByRole("heading", { name: /delete record/i });
    await expect(dialogHeading).toBeVisible({ timeout: 3000 });
    await dialogHeading.locator("..").getByRole("button", { name: "Delete", exact: true }).click();

    // Verify product is gone
    await expect(page.getByRole("cell", { name: "Laptop", exact: true })).not.toBeVisible({ timeout: 5000 });

    // Should show empty state or "no rows"
    await expect(page.getByText(/no.*rows|no.*records|0.*items/i)).toBeVisible();

    console.log("✅ Full CRUD test passed");
  });
});
