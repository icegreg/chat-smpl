import { WebDriver } from 'selenium-webdriver'
import { RegisterPage, RegisterData } from '../pages/RegisterPage.js'
import { LoginPage } from '../pages/LoginPage.js'
import { BASE_URL } from '../config/webdriver.js'

// Generate unique test data
export function generateTestUser(): RegisterData {
  const timestamp = Date.now()
  const random = Math.random().toString(36).substring(2, 8)
  return {
    email: `test_${timestamp}_${random}@example.com`,
    username: `user_${timestamp}_${random}`,
    password: 'TestPassword123!',
    displayName: `Test User ${random}`,
  }
}

// Create and register a test user
export async function createTestUser(
  driver: WebDriver
): Promise<RegisterData> {
  const registerPage = new RegisterPage(driver)
  const user = generateTestUser()

  await registerPage.goto()
  await registerPage.register(user)
  await registerPage.waitForUrl('/chat', 15000)

  return user
}

// Create user and then logout
export async function createUserAndLogout(
  driver: WebDriver
): Promise<RegisterData> {
  const user = await createTestUser(driver)

  // Clear local storage to logout
  await driver.executeScript('localStorage.clear()')

  return user
}

// Login with existing user
export async function loginUser(
  driver: WebDriver,
  email: string,
  password: string
): Promise<void> {
  const loginPage = new LoginPage(driver)
  await loginPage.goto()
  await loginPage.login(email, password)
  await loginPage.waitForUrl('/chat', 15000)
}

// Clear browser state
export async function clearBrowserState(driver: WebDriver): Promise<void> {
  // Navigate to base URL first to access localStorage
  const currentUrl = await driver.getCurrentUrl()
  if (currentUrl === 'about:blank' || currentUrl.startsWith('data:')) {
    await driver.get(BASE_URL)
    await wait(500)
  }

  await driver.executeScript('localStorage.clear()')
  await driver.executeScript('sessionStorage.clear()')
  await driver.manage().deleteAllCookies()
}

// Wait helper
export function wait(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms))
}
