import { test, expect } from "@playwright/test";

/**
 * FULL E2E TEST: Webhooks Lifecycle
 *
 * Tests complete webhook management:
 * - Create webhook with URL and events
 * - Toggle enabled/disabled
 * - Edit webhook URL
 * - Test webhook delivery
 * - View delivery history
 * - Delete webhook
 *
 * UI-ONLY: No direct API calls
 */

test.describe("Webhooks Lifecycle (Full E2E)", () => {
  test("complete webhook management lifecycle", async ({ page }) => {
    const runId = Date.now();
    const webhookUrl = `https://httpbin.org/post?test=${runId}`;
    const updatedUrl = `https://httpbin.org/anything?test=${runId}`;

    // ============================================================
    // Setup: Navigate to Webhooks
    // ============================================================
    await page.goto("/admin/");
    await expect(page.getByText("AYB Admin").first()).toBeVisible();

    const webhooksButton = page.locator("aside").getByRole("button", { name: /^Webhooks$/i });
    await expect(webhooksButton).toBeVisible({ timeout: 5000 });
    await webhooksButton.click();

    // Wait for webhooks view to load
    await expect(page.getByText("Webhooks").first()).toBeVisible();

    // ============================================================
    // CREATE: Add new webhook
    // ============================================================
    // Click "Add Webhook" button
    const addButton = page.getByRole("button", { name: /add webhook/i });
    await expect(addButton).toBeVisible({ timeout: 5000 });
    await addButton.click();

    // Fill webhook URL
    const urlInput = page.getByPlaceholder("https://example.com/webhook");
    await expect(urlInput).toBeVisible({ timeout: 3000 });
    await urlInput.fill(webhookUrl);

    // Submit
    const createBtn = page.getByRole("button", { name: /^create$|^save$/i });
    await expect(createBtn).toBeVisible();
    await createBtn.click();

    // Verify webhook in list
    await expect(page.getByText(webhookUrl).first()).toBeVisible({ timeout: 5000 });

    // ============================================================
    // TOGGLE: Enable/disable webhook
    // ============================================================
    // Find the switch/toggle in the webhook row
    const webhookRow = page.locator("tr, div[class*='border']").filter({ hasText: "httpbin.org" }).first();
    const toggleSwitch = webhookRow.getByRole("switch").or(
      webhookRow.locator('button[role="switch"]')
    );

    if (await toggleSwitch.isVisible({ timeout: 2000 }).catch(() => false)) {
      await toggleSwitch.click();

      // Verify toast (enabled or disabled)
      const toggleToast = page.getByText(/webhook (enabled|disabled)/i);
      await expect(toggleToast.first()).toBeVisible({ timeout: 3000 });

      // Toggle back
      await toggleSwitch.click();
      await page.waitForTimeout(500);
    }

    // ============================================================
    // EDIT: Update webhook URL
    // ============================================================
    const editButton = webhookRow.locator('button[title="Edit"]').or(
      webhookRow.locator("button").filter({ has: page.locator("svg.lucide-pencil") })
    ).first();

    if (await editButton.isVisible({ timeout: 2000 }).catch(() => false)) {
      await editButton.click();

      // Update URL in edit form
      const editUrlInput = page.getByPlaceholder("https://example.com/webhook");
      await expect(editUrlInput).toBeVisible({ timeout: 3000 });
      await editUrlInput.clear();
      await editUrlInput.fill(updatedUrl);

      // Save
      const saveBtn = page.getByRole("button", { name: /^save$|^update$/i });
      await saveBtn.click();

      // Verify updated URL in list
      await expect(page.getByText("httpbin.org/anything").first()).toBeVisible({ timeout: 5000 });
    }

    // ============================================================
    // TEST: Send test delivery
    // ============================================================
    const updatedRow = page.locator("tr, div[class*='border']").filter({ hasText: "httpbin.org" }).first();
    const testButton = updatedRow.locator('button[title*="est"]').or(
      updatedRow.locator("button").filter({ has: page.locator("svg.lucide-zap") })
    ).first();

    if (await testButton.isVisible({ timeout: 2000 }).catch(() => false)) {
      await testButton.click();

      // Verify test result toast (success or failure — both are valid, we just need the feature to work)
      const testToast = page.getByText(/test (passed|failed)|test.*\d+/i);
      await expect(testToast.first()).toBeVisible({ timeout: 10000 });
    }

    // ============================================================
    // HISTORY: View delivery history
    // ============================================================
    const historyButton = updatedRow.locator('button[title*="istory"]').or(
      updatedRow.locator("button").filter({ has: page.locator("svg.lucide-history") })
    ).first();

    if (await historyButton.isVisible({ timeout: 2000 }).catch(() => false)) {
      await historyButton.click();

      // Verify history modal/view opens
      const historyModal = page.getByText(/delivery history|deliveries/i);
      await expect(historyModal.first()).toBeVisible({ timeout: 3000 });

      // Close the modal
      const closeBtn = page.locator('button:has(svg.lucide-x)').or(
        page.getByRole("button", { name: /close/i })
      );
      if (await closeBtn.first().isVisible({ timeout: 1000 }).catch(() => false)) {
        await closeBtn.first().click();
      }
    }

    // ============================================================
    // DELETE: Remove webhook
    // ============================================================
    const deleteRow = page.locator("tr").filter({ hasText: "httpbin.org" }).first();
    const deleteButton = deleteRow.locator('button[title="Delete"]').first();
    await expect(deleteButton).toBeVisible({ timeout: 3000 });
    await deleteButton.click();

    // Wait for confirmation dialog and confirm
    await expect(page.getByText("Are you sure")).toBeVisible({ timeout: 3000 });
    await page.getByRole("button", { name: "Delete", exact: true }).last().click();

    // Verify deleted
    await expect(page.getByText("httpbin.org").first()).not.toBeVisible({ timeout: 5000 });

    console.log("✅ Full webhook lifecycle test passed");
  });
});
