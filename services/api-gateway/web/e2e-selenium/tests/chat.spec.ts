import { WebDriver } from 'selenium-webdriver'
import { expect } from 'chai'
import { createDriver, quitDriver } from '../config/webdriver.js'
import { ChatPage } from '../pages/ChatPage.js'
import { createTestUser, clearBrowserState } from '../helpers/testHelpers.js'

describe('Chat Creation', function () {
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

  it('should show create chat button on chat page', async function () {
    await createTestUser(driver)
    await chatPage.waitForChatPage()

    expect(await chatPage.isSidebarVisible()).to.be.true
  })

  it('should open create chat modal when clicking create button', async function () {
    await createTestUser(driver)
    await chatPage.waitForChatPage()

    await chatPage.openCreateChatModal()

    expect(await chatPage.isCreateChatModalVisible()).to.be.true
  })

  it('should close modal when clicking cancel', async function () {
    await createTestUser(driver)
    await chatPage.waitForChatPage()

    await chatPage.openCreateChatModal()
    expect(await chatPage.isCreateChatModalVisible()).to.be.true

    await chatPage.cancelCreateChat()
    await chatPage.waitForModalToClose()

    expect(await chatPage.isModalClosed()).to.be.true
  })

  it('should create a group chat successfully', async function () {
    await createTestUser(driver)
    await chatPage.waitForChatPage()

    const chatName = `Test Group ${Date.now()}`
    await chatPage.createChat(chatName, 'group')

    await chatPage.waitForModalToClose()

    // Verify chat appears in list
    await chatPage.sleep(500)
    const chatNames = await chatPage.getChatNames()
    expect(chatNames).to.include(chatName)
  })

  it('should create a channel chat successfully', async function () {
    await createTestUser(driver)
    await chatPage.waitForChatPage()

    const chatName = `Test Channel ${Date.now()}`
    await chatPage.createChat(chatName, 'channel')

    await chatPage.waitForModalToClose()

    // Verify chat appears in list
    await chatPage.sleep(500)
    const chatNames = await chatPage.getChatNames()
    expect(chatNames).to.include(chatName)
  })

  it('should create chat with description', async function () {
    await createTestUser(driver)
    await chatPage.waitForChatPage()

    const chatName = `Described Chat ${Date.now()}`
    await chatPage.createChat(chatName, 'group', 'This is a test description')

    await chatPage.waitForModalToClose()

    // Verify chat appears in list
    await chatPage.sleep(500)
    const chatNames = await chatPage.getChatNames()
    expect(chatNames).to.include(chatName)
  })

  it('should show error when creating chat without name', async function () {
    await createTestUser(driver)
    await chatPage.waitForChatPage()

    await chatPage.openCreateChatModal()
    await chatPage.submitCreateChat()

    // Modal should stay open (form validation prevents submission)
    expect(await chatPage.isCreateChatModalVisible()).to.be.true
  })

  it('should be able to create multiple chats', async function () {
    await createTestUser(driver)
    await chatPage.waitForChatPage()

    const timestamp = Date.now()
    const chatName1 = `Multi Chat 1 ${timestamp}`
    const chatName2 = `Multi Chat 2 ${timestamp}`

    // Create first chat
    await chatPage.createChat(chatName1, 'group')
    await chatPage.waitForModalToClose()
    await chatPage.sleep(500)

    // Create second chat
    await chatPage.createChat(chatName2, 'group')
    await chatPage.waitForModalToClose()
    await chatPage.sleep(500)

    // Verify both chats appear
    const chatNames = await chatPage.getChatNames()
    expect(chatNames).to.include(chatName1)
    expect(chatNames).to.include(chatName2)
  })
})

describe('Chat List', function () {
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

  it('should show empty state for new user', async function () {
    await createTestUser(driver)
    await chatPage.waitForChatPage()

    expect(await chatPage.isEmptyStateVisible()).to.be.true
  })

  it('should show chat in sidebar after creation', async function () {
    await createTestUser(driver)
    await chatPage.waitForChatPage()

    const chatName = `Sidebar Chat ${Date.now()}`
    await chatPage.createChat(chatName, 'group')
    await chatPage.waitForModalToClose()
    await chatPage.sleep(500)

    const count = await chatPage.getChatCount()
    expect(count).to.be.at.least(1)
  })
})
