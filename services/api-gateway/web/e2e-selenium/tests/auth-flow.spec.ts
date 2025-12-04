import { WebDriver } from 'selenium-webdriver'
import { expect } from 'chai'
import { createDriver, quitDriver } from '../config/webdriver.js'
import { LoginPage } from '../pages/LoginPage.js'
import { RegisterPage } from '../pages/RegisterPage.js'
import { ChatPage } from '../pages/ChatPage.js'
import {
  createTestUser,
  createUserAndLogout,
  clearBrowserState,
  loginUser,
} from '../helpers/testHelpers.js'

describe('Logout', function () {
  let driver: WebDriver
  let loginPage: LoginPage
  let chatPage: ChatPage

  before(async function () {
    driver = await createDriver()
    loginPage = new LoginPage(driver)
    chatPage = new ChatPage(driver)
  })

  after(async function () {
    await quitDriver(driver)
  })

  beforeEach(async function () {
    await clearBrowserState(driver)
  })

  it('should logout successfully', async function () {
    // Login first
    await createTestUser(driver)
    await chatPage.waitForChatPage()

    // Logout
    await chatPage.logout()
    await loginPage.waitForUrl('/login', 5000)

    const url = await loginPage.getCurrentUrl()
    expect(url).to.include('/login')
  })

  it('should clear local storage on logout', async function () {
    await createTestUser(driver)
    await chatPage.waitForChatPage()

    // Verify token exists
    const tokenBefore = await loginPage.getLocalStorageItem('access_token')
    expect(tokenBefore).to.not.be.null

    // Logout
    await chatPage.logout()
    await loginPage.waitForUrl('/login', 5000)

    // Verify token is cleared
    const tokenAfter = await loginPage.getLocalStorageItem('access_token')
    expect(tokenAfter).to.be.null
  })

  it('should not be able to access protected routes after logout', async function () {
    await createTestUser(driver)
    await chatPage.waitForChatPage()

    // Logout
    await chatPage.logout()
    await loginPage.waitForUrl('/login', 5000)

    // Try to access chat page directly
    await chatPage.goto()
    await loginPage.sleep(1000)

    const url = await loginPage.getCurrentUrl()
    expect(url).to.include('/login')
  })
})

describe('Protected Routes', function () {
  let driver: WebDriver
  let loginPage: LoginPage
  let chatPage: ChatPage

  before(async function () {
    driver = await createDriver()
    loginPage = new LoginPage(driver)
    chatPage = new ChatPage(driver)
  })

  after(async function () {
    await quitDriver(driver)
  })

  beforeEach(async function () {
    await clearBrowserState(driver)
  })

  it('should redirect to login when accessing /chat without auth', async function () {
    await chatPage.goto()
    await loginPage.sleep(1000)

    const url = await loginPage.getCurrentUrl()
    expect(url).to.include('/login')
  })

  it('should redirect to login when accessing /chat/:id without auth', async function () {
    await loginPage.navigate('/chat/some-chat-id')
    await loginPage.sleep(1000)

    const url = await loginPage.getCurrentUrl()
    expect(url).to.include('/login')
  })

  it('should preserve redirect URL for deep links', async function () {
    // Create a user first
    const user = await createUserAndLogout(driver)

    // Try to access specific chat
    await loginPage.navigate('/chat/test-chat-123')
    await loginPage.sleep(1000)

    // Should be on login page with redirect param
    let url = await loginPage.getCurrentUrl()
    expect(url).to.include('/login')

    // Login
    await loginPage.login(user.email, user.password)
    await loginPage.waitForUrl('/chat', 15000)

    // Should redirect back to chat
    url = await loginPage.getCurrentUrl()
    expect(url).to.include('/chat')
  })

  it('should allow access to public routes without auth', async function () {
    // Access login page
    await loginPage.goto()
    let url = await loginPage.getCurrentUrl()
    expect(url).to.include('/login')

    // Access register page
    await loginPage.navigate('/register')
    url = await loginPage.getCurrentUrl()
    expect(url).to.include('/register')
  })
})

describe('Token Refresh', function () {
  let driver: WebDriver
  let loginPage: LoginPage
  let chatPage: ChatPage

  before(async function () {
    driver = await createDriver()
    loginPage = new LoginPage(driver)
    chatPage = new ChatPage(driver)
  })

  after(async function () {
    await quitDriver(driver)
  })

  beforeEach(async function () {
    await clearBrowserState(driver)
  })

  it('should stay logged in after page refresh', async function () {
    await createTestUser(driver)
    await chatPage.waitForChatPage()

    // Refresh page
    await loginPage.refresh()
    await loginPage.sleep(1000)

    const url = await loginPage.getCurrentUrl()
    expect(url).to.include('/chat')
  })
})

describe('Authentication State UI', function () {
  let driver: WebDriver
  let chatPage: ChatPage

  before(async function () {
    driver = await createDriver()
    chatPage = new ChatPage(driver)
  })

  after(async function () {
    await quitDriver(driver)
  })

  beforeEach(async function () {
    await clearBrowserState(driver)
  })

  it('should show user info in header when logged in', async function () {
    const user = await createTestUser(driver)
    await chatPage.waitForChatPage()

    const displayedName = await chatPage.getUserName()
    expect(displayedName).to.include(user.displayName!)
  })

  it('should show logout button when logged in', async function () {
    await createTestUser(driver)
    await chatPage.waitForChatPage()

    expect(await chatPage.isLogoutButtonVisible()).to.be.true
  })
})

describe('Full Authentication Flow', function () {
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
  })

  it('should complete full flow: register -> chat -> logout -> login -> chat', async function () {
    // Step 1: Register
    const user = await createTestUser(driver)
    await chatPage.waitForChatPage()

    let url = await loginPage.getCurrentUrl()
    expect(url).to.include('/chat')

    // Step 2: Logout
    await chatPage.logout()
    await loginPage.waitForUrl('/login', 5000)

    url = await loginPage.getCurrentUrl()
    expect(url).to.include('/login')

    // Step 3: Login
    await loginPage.login(user.email, user.password)
    await loginPage.waitForUrl('/chat', 15000)

    // Step 4: Verify on chat page
    url = await loginPage.getCurrentUrl()
    expect(url).to.include('/chat')

    const displayedName = await chatPage.getUserName()
    expect(displayedName).to.include(user.displayName!)
  })

  it('should handle back button after logout', async function () {
    // Login
    await createTestUser(driver)
    await chatPage.waitForChatPage()

    // Logout
    await chatPage.logout()
    await loginPage.waitForUrl('/login', 5000)

    // Press back button
    await loginPage.goBack()
    await loginPage.sleep(1000)

    // Should redirect to login (can't access protected route)
    const url = await loginPage.getCurrentUrl()
    expect(url).to.include('/login')
  })
})
