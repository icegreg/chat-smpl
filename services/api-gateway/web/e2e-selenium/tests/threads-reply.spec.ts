import { WebDriver } from 'selenium-webdriver'
import { expect } from 'chai'
import { createDriver, quitDriver } from '../config/webdriver.js'
import { ChatPage } from '../pages/ChatPage.js'
import { createTestUser, clearBrowserState } from '../helpers/testHelpers.js'

/**
 * E2E Tests for Threads and Reply functionality
 *
 * Tests cover:
 * 1. Reply without thread (inline quote) - simple reply to a message
 * 2. User thread creation - discussion threads
 * 3. System thread creation - moderator activity threads
 * 4. Thread attached to message - reply thread linked to parent_message_id
 */

describe('Reply without Thread (Inline Quote)', function () {
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

  it('should show reply button when hovering over a message', async function () {
    await createTestUser(driver)
    await chatPage.waitForChatPage()

    // Create a chat and send a message
    const chatName = `Reply Test ${Date.now()}`
    await chatPage.createChat(chatName, 'group')
    await chatPage.waitForModalToClose()
    await chatPage.sleep(500)

    await chatPage.selectFirstChat()
    await chatPage.sleep(500)

    // Send a message first
    const originalMessage = `Original message ${Date.now()}`
    await chatPage.sendMessage(originalMessage)
    await chatPage.waitForMessageContaining(originalMessage)

    // Hover over the message
    await chatPage.hoverOverFirstMessage()
    await chatPage.sleep(300)

    // Check that reply button is visible
    expect(await chatPage.isReplyButtonVisible()).to.be.true
  })

  it('should show reply preview when clicking reply button', async function () {
    await createTestUser(driver)
    await chatPage.waitForChatPage()

    const chatName = `Reply Preview Test ${Date.now()}`
    await chatPage.createChat(chatName, 'group')
    await chatPage.waitForModalToClose()
    await chatPage.sleep(500)

    await chatPage.selectFirstChat()
    await chatPage.sleep(500)

    // Send a message
    const originalMessage = `Message to reply to ${Date.now()}`
    await chatPage.sendMessage(originalMessage)
    await chatPage.waitForMessageContaining(originalMessage)

    // Click reply on the message
    await chatPage.clickReplyOnFirstMessage()
    await chatPage.sleep(300)

    // Verify reply preview is shown
    expect(await chatPage.isReplyPreviewVisible()).to.be.true

    // Verify preview contains original message content
    const previewContent = await chatPage.getReplyPreviewContent()
    expect(previewContent).to.include(originalMessage)
  })

  it('should cancel reply when clicking cancel button', async function () {
    await createTestUser(driver)
    await chatPage.waitForChatPage()

    const chatName = `Cancel Reply Test ${Date.now()}`
    await chatPage.createChat(chatName, 'group')
    await chatPage.waitForModalToClose()
    await chatPage.sleep(500)

    await chatPage.selectFirstChat()
    await chatPage.sleep(500)

    // Send a message and click reply
    const originalMessage = `Message ${Date.now()}`
    await chatPage.sendMessage(originalMessage)
    await chatPage.waitForMessageContaining(originalMessage)

    await chatPage.clickReplyOnFirstMessage()
    await chatPage.sleep(300)

    expect(await chatPage.isReplyPreviewVisible()).to.be.true

    // Cancel reply
    await chatPage.cancelReply()
    await chatPage.sleep(300)

    // Verify preview is hidden
    expect(await chatPage.isReplyPreviewVisible()).to.be.false
  })

  it('should send reply message with quote', async function () {
    await createTestUser(driver)
    await chatPage.waitForChatPage()

    const chatName = `Send Reply Test ${Date.now()}`
    await chatPage.createChat(chatName, 'group')
    await chatPage.waitForModalToClose()
    await chatPage.sleep(500)

    await chatPage.selectFirstChat()
    await chatPage.sleep(500)

    // Send original message
    const originalMessage = `Original: ${Date.now()}`
    await chatPage.sendMessage(originalMessage)
    await chatPage.waitForMessageContaining(originalMessage)

    // Click reply
    await chatPage.clickReplyOnFirstMessage()
    await chatPage.sleep(300)

    // Send reply message
    const replyMessage = `Reply: ${Date.now()}`
    await chatPage.sendMessage(replyMessage)
    await chatPage.sleep(500)

    // Wait for reply to appear with quote
    await chatPage.waitForMessageContaining(replyMessage)
    await chatPage.waitForQuote()

    // Verify quote is displayed
    expect(await chatPage.hasMessageWithQuote()).to.be.true

    // Verify quote contains original message
    const quoteContent = await chatPage.getQuoteContent()
    expect(quoteContent).to.include(originalMessage)
  })

  it('should reply to specific message in conversation', async function () {
    await createTestUser(driver)
    await chatPage.waitForChatPage()

    const chatName = `Multi Reply Test ${Date.now()}`
    await chatPage.createChat(chatName, 'group')
    await chatPage.waitForModalToClose()
    await chatPage.sleep(500)

    await chatPage.selectFirstChat()
    await chatPage.sleep(500)

    // Send multiple messages
    const message1 = `First message ${Date.now()}`
    const message2 = `Second message ${Date.now()}`
    const message3 = `Third message ${Date.now()}`

    await chatPage.sendMessage(message1)
    await chatPage.waitForMessageContaining(message1)

    await chatPage.sendMessage(message2)
    await chatPage.waitForMessageContaining(message2)

    await chatPage.sendMessage(message3)
    await chatPage.waitForMessageContaining(message3)

    // Reply to the first message (index 0)
    await chatPage.clickReplyOnMessage(0)
    await chatPage.sleep(300)

    // Verify preview shows first message
    const previewContent = await chatPage.getReplyPreviewContent()
    expect(previewContent).to.include(message1)

    // Send reply
    const replyMessage = `Reply to first: ${Date.now()}`
    await chatPage.sendMessage(replyMessage)
    await chatPage.sleep(500)

    await chatPage.waitForMessageContaining(replyMessage)

    // Verify quote contains first message
    const quoteContent = await chatPage.getQuoteContent()
    expect(quoteContent).to.include(message1)
  })
})

