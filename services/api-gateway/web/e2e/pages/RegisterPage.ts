import { Page, Locator, expect } from '@playwright/test'

export class RegisterPage {
  readonly page: Page
  readonly emailInput: Locator
  readonly usernameInput: Locator
  readonly displayNameInput: Locator
  readonly passwordInput: Locator
  readonly confirmPasswordInput: Locator
  readonly submitButton: Locator
  readonly errorMessage: Locator
  readonly loginLink: Locator
  readonly pageTitle: Locator

  constructor(page: Page) {
    this.page = page
    this.emailInput = page.locator('input[name="email"]')
    this.usernameInput = page.locator('input[name="username"]')
    this.displayNameInput = page.locator('input[name="displayName"]')
    this.passwordInput = page.locator('input[name="password"]')
    this.confirmPasswordInput = page.locator('input[name="confirmPassword"]')
    this.submitButton = page.locator('button[type="submit"]')
    this.errorMessage = page.locator('.bg-red-50 p')
    this.loginLink = page.locator('a[href="/login"]')
    this.pageTitle = page.locator('h2')
  }

  async goto() {
    await this.page.goto('/register')
  }

  async register(data: {
    email: string
    username: string
    password: string
    confirmPassword?: string
    displayName?: string
  }) {
    await this.emailInput.fill(data.email)
    await this.usernameInput.fill(data.username)

    if (data.displayName) {
      await this.displayNameInput.fill(data.displayName)
    }

    await this.passwordInput.fill(data.password)
    await this.confirmPasswordInput.fill(data.confirmPassword || data.password)
    await this.submitButton.click()
  }

  async expectError(message: string) {
    await expect(this.errorMessage).toContainText(message)
  }

  async expectOnRegisterPage() {
    await expect(this.pageTitle).toContainText('Create your account')
    await expect(this.emailInput).toBeVisible()
    await expect(this.usernameInput).toBeVisible()
  }

  async goToLogin() {
    await this.loginLink.click()
    await this.page.waitForURL('/login')
  }
}
