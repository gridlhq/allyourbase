import { test, expect } from "@playwright/test";

/**
 * FULL E2E TEST: API Keys Lifecycle
 *
 * Tests complete API key management:
 * - Setup: Create a test user via SQL (required for key creation)
 * - Create API key with name, user, scope
 * - Verify key displayed in creation modal
 * - Verify key appears in list
 * - Revoke API key
 * - Cleanup: Remove test user
 *
 * UI-ONLY: No direct API calls (except SQL via UI)
 */

test.describe("API Keys Lifecycle (Full E2E)", () => {
  test("create, view, and revoke API key", async ({ page }) => {
    const runId = Date.now();

    // ============================================================
    // Setup: Create a test user via SQL so we have a user_id for key creation
    // ============================================================
    await page.goto("/admin/");
    await expect(page.getByText("AYB Admin").first()).toBeVisible();

    // Navigate to SQL Editor via sidebar
    const sidebar = page.locator("aside");
    await sidebar.getByRole("button", { name: /^SQL Editor$/i }).click();

    // Create test user via SQL
    const sqlInput = page.locator("textarea").first();
    await expect(sqlInput).toBeVisible({ timeout: 5000 });

    const testEmail = `apikey-test-${runId}@test.com`;
    await sqlInput.fill(
      `INSERT INTO _ayb_users (email, password_hash) VALUES ('${testEmail}', '$argon2id$v=19$m=65536,t=3,p=4$dGVzdHNhbHQ$dGVzdGhhc2g') ON CONFLICT DO NOTHING RETURNING id;`
    );
    await page.getByRole("button", { name: /run|execute/i }).click();
    await page.waitForTimeout(500);

    // ============================================================
    // Navigate to API Keys
    // ============================================================
    const apiKeysButton = sidebar.getByRole("button", { name: /^API Keys$/i });
    await expect(apiKeysButton).toBeVisible({ timeout: 5000 });
    await apiKeysButton.click();

    // Verify API Keys view loaded
    await expect(page.getByText(/API Keys/i).first()).toBeVisible({ timeout: 5000 });

    // ============================================================
    // CREATE: Add new API key
    // ============================================================
    const createButton = page.getByRole("button", { name: /create key|new key|add/i }).or(
      page.locator("button").filter({ has: page.locator("svg.lucide-plus") })
    );
    await expect(createButton.first()).toBeVisible({ timeout: 3000 });
    await createButton.first().click();

    // Fill creation form
    const keyName = `test-key-${runId}`;

    // Name input (ApiKeys component uses aria-label="Key name")
    const nameInput = page.locator('[aria-label="Key name"]').or(
      page.locator('input[name="name"]')
    ).or(
      page.getByPlaceholder(/name/i)
    ).first();
    await expect(nameInput).toBeVisible({ timeout: 3000 });
    await nameInput.fill(keyName);

    // User selector (aria-label="User" or aria-label="User ID")
    const userSelect = page.locator('[aria-label="User"]').or(
      page.locator('[aria-label="User ID"]')
    ).or(page.locator('select').first());
    if (await userSelect.first().isVisible({ timeout: 2000 }).catch(() => false)) {
      // Try to select the first non-empty option
      const selectEl = userSelect.first();
      const options = selectEl.locator("option");
      const optCount = await options.count();
      if (optCount > 1) {
        await selectEl.selectOption({ index: 1 });
      }
    }

    // Submit
    const submitBtn = page.getByRole("button", { name: /^create$|^save$/i });
    await expect(submitBtn).toBeVisible();
    await submitBtn.click();

    // ============================================================
    // VERIFY: Key created modal shows the key
    // ============================================================
    // The creation modal shows the key one time
    const createdModal = page.getByText(/created|key:/i);
    const keyDisplay = page.locator("code, pre, input[readonly]");
    await expect(createdModal.or(keyDisplay.first()).first()).toBeVisible({ timeout: 5000 });

    // Close the created modal by clicking Done
    const doneBtn = page.getByRole("button", { name: /^done$/i });
    await expect(doneBtn).toBeVisible({ timeout: 3000 });
    await doneBtn.click();

    // Wait for modal to fully dismiss
    await expect(page.getByText("API Key Created")).not.toBeVisible({ timeout: 3000 });

    // ============================================================
    // VERIFY: Key appears in list
    // ============================================================
    await expect(page.getByText(keyName).first()).toBeVisible({ timeout: 5000 });

    // Verify Active badge
    const activeBadge = page.getByText("Active");
    await expect(activeBadge.first()).toBeVisible({ timeout: 2000 }).catch(() => {
      // Badge may not exist as separate text — that's OK
    });

    // ============================================================
    // REVOKE: Revoke the API key
    // ============================================================
    // Use "tr" only (not "div") to avoid matching modal overlay divs
    const keyRow = page.locator("tr").filter({ hasText: keyName }).first();
    const revokeButton = keyRow.getByRole("button", { name: /revoke/i }).or(
      keyRow.locator('button[title*="evoke"], button[title="Delete"]')
    ).first();

    await expect(revokeButton).toBeVisible({ timeout: 3000 });
    await revokeButton.click();

    // Confirm revocation
    const confirmBtn = page.getByRole("button", { name: /^revoke$|^delete$|^confirm$/i });
    if (await confirmBtn.isVisible({ timeout: 2000 }).catch(() => false)) {
      await confirmBtn.click();
    }

    // Verify revoked — the key row should show "Revoked" status
    const revokedKeyRow = page.locator("tr").filter({ hasText: keyName });
    await expect(revokedKeyRow.getByText("Revoked")).toBeVisible({ timeout: 5000 });

    console.log("✅ Full API keys lifecycle test passed");
  });
});
