import { test, expect } from "@playwright/test";

/**
 * SMOKE TEST: Admin Dashboard - Login
 *
 * Critical Path: Admin enters password → Dashboard loads
 *
 * This test validates:
 * 1. Login form displays when auth is configured
 * 2. Correct password grants access
 * 3. Wrong password shows error
 *
 * UI-ONLY: No direct API calls allowed
 *
 * Note: This test uses its OWN storage state (no pre-auth)
 * since it tests the login flow itself.
 */

// Override storageState — login test must start unauthenticated
test.use({ storageState: { cookies: [], origins: [] } });

test.describe("Smoke: Admin Login", () => {
  test("admin can log in with correct password", async ({ page, request }) => {
    // Check if auth is enabled
    const status = await request.get("/api/admin/status");
    const { auth } = await status.json();
    test.skip(!auth, "admin.password not configured — no-auth mode");

    // Step 1: Navigate to admin dashboard
    await page.goto("/admin/");

    // Step 2: Verify login form is shown
    await expect(page.getByText("Enter the admin password")).toBeVisible();
    await expect(page.getByRole("button", { name: "Sign in" })).toBeVisible();

    // Step 3: Enter admin password from env
    const adminPassword = process.env.AYB_ADMIN_PASSWORD || "admin";
    await page.getByLabel("Password").fill(adminPassword);

    // Step 4: Click Sign in
    await page.getByRole("button", { name: "Sign in" }).click();

    // Step 5: Verify dashboard loads
    await expect(page.locator("aside").getByText("AYB Admin")).toBeVisible({ timeout: 10000 });

    console.log("✅ Smoke test passed: Admin login");
  });

  test("admin login rejects wrong password", async ({ page, request }) => {
    // Check if auth is enabled
    const status = await request.get("/api/admin/status");
    const { auth } = await status.json();
    test.skip(!auth, "admin.password not configured — no-auth mode");

    // Step 1: Navigate to admin dashboard
    await page.goto("/admin/");

    // Step 2: Enter wrong password
    await page.getByLabel("Password").fill("wrongpassword123");

    // Step 3: Click Sign in
    await page.getByRole("button", { name: "Sign in" }).click();

    // Step 4: Verify error message
    await expect(page.getByText("invalid password")).toBeVisible({ timeout: 5000 });

    // Step 5: Verify we're still on login form
    await expect(page.getByRole("button", { name: "Sign in" })).toBeVisible();

    console.log("✅ Smoke test passed: Admin login rejects wrong password");
  });
});
