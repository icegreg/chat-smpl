import { Page, Locator, expect } from '@playwright/test'

export class LoginPage {
  readonly page: Page
  readonly emailInput: Locator
  readonly passwordInput: Locator
  readonly submitButton: Locator
  readonly errorMessage: Locator
  readonly registerLink: Locator
  readonly pageTitle: Locator

  constructor(page: Page) {
    this.page = page
    this.emailInput = page.locator('input[name="email"]')
    this.passwordInput = page.locator('input[name="password"]')
    this.submitButton = page.locator('button[type="submit"]')
    this.errorMessage = page.locator('.bg-red-50 p')
    this.registerLink = page.locator('a[href="/register"]')
    this.pageTitle = page.locator('h2')
  }

  async goto() {
    await this.page.goto('/login')
  }

  async login(email: string, password: string) {
    await this.emailInput.fill(email)
    await this.passwordInput.fill(password)
    await this.submitButton.click()
  }

  async expectError(message: string) {
    await expect(this.errorMessage).toContainText(message)
  }

  async expectOnLoginPage() {
    await expect(this.pageTitle).toContainText('Sign in')
    await expect(this.emailInput).toBeVisible()
    await expect(this.passwordInput).toBeVisible()
  }

  async expectSubmitDisabled() {
    await expect(this.submitButton).toBeDisabled()
  }

  async expectSubmitEnabled() {
    await expect(this.submitButton).toBeEnabled()
  }

  async goToRegister() {
    await this.registerLink.click()
    await this.page.waitForURL('/register')
  }
}
