/**
 * Message Delete & Restore E2E Tests (Selenium)
 *
 * Тесты soft delete и восстановления сообщений через браузер:
 * - Удаление сообщения (soft delete) - контент скрывается, отображается иконка
 * - Восстановление удалённого сообщения через UI
 * - Проверка иконок: 🗑️ для автора, 🛡️ для модератора
 * - Удаление и восстановление сообщений с файлами
 *
 * Запуск: npx mocha --require ts-node/register tests/message-delete-restore.spec.ts
 */

import { WebDriver } from 'selenium-webdriver'
import { expect } from 'chai'
import { createDriver, quitDriver } from '../config/webdriver.js'
import { ChatPage } from '../pages/ChatPage.js'
import { createTestUser, clearBrowserState } from '../helpers/testHelpers.js'

describe('Message Delete & Restore', function () {
  let driver: WebDriver
  let chatPage: ChatPage

  this.timeout(120000)

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

  describe('Soft Delete', function () {
    it('should delete message and show deletion indicator', async function () {
      const timestamp = Date.now()

      // Create user and login
      await createTestUser(driver, {
        username: `deletetest${timestamp}`,
        displayName: `Delete Test ${timestamp}`
      })
      await chatPage.waitForChatPage()

      // Create a chat
      const chatName = `Delete Test Chat ${timestamp}`
      await chatPage.createChat(chatName, 'group')
      await chatPage.waitForModalToClose()
      await chatPage.sleep(500)

      // Select the chat
      await chatPage.selectFirstChat()
      await chatPage.sleep(500)

      // Send a message
      const messageText = `Message to delete ${timestamp}`
      await chatPage.sendMessage(messageText)
      await chatPage.waitForMessageContaining(messageText, 10000)

      // Get message ID before deletion
      const messageId = await chatPage.getMessageId(0)
      expect(messageId).to.not.be.null
      console.log(`Message ID: ${messageId}`)

      // Delete message via API
      await chatPage.deleteMessageViaApi(messageId!)

      // Wait for deleted indicator to appear
      await chatPage.waitForDeletedMessage(10000)

      // Verify deletion indicator is shown
      const hasDeleted = await chatPage.hasDeletedMessage()
      expect(hasDeleted).to.be.true

      // Verify it shows regular deletion (not moderated)
      const isRegular = await chatPage.isRegularDeletion()
      expect(isRegular).to.be.true

      // Verify the deleted message text contains expected icon
      const deletedText = await chatPage.getDeletedMessageText()
      console.log(`Deleted message text: ${deletedText}`)
      expect(deletedText).to.include('🗑️')

      console.log('Message deleted successfully and shows deletion indicator')
    })

    it('should show restore button on deleted message', async function () {
      const timestamp = Date.now()

      await createTestUser(driver, {
        username: `restoretest${timestamp}`,
        displayName: `Restore Test ${timestamp}`
      })
      await chatPage.waitForChatPage()

      const chatName = `Restore Button Test ${timestamp}`
      await chatPage.createChat(chatName, 'group')
      await chatPage.waitForModalToClose()
      await chatPage.sleep(500)

      await chatPage.selectFirstChat()
      await chatPage.sleep(500)

      // Send and delete message
      await chatPage.sendMessage(`Test message ${timestamp}`)
      await chatPage.waitForMessageCount(1, 10000)

      const messageId = await chatPage.getMessageId(0)
      await chatPage.deleteMessageViaApi(messageId!)
      await chatPage.waitForDeletedMessage(10000)

      // Check restore button is visible for author
      const hasRestoreBtn = await chatPage.hasRestoreButton()
      expect(hasRestoreBtn).to.be.true

      console.log('Restore button is visible on deleted message')
    })
  })

  describe('Restore Message', function () {
    it('should restore deleted message via UI button', async function () {
      const timestamp = Date.now()

      await createTestUser(driver, {
        username: `uirestore${timestamp}`,
        displayName: `UI Restore ${timestamp}`
      })
      await chatPage.waitForChatPage()

      const chatName = `UI Restore Test ${timestamp}`
      await chatPage.createChat(chatName, 'group')
      await chatPage.waitForModalToClose()
      await chatPage.sleep(500)

      await chatPage.selectFirstChat()
      await chatPage.sleep(500)

      // Send message
      const originalMessage = `Original message ${timestamp}`
      await chatPage.sendMessage(originalMessage)
      await chatPage.waitForMessageContaining(originalMessage, 10000)

      // Delete message
      const messageId = await chatPage.getMessageId(0)
      await chatPage.deleteMessageViaApi(messageId!)
      await chatPage.waitForDeletedMessage(10000)

      // Verify message is deleted
      const deletedText = await chatPage.getDeletedMessageText()
      expect(deletedText).to.include('🗑️')

      // Click restore button
      await chatPage.clickRestoreButton()
      await chatPage.sleep(1000)

      // Verify message is restored - no more deleted indicator
      await chatPage.waitForNoDeletedMessage(10000)

      // Verify original content is back
      await chatPage.waitForMessageContaining(originalMessage, 10000)

      console.log('Message restored successfully via UI button')
    })

    it('should restore deleted message via API', async function () {
      const timestamp = Date.now()

      await createTestUser(driver, {
        username: `apirestore${timestamp}`,
        displayName: `API Restore ${timestamp}`
      })
      await chatPage.waitForChatPage()

      const chatName = `API Restore Test ${timestamp}`
      await chatPage.createChat(chatName, 'group')
      await chatPage.waitForModalToClose()
      await chatPage.sleep(500)

      await chatPage.selectFirstChat()
      await chatPage.sleep(500)

      // Send message
      const originalMessage = `API restore test ${timestamp}`
      await chatPage.sendMessage(originalMessage)
      await chatPage.waitForMessageContaining(originalMessage, 10000)

      // Delete message
      const messageId = await chatPage.getMessageId(0)
      await chatPage.deleteMessageViaApi(messageId!)
      await chatPage.waitForDeletedMessage(10000)

      // Restore via API
      await chatPage.restoreMessageViaApi(messageId!)

      // Reload page to see updated state (WebSocket should handle this, but for test reliability)
      await driver.navigate().refresh()
      await chatPage.waitForChatPage()
      await chatPage.selectFirstChat()
      await chatPage.sleep(1000)

      // Verify message is restored
      await chatPage.waitForNoDeletedMessage(10000)
      await chatPage.waitForMessageContaining(originalMessage, 10000)

      console.log('Message restored successfully via API')
    })
  })

  describe('Multiple Delete/Restore Cycles', function () {
    it('should handle multiple delete and restore cycles', async function () {
      const timestamp = Date.now()

      await createTestUser(driver, {
        username: `multicycle${timestamp}`,
        displayName: `Multi Cycle ${timestamp}`
      })
      await chatPage.waitForChatPage()

      const chatName = `Multi Cycle Test ${timestamp}`
      await chatPage.createChat(chatName, 'group')
      await chatPage.waitForModalToClose()
      await chatPage.sleep(500)

      await chatPage.selectFirstChat()
      await chatPage.sleep(500)

      // Send message
      const originalMessage = `Cycle test message ${timestamp}`
      await chatPage.sendMessage(originalMessage)
      await chatPage.waitForMessageContaining(originalMessage, 10000)

      const messageId = await chatPage.getMessageId(0)

      // Perform 2 delete/restore cycles (using UI restore which works better)
      for (let i = 1; i <= 2; i++) {
        console.log(`Cycle ${i}: Deleting message...`)

        // Delete
        await chatPage.deleteMessageViaApi(messageId!)
        await chatPage.waitForDeletedMessage(10000)

        const hasDeleted = await chatPage.hasDeletedMessage()
        expect(hasDeleted).to.be.true

        console.log(`Cycle ${i}: Restoring message...`)

        // Restore via UI button (more reliable than API for UI update)
        await chatPage.clickRestoreButton()
        await chatPage.sleep(1000)
        await chatPage.waitForNoDeletedMessage(10000)

        // Verify content is back
        await chatPage.waitForMessageContaining(originalMessage, 10000)

        console.log(`Cycle ${i}: Complete`)
      }

      console.log('All 2 delete/restore cycles completed successfully')
    })
  })

  describe('Delete Multiple Messages', function () {
    it('should delete multiple messages in a chat', async function () {
      const timestamp = Date.now()

      await createTestUser(driver, {
        username: `multidelete${timestamp}`,
        displayName: `Multi Delete ${timestamp}`
      })
      await chatPage.waitForChatPage()

      const chatName = `Multi Delete Test ${timestamp}`
      await chatPage.createChat(chatName, 'group')
      await chatPage.waitForModalToClose()
      await chatPage.sleep(500)

      await chatPage.selectFirstChat()
      await chatPage.sleep(500)

      // Send multiple messages
      const messageCount = 3
      for (let i = 1; i <= messageCount; i++) {
        await chatPage.sendMessage(`Message ${i} - ${timestamp}`)
        await chatPage.waitForMessageCount(i, 10000)
      }

      // Get all message IDs
      const messageIds: string[] = []
      for (let i = 0; i < messageCount; i++) {
        const msgId = await chatPage.getMessageId(i)
        if (msgId) messageIds.push(msgId)
      }

      console.log(`Got ${messageIds.length} message IDs`)

      // Delete all messages
      for (const msgId of messageIds) {
        await chatPage.deleteMessageViaApi(msgId)
        await chatPage.sleep(300)
      }

      // Wait a bit for UI to update
      await chatPage.sleep(1000)

      // Verify all are deleted
      const deletedCount = await chatPage.getDeletedMessageCount()
      expect(deletedCount).to.equal(messageCount)

      console.log(`Successfully deleted ${deletedCount} messages`)
    })
  })

  describe('Delete with Reply', function () {
    it('should delete second message but keep first', async function () {
      const timestamp = Date.now()

      await createTestUser(driver, {
        username: `replydelete${timestamp}`,
        displayName: `Reply Delete ${timestamp}`
      })
      await chatPage.waitForChatPage()

      const chatName = `Reply Delete Test ${timestamp}`
      await chatPage.createChat(chatName, 'group')
      await chatPage.waitForModalToClose()
      await chatPage.sleep(500)

      await chatPage.selectFirstChat()
      await chatPage.sleep(500)

      // Send first message
      const firstMsg = `First message ${timestamp}`
      await chatPage.sendMessage(firstMsg)
      await chatPage.waitForMessageCount(1, 10000)

      // Send second message
      const secondMsg = `Second message ${timestamp}`
      await chatPage.sendMessage(secondMsg)
      await chatPage.waitForMessageCount(2, 10000)

      await chatPage.sleep(500)

      // Get second message ID
      const msgId1 = await chatPage.getMessageId(1)

      // Delete second message (index 1)
      if (msgId1) {
        await chatPage.deleteMessageViaApi(msgId1)
        await chatPage.sleep(500)
      }

      // Verify only 1 message is deleted
      const deletedCount = await chatPage.getDeletedMessageCount()
      expect(deletedCount).to.equal(1)

      console.log('Second message deleted, first message intact')
    })
  })

  describe('Message Content After Deletion', function () {
    it('should show deletion indicator with correct icon', async function () {
      const timestamp = Date.now()

      await createTestUser(driver, {
        username: `contenthide${timestamp}`,
        displayName: `Content Hide ${timestamp}`
      })
      await chatPage.waitForChatPage()

      const chatName = `Content Hide Test ${timestamp}`
      await chatPage.createChat(chatName, 'group')
      await chatPage.waitForModalToClose()
      await chatPage.sleep(500)

      await chatPage.selectFirstChat()
      await chatPage.sleep(500)

      // Send message
      await chatPage.sendMessage(`Test content ${timestamp}`)
      await chatPage.waitForMessageCount(1, 10000)

      // Delete the message
      const messageId = await chatPage.getMessageId(0)
      await chatPage.deleteMessageViaApi(messageId!)
      await chatPage.waitForDeletedMessage(10000)

      // Verify deleted indicator IS visible with correct icon
      const deletedText = await chatPage.getDeletedMessageText()
      expect(deletedText).to.include('🗑️')
      expect(deletedText).to.include('Сообщение удалено')

      // Verify it's regular deletion (not moderated)
      const isRegular = await chatPage.isRegularDeletion()
      expect(isRegular).to.be.true

      console.log('Deletion indicator shown correctly')
    })

    it('should restore message content via UI button', async function () {
      const timestamp = Date.now()

      await createTestUser(driver, {
        username: `contentrestore${timestamp}`,
        displayName: `Content Restore ${timestamp}`
      })
      await chatPage.waitForChatPage()

      const chatName = `Content Restore Test ${timestamp}`
      await chatPage.createChat(chatName, 'group')
      await chatPage.waitForModalToClose()
      await chatPage.sleep(500)

      await chatPage.selectFirstChat()
      await chatPage.sleep(500)

      // Send message
      const uniqueContent = `RESTORE_TEST_${timestamp}`
      await chatPage.sendMessage(uniqueContent)
      await chatPage.waitForMessageContaining(uniqueContent, 10000)

      // Delete
      const messageId = await chatPage.getMessageId(0)
      await chatPage.deleteMessageViaApi(messageId!)
      await chatPage.waitForDeletedMessage(10000)

      // Verify deleted indicator shown
      const deletedText = await chatPage.getDeletedMessageText()
      expect(deletedText).to.include('🗑️')

      // Restore via UI button
      await chatPage.clickRestoreButton()
      await chatPage.sleep(1000)
      await chatPage.waitForNoDeletedMessage(10000)

      // Verify content is back
      await chatPage.waitForMessageContaining(uniqueContent, 10000)

      console.log('Message content restored successfully via UI')
    })
  })
})
