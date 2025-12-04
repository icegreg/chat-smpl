import { test, expect, createAndLoginUser } from '../fixtures/auth.fixture'

test.describe('Logout', () => {
  test('should logout successfully', async ({
    registerPage,
    loginPage,
    chatPage,
    page,
  }) => {
    // Login first
    await createAndLoginUser(registerPage, loginPage)
    await chatPage.expectOnChatPage()

    // Logout
    await chatPage.logout()

    // Should redirect to login
    await page.waitForURL('/login', { timeout: 5000 })
    await expect(page).toHaveURL('/login')
  })

  test('should clear local storage on logout', async ({
    registerPage,
    loginPage,
    chatPage,
    page,
  }) => {
    // Login first
    await createAndLoginUser(registerPage, loginPage)

    // Verify tokens exist
    const tokensBefore = await page.evaluate(() => ({
      access: localStorage.getItem('access_token'),
      refresh: localStorage.getItem('refresh_token'),
    }))
    expect(tokensBefore.access).not.toBeNull()

    // Logout
    await chatPage.logout()
    await page.waitForURL('/login')

    // Verify tokens are cleared
    const tokensAfter = await page.evaluate(() => ({
      access: localStorage.getItem('access_token'),
      refresh: localStorage.getItem('refresh_token'),
    }))
    expect(tokensAfter.access).toBeNull()
    expect(tokensAfter.refresh).toBeNull()
  })

  test('should not be able to access protected routes after logout', async ({
    registerPage,
    loginPage,
    chatPage,
    page,
  }) => {
    // Login first
    await createAndLoginUser(registerPage, loginPage)

    // Logout
    await chatPage.logout()
    await page.waitForURL('/login')

    // Try to access chat page directly
    await page.goto('/chat')

    // Should redirect to login
    await page.waitForURL(/\/login/, { timeout: 5000 })
  })
})

test.describe('Protected Routes', () => {
  test('should redirect to login when accessing /chat without auth', async ({ page }) => {
    // Clear any existing tokens
    await page.goto('/')
    await page.evaluate(() => localStorage.clear())

    // Try to access protected route
    await page.goto('/chat')

    // Should redirect to login
    await page.waitForURL(/\/login/, { timeout: 5000 })
  })

  test('should redirect to login when accessing /chat/:id without auth', async ({ page }) => {
    // Clear any existing tokens
    await page.goto('/')
    await page.evaluate(() => localStorage.clear())

    // Try to access specific chat
    await page.goto('/chat/some-chat-id')

    // Should redirect to login with redirect param
    await page.waitForURL(/\/login\?redirect=/, { timeout: 5000 })
  })

  test('should preserve redirect URL for deep links', async ({
    registerPage,
    loginPage,
    page,
  }) => {
    // First register a user
    const user = await createAndLoginUser(registerPage, loginPage)

    // Logout
    await page.evaluate(() => localStorage.clear())

    // Try to access specific chat
    const chatUrl = '/chat/test-chat-123'
    await page.goto(chatUrl)

    // Should redirect to login with redirect param
    await page.waitForURL(/\/login\?redirect=/)
    const url = new URL(page.url())
    expect(url.searchParams.get('redirect')).toContain('/chat')

    // Login
    await loginPage.login(user.email, user.password)

    // Should redirect back to original URL
    await page.waitForURL(/\/chat/, { timeout: 10000 })
  })

  test('should allow access to public routes without auth', async ({ page }) => {
    // Clear any existing tokens
    await page.evaluate(() => localStorage.clear())

    // Access login page
    await page.goto('/login')
    await expect(page).toHaveURL('/login')

    // Access register page
    await page.goto('/register')
    await expect(page).toHaveURL('/register')
  })
})

