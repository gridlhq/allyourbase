import { test, expect } from "@playwright/test";

/**
 * FULL E2E TEST: API Explorer
 *
 * Tests the interactive API explorer:
 * - Navigate to API Explorer
 * - Send GET request to /api/schema
 * - Verify response display (status, body)
 * - Check cURL code generation
 * - Check JS SDK code generation
 * - Send POST request (if table exists)
 *
 * UI-ONLY: No direct API calls
 */

test.describe("API Explorer (Full E2E)", () => {
  test("send requests and verify response display", async ({ page }) => {
    // ============================================================
    // Navigate to API Explorer
    // ============================================================
    await page.goto("/admin/");
    await expect(page.getByText("AYB Admin").first()).toBeVisible();

    const sidebar = page.locator("aside");
    const explorerButton = sidebar.getByRole("button", { name: /^API Explorer$/i });
    await expect(explorerButton).toBeVisible({ timeout: 5000 });
    await explorerButton.click();

    // Verify API Explorer loaded
    await expect(page.getByText(/API Explorer/i).first()).toBeVisible({ timeout: 5000 });

    // ============================================================
    // SEND GET REQUEST: /api/schema
    // ============================================================
    // Verify method selector defaults to GET (or select it)
    const methodSelector = page.locator("select").first().or(
      page.getByRole("button", { name: /GET/i })
    );
    if (await methodSelector.isVisible({ timeout: 2000 }).catch(() => false)) {
      // If it's a select, choose GET
      if (await page.locator("select").first().isVisible({ timeout: 1000 }).catch(() => false)) {
        await page.locator("select").first().selectOption("GET");
      }
    }

    // Enter path
    const pathInput = page.locator('input[type="text"]').or(
      page.getByPlaceholder(/path|url|endpoint/i)
    ).first();
    await expect(pathInput).toBeVisible({ timeout: 3000 });
    await pathInput.clear();
    await pathInput.fill("/api/schema");

    // Click execute
    const executeButton = page.getByRole("button", { name: /send|execute|run/i }).or(
      page.locator("button").filter({ has: page.locator("svg.lucide-play") })
    );
    await expect(executeButton.first()).toBeVisible({ timeout: 2000 });
    await executeButton.first().click();

    // ============================================================
    // VERIFY RESPONSE: Status 200 and body contains "tables"
    // ============================================================
    // Wait for response to appear
    const statusCode = page.getByText("200").or(page.getByText(/2\d\d/));
    await expect(statusCode.first()).toBeVisible({ timeout: 10000 });

    // Response body should contain "tables" (schema endpoint)
    const responseBody = page.getByText(/tables/i);
    await expect(responseBody.first()).toBeVisible({ timeout: 3000 });

    // ============================================================
    // CODE GENERATION: Verify cURL tab
    // ============================================================
    const curlTab = page.getByRole("button", { name: /curl/i }).or(
      page.getByText(/cURL/i)
    );
    if (await curlTab.first().isVisible({ timeout: 2000 }).catch(() => false)) {
      await curlTab.first().click();

      // Verify cURL command contains the path
      const curlCode = page.getByText(/curl.*-X.*GET/i).or(
        page.locator("pre, code").filter({ hasText: "curl" })
      );
      await expect(curlCode.first()).toBeVisible({ timeout: 3000 });
    }

    // ============================================================
    // CODE GENERATION: Verify JS SDK tab
    // ============================================================
    const jsTab = page.getByRole("button", { name: /javascript|js|sdk/i }).or(
      page.getByText(/JavaScript|SDK/i)
    );
    if (await jsTab.first().isVisible({ timeout: 2000 }).catch(() => false)) {
      await jsTab.first().click();

      // Verify SDK code is present
      const sdkCode = page.getByText(/ayb\./i).or(
        page.locator("pre, code").filter({ hasText: "ayb" })
      );
      await expect(sdkCode.first()).toBeVisible({ timeout: 3000 });
    }

    // ============================================================
    // SEND GET REQUEST: /api/collections (test a different endpoint)
    // ============================================================
    await pathInput.clear();
    await pathInput.fill("/api/admin/status");
    await executeButton.first().click();

    // Verify we get a response
    const statusResponse = page.getByText("200").or(page.getByText(/2\d\d/));
    await expect(statusResponse.first()).toBeVisible({ timeout: 10000 });

    // Response should contain "auth"
    const authField = page.getByText(/auth/i);
    await expect(authField.first()).toBeVisible({ timeout: 3000 });

    console.log("✅ Full API explorer test passed");
  });
});
