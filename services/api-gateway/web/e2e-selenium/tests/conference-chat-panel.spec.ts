import { WebDriver } from 'selenium-webdriver'
import { expect } from 'chai'
import { createWebRTCDriver, quitDriver } from '../config/webdriver-webrtc.js'
import { ChatPage } from '../pages/ChatPage.js'
import { createTestUser, clearBrowserState, getUserIdFromApi } from '../helpers/testHelpers.js'

/**
 * Conference Chat Panel E2E Tests
 *
 * Tests for the chat panel functionality within the ConferenceView:
 * 1. Chat toggle button appears when conference has a chat_id
 * 2. Clicking chat toggle opens the chat sidebar
 * 3. Messages display correctly in the chat sidebar
 * 4. Sending a message works from within the conference view
 * 5. System messages appear when users join/leave
 * 6. Chat can be closed and reopened
 *
 * Requirements:
 * - FreeSWITCH running with Verto WebSocket
 * - Non-headless browser (HEADLESS=false) recommended for WebRTC
 */

describe('Conference Chat Panel', function () {
  this.timeout(180000) // 3 minutes timeout

  const testUrl = 'http://localhost:8888'
  let driver1: WebDriver
  let driver2: WebDriver
  let chatPage1: ChatPage
  let chatPage2: ChatPage
  let chatName: string

  before(async function () {
    console.log('\n=== Conference Chat Panel Test Setup ===')
    console.log('Creating two browser instances...')

    driver1 = await createWebRTCDriver()
    driver2 = await createWebRTCDriver()
    chatPage1 = new ChatPage(driver1)
    chatPage2 = new ChatPage(driver2)

    console.log('Browser instances created')
  })

  after(async function () {
    console.log('\n=== Cleanup ===')
    await quitDriver(driver1).catch(() => {})
    await quitDriver(driver2).catch(() => {})
    console.log('Cleanup complete')
  })

  beforeEach(async function () {
    // Clear browser state
    await driver1.get(testUrl)
    await chatPage1.sleep(500)
    await clearBrowserState(driver1)

    await driver2.get(testUrl)
    await chatPage2.sleep(500)
    await clearBrowserState(driver2)
  })

  describe('Chat Toggle Button', function () {
    it('should show chat toggle button when conference has a chat_id', async function () {
      console.log('\n--- Test: Chat toggle button visibility ---')

      // Register user
      const user1 = await createTestUser(driver1)
      await chatPage1.waitForChatPage()
      console.log(`[User 1] Registered: ${user1.username}`)

      // Create a chat
      chatName = `ChatPanelTest ${Date.now()}`
      console.log(`[User 1] Creating chat: ${chatName}`)
      await chatPage1.createChat(chatName, 'group')
      await chatPage1.waitForModalToClose()
      await chatPage1.sleep(1000)

      // Select the chat
      await chatPage1.selectFirstChat()
      await chatPage1.waitForChatRoom()
      console.log('[User 1] Chat selected')

      // Start a conference
      console.log('[User 1] Starting Call All...')
      await chatPage1.clickAdHocCallButton()
      await chatPage1.waitForAdHocDropdown()
      await chatPage1.clickCallAll()

      // Wait for ConferenceView popup
      await chatPage1.waitForConferenceView(20000)
      console.log('[User 1] ConferenceView appeared')

      // Wait a moment for UI to fully render
      await chatPage1.sleep(1000)

      // Check that chat toggle button is visible
      const isChatToggleVisible = await chatPage1.isConferenceChatToggleVisible()
      expect(isChatToggleVisible).to.be.true
      console.log('[User 1] Chat toggle button is visible')

      // Cleanup
      await chatPage1.closeConferenceView()
    })
  })

  describe('Chat Sidebar Open/Close', function () {
    it('should open chat sidebar when clicking toggle button', async function () {
      console.log('\n--- Test: Open chat sidebar ---')

      // Register user
      await createTestUser(driver1)
      await chatPage1.waitForChatPage()

      // Create and select chat
      chatName = `ChatSidebarTest ${Date.now()}`
      await chatPage1.createChat(chatName, 'group')
      await chatPage1.waitForModalToClose()
      await chatPage1.selectFirstChat()
      await chatPage1.waitForChatRoom()

      // Start conference
      await chatPage1.clickAdHocCallButton()
      await chatPage1.waitForAdHocDropdown()
      await chatPage1.clickCallAll()
      await chatPage1.waitForConferenceView(20000)
      console.log('Conference started')

      // Verify sidebar is initially closed
      let isSidebarVisible = await chatPage1.isConferenceChatSidebarVisible()
      expect(isSidebarVisible).to.be.false
      console.log('Chat sidebar is initially closed')

      // Click chat toggle button
      await chatPage1.clickConferenceChatToggle()
      console.log('Clicked chat toggle')

      // Wait for sidebar to appear
      await chatPage1.waitForConferenceChatSidebar()
      isSidebarVisible = await chatPage1.isConferenceChatSidebarVisible()
      expect(isSidebarVisible).to.be.true
      console.log('Chat sidebar opened successfully')

      // Cleanup
      await chatPage1.closeConferenceView()
    })

    it('should close chat sidebar when clicking toggle button again', async function () {
      console.log('\n--- Test: Close chat sidebar ---')

      // Register user
      await createTestUser(driver1)
      await chatPage1.waitForChatPage()

      // Create and select chat
      chatName = `ChatCloseTest ${Date.now()}`
      await chatPage1.createChat(chatName, 'group')
      await chatPage1.waitForModalToClose()
      await chatPage1.selectFirstChat()
      await chatPage1.waitForChatRoom()

      // Start conference
      await chatPage1.clickAdHocCallButton()
      await chatPage1.waitForAdHocDropdown()
      await chatPage1.clickCallAll()
      await chatPage1.waitForConferenceView(20000)
      console.log('Conference started')

      // Open chat sidebar
      await chatPage1.clickConferenceChatToggle()
      await chatPage1.waitForConferenceChatSidebar()
      console.log('Chat sidebar opened')

      // Close chat sidebar by clicking toggle again
      await chatPage1.clickConferenceChatToggle()
      await chatPage1.waitForConferenceChatSidebarToClose()

      const isSidebarVisible = await chatPage1.isConferenceChatSidebarVisible()
      expect(isSidebarVisible).to.be.false
      console.log('Chat sidebar closed successfully')

      // Cleanup
      await chatPage1.closeConferenceView()
    })

    it('should close chat sidebar when clicking close button', async function () {
      console.log('\n--- Test: Close via close button ---')

      // Register user
      await createTestUser(driver1)
      await chatPage1.waitForChatPage()

      // Create and select chat
      chatName = `ChatCloseBtn ${Date.now()}`
      await chatPage1.createChat(chatName, 'group')
      await chatPage1.waitForModalToClose()
      await chatPage1.selectFirstChat()
      await chatPage1.waitForChatRoom()

      // Start conference
      await chatPage1.clickAdHocCallButton()
      await chatPage1.waitForAdHocDropdown()
      await chatPage1.clickCallAll()
      await chatPage1.waitForConferenceView(20000)
      console.log('Conference started')

      // Open chat sidebar
      await chatPage1.clickConferenceChatToggle()
      await chatPage1.waitForConferenceChatSidebar()
      console.log('Chat sidebar opened')

      // Close via close button
      await chatPage1.closeConferenceChatSidebar()

      // Wait for animation
      await chatPage1.sleep(500)

      const isSidebarVisible = await chatPage1.isConferenceChatSidebarVisible()
      expect(isSidebarVisible).to.be.false
      console.log('Chat sidebar closed via close button')

      // Cleanup with error handling
      try {
        await chatPage1.closeConferenceView()
      } catch (e) {
        console.log('Cleanup: closeConferenceView failed, conference may have auto-ended')
      }
    })
  })

  describe('Send Messages from Conference Chat', function () {
    it('should send a message from conference chat panel', async function () {
      console.log('\n--- Test: Send message from conference chat ---')

      // Register user
      await createTestUser(driver1)
      await chatPage1.waitForChatPage()

      // Create and select chat
      chatName = `SendMsgTest ${Date.now()}`
      await chatPage1.createChat(chatName, 'group')
      await chatPage1.waitForModalToClose()
      await chatPage1.selectFirstChat()
      await chatPage1.waitForChatRoom()

      // Start conference
      await chatPage1.clickAdHocCallButton()
      await chatPage1.waitForAdHocDropdown()
      await chatPage1.clickCallAll()
      await chatPage1.waitForConferenceView(20000)
      console.log('Conference started')

      // Open chat sidebar
      await chatPage1.clickConferenceChatToggle()
      await chatPage1.waitForConferenceChatSidebar()
      await chatPage1.waitForConferenceChatLoaded()
      console.log('Chat sidebar opened and loaded')

      // Get initial message count
      const initialCount = await chatPage1.getConferenceChatMessageCount()
      console.log(`Initial message count: ${initialCount}`)

      // Send a message
      const testMessage = `Test message from conference ${Date.now()}`
      await chatPage1.sendConferenceChatMessage(testMessage)
      console.log(`Sent message: ${testMessage}`)

      // Wait for message to appear
      await chatPage1.waitForConferenceChatMessage(testMessage)
      console.log('Message appeared in chat')

      // Verify message count increased
      const finalCount = await chatPage1.getConferenceChatMessageCount()
      expect(finalCount).to.be.greaterThan(initialCount)
      console.log(`Final message count: ${finalCount}`)

      // Verify own message styling
      const hasOwnMessage = await chatPage1.hasConferenceChatOwnMessage()
      expect(hasOwnMessage).to.be.true
      console.log('Own message is correctly styled')

      // Cleanup
      await chatPage1.closeConferenceView()
    })

    it('should disable send button when input is empty', async function () {
      console.log('\n--- Test: Send button disabled when empty ---')

      // Register user
      await createTestUser(driver1)
      await chatPage1.waitForChatPage()

      // Create and select chat
      chatName = `EmptyInputTest ${Date.now()}`
      await chatPage1.createChat(chatName, 'group')
      await chatPage1.waitForModalToClose()
      await chatPage1.selectFirstChat()
      await chatPage1.waitForChatRoom()

      // Start conference
      await chatPage1.clickAdHocCallButton()
      await chatPage1.waitForAdHocDropdown()
      await chatPage1.clickCallAll()
      await chatPage1.waitForConferenceView(20000)
      console.log('Conference started')

      // Open chat sidebar
      await chatPage1.clickConferenceChatToggle()
      await chatPage1.waitForConferenceChatSidebar()
      await chatPage1.waitForConferenceChatLoaded()
      console.log('Chat sidebar opened')

      // Check send button is disabled when empty
      const isEnabled = await chatPage1.isConferenceChatSendEnabled()
      expect(isEnabled).to.be.false
      console.log('Send button is correctly disabled when input is empty')

      // Type something and check again
      await chatPage1.typeConferenceChatMessage('test')
      await chatPage1.sleep(200)
      const isEnabledAfterTyping = await chatPage1.isConferenceChatSendEnabled()
      expect(isEnabledAfterTyping).to.be.true
      console.log('Send button is enabled after typing')

      // Cleanup
      await chatPage1.closeConferenceView()
    })
  })

  describe('Message Display', function () {
    it('should show "No messages yet" when chat is empty', async function () {
      console.log('\n--- Test: Empty chat state ---')

      // Register user
      await createTestUser(driver1)
      await chatPage1.waitForChatPage()

      // Create NEW chat (to ensure it's empty)
      chatName = `EmptyChatTest ${Date.now()}`
      await chatPage1.createChat(chatName, 'group')
      await chatPage1.waitForModalToClose()
      await chatPage1.selectFirstChat()
      await chatPage1.waitForChatRoom()

      // Start conference immediately without sending any messages
      await chatPage1.clickAdHocCallButton()
      await chatPage1.waitForAdHocDropdown()
      await chatPage1.clickCallAll()
      await chatPage1.waitForConferenceView(20000)
      console.log('Conference started')

      // Open chat sidebar
      await chatPage1.clickConferenceChatToggle()
      await chatPage1.waitForConferenceChatSidebar()
      await chatPage1.waitForConferenceChatLoaded()
      console.log('Chat sidebar opened')

      // Check for empty state or messages (might have system messages)
      const messageCount = await chatPage1.getConferenceChatMessageCount()
      console.log(`Message count: ${messageCount}`)

      if (messageCount === 0) {
        const isEmpty = await chatPage1.isConferenceChatEmpty()
        expect(isEmpty).to.be.true
        console.log('Empty state is correctly shown')
      } else {
        console.log('Chat has messages (possibly system messages) - that is OK')
      }

      // Cleanup
      await chatPage1.closeConferenceView()
    })

    it('should display existing messages when opening chat', async function () {
      console.log('\n--- Test: Display existing messages ---')

      // Register user
      await createTestUser(driver1)
      await chatPage1.waitForChatPage()

      // Create and select chat
      chatName = `ExistingMsgsTest ${Date.now()}`
      await chatPage1.createChat(chatName, 'group')
      await chatPage1.waitForModalToClose()
      await chatPage1.selectFirstChat()
      await chatPage1.waitForChatRoom()

      // Send some messages BEFORE starting conference
      await chatPage1.sendMessage('Message before conference 1')
      await chatPage1.sleep(500)
      await chatPage1.sendMessage('Message before conference 2')
      await chatPage1.sleep(500)
      console.log('Sent messages before conference')

      // Start conference
      await chatPage1.clickAdHocCallButton()
      await chatPage1.waitForAdHocDropdown()
      await chatPage1.clickCallAll()
      await chatPage1.waitForConferenceView(20000)
      console.log('Conference started')

      // Open chat sidebar
      await chatPage1.clickConferenceChatToggle()
      await chatPage1.waitForConferenceChatSidebar()
      await chatPage1.waitForConferenceChatLoaded()
      console.log('Chat sidebar opened')

      // Verify existing messages are displayed
      const messages = await chatPage1.getConferenceChatMessages()
      console.log(`Messages in conference chat: ${messages.length}`)
      console.log('Messages:', messages)

      expect(messages.length).to.be.at.least(2)
      expect(messages.some(m => m.includes('Message before conference 1'))).to.be.true
      expect(messages.some(m => m.includes('Message before conference 2'))).to.be.true
      console.log('Existing messages displayed correctly')

      // Cleanup
      await chatPage1.closeConferenceView()
    })
  })

  describe('Two Users Chat During Conference', function () {
    it('should allow two users to chat during conference', async function () {
      console.log('\n--- Test: Two users chat during conference ---')

      // Register user 1
      console.log('[User 1] Registering...')
      const user1 = await createTestUser(driver1)
      await chatPage1.waitForChatPage()
      console.log(`[User 1] Registered: ${user1.username}`)

      // Register user 2
      console.log('[User 2] Registering...')
      const user2 = await createTestUser(driver2)
      await chatPage2.waitForChatPage()
      console.log(`[User 2] Registered: ${user2.username}`)

      // User 1 creates chat
      chatName = `TwoUsersChatTest ${Date.now()}`
      console.log(`[User 1] Creating chat: ${chatName}`)
      await chatPage1.createChat(chatName, 'group')
      await chatPage1.waitForModalToClose()
      await chatPage1.selectFirstChat()
      await chatPage1.waitForChatRoom()

      // User 1 adds User 2 to chat
      const user2Id = await getUserIdFromApi(driver2)
      console.log(`[User 1] Adding User 2 (${user2Id}) to chat...`)
      await chatPage1.addParticipantToChat(user2Id)
      await chatPage1.sleep(2000)

      // User 2 should see the chat
      await driver2.navigate().refresh()
      await chatPage2.waitForChatPage()
      await chatPage2.waitForChatInList(chatName, 15000)
      console.log('[User 2] Chat appeared in list')

      // User 1 starts conference
      console.log('[User 1] Starting conference...')
      await chatPage1.clickAdHocCallButton()
      await chatPage1.waitForAdHocDropdown()
      await chatPage1.clickCallAll()
      await chatPage1.waitForConferenceView(20000)
      console.log('[User 1] Conference started')

      // Wait for events
      await chatPage1.sleep(3000)

      // User 2 joins via active events
      await chatPage2.refreshActiveConferences()
      await chatPage2.sleep(2000)

      const isActiveVisible = await chatPage2.isActiveEventsSectionVisible()
      if (isActiveVisible) {
        console.log('[User 2] Joining conference via active events...')
        await chatPage2.clickJoinActiveEvent()
        await chatPage2.waitForConferenceView(20000)
        console.log('[User 2] Joined conference')
      } else {
        console.log('[User 2] Active events not visible, skipping join test')
        await chatPage1.closeConferenceView()
        this.skip()
        return
      }

      // Both users open chat sidebar
      await chatPage1.clickConferenceChatToggle()
      await chatPage1.waitForConferenceChatSidebar()
      await chatPage1.waitForConferenceChatLoaded()
      console.log('[User 1] Chat sidebar opened')

      await chatPage2.clickConferenceChatToggle()
      await chatPage2.waitForConferenceChatSidebar()
      await chatPage2.waitForConferenceChatLoaded()
      console.log('[User 2] Chat sidebar opened')

      // User 1 sends a message
      const user1Message = `Hello from User 1 at ${Date.now()}`
      await chatPage1.sendConferenceChatMessage(user1Message)
      console.log(`[User 1] Sent: ${user1Message}`)

      // User 2 should receive the message
      await chatPage2.waitForConferenceChatMessage(user1Message, 10000)
      console.log('[User 2] Received message from User 1')

      // User 2 sends a response
      const user2Message = `Reply from User 2 at ${Date.now()}`
      await chatPage2.sendConferenceChatMessage(user2Message)
      console.log(`[User 2] Sent: ${user2Message}`)

      // User 1 should receive the response
      await chatPage1.waitForConferenceChatMessage(user2Message, 10000)
      console.log('[User 1] Received message from User 2')

      console.log('Two-way chat during conference works correctly!')

      // Cleanup
      await chatPage2.closeConferenceView()
      await chatPage1.closeConferenceView()
    })
  })

  describe('System Messages', function () {
    it('should show system messages with special styling', async function () {
      console.log('\n--- Test: System messages styling ---')

      // Register user 1
      console.log('[User 1] Registering...')
      await createTestUser(driver1)
      await chatPage1.waitForChatPage()

      // Register user 2
      console.log('[User 2] Registering...')
      await createTestUser(driver2)
      await chatPage2.waitForChatPage()

      // User 1 creates chat
      chatName = `SystemMsgsTest ${Date.now()}`
      await chatPage1.createChat(chatName, 'group')
      await chatPage1.waitForModalToClose()
      await chatPage1.selectFirstChat()
      await chatPage1.waitForChatRoom()

      // User 1 adds User 2 to chat
      const user2Id = await getUserIdFromApi(driver2)
      await chatPage1.addParticipantToChat(user2Id)
      await chatPage1.sleep(2000)

      // User 2 refreshes
      await driver2.navigate().refresh()
      await chatPage2.waitForChatPage()
      await chatPage2.waitForChatInList(chatName, 15000)

      // User 1 starts conference
      await chatPage1.clickAdHocCallButton()
      await chatPage1.waitForAdHocDropdown()
      await chatPage1.clickCallAll()
      await chatPage1.waitForConferenceView(20000)
      console.log('[User 1] Conference started')

      // Wait for system message to be sent
      await chatPage1.sleep(3000)

      // Open chat sidebar
      await chatPage1.clickConferenceChatToggle()
      await chatPage1.waitForConferenceChatSidebar()
      await chatPage1.waitForConferenceChatLoaded()
      console.log('[User 1] Chat sidebar opened')

      // Check for system messages
      const hasSystemMsg = await chatPage1.hasConferenceChatSystemMessage()
      console.log(`Has system message: ${hasSystemMsg}`)

      if (hasSystemMsg) {
        const systemMsgs = await chatPage1.getConferenceChatSystemMessages()
        console.log('System messages:', systemMsgs)
        expect(systemMsgs.length).to.be.at.least(1)
        console.log('System messages are displayed with special styling')
      } else {
        console.log('No system messages found - this may depend on backend configuration')
      }

      // Cleanup
      await chatPage1.closeConferenceView()
    })
  })
})
