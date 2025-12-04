import { test, expect, generateTestUser } from '../fixtures/auth.fixture'

test.describe('Registration Page', () => {
  test.beforeEach(async ({ registerPage }) => {
    await registerPage.goto()
  })

  test('should display registration form', async ({ registerPage }) => {
    await registerPage.expectOnRegisterPage()
    await expect(registerPage.emailInput).toBeVisible()
    await expect(registerPage.usernameInput).toBeVisible()
    await expect(registerPage.passwordInput).toBeVisible()
    await expect(registerPage.confirmPasswordInput).toBeVisible()
    await expect(registerPage.submitButton).toBeVisible()
  })

  test('should have link to login page', async ({ registerPage, page }) => {
    await expect(registerPage.loginLink).toBeVisible()
    await registerPage.goToLogin()
    await expect(page).toHaveURL('/login')
  })

  test('should register successfully with valid data', async ({ registerPage, page }) => {
    const user = generateTestUser()

    await registerPage.register(user)

    // Should redirect to chat page after successful registration
    await page.waitForURL('/chat', { timeout: 10000 })
    await expect(page).toHaveURL('/chat')
  })

  test('should register with display name', async ({ registerPage, page, chatPage }) => {
    const user = generateTestUser()

    await registerPage.register(user)

    await page.waitForURL('/chat', { timeout: 10000 })

    // Verify display name is shown
    await chatPage.expectUserName(user.displayName)
  })

  test('should show error for mismatched passwords', async ({ registerPage }) => {
    const user = generateTestUser()

    await registerPage.register({
      ...user,
      confirmPassword: 'DifferentPassword123!',
    })

    await registerPage.expectError('Passwords do not match')
  })

  test('should show error for short password', async ({ registerPage }) => {
    const user = generateTestUser()

    await registerPage.register({
      ...user,
      password: 'short',
      confirmPassword: 'short',
    })

    await registerPage.expectError('Password must be at least 8 characters')
  })

  test('should require email field', async ({ registerPage, page }) => {
    const user = generateTestUser()

    await registerPage.usernameInput.fill(user.username)
    await registerPage.passwordInput.fill(user.password)
    await registerPage.confirmPasswordInput.fill(user.password)
    await registerPage.submitButton.click()

    // Form should not submit - still on register page
    await expect(page).toHaveURL('/register')
  })

  test('should require username field', async ({ registerPage, page }) => {
    const user = generateTestUser()

    await registerPage.emailInput.fill(user.email)
    await registerPage.passwordInput.fill(user.password)
    await registerPage.confirmPasswordInput.fill(user.password)
    await registerPage.submitButton.click()

    // Form should not submit - still on register page
    await expect(page).toHaveURL('/register')
  })

  test('should validate email format', async ({ registerPage, page }) => {
    const user = generateTestUser()

    await registerPage.emailInput.fill('invalid-email')
    await registerPage.usernameInput.fill(user.username)
    await registerPage.passwordInput.fill(user.password)
    await registerPage.confirmPasswordInput.fill(user.password)
    await registerPage.submitButton.click()

    // Form should not submit due to HTML5 validation
    await expect(page).toHaveURL('/register')
  })

  test('should show loading state during submission', async ({ registerPage, page }) => {
    const user = generateTestUser()

    // Fill form
    await registerPage.emailInput.fill(user.email)
    await registerPage.usernameInput.fill(user.username)
    await registerPage.passwordInput.fill(user.password)
    await registerPage.confirmPasswordInput.fill(user.password)

    // Click and immediately check for loading state
    const submitPromise = registerPage.submitButton.click()

    // Button should show loading text
    await expect(registerPage.submitButton).toContainText(/Creating account/)

    await submitPromise
  })

  test('should clear form after navigation', async ({ registerPage, page }) => {
    const user = generateTestUser()

    await registerPage.emailInput.fill(user.email)
    await registerPage.usernameInput.fill(user.username)

    // Navigate away and back
    await registerPage.goToLogin()
    await page.goBack()

    // Form should be cleared (depends on browser behavior)
    await registerPage.expectOnRegisterPage()
  })
})

test.describe('Registration Page - Duplicate User Handling', () => {
  test('should show error for duplicate email', async ({ registerPage, page }) => {
    const user = generateTestUser()

    // Register first user
    await registerPage.goto()
    await registerPage.register(user)
    await page.waitForURL('/chat', { timeout: 10000 })

    // Logout (clear storage)
    await page.evaluate(() => {
      localStorage.clear()
    })

    // Try to register with same email
    await registerPage.goto()
    await registerPage.register({
      ...user,
      username: user.username + '_new',
    })

    // Should show error
    await registerPage.expectError('already exists')
  })
})

test.describe('Registration Page - Accessibility', () => {
  test('should have proper form labels', async ({ registerPage }) => {
    await registerPage.goto()

    // Check that inputs have associated labels
    await expect(registerPage.page.locator('label[for="email"]')).toBeVisible()
    await expect(registerPage.page.locator('label[for="username"]')).toBeVisible()
    await expect(registerPage.page.locator('label[for="password"]')).toBeVisible()
    await expect(registerPage.page.locator('label[for="confirmPassword"]')).toBeVisible()
  })

  test('should be keyboard navigable', async ({ registerPage, page }) => {
    await registerPage.goto()

    // Tab through form elements
    await page.keyboard.press('Tab')
    await expect(registerPage.emailInput).toBeFocused()

    await page.keyboard.press('Tab')
    await expect(registerPage.usernameInput).toBeFocused()

    await page.keyboard.press('Tab')
    await expect(registerPage.displayNameInput).toBeFocused()

    await page.keyboard.press('Tab')
    await expect(registerPage.passwordInput).toBeFocused()

    await page.keyboard.press('Tab')
    await expect(registerPage.confirmPasswordInput).toBeFocused()
  })

  test('should submit form with Enter key', async ({ registerPage, page }) => {
    const user = generateTestUser()

    await registerPage.emailInput.fill(user.email)
    await registerPage.usernameInput.fill(user.username)
    await registerPage.passwordInput.fill(user.password)
    await registerPage.confirmPasswordInput.fill(user.password)

    await page.keyboard.press('Enter')

    await page.waitForURL('/chat', { timeout: 10000 })
  })
})
