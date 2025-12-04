import { WebDriver } from 'selenium-webdriver'
import { expect } from 'chai'
import { createDriver, quitDriver } from '../config/webdriver.js'
import { RegisterPage } from '../pages/RegisterPage.js'
import { ChatPage } from '../pages/ChatPage.js'
import { generateTestUser, clearBrowserState } from '../helpers/testHelpers.js'

describe('Registration Page', function () {
  let driver: WebDriver
  let registerPage: RegisterPage
  let chatPage: ChatPage

  before(async function () {
    driver = await createDriver()
    registerPage = new RegisterPage(driver)
    chatPage = new ChatPage(driver)
  })

  after(async function () {
    await quitDriver(driver)
  })

  beforeEach(async function () {
    await clearBrowserState(driver)
    await registerPage.goto()
  })

  describe('Form Display', function () {
    it('should display registration form', async function () {
      expect(await registerPage.isOnRegisterPage()).to.be.true
      expect(await registerPage.isEmailInputVisible()).to.be.true
      expect(await registerPage.isUsernameInputVisible()).to.be.true
      expect(await registerPage.isPasswordInputVisible()).to.be.true
      expect(await registerPage.isConfirmPasswordInputVisible()).to.be.true
      expect(await registerPage.isSubmitButtonVisible()).to.be.true
    })

    it('should have link to login page', async function () {
      await registerPage.goToLogin()
      const url = await registerPage.getCurrentUrl()
      expect(url).to.include('/login')
    })
  })

  describe('Successful Registration', function () {
    it('should register successfully with valid data', async function () {
      const user = generateTestUser()

      await registerPage.register(user)
      await registerPage.waitForUrl('/chat', 15000)

      const url = await registerPage.getCurrentUrl()
      expect(url).to.include('/chat')
    })

    it('should register with display name and show it', async function () {
      const user = generateTestUser()

      await registerPage.register(user)
      await chatPage.waitForChatPage()

      const displayedName = await chatPage.getUserName()
      expect(displayedName).to.include(user.displayName!)
    })

    it('should show loading state during submission', async function () {
      const user = generateTestUser()

      await registerPage.fillEmail(user.email)
      await registerPage.fillUsername(user.username)
      await registerPage.fillPassword(user.password)
      await registerPage.fillConfirmPassword(user.password)

      await registerPage.submitForm()

      // Button text should change to loading state
      const buttonText = await registerPage.getSubmitButtonText()
      // Either shows loading or we've already redirected
      expect(buttonText.includes('Creating') || (await registerPage.getCurrentUrl()).includes('/chat')).to.be.true
    })
  })

  describe('Validation Errors', function () {
    it('should show error for mismatched passwords', async function () {
      const user = generateTestUser()

      await registerPage.register({
        ...user,
        confirmPassword: 'DifferentPassword123!',
      })

      const hasError = await registerPage.expectError('Passwords do not match')
      expect(hasError).to.be.true
    })

    it('should show error for short password', async function () {
      const user = generateTestUser()

      await registerPage.register({
        ...user,
        password: 'short',
        confirmPassword: 'short',
      })

      const hasError = await registerPage.expectError('at least 8 characters')
      expect(hasError).to.be.true
    })

    it('should not submit without email', async function () {
      const user = generateTestUser()

      await registerPage.fillUsername(user.username)
      await registerPage.fillPassword(user.password)
      await registerPage.fillConfirmPassword(user.password)
      await registerPage.submitForm()

      // Should stay on register page (HTML5 validation)
      await registerPage.sleep(500)
      const url = await registerPage.getCurrentUrl()
      expect(url).to.include('/register')
    })

    it('should not submit without username', async function () {
      const user = generateTestUser()

      await registerPage.fillEmail(user.email)
      await registerPage.fillPassword(user.password)
      await registerPage.fillConfirmPassword(user.password)
      await registerPage.submitForm()

      // Should stay on register page
      await registerPage.sleep(500)
      const url = await registerPage.getCurrentUrl()
      expect(url).to.include('/register')
    })
  })

  describe('Duplicate User Handling', function () {
    it('should show error for duplicate email', async function () {
      const user = generateTestUser()

      // Register first user
      await registerPage.register(user)
      await registerPage.waitForUrl('/chat', 15000)

      // Clear storage and go back to register
      await clearBrowserState(driver)
      await registerPage.goto()

      // Try to register with same email
      await registerPage.register({
        ...user,
        username: user.username + '_new',
      })

      const hasError = await registerPage.hasError()
      expect(hasError).to.be.true
    })
  })

  describe('Accessibility', function () {
    it('should submit form with Enter key', async function () {
      const user = generateTestUser()

      await registerPage.fillEmail(user.email)
      await registerPage.fillUsername(user.username)
      await registerPage.fillPassword(user.password)
      await registerPage.fillConfirmPassword(user.password)

      await registerPage.submitWithEnter()
      await registerPage.waitForUrl('/chat', 15000)

      const url = await registerPage.getCurrentUrl()
      expect(url).to.include('/chat')
    })
  })
})
