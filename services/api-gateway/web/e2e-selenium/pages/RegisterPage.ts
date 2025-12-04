import { WebDriver, By } from 'selenium-webdriver'
import { BasePage } from './BasePage.js'

export interface RegisterData {
  email: string
  username: string
  password: string
  confirmPassword?: string
  displayName?: string
}

export class RegisterPage extends BasePage {
  // Locators
  private readonly emailInput = By.css('input[name="email"]')
  private readonly usernameInput = By.css('input[name="username"]')
  private readonly displayNameInput = By.css('input[name="displayName"]')
  private readonly passwordInput = By.css('input[name="password"]')
  private readonly confirmPasswordInput = By.css('input[name="confirmPassword"]')
  private readonly submitButton = By.css('button[type="submit"]')
  private readonly errorMessage = By.css('.bg-red-50 p')
  private readonly loginLink = By.css('a[href="/login"]')
  private readonly pageTitle = By.css('h2')

  constructor(driver: WebDriver) {
    super(driver)
  }

  async goto(): Promise<void> {
    await this.navigate('/register')
  }

  async register(data: RegisterData): Promise<void> {
    await this.type(this.emailInput, data.email)
    await this.type(this.usernameInput, data.username)

    if (data.displayName) {
      await this.type(this.displayNameInput, data.displayName)
    }

    await this.type(this.passwordInput, data.password)
    await this.type(this.confirmPasswordInput, data.confirmPassword || data.password)
    await this.click(this.submitButton)
  }

  async getErrorMessage(): Promise<string> {
    return this.getText(this.errorMessage)
  }

  async hasError(): Promise<boolean> {
    return this.isDisplayed(this.errorMessage)
  }

  async expectError(message: string): Promise<boolean> {
    await this.waitForTextInElement(this.errorMessage, message)
    const errorText = await this.getErrorMessage()
    return errorText.includes(message)
  }

  async isOnRegisterPage(): Promise<boolean> {
    const title = await this.getText(this.pageTitle)
    return title.includes('Create your account')
  }

  async isEmailInputVisible(): Promise<boolean> {
    return this.isDisplayed(this.emailInput)
  }

  async isUsernameInputVisible(): Promise<boolean> {
    return this.isDisplayed(this.usernameInput)
  }

  async isPasswordInputVisible(): Promise<boolean> {
    return this.isDisplayed(this.passwordInput)
  }

  async isConfirmPasswordInputVisible(): Promise<boolean> {
    return this.isDisplayed(this.confirmPasswordInput)
  }

  async isSubmitButtonVisible(): Promise<boolean> {
    return this.isDisplayed(this.submitButton)
  }

  async getSubmitButtonText(): Promise<string> {
    return this.getText(this.submitButton)
  }

  async goToLogin(): Promise<void> {
    await this.click(this.loginLink)
    await this.waitForUrl('/login')
  }

  async fillEmail(email: string): Promise<void> {
    await this.type(this.emailInput, email)
  }

  async fillUsername(username: string): Promise<void> {
    await this.type(this.usernameInput, username)
  }

  async fillPassword(password: string): Promise<void> {
    await this.type(this.passwordInput, password)
  }

  async fillConfirmPassword(password: string): Promise<void> {
    await this.type(this.confirmPasswordInput, password)
  }

  async submitForm(): Promise<void> {
    await this.click(this.submitButton)
  }

  async submitWithEnter(): Promise<void> {
    await this.pressEnter(this.confirmPasswordInput)
  }
}