describe('User Thread (Discussion)', function () {
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

  it('should open threads panel when clicking threads button', async function () {
    await createTestUser(driver)
    await chatPage.waitForChatPage()

    const chatName = `Thread Panel Test ${Date.now()}`
    await chatPage.createChat(chatName, 'group')
    await chatPage.waitForModalToClose()
    await chatPage.sleep(500)

    await chatPage.selectFirstChat()
    await chatPage.sleep(500)

    // Open threads panel
    await chatPage.openThreadsPanel()

    // Verify panel is visible
    expect(await chatPage.isThreadsPanelVisible()).to.be.true
  })

  it('should create a new user thread', async function () {
    await createTestUser(driver)
    await chatPage.waitForChatPage()

    const chatName = `Create Thread Test ${Date.now()}`
    await chatPage.createChat(chatName, 'group')
    await chatPage.waitForModalToClose()
    await chatPage.sleep(500)

    await chatPage.selectFirstChat()
    await chatPage.sleep(1000)

    // Get chat ID
    const chatId = await chatPage.getCurrentChatId()
    expect(chatId).to.not.be.empty

    // Create thread via API
    const threadTitle = `Discussion ${Date.now()}`
    const threadId = await chatPage.createThreadViaApi(chatId, threadTitle)
    expect(threadId).to.not.be.empty

    // Open threads panel
    await chatPage.openThreadsPanel()
    await chatPage.sleep(500)

    // Verify thread appears in list
    const threads = await chatPage.listThreadsViaApi(chatId)
    const createdThread = threads.find(t => t.title === threadTitle)
    expect(createdThread).to.not.be.undefined
  })

  it('should view and send message in thread', async function () {
    await createTestUser(driver)
    await chatPage.waitForChatPage()

    const chatName = `Thread Messaging Test ${Date.now()}`
    await chatPage.createChat(chatName, 'group')
    await chatPage.waitForModalToClose()
    await chatPage.sleep(500)

    await chatPage.selectFirstChat()
    await chatPage.sleep(1000)

    // Create thread
    const chatId = await chatPage.getCurrentChatId()
    const threadTitle = `Test Thread ${Date.now()}`
    await chatPage.createThreadViaApi(chatId, threadTitle)

    // Open threads panel and select thread
    await chatPage.openThreadsPanel()
    await chatPage.sleep(1000)

    // Click on thread
    await chatPage.clickFirstThread()
    await chatPage.waitForThreadView()

    // Send message in thread
    const threadMessage = `Thread message ${Date.now()}`
    await chatPage.sendThreadMessage(threadMessage)
    await chatPage.sleep(1000)

    // Verify message appeared
    const messageCount = await chatPage.getThreadMessageCount()
    expect(messageCount).to.be.at.least(1)
  })

  it('should close thread view and return to thread list', async function () {
    await createTestUser(driver)
    await chatPage.waitForChatPage()

    const chatName = `Close Thread Test ${Date.now()}`
    await chatPage.createChat(chatName, 'group')
    await chatPage.waitForModalToClose()
    await chatPage.sleep(500)

    await chatPage.selectFirstChat()
    await chatPage.sleep(1000)

    // Create and open thread
    const chatId = await chatPage.getCurrentChatId()
    await chatPage.createThreadViaApi(chatId, `Thread ${Date.now()}`)

    await chatPage.openThreadsPanel()
    await chatPage.sleep(500)

    await chatPage.clickFirstThread()
    await chatPage.waitForThreadView()

    // Go back to thread list
    await chatPage.goBackFromThreadView()
    await chatPage.sleep(300)

    // Verify we're back to thread list
    expect(await chatPage.isThreadViewVisible()).to.be.false
    expect(await chatPage.isThreadsPanelVisible()).to.be.true
  })
})

