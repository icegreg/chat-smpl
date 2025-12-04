import { WebDriver, By } from 'selenium-webdriver'
import { BasePage } from './BasePage.js'

export class LoginPage extends BasePage {
  // Locators
  private readonly emailInput = By.css('input[name="email"]')
  private readonly passwordInput = By.css('input[name="password"]')
  private readonly submitButton = By.css('button[type="submit"]')
  private readonly errorMessage = By.css('.bg-red-50 p')
  private readonly registerLink = By.css('a[href="/register"]')
  private readonly pageTitle = By.css('h2')

  constructor(driver: WebDriver) {
    super(driver)
  }

  async goto(): Promise<void> {
    await this.navigate('/login')
  }

  async login(email: string, password: string): Promise<void> {
    await this.type(this.emailInput, email)
    await this.type(this.passwordInput, password)
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

  async isOnLoginPage(): Promise<boolean> {
    const title = await this.getText(this.pageTitle)
    return title.includes('Sign in')
  }

  async isEmailInputVisible(): Promise<boolean> {
    return this.isDisplayed(this.emailInput)
  }

  async isPasswordInputVisible(): Promise<boolean> {
    return this.isDisplayed(this.passwordInput)
  }

  async isSubmitButtonEnabled(): Promise<boolean> {
    return this.isEnabled(this.submitButton)
  }

  async getSubmitButtonText(): Promise<string> {
    return this.getText(this.submitButton)
  }

  async goToRegister(): Promise<void> {
    await this.click(this.registerLink)
    await this.waitForUrl('/register')
  }

  async fillEmail(email: string): Promise<void> {
    await this.type(this.emailInput, email)
  }

  async fillPassword(password: string): Promise<void> {
    await this.type(this.passwordInput, password)
  }

  async submitForm(): Promise<void> {
    await this.click(this.submitButton)
  }

  async submitWithEnter(): Promise<void> {
    await this.pressEnter(this.passwordInput)
  }

  async getEmailInputType(): Promise<string | null> {
    return this.getAttribute(this.emailInput, 'type')
  }

  async getPasswordInputType(): Promise<string | null> {
    return this.getAttribute(this.passwordInput, 'type')
  }

  async getEmailAutocomplete(): Promise<string | null> {
    return this.getAttribute(this.emailInput, 'autocomplete')
  }

  async getPasswordAutocomplete(): Promise<string | null> {
    return this.getAttribute(this.passwordInput, 'autocomplete')
  }
}
