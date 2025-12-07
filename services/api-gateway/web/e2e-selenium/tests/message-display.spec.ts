import { WebDriver } from 'selenium-webdriver'
import { expect } from 'chai'
import { createDriver, quitDriver } from '../config/webdriver.js'
import { ChatPage } from '../pages/ChatPage.js'
import { createTestUser, clearBrowserState } from '../helpers/testHelpers.js'

describe('Message Display', function () {
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

  describe('Sender Information', function () {
    it('should display sender name on messages', async function () {
      // Create user with known display name
      const timestamp = Date.now()
      const displayName = `Test User ${timestamp}`
      await createTestUser(driver, {
        username: `testuser${timestamp}`,
        displayName: displayName
      })
      await chatPage.waitForChatPage()

      // Create a chat
      const chatName = `Sender Name Test ${timestamp}`
      await chatPage.createChat(chatName, 'group')
      await chatPage.waitForModalToClose()
      await chatPage.sleep(500)

      // Select the chat
      await chatPage.selectFirstChat()
      await chatPage.sleep(500)

      // Send a message
      const messageText = `Test message ${timestamp}`
      await chatPage.sendMessage(messageText)

      // Wait for message to appear
      await chatPage.waitForMessageContaining(messageText, 10000)

      // Wait for sender name
      await chatPage.waitForSenderName(displayName, 10000)

      // Verify sender name is displayed correctly
      const senderNames = await chatPage.getMessageSenderNames()
      expect(senderNames.length).to.be.greaterThan(0)
      expect(senderNames[0]).to.include(displayName)
    })

    it('should not display "Unknown" as sender name', async function () {
      const timestamp = Date.now()
      await createTestUser(driver, {
        username: `testuser${timestamp}`,
        displayName: `Known User ${timestamp}`
      })
      await chatPage.waitForChatPage()

      // Create a chat
      const chatName = `No Unknown Test ${timestamp}`
      await chatPage.createChat(chatName, 'group')
      await chatPage.waitForModalToClose()
      await chatPage.sleep(500)

      // Select the chat
      await chatPage.selectFirstChat()
      await chatPage.sleep(500)

      // Send a message
      await chatPage.sendMessage(`Message ${timestamp}`)
      await chatPage.waitForMessageCount(1, 10000)

      // Wait a bit for sender info to load
      await chatPage.sleep(500)

      // Verify no "Unknown" sender names
      const senderNames = await chatPage.getMessageSenderNames()
      expect(senderNames.length).to.be.greaterThan(0)
      for (const name of senderNames) {
        expect(name).to.not.equal('Unknown')
      }
    })
  })

  describe('Avatar Display', function () {
    it('should display avatar image or placeholder for messages', async function () {
      const timestamp = Date.now()
      await createTestUser(driver, {
        username: `avatartest${timestamp}`,
        displayName: `Avatar User ${timestamp}`
      })
      await chatPage.waitForChatPage()

      // Create a chat
      const chatName = `Avatar Test ${timestamp}`
      await chatPage.createChat(chatName, 'group')
      await chatPage.waitForModalToClose()
      await chatPage.sleep(500)

      // Select the chat
      await chatPage.selectFirstChat()
      await chatPage.sleep(500)

      // Send a message
      await chatPage.sendMessage(`Avatar test message ${timestamp}`)
      await chatPage.waitForMessageCount(1, 10000)
      await chatPage.sleep(500)

      // Check that either avatar image or placeholder exists
      const hasAvatar = await chatPage.hasMessageAvatar()
      const hasPlaceholder = await chatPage.hasMessageAvatarPlaceholder()

      expect(hasAvatar || hasPlaceholder).to.be.true
    })

    it('should display avatar image when user has avatar_url', async function () {
      const timestamp = Date.now()
      // Users registered via API get random cat avatar from cataas.com
      await createTestUser(driver, {
        username: `avatarimg${timestamp}`,
        displayName: `Avatar Image User ${timestamp}`
      })
      await chatPage.waitForChatPage()

      // Create a chat
      const chatName = `Avatar Image Test ${timestamp}`
      await chatPage.createChat(chatName, 'group')
      await chatPage.waitForModalToClose()
      await chatPage.sleep(500)

      // Select the chat
      await chatPage.selectFirstChat()
      await chatPage.sleep(500)

      // Send a message
      await chatPage.sendMessage(`Test with avatar ${timestamp}`)
      await chatPage.waitForMessageCount(1, 10000)
      await chatPage.sleep(1000)

      // Check avatar src contains cat image URL
      const avatarSrc = await chatPage.getAvatarSrc()
      if (avatarSrc) {
        expect(avatarSrc).to.include('cataas.com')
      } else {
        // If no avatar image, placeholder should have first letter of display name
        const placeholder = await chatPage.getAvatarPlaceholderText()
        expect(placeholder).to.equal('A') // "Avatar Image User" starts with A
      }
    })

    it('should display placeholder with first letter when no avatar', async function () {
      const timestamp = Date.now()
      const displayName = `Zorro ${timestamp}`
      await createTestUser(driver, {
        username: `zorro${timestamp}`,
        displayName: displayName
      })
      await chatPage.waitForChatPage()

      // Create a chat
      const chatName = `Placeholder Test ${timestamp}`
      await chatPage.createChat(chatName, 'group')
      await chatPage.waitForModalToClose()
      await chatPage.sleep(500)

      // Select the chat
      await chatPage.selectFirstChat()
      await chatPage.sleep(500)

      // Send a message
      await chatPage.sendMessage(`Placeholder message ${timestamp}`)
      await chatPage.waitForMessageCount(1, 10000)
      await chatPage.sleep(500)

      // If using placeholder (no avatar), it should show first letter
      const hasPlaceholder = await chatPage.hasMessageAvatarPlaceholder()
      if (hasPlaceholder) {
        const placeholderText = await chatPage.getAvatarPlaceholderText()
        expect(placeholderText).to.equal('Z') // "Zorro" starts with Z
      }
    })
  })

  describe('Message Time', function () {
    it('should display time on messages', async function () {
      const timestamp = Date.now()
      await createTestUser(driver)
      await chatPage.waitForChatPage()

      // Create a chat
      const chatName = `Time Test ${timestamp}`
      await chatPage.createChat(chatName, 'group')
      await chatPage.waitForModalToClose()
      await chatPage.sleep(500)

      // Select the chat
      await chatPage.selectFirstChat()
      await chatPage.sleep(500)

      // Send a message
      await chatPage.sendMessage(`Time test message ${timestamp}`)
      await chatPage.waitForMessageCount(1, 10000)
      await chatPage.sleep(500)

      // Verify time is displayed
      const times = await chatPage.getMessageTimes()
      expect(times.length).to.be.greaterThan(0)

      // Time should match HH:MM format (e.g., "14:35" or "2:35 PM")
      const timePattern = /^\d{1,2}:\d{2}(\s?(AM|PM))?$/i
      expect(times[0]).to.match(timePattern)
    })

    it('should display time for all messages', async function () {
      const timestamp = Date.now()
      await createTestUser(driver)
      await chatPage.waitForChatPage()

      // Create a chat
      const chatName = `Multi Time Test ${timestamp}`
      await chatPage.createChat(chatName, 'group')
      await chatPage.waitForModalToClose()
      await chatPage.sleep(500)

      // Select the chat
      await chatPage.selectFirstChat()
      await chatPage.sleep(500)

      // Send multiple messages
      await chatPage.sendMessage(`First ${timestamp}`)
      await chatPage.waitForMessageCount(1)

      await chatPage.sendMessage(`Second ${timestamp}`)
      await chatPage.waitForMessageCount(2)

      await chatPage.sendMessage(`Third ${timestamp}`)
      await chatPage.waitForMessageCount(3)
      await chatPage.sleep(500)

      // Verify all messages have time
      const times = await chatPage.getMessageTimes()
      const messageCount = await chatPage.getMessageCount()
      expect(times.length).to.equal(messageCount)
    })
  })

  describe('Date Separators', function () {
    it('should display date separator when messages exist', async function () {
      const timestamp = Date.now()
      await createTestUser(driver)
      await chatPage.waitForChatPage()

      // Create a chat
      const chatName = `Date Separator Test ${timestamp}`
      await chatPage.createChat(chatName, 'group')
      await chatPage.waitForModalToClose()
      await chatPage.sleep(500)

      // Select the chat
      await chatPage.selectFirstChat()
      await chatPage.sleep(500)

      // Send a message
      await chatPage.sendMessage(`Date separator test ${timestamp}`)
      await chatPage.waitForMessageCount(1, 10000)

      // Wait for date separator to appear
      await chatPage.waitForDateSeparator(10000)

      // Verify date separator is displayed
      const separatorCount = await chatPage.getDateSeparatorsCount()
      expect(separatorCount).to.be.greaterThan(0)
    })

    it('should display "Today" for today\'s messages', async function () {
      const timestamp = Date.now()
      await createTestUser(driver)
      await chatPage.waitForChatPage()

      // Create a chat
      const chatName = `Today Test ${timestamp}`
      await chatPage.createChat(chatName, 'group')
      await chatPage.waitForModalToClose()
      await chatPage.sleep(500)

      // Select the chat
      await chatPage.selectFirstChat()
      await chatPage.sleep(500)

      // Send a message (will be today)
      await chatPage.sendMessage(`Today's message ${timestamp}`)
      await chatPage.waitForMessageCount(1, 10000)
      await chatPage.waitForDateSeparator(10000)

      // Verify "Today" label is displayed
      const dateLabels = await chatPage.getDateLabels()
      expect(dateLabels.length).to.be.greaterThan(0)
      expect(dateLabels).to.include('Today')
    })

    it('should have one date separator for messages on same day', async function () {
      const timestamp = Date.now()
      await createTestUser(driver)
      await chatPage.waitForChatPage()

      // Create a chat
      const chatName = `Single Separator Test ${timestamp}`
      await chatPage.createChat(chatName, 'group')
      await chatPage.waitForModalToClose()
      await chatPage.sleep(500)

      // Select the chat
      await chatPage.selectFirstChat()
      await chatPage.sleep(500)

      // Send multiple messages (all will be today)
      await chatPage.sendMessage(`Message 1 ${timestamp}`)
      await chatPage.waitForMessageCount(1)

      await chatPage.sendMessage(`Message 2 ${timestamp}`)
      await chatPage.waitForMessageCount(2)

      await chatPage.sendMessage(`Message 3 ${timestamp}`)
      await chatPage.waitForMessageCount(3)
      await chatPage.sleep(500)

      // Should have only one date separator (all messages are today)
      const separatorCount = await chatPage.getDateSeparatorsCount()
      expect(separatorCount).to.equal(1)

      const dateLabels = await chatPage.getDateLabels()
      expect(dateLabels).to.deep.equal(['Today'])
    })
  })

  describe('Complete Message Info Display', function () {
    it('should display sender name, avatar, time and date separator together', async function () {
      const timestamp = Date.now()
      const displayName = `Complete Test User ${timestamp}`
      await createTestUser(driver, {
        username: `completetest${timestamp}`,
        displayName: displayName
      })
      await chatPage.waitForChatPage()

      // Create a chat
      const chatName = `Complete Info Test ${timestamp}`
      await chatPage.createChat(chatName, 'group')
      await chatPage.waitForModalToClose()
      await chatPage.sleep(500)

      // Select the chat
      await chatPage.selectFirstChat()
      await chatPage.sleep(500)

      // Send a message
      const messageText = `Complete info message ${timestamp}`
      await chatPage.sendMessage(messageText)
      await chatPage.waitForMessageContaining(messageText, 10000)
      await chatPage.sleep(1000)

      // Verify all elements are present
      // 1. Sender name
      const senderNames = await chatPage.getMessageSenderNames()
      expect(senderNames.length).to.be.greaterThan(0)
      expect(senderNames[0]).to.include(displayName)

      // 2. Avatar or placeholder
      const hasAvatar = await chatPage.hasMessageAvatar()
      const hasPlaceholder = await chatPage.hasMessageAvatarPlaceholder()
      expect(hasAvatar || hasPlaceholder).to.be.true

      // 3. Time
      const times = await chatPage.getMessageTimes()
      expect(times.length).to.be.greaterThan(0)
      expect(times[0]).to.match(/^\d{1,2}:\d{2}/)

      // 4. Date separator
      const separatorCount = await chatPage.getDateSeparatorsCount()
      expect(separatorCount).to.be.greaterThan(0)

      // 5. Date label is "Today"
      const dateLabels = await chatPage.getDateLabels()
      expect(dateLabels).to.include('Today')

      console.log(`Verified complete message display:`)
      console.log(`  - Sender: ${senderNames[0]}`)
      console.log(`  - Has avatar: ${hasAvatar}`)
      console.log(`  - Has placeholder: ${hasPlaceholder}`)
      console.log(`  - Time: ${times[0]}`)
      console.log(`  - Date label: ${dateLabels[0]}`)
    })
  })
})