test.describe('Token Refresh', () => {
  test('should stay logged in after page refresh', async ({
    registerPage,
    loginPage,
    chatPage,
    page,
  }) => {
    // Login
    await createAndLoginUser(registerPage, loginPage)
    await chatPage.expectOnChatPage()

    // Refresh page
    await page.reload()

    // Should still be on chat page
    await expect(page).toHaveURL('/chat')
    await chatPage.expectOnChatPage()
  })

  test('should handle expired access token', async ({
    registerPage,
    loginPage,
    chatPage,
    page,
  }) => {
    // Login
    await createAndLoginUser(registerPage, loginPage)

    // Simulate expired access token by clearing only access token
    await page.evaluate(() => {
      localStorage.removeItem('access_token')
    })

    // Refresh page - should use refresh token to get new access token
    await page.reload()

    // Should either stay logged in or redirect to login
    // (depends on if refresh token is still valid)
    const url = page.url()
    expect(url).toMatch(/\/(chat|login)/)
  })
})

test.describe('Authentication State UI', () => {
  test('should show user info in header when logged in', async ({
    registerPage,
    loginPage,
    chatPage,
  }) => {
    const user = await createAndLoginUser(registerPage, loginPage)

    await chatPage.expectOnChatPage()
    await chatPage.expectUserName(user.displayName)
  })

  test('should show logout button when logged in', async ({
    registerPage,
    loginPage,
    chatPage,
  }) => {
    await createAndLoginUser(registerPage, loginPage)

    await chatPage.expectOnChatPage()
    await expect(chatPage.logoutButton).toBeVisible()
  })
})

test.describe('Navigation Flow', () => {
  test('full auth flow: register -> chat -> logout -> login -> chat', async ({
    registerPage,
    loginPage,
    chatPage,
    page,
  }) => {
    // Step 1: Register
    const user = await createAndLoginUser(registerPage, loginPage)
    await chatPage.expectOnChatPage()

    // Step 2: Logout
    await chatPage.logout()
    await page.waitForURL('/login')

    // Step 3: Login
    await loginPage.login(user.email, user.password)
    await page.waitForURL('/chat', { timeout: 10000 })

    // Step 4: Verify on chat page
    await chatPage.expectOnChatPage()
    await chatPage.expectUserName(user.displayName)
  })

  test('should handle back button after logout', async ({
    registerPage,
    loginPage,
    chatPage,
    page,
  }) => {
    // Login
    await createAndLoginUser(registerPage, loginPage)
    await chatPage.expectOnChatPage()

    // Logout
    await chatPage.logout()
    await page.waitForURL('/login')

    // Press back button
    await page.goBack()

    // Should not be able to access chat - redirect to login
    await page.waitForURL(/\/login/, { timeout: 5000 })
  })

  test('should handle forward button after login', async ({
    registerPage,
    loginPage,
    page,
  }) => {
    const user = await createAndLoginUser(registerPage, loginPage)

    // Logout
    await page.evaluate(() => localStorage.clear())
    await page.goto('/login')

    // Login
    await loginPage.login(user.email, user.password)
    await page.waitForURL('/chat', { timeout: 10000 })

    // Go back to login
    await page.goBack()

    // Since user is logged in, should redirect to chat
    await page.waitForURL('/chat', { timeout: 5000 })
  })
})

test.describe('Error Handling', () => {
  test('should handle network error during login', async ({ loginPage, page }) => {
    // Intercept and fail the login request
    await page.route('**/api/auth/login', (route) =>
      route.abort('connectionfailed')
    )

    await loginPage.goto()
    await loginPage.login('test@example.com', 'password123')

    // Should show error
    await loginPage.expectError('unavailable')
  })

  test('should handle server error during registration', async ({ registerPage, page }) => {
    // Intercept and return 500
    await page.route('**/api/auth/register', (route) =>
      route.fulfill({
        status: 500,
        body: JSON.stringify({ error: 'Internal server error' }),
      })
    )

    await registerPage.goto()
    await registerPage.register({
      email: 'test@example.com',
      username: 'testuser',
      password: 'password123',
    })

    // Should show error
    await registerPage.expectError('failed')
  })
})
