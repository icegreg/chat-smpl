import { test as base, expect } from '@playwright/test'
import { LoginPage } from '../pages/LoginPage'
import { RegisterPage } from '../pages/RegisterPage'
import { ChatPage } from '../pages/ChatPage'

// Generate unique test data
export function generateTestUser() {
  const timestamp = Date.now()
  const random = Math.random().toString(36).substring(2, 8)
  return {
    email: `test_${timestamp}_${random}@example.com`,
    username: `user_${timestamp}_${random}`,
    password: 'TestPassword123!',
    displayName: `Test User ${random}`,
  }
}

// Extend Playwright test with page objects
export const test = base.extend<{
  loginPage: LoginPage
  registerPage: RegisterPage
  chatPage: ChatPage
}>({
  loginPage: async ({ page }, use) => {
    const loginPage = new LoginPage(page)
    await use(loginPage)
  },

  registerPage: async ({ page }, use) => {
    const registerPage = new RegisterPage(page)
    await use(registerPage)
  },

  chatPage: async ({ page }, use) => {
    const chatPage = new ChatPage(page)
    await use(chatPage)
  },
})

export { expect }

// Helper to create and login a test user
export async function createAndLoginUser(
  registerPage: RegisterPage,
  loginPage: LoginPage
): Promise<{ email: string; username: string; password: string; displayName: string }> {
  const user = generateTestUser()

  // Register
  await registerPage.goto()
  await registerPage.register(user)

  // Wait for redirect to chat
  await registerPage.page.waitForURL('/chat')

  return user
}

// Helper to mock API responses (for isolated tests)
export async function mockAuthAPI(page: any) {
  // Mock successful login
  await page.route('**/api/auth/login', async (route: any) => {
    const request = route.request()
    const postData = JSON.parse(request.postData() || '{}')

    if (postData.email === 'test@example.com' && postData.password === 'password123') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          access_token: 'mock_access_token',
          refresh_token: 'mock_refresh_token',
        }),
      })
    } else {
      await route.fulfill({
        status: 401,
        contentType: 'application/json',
        body: JSON.stringify({ error: 'Invalid credentials' }),
      })
    }
  })

  // Mock user info
  await page.route('**/api/auth/me', async (route: any) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        id: 'user-123',
        email: 'test@example.com',
        username: 'testuser',
        display_name: 'Test User',
        role: 'user',
      }),
    })
  })
}
