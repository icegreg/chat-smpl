import { WebDriver } from 'selenium-webdriver'
import { expect } from 'chai'
import { createDriver, quitDriver } from '../config/webdriver.js'
import { LoginPage } from '../pages/LoginPage.js'
import { RegisterPage } from '../pages/RegisterPage.js'
import { ChatPage } from '../pages/ChatPage.js'
import {
  generateTestUser,
  createUserAndLogout,
  clearBrowserState,
} from '../helpers/testHelpers.js'

describe('Login Page', function () {
  let driver: WebDriver
  let loginPage: LoginPage
  let registerPage: RegisterPage
  let chatPage: ChatPage

  before(async function () {
    driver = await createDriver()
    loginPage = new LoginPage(driver)
    registerPage = new RegisterPage(driver)
    chatPage = new ChatPage(driver)
  })

  after(async function () {
    await quitDriver(driver)
  })

  beforeEach(async function () {
    await clearBrowserState(driver)
    await loginPage.goto()
  })

  describe('Form Display', function () {
    it('should display login form', async function () {
      expect(await loginPage.isOnLoginPage()).to.be.true
      expect(await loginPage.isEmailInputVisible()).to.be.true
      expect(await loginPage.isPasswordInputVisible()).to.be.true
    })

    it('should have link to register page', async function () {
      await loginPage.goToRegister()
      const url = await loginPage.getCurrentUrl()
      expect(url).to.include('/register')
    })
  })

  describe('Successful Login', function () {
    it('should login successfully with valid credentials', async function () {
      // First create a user
      const user = await createUserAndLogout(driver)

      // Now login
      await loginPage.goto()
      await loginPage.login(user.email, user.password)
      await loginPage.waitForUrl('/chat', 15000)

      const url = await loginPage.getCurrentUrl()
      expect(url).to.include('/chat')
    })

    it('should show user name after login', async function () {
      const user = await createUserAndLogout(driver)

      await loginPage.goto()
      await loginPage.login(user.email, user.password)
      await chatPage.waitForChatPage()

      const displayedName = await chatPage.getUserName()
      expect(displayedName).to.include(user.displayName!)
    })

    it('should show loading state during submission', async function () {
      const user = await createUserAndLogout(driver)

      await loginPage.goto()
      await loginPage.fillEmail(user.email)
      await loginPage.fillPassword(user.password)
      await loginPage.submitForm()

      const buttonText = await loginPage.getSubmitButtonText()
      expect(buttonText.includes('Signing') || (await loginPage.getCurrentUrl()).includes('/chat')).to.be.true
    })
  })

  describe('Failed Login', function () {
    it('should show error for invalid credentials', async function () {
      await loginPage.login('invalid@example.com', 'wrongpassword')

      const hasError = await loginPage.expectError('Invalid')
      expect(hasError).to.be.true
    })

    it('should show error for non-existent user', async function () {
      await loginPage.login('nonexistent@example.com', 'password123')

      const hasError = await loginPage.expectError('Invalid')
      expect(hasError).to.be.true
    })

    it('should not submit without email', async function () {
      await loginPage.fillPassword('password123')
      await loginPage.submitForm()

      await loginPage.sleep(500)
      const url = await loginPage.getCurrentUrl()
      expect(url).to.include('/login')
    })

    it('should not submit without password', async function () {
      await loginPage.fillEmail('test@example.com')
      await loginPage.submitForm()

      await loginPage.sleep(500)
      const url = await loginPage.getCurrentUrl()
      expect(url).to.include('/login')
    })
  })

  describe('Session Management', function () {
    it('should persist session after page reload', async function () {
      const user = await createUserAndLogout(driver)

      await loginPage.goto()
      await loginPage.login(user.email, user.password)
      await chatPage.waitForChatPage()

      // Reload page
      await loginPage.refresh()
      await loginPage.sleep(1000)

      const url = await loginPage.getCurrentUrl()
      expect(url).to.include('/chat')
    })

    it('should redirect to chat if already logged in', async function () {
      const user = await createUserAndLogout(driver)

      await loginPage.goto()
      await loginPage.login(user.email, user.password)
      await chatPage.waitForChatPage()

      // Try to visit login page
      await loginPage.goto()
      await loginPage.sleep(1000)

      const url = await loginPage.getCurrentUrl()
      expect(url).to.include('/chat')
    })
  })

  describe('Security', function () {
    it('should not expose password in URL', async function () {
      await loginPage.fillEmail('test@example.com')
      await loginPage.fillPassword('secretpassword')
      await loginPage.submitForm()

      const url = await loginPage.getCurrentUrl()
      expect(url).to.not.include('secretpassword')
    })

    it('should use password input type', async function () {
      const inputType = await loginPage.getPasswordInputType()
      expect(inputType).to.equal('password')
    })

    it('should have autocomplete attributes', async function () {
      const emailAutocomplete = await loginPage.getEmailAutocomplete()
      const passwordAutocomplete = await loginPage.getPasswordAutocomplete()

      expect(emailAutocomplete).to.equal('email')
      expect(passwordAutocomplete).to.equal('current-password')
    })
  })

  describe('Accessibility', function () {
    it('should submit form with Enter key', async function () {
      const user = await createUserAndLogout(driver)

      await loginPage.goto()
      await loginPage.fillEmail(user.email)
      await loginPage.fillPassword(user.password)
      await loginPage.submitWithEnter()

      await loginPage.waitForUrl('/chat', 15000)
      const url = await loginPage.getCurrentUrl()
      expect(url).to.include('/chat')
    })
  })
})
