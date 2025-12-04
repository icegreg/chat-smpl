import { test, expect, generateTestUser, createAndLoginUser } from '../fixtures/auth.fixture'

test.describe('Login Page', () => {
  test.beforeEach(async ({ loginPage }) => {
    await loginPage.goto()
  })

  test('should display login form', async ({ loginPage }) => {
    await loginPage.expectOnLoginPage()
    await expect(loginPage.emailInput).toBeVisible()
    await expect(loginPage.passwordInput).toBeVisible()
    await expect(loginPage.submitButton).toBeVisible()
  })

  test('should have link to register page', async ({ loginPage, page }) => {
    await expect(loginPage.registerLink).toBeVisible()
    await loginPage.goToRegister()
    await expect(page).toHaveURL('/register')
  })

  test('should login successfully with valid credentials', async ({
    registerPage,
    loginPage,
    page,
  }) => {
    // First register a user
    const user = await createAndLoginUser(registerPage, loginPage)

    // Logout
    await page.evaluate(() => localStorage.clear())
    await loginPage.goto()

    // Login
    await loginPage.login(user.email, user.password)

    // Should redirect to chat page
    await page.waitForURL('/chat', { timeout: 10000 })
    await expect(page).toHaveURL('/chat')
  })

  test('should show error for invalid credentials', async ({ loginPage }) => {
    await loginPage.login('invalid@example.com', 'wrongpassword')

    await loginPage.expectError('Invalid')
  })

  test('should show error for non-existent user', async ({ loginPage }) => {
    await loginPage.login('nonexistent@example.com', 'password123')

    await loginPage.expectError('Invalid')
  })

  test('should require email field', async ({ loginPage, page }) => {
    await loginPage.passwordInput.fill('password123')
    await loginPage.submitButton.click()

    // Form should not submit - still on login page
    await expect(page).toHaveURL('/login')
  })

  test('should require password field', async ({ loginPage, page }) => {
    await loginPage.emailInput.fill('test@example.com')
    await loginPage.submitButton.click()

    // Form should not submit - still on login page
    await expect(page).toHaveURL('/login')
  })

  test('should validate email format', async ({ loginPage, page }) => {
    await loginPage.emailInput.fill('invalid-email')
    await loginPage.passwordInput.fill('password123')
    await loginPage.submitButton.click()

    // Form should not submit due to HTML5 validation
    await expect(page).toHaveURL('/login')
  })

  test('should show loading state during submission', async ({ loginPage }) => {
    await loginPage.emailInput.fill('test@example.com')
    await loginPage.passwordInput.fill('password123')

    const submitPromise = loginPage.submitButton.click()

    // Button should show loading text
    await expect(loginPage.submitButton).toContainText(/Signing in/)

    await submitPromise
  })

  test('should preserve redirect URL after login', async ({
    registerPage,
    loginPage,
    page,
  }) => {
    // First register a user
    const user = await createAndLoginUser(registerPage, loginPage)

    // Logout
    await page.evaluate(() => localStorage.clear())

    // Try to access protected route
    await page.goto('/chat/some-chat-id')

    // Should redirect to login with redirect param
    await page.waitForURL(/\/login\?redirect=/)

    // Login
    await loginPage.login(user.email, user.password)

    // Should redirect back to original URL (or /chat if chat doesn't exist)
    await page.waitForURL(/\/chat/, { timeout: 10000 })
  })
})

test.describe('Login Page - Session Management', () => {
  test('should persist session after page reload', async ({
    registerPage,
    loginPage,
    page,
    chatPage,
  }) => {
    // Register and login
    const user = await createAndLoginUser(registerPage, loginPage)

    // Verify on chat page
    await chatPage.expectOnChatPage()

    // Reload page
    await page.reload()

    // Should still be on chat page
    await expect(page).toHaveURL('/chat')
    await chatPage.expectOnChatPage()
  })

  test('should redirect to chat if already logged in', async ({
    registerPage,
    loginPage,
    page,
  }) => {
    // Register and login
    await createAndLoginUser(registerPage, loginPage)

    // Try to visit login page
    await page.goto('/login')

    // Should redirect to chat
    await page.waitForURL('/chat', { timeout: 5000 })
  })

  test('should redirect to chat from register if already logged in', async ({
    registerPage,
    loginPage,
    page,
  }) => {
    // Register and login
    await createAndLoginUser(registerPage, loginPage)

    // Try to visit register page
    await page.goto('/register')

    // Should redirect to chat
    await page.waitForURL('/chat', { timeout: 5000 })
  })
})

test.describe('Login Page - Accessibility', () => {
  test('should have proper form labels', async ({ loginPage }) => {
    await loginPage.goto()

    // Check for labels (sr-only in this case)
    await expect(loginPage.page.locator('label[for="email"]')).toBeAttached()
    await expect(loginPage.page.locator('label[for="password"]')).toBeAttached()
  })

  test('should be keyboard navigable', async ({ loginPage, page }) => {
    await loginPage.goto()

    // Tab through form elements
    await page.keyboard.press('Tab')
    await expect(loginPage.emailInput).toBeFocused()

    await page.keyboard.press('Tab')
    await expect(loginPage.passwordInput).toBeFocused()

    await page.keyboard.press('Tab')
    await expect(loginPage.submitButton).toBeFocused()
  })

  test('should submit form with Enter key', async ({
    registerPage,
    loginPage,
    page,
  }) => {
    // First register a user
    const user = await createAndLoginUser(registerPage, loginPage)

    // Logout and go to login
    await page.evaluate(() => localStorage.clear())
    await loginPage.goto()

    await loginPage.emailInput.fill(user.email)
    await loginPage.passwordInput.fill(user.password)

    await page.keyboard.press('Enter')

    await page.waitForURL('/chat', { timeout: 10000 })
  })
})

test.describe('Login Page - Security', () => {
  test('should not expose password in URL', async ({ loginPage, page }) => {
    await loginPage.emailInput.fill('test@example.com')
    await loginPage.passwordInput.fill('secretpassword')
    await loginPage.submitButton.click()

    // URL should not contain password
    expect(page.url()).not.toContain('secretpassword')
  })

  test('should use password input type', async ({ loginPage }) => {
    const inputType = await loginPage.passwordInput.getAttribute('type')
    expect(inputType).toBe('password')
  })

  test('should have autocomplete attributes', async ({ loginPage }) => {
    const emailAutocomplete = await loginPage.emailInput.getAttribute('autocomplete')
    const passwordAutocomplete = await loginPage.passwordInput.getAttribute('autocomplete')

    expect(emailAutocomplete).toBe('email')
    expect(passwordAutocomplete).toBe('current-password')
  })
})