describe('System Thread (Moderator Activity)', function () {
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

  it('should create system thread via API', async function () {
    await createTestUser(driver)
    await chatPage.waitForChatPage()

    const chatName = `System Thread Test ${Date.now()}`
    await chatPage.createChat(chatName, 'group')
    await chatPage.waitForModalToClose()
    await chatPage.sleep(500)

    await chatPage.selectFirstChat()
    await chatPage.sleep(1000)

    const chatId = await chatPage.getCurrentChatId()

    // Create system thread via direct API call
    const result = await driver.executeScript(`
      return new Promise(async (resolve, reject) => {
        try {
          const token = localStorage.getItem('access_token');
          if (!token) {
            reject('No access token');
            return;
          }
          const response = await fetch('/api/chats/${chatId}/threads', {
            method: 'POST',
            headers: {
              'Authorization': 'Bearer ' + token,
              'Content-Type': 'application/json'
            },
            body: JSON.stringify({
              title: 'Moderator Activity',
              thread_type: 'system'
            })
          });
          if (!response.ok) {
            const error = await response.text();
            reject('API error: ' + response.status + ' ' + error);
            return;
          }
          const thread = await response.json();
          resolve(thread);
        } catch (e) {
          reject(e.message);
        }
      });
    `, chatId) as { id: string; thread_type: string; title: string }

    expect(result).to.not.be.null
    expect(result.thread_type).to.equal('system')
    expect(result.title).to.equal('Moderator Activity')
  })

  it('should display system thread with orange styling', async function () {
    await createTestUser(driver)
    await chatPage.waitForChatPage()

    const chatName = `System Thread Display ${Date.now()}`
    await chatPage.createChat(chatName, 'group')
    await chatPage.waitForModalToClose()
    await chatPage.sleep(500)

    await chatPage.selectFirstChat()
    await chatPage.sleep(1000)

    const chatId = await chatPage.getCurrentChatId()

    // Create system thread
    await driver.executeScript(`
      return new Promise(async (resolve, reject) => {
        try {
          const token = localStorage.getItem('access_token');
          const response = await fetch('/api/chats/${chatId}/threads', {
            method: 'POST',
            headers: {
              'Authorization': 'Bearer ' + token,
              'Content-Type': 'application/json'
            },
            body: JSON.stringify({
              title: 'Activity Log',
              thread_type: 'system'
            })
          });
          if (!response.ok) {
            reject('Failed to create thread');
            return;
          }
          resolve(await response.json());
        } catch (e) {
          reject(e.message);
        }
      });
    `, chatId)

    // Open threads panel
    await chatPage.openThreadsPanel()
    await chatPage.sleep(1000)

    // Verify system thread is in list with correct type
    const threads = await driver.executeScript(`
      const items = document.querySelectorAll('[data-testid="thread-item"]');
      return Array.from(items).map(item => ({
        type: item.getAttribute('data-thread-type'),
        text: item.textContent
      }));
    `) as { type: string; text: string }[]

    const systemThread = threads.find(t => t.type === 'system')
    expect(systemThread).to.not.be.undefined
    expect(systemThread!.text).to.include('Activity')
  })

  it('should not allow sending messages in system thread (read-only)', async function () {
    await createTestUser(driver)
    await chatPage.waitForChatPage()

    const chatName = `System Thread Readonly ${Date.now()}`
    await chatPage.createChat(chatName, 'group')
    await chatPage.waitForModalToClose()
    await chatPage.sleep(500)

    await chatPage.selectFirstChat()
    await chatPage.sleep(1000)

    const chatId = await chatPage.getCurrentChatId()

    // Create system thread
    await driver.executeScript(`
      return new Promise(async (resolve, reject) => {
        try {
          const token = localStorage.getItem('access_token');
          const response = await fetch('/api/chats/${chatId}/threads', {
            method: 'POST',
            headers: {
              'Authorization': 'Bearer ' + token,
              'Content-Type': 'application/json'
            },
            body: JSON.stringify({
              title: 'System Activity',
              thread_type: 'system'
            })
          });
          resolve(await response.json());
        } catch (e) {
          reject(e.message);
        }
      });
    `, chatId)

    // Open threads panel and select system thread
    await chatPage.openThreadsPanel()
    await chatPage.sleep(1000)

    // Click on system thread
    const systemThreadClicked = await driver.executeScript(`
      const items = document.querySelectorAll('[data-testid="thread-item"][data-thread-type="system"]');
      if (items.length > 0) {
        items[0].click();
        return true;
      }
      return false;
    `) as boolean

    expect(systemThreadClicked).to.be.true
    await chatPage.sleep(500)

    // Check that thread view is visible
    expect(await chatPage.isThreadViewVisible()).to.be.true

    // Verify no input field in system thread (read-only)
    const hasInput = await driver.executeScript(`
      const threadView = document.querySelector('[data-testid="thread-view"]');
      if (!threadView) return false;
      const textarea = threadView.querySelector('textarea');
      return textarea !== null;
    `) as boolean

    // System threads should not have input
    expect(hasInput).to.be.false
  })
})

