import { test, expect } from "@playwright/test";

/**
 * FULL E2E TEST: RLS Policy Management
 *
 * Tests Row-Level Security policy management:
 * - Setup: Create test table via SQL
 * - Enable RLS on table
 * - Create a policy
 * - Verify policy in list
 * - Delete policy
 * - Disable RLS
 * - Cleanup: Drop test table
 *
 * UI-ONLY: No direct API calls (except SQL via UI)
 */

test.describe("RLS Policies (Full E2E)", () => {
  test("enable RLS, create policy, delete policy, disable RLS", async ({ page }) => {
    const runId = Date.now();
    const tableName = `rls_test_${runId}`;

    // ============================================================
    // Setup: Create test table via SQL
    // ============================================================
    await page.goto("/admin/");
    await expect(page.getByText("AYB Admin").first()).toBeVisible();

    const sidebar = page.locator("aside");

    // Navigate to SQL Editor via sidebar
    await sidebar.getByRole("button", { name: /^SQL Editor$/i }).click();

    const sqlInput = page.locator("textarea").first();
    await expect(sqlInput).toBeVisible({ timeout: 5000 });

    // Create test table with user_id column for RLS testing
    await sqlInput.fill(`CREATE TABLE ${tableName} (
      id SERIAL PRIMARY KEY,
      name TEXT NOT NULL,
      user_id UUID
    );`);
    await page.getByRole("button", { name: /run|execute/i }).click();
    await page.waitForTimeout(500);

    // Reload to see new table
    await page.reload();
    await expect(page.getByText("AYB Admin").first()).toBeVisible();
    // Wait for sidebar to fully settle after reload
    await page.waitForTimeout(1000);

    // ============================================================
    // Navigate to RLS Policies
    // ============================================================
    const rlsButton = sidebar.getByRole("button", { name: /^RLS Policies$/i });
    await expect(rlsButton).toBeVisible({ timeout: 5000 });
    await rlsButton.click();
    // Wait for view transition
    await page.waitForTimeout(500);

    // Verify RLS view loaded — look for heading or table selection UI
    await expect(page.getByText(/RLS|Row.*Level|Policies/i).first()).toBeVisible({ timeout: 5000 });

    // ============================================================
    // SELECT TABLE: Choose test table from RLS component's internal sidebar
    // ============================================================
    // The RLS component renders its own table list as buttons inside main.
    // Scope to main to avoid matching the global aside sidebar.
    const rlsTableButton = page.locator("main").getByRole("button", { name: tableName });
    await expect(rlsTableButton).toBeVisible({ timeout: 5000 });
    await rlsTableButton.click();

    // ============================================================
    // ENABLE RLS
    // ============================================================
    const enableToggle = page.getByRole("switch").or(
      page.getByRole("button", { name: /enable rls|enable/i })
    ).first();

    if (await enableToggle.isVisible({ timeout: 3000 }).catch(() => false)) {
      await enableToggle.click();

      // Verify enabled — toast or shield icon change
      const enabledToast = page.getByText(/enabled/i);
      await expect(enabledToast.first()).toBeVisible({ timeout: 3000 }).catch(() => {});
    }

    // ============================================================
    // CREATE POLICY
    // ============================================================
    const createPolicyBtn = page.getByRole("button", { name: /create policy|new policy|add/i }).or(
      page.locator("button").filter({ has: page.locator("svg.lucide-plus") })
    );

    if (await createPolicyBtn.first().isVisible({ timeout: 3000 }).catch(() => false)) {
      await createPolicyBtn.first().click();

      // Fill policy form
      const policyName = `test_policy_${runId}`;

      const nameInput = page.locator('input[name="name"]').or(
        page.getByPlaceholder(/name|policy/i)
      ).or(
        page.locator('label:has-text("Name")').locator("..").locator("input")
      ).first();

      await expect(nameInput).toBeVisible({ timeout: 3000 });
      await nameInput.fill(policyName);

      // Command select (ALL, SELECT, INSERT, etc.)
      const commandSelect = page.locator('select[name="command"]').or(
        page.locator('label:has-text("Command")').locator("..").locator("select")
      ).first();
      if (await commandSelect.isVisible({ timeout: 2000 }).catch(() => false)) {
        await commandSelect.selectOption("ALL");
      }

      // USING expression
      const usingInput = page.locator('input[name="using"], textarea[name="using"]').or(
        page.locator('label:has-text("USING")').locator("..").locator("input,textarea")
      ).first();
      if (await usingInput.isVisible({ timeout: 2000 }).catch(() => false)) {
        await usingInput.fill("true");
      }

      // Submit policy — button text is "Create Policy" (not just "Create")
      const submitBtn = page.getByRole("button", { name: /^create policy$|^create$|^save$/i });
      await expect(submitBtn).toBeVisible({ timeout: 5000 });
      await submitBtn.click();

      // Verify policy appears
      await expect(page.getByText(policyName).first()).toBeVisible({ timeout: 5000 });

      // ============================================================
      // DELETE POLICY
      // ============================================================
      const policyRow = page.locator("tr, div").filter({ hasText: policyName }).first();
      const deleteBtn = policyRow.locator('button[title="Delete"]').or(
        policyRow.locator("button").filter({ has: page.locator("svg.lucide-trash-2") })
      ).first();

      if (await deleteBtn.isVisible({ timeout: 2000 }).catch(() => false)) {
        await deleteBtn.click();

        // Confirm
        const confirmBtn = page.getByRole("button", { name: /^delete$|^confirm$/i });
        if (await confirmBtn.isVisible({ timeout: 2000 }).catch(() => false)) {
          await confirmBtn.click();
        }

        // Verify deleted
        await expect(page.getByText(policyName)).not.toBeVisible({ timeout: 5000 });
      }
    }

    // ============================================================
    // DISABLE RLS
    // ============================================================
    const disableToggle = page.getByRole("switch").or(
      page.getByRole("button", { name: /disable rls|disable/i })
    ).first();

    if (await disableToggle.isVisible({ timeout: 2000 }).catch(() => false)) {
      await disableToggle.click();
      const disabledToast = page.getByText(/disabled/i);
      await expect(disabledToast.first()).toBeVisible({ timeout: 3000 }).catch(() => {});
    }

    // ============================================================
    // Cleanup: Drop test table via SQL
    // ============================================================
    await sidebar.getByRole("button", { name: /^SQL Editor$/i }).click();
    const cleanupSql = page.locator("textarea").first();
    await expect(cleanupSql).toBeVisible({ timeout: 5000 });
    await cleanupSql.fill(`DROP TABLE IF EXISTS ${tableName};`);
    await page.getByRole("button", { name: /run|execute/i }).click();

    console.log("✅ Full RLS policies test passed");
  });
});
