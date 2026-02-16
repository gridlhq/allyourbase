import { test, expect } from "@playwright/test";

/**
 * SMOKE TEST: Webhooks - Create and Delete
 *
 * Critical Path: Navigate to Webhooks → Create webhook → Verify in list → Delete
 *
 * This test validates:
 * 1. Webhooks section loads from sidebar
 * 2. Webhook creation form works
 * 3. Created webhook appears in list
 * 4. Webhook deletion works
 *
 * UI-ONLY: No direct API calls allowed
 */

test.describe("Smoke: Webhooks CRUD", () => {
  test("create and delete a webhook via UI", async ({ page }) => {
    const runId = Date.now();

    // Step 1: Navigate to admin dashboard
    await page.goto("/admin/");
    await expect(page.getByText("AYB Admin").first()).toBeVisible();

    // Step 2: Navigate to Webhooks section
    const webhooksButton = page.locator("aside").getByRole("button", { name: /^Webhooks$/i });
    await expect(webhooksButton).toBeVisible({ timeout: 5000 });
    await webhooksButton.click();

    // Step 3: Verify webhooks view loaded
    await expect(page.getByText("Webhooks").first()).toBeVisible();

    // Step 4: Click "Add Webhook" button
    const addWebhookBtn = page.getByRole("button", { name: /add webhook/i });
    await expect(addWebhookBtn).toBeVisible({ timeout: 5000 });
    await addWebhookBtn.click();

    // Step 5: Fill webhook URL
    const webhookUrl = `https://httpbin.org/post?run=${runId}`;
    const urlInput = page.getByPlaceholder("https://example.com/webhook");
    await expect(urlInput).toBeVisible({ timeout: 5000 });
    await urlInput.fill(webhookUrl);

    // Step 6: Submit the form
    const submitButton = page.getByRole("button", { name: /^create$|^save$/i });
    await expect(submitButton).toBeVisible();
    await submitButton.click();

    // Step 7: Verify webhook appears in list
    await expect(page.getByText(webhookUrl).first()).toBeVisible({ timeout: 5000 });

    // Step 8: Delete the webhook — find the trash icon in the row
    const webhookRow = page.locator("tr").filter({ hasText: webhookUrl }).first();
    const deleteBtn = webhookRow.locator('button[title="Delete"]').first();
    await expect(deleteBtn).toBeVisible({ timeout: 3000 });
    await deleteBtn.click();

    // Step 9: Wait for confirmation dialog and confirm
    await expect(page.getByText("Are you sure")).toBeVisible({ timeout: 3000 });
    await page.getByRole("button", { name: "Delete", exact: true }).last().click();

    // Step 10: Verify webhook is removed
    await expect(page.getByText(webhookUrl)).not.toBeVisible({ timeout: 5000 });

    console.log("✅ Smoke test passed: Webhooks create and delete");
  });
});
