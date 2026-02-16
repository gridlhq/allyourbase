import { test, expect } from "@playwright/test";

/**
 * SMOKE TEST: Storage - Upload, Download, Delete
 *
 * Critical Path: Navigate to Storage → Upload file → Verify in list → Delete
 *
 * This test validates:
 * 1. Storage interface loads
 * 2. File upload works
 * 3. Uploaded files appear in file list
 * 4. File deletion works
 *
 * UI-ONLY: No direct API calls allowed
 */

test.describe("Smoke: Storage", () => {
  test("upload file and delete via storage UI", async ({ page }) => {
    const runId = Date.now();

    // Step 1: Navigate to admin dashboard
    await page.goto("/admin/");
    await expect(page.getByText("AYB Admin").first()).toBeVisible();

    // Step 2: Navigate to Storage section (button in admin sidebar)
    const storageButton = page.locator("aside").getByRole("button", { name: /^Storage$/i });
    await expect(storageButton).toBeVisible({ timeout: 5000 });
    await storageButton.click();

    // Step 3: Wait for storage view to load (Upload button in toolbar — exact match to avoid "Upload your first file")
    const uploadButton = page.getByRole("button", { name: "Upload", exact: true });
    await expect(uploadButton).toBeVisible({ timeout: 5000 });

    // Step 4: Upload a file via the hidden file input
    const fileName = `smoke-test-${runId}.txt`;
    const fileInput = page.locator('input[type="file"]');
    await fileInput.setInputFiles({
      name: fileName,
      mimeType: "text/plain",
      buffer: Buffer.from("This is a smoke test file for upload and delete"),
    });

    // Step 5: Verify upload success (toast or file in list)
    const fileInList = page.getByText(fileName);
    const successToast = page.getByText(/uploaded/i);
    await expect(fileInList.or(successToast).first()).toBeVisible({ timeout: 10000 });

    // Step 6: Verify file is in the list
    await expect(page.getByText(fileName)).toBeVisible({ timeout: 5000 });

    // Step 7: Delete the file — find the trash icon in the file row
    const fileRow = page.locator("tr, div").filter({ hasText: fileName }).first();
    const deleteButton = fileRow.locator('button[title="Delete"]').first();
    await expect(deleteButton).toBeVisible({ timeout: 3000 });
    await deleteButton.click();

    // Step 8: Wait for confirmation dialog and confirm
    await expect(page.getByText("Are you sure")).toBeVisible({ timeout: 3000 });
    await page.getByRole("button", { name: "Delete", exact: true }).last().click();

    // Step 9: Verify file removed
    await expect(page.getByText(fileName)).not.toBeVisible({ timeout: 5000 });

    console.log("✅ Smoke test passed: Storage upload and delete");
  });
});
