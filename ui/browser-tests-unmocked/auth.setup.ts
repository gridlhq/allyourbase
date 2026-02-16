import { test as setup, expect } from "@playwright/test";

/**
 * AUTH SETUP: Log into the admin dashboard and save auth state.
 *
 * All smoke and full test projects depend on this setup.
 * The saved storageState includes the JWT token in localStorage,
 * which is automatically loaded before each test.
 */

const authFile = "browser-tests-unmocked/.auth/admin.json";

setup("authenticate as admin", async ({ page }) => {
  // Navigate to admin login
  await page.goto("/admin/");

  // Wait for login form
  await expect(page.getByText("Enter the admin password")).toBeVisible({
    timeout: 15000,
  });

  // Enter admin password
  const adminPassword = process.env.AYB_ADMIN_PASSWORD || "admin";
  await page.getByLabel("Password").fill(adminPassword);

  // Click Sign in
  await page.getByRole("button", { name: "Sign in" }).click();

  // Wait for dashboard to load (sidebar with "AYB Admin" text)
  await expect(page.locator("aside")).toBeVisible({ timeout: 10000 });

  // Save auth state (localStorage with JWT token)
  await page.context().storageState({ path: authFile });
});