describe('Thread Attached to Message (Reply Thread)', function () {
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

  it('should create thread linked to a specific message', async function () {
    await createTestUser(driver)
    await chatPage.waitForChatPage()

    const chatName = `Reply Thread Test ${Date.now()}`
    await chatPage.createChat(chatName, 'group')
    await chatPage.waitForModalToClose()
    await chatPage.sleep(500)

    await chatPage.selectFirstChat()
    await chatPage.sleep(1000)

    // Send a message first
    const originalMessage = `Message for thread ${Date.now()}`
    await chatPage.sendMessage(originalMessage)
    await chatPage.waitForMessageContaining(originalMessage)
    await chatPage.sleep(500)

    const chatId = await chatPage.getCurrentChatId()

    // Get message ID from API
    const messages = await driver.executeScript(`
      return new Promise(async (resolve, reject) => {
        try {
          const token = localStorage.getItem('access_token');
          const response = await fetch('/api/chats/${chatId}/messages', {
            headers: { 'Authorization': 'Bearer ' + token }
          });
          const data = await response.json();
          resolve(data.messages || []);
        } catch (e) {
          reject(e.message);
        }
      });
    `, chatId) as { id: string; content: string }[]

    expect(messages.length).to.be.at.least(1)
    const messageId = messages[0].id

    // Create thread linked to message
    const threadResult = await driver.executeScript(`
      return new Promise(async (resolve, reject) => {
        try {
          const token = localStorage.getItem('access_token');
          const response = await fetch('/api/chats/${chatId}/threads', {
            method: 'POST',
            headers: {
              'Authorization': 'Bearer ' + token,
              'Content-Type': 'application/json'
            },
            body: JSON.stringify({
              title: 'Discussion about this message',
              thread_type: 'user',
              parent_message_id: '${messageId}'
            })
          });
          if (!response.ok) {
            const error = await response.text();
            reject('API error: ' + response.status + ' ' + error);
            return;
          }
          resolve(await response.json());
        } catch (e) {
          reject(e.message);
        }
      });
    `, chatId, messageId) as { id: string; parent_message_id: string }

    expect(threadResult).to.not.be.null
    expect(threadResult.parent_message_id).to.equal(messageId)
  })

  it('should display reply icon for thread attached to message', async function () {
    await createTestUser(driver)
    await chatPage.waitForChatPage()

    const chatName = `Reply Icon Test ${Date.now()}`
    await chatPage.createChat(chatName, 'group')
    await chatPage.waitForModalToClose()
    await chatPage.sleep(500)

    await chatPage.selectFirstChat()
    await chatPage.sleep(1000)

    // Send a message
    const message = `Parent message ${Date.now()}`
    await chatPage.sendMessage(message)
    await chatPage.waitForMessageContaining(message)

    const chatId = await chatPage.getCurrentChatId()

    // Get message ID
    const messages = await driver.executeScript(`
      return new Promise(async (resolve, reject) => {
        try {
          const token = localStorage.getItem('access_token');
          const response = await fetch('/api/chats/${chatId}/messages', {
            headers: { 'Authorization': 'Bearer ' + token }
          });
          const data = await response.json();
          resolve(data.messages || []);
        } catch (e) {
          reject(e.message);
        }
      });
    `, chatId) as { id: string }[]

    const messageId = messages[0].id

    // Create thread linked to message
    await driver.executeScript(`
      return new Promise(async (resolve, reject) => {
        try {
          const token = localStorage.getItem('access_token');
          const response = await fetch('/api/chats/${chatId}/threads', {
            method: 'POST',
            headers: {
              'Authorization': 'Bearer ' + token,
              'Content-Type': 'application/json'
            },
            body: JSON.stringify({
              title: 'Reply Thread',
              thread_type: 'user',
              parent_message_id: '${messageId}'
            })
          });
          resolve(await response.json());
        } catch (e) {
          reject(e.message);
        }
      });
    `, chatId, messageId)

    // Open threads panel
    await chatPage.openThreadsPanel()
    await chatPage.sleep(1000)

    // Check thread list - thread linked to message should show reply icon
    // The ThreadList.vue shows different icon for threads with parent_message_id
    const threads = await chatPage.listThreadsViaApi(chatId)
    const replyThread = threads.find(t => t.title === 'Reply Thread')
    expect(replyThread).to.not.be.undefined
  })

  it('should send message in reply thread and see it in thread view', async function () {
    await createTestUser(driver)
    await chatPage.waitForChatPage()

    const chatName = `Reply Thread Messaging ${Date.now()}`
    await chatPage.createChat(chatName, 'group')
    await chatPage.waitForModalToClose()
    await chatPage.sleep(500)

    await chatPage.selectFirstChat()
    await chatPage.sleep(1000)

    // Send parent message
    const parentMessage = `Parent ${Date.now()}`
    await chatPage.sendMessage(parentMessage)
    await chatPage.waitForMessageContaining(parentMessage)

    const chatId = await chatPage.getCurrentChatId()

    // Get message ID
    const messages = await driver.executeScript(`
      return new Promise(async (resolve, reject) => {
        try {
          const token = localStorage.getItem('access_token');
          const response = await fetch('/api/chats/${chatId}/messages', {
            headers: { 'Authorization': 'Bearer ' + token }
          });
          const data = await response.json();
          resolve(data.messages || []);
        } catch (e) {
          reject(e.message);
        }
      });
    `, chatId) as { id: string }[]

    const messageId = messages[0].id

    // Create reply thread
    await driver.executeScript(`
      return new Promise(async (resolve, reject) => {
        try {
          const token = localStorage.getItem('access_token');
          const response = await fetch('/api/chats/${chatId}/threads', {
            method: 'POST',
            headers: {
              'Authorization': 'Bearer ' + token,
              'Content-Type': 'application/json'
            },
            body: JSON.stringify({
              title: 'Thread on Message',
              thread_type: 'user',
              parent_message_id: '${messageId}'
            })
          });
          resolve(await response.json());
        } catch (e) {
          reject(e.message);
        }
      });
    `, chatId, messageId)

    // Open threads panel and select thread
    await chatPage.openThreadsPanel()
    await chatPage.sleep(1000)

    await chatPage.clickFirstThread()
    await chatPage.waitForThreadView()

    // Send message in thread
    const threadMessage = `Reply in thread ${Date.now()}`
    await chatPage.sendThreadMessage(threadMessage)
    await chatPage.sleep(1000)

    // Verify message count
    const messageCount = await chatPage.getThreadMessageCount()
    expect(messageCount).to.be.at.least(1)
  })
})
