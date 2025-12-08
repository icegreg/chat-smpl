import { WebDriver } from 'selenium-webdriver'
import { expect } from 'chai'
import * as path from 'path'
import * as fs from 'fs'
import * as os from 'os'
import { createDriver, quitDriver } from '../config/webdriver.js'
import { ChatPage } from '../pages/ChatPage.js'
import { RegisterPage } from '../pages/RegisterPage.js'
import {
  generateTestUserWithIndex,
  getUserIdFromApi,
} from '../helpers/testHelpers.js'

/**
 * Message Forward E2E Test
 *
 * Tests message forwarding with file attachments:
 * 1. User1 creates Chat1 with User2 (but NOT User3)
 * 2. User1 sends message with file in Chat1
 * 3. User1 creates Chat2 with User3 (but NOT User2)
 * 4. User1 forwards the message from Chat1 to Chat2
 * 5. User3 (participant of Chat2) can access the file via new link
 * 6. User2 (participant of Chat1, but NOT Chat2) cannot access the new file link
 * 7. User2 CAN still access the original file link from Chat1
 */
describe('Message Forward', function () {
  this.timeout(180000) // 3 minutes timeout

  let user1Driver: WebDriver
  let user2Driver: WebDriver
  let user3Driver: WebDriver

  let user1ChatPage: ChatPage
  let user2ChatPage: ChatPage
  let user3ChatPage: ChatPage

  let user2Id: string
  let user3Id: string

  let testFile: string
  let chat1Name: string
  let chat2Name: string
  let originalFileLinkId: string
  let forwardedFileLinkId: string

  before(async function () {
    console.log('\n=== Message Forward Test Setup ===')

    // Create test file
    const tempDir = os.tmpdir()
    testFile = path.join(tempDir, `forward-test-${Date.now()}.txt`)
    fs.writeFileSync(testFile, 'This is a test file for forward testing.')
    console.log(`Created test file: ${testFile}`)

    // Create 3 browser instances in parallel
    console.log('Creating 3 browser instances...')
    const [driver1, driver2, driver3] = await Promise.all([
      createDriver(),
      createDriver(),
      createDriver(),
    ])

    user1Driver = driver1
    user2Driver = driver2
    user3Driver = driver3

    user1ChatPage = new ChatPage(user1Driver)
    user2ChatPage = new ChatPage(user2Driver)
    user3ChatPage = new ChatPage(user3Driver)

    // Register all 3 users
    console.log('Registering users...')

    const registerUser = async (
      driver: WebDriver,
      index: number
    ): Promise<string> => {
      const registerPage = new RegisterPage(driver)
      const userData = generateTestUserWithIndex(index)

      await registerPage.goto()
      await registerPage.register(userData)
      await registerPage.waitForUrl('/chat', 15000)

      const userId = await getUserIdFromApi(driver)
      console.log(`User ${index} registered: ${userData.displayName} (${userId})`)

      return userId
    }

    const [id1, id2, id3] = await Promise.all([
      registerUser(user1Driver, 1),
      registerUser(user2Driver, 2),
      registerUser(user3Driver, 3),
    ])

    void id1 // user1Id not needed in tests
    user2Id = id2
    user3Id = id3

    console.log('All users registered')
  })

  after(async function () {
    console.log('\n=== Cleanup ===')

    // Close all browsers
    await Promise.all([
      quitDriver(user1Driver).catch(() => {}),
      quitDriver(user2Driver).catch(() => {}),
      quitDriver(user3Driver).catch(() => {}),
    ])

    // Clean up test file
    try {
      if (fs.existsSync(testFile)) {
        fs.unlinkSync(testFile)
      }
    } catch {
      // Ignore cleanup errors
    }

    console.log('Cleanup complete')
  })

  it('should forward message with file and enforce file permissions correctly', async function () {
    const timestamp = Date.now()
    chat1Name = `Chat1 Forward Test ${timestamp}`
    chat2Name = `Chat2 Forward Test ${timestamp}`

    // ==========================================
    // Step 1: User1 creates Chat1 with User2
    // ==========================================
    console.log('\n--- Step 1: User1 creates Chat1 with User2 ---')

    await user1ChatPage.createChatWithParticipants(chat1Name, [user2Id], 'group')
    await user1ChatPage.waitForModalToClose()
    console.log(`Chat1 created: "${chat1Name}"`)

    // User1 enters Chat1
    await user1ChatPage.waitForChatInList(chat1Name, 10000)
    await user1ChatPage.clickChatByNameInList(chat1Name)
    await user1ChatPage.waitForChatRoom()
    console.log('User1 entered Chat1')

    // Wait for User2 to see Chat1
    await user2ChatPage.waitForChatInList(chat1Name, 30000)
    await user2ChatPage.clickChatByNameInList(chat1Name)
    await user2ChatPage.waitForChatRoom()
    console.log('User2 entered Chat1')

    // ==========================================
    // Step 2: User1 sends message with file in Chat1
    // ==========================================
    console.log('\n--- Step 2: User1 sends message with file in Chat1 ---')

    const originalMessage = `Original message with file ${timestamp}`
    await user1ChatPage.sendMessageWithFile(originalMessage, testFile)
    await user1ChatPage.waitForFileAttachment(15000)
    console.log('File message sent in Chat1')

    // Get original file link ID
    const hrefs = await user1ChatPage.getFileAttachmentHrefs()
    expect(hrefs.length).to.be.greaterThan(0, 'Should have file attachment in Chat1')

    const hrefMatch = hrefs[0].match(/\/api\/files\/([a-f0-9-]+)/)
    expect(hrefMatch).to.not.be.null
    originalFileLinkId = hrefMatch![1]
    console.log(`Original file link ID: ${originalFileLinkId}`)

    // Get message ID from the last message for forwarding
    // We need to get messageId from the API
    const chat1Id = await getFirstChatId(user1Driver)
    console.log(`Chat1 ID: ${chat1Id}`)

    const messageId = await getLastMessageId(user1Driver, chat1Id)
    console.log(`Message ID to forward: ${messageId}`)

    // Verify User2 can access the original file
    console.log('\n--- Verify User2 can access original file ---')
    await new Promise(resolve => setTimeout(resolve, 2000))

    const user2OriginalAccess = await checkFileAccess(user2Driver, originalFileLinkId)
    console.log(`User2 original file access: ${JSON.stringify(user2OriginalAccess)}`)
    expect(user2OriginalAccess.status).to.equal(200, 'User2 (Chat1 participant) should access original file')

    // ==========================================
    // Step 3: User1 creates Chat2 with User3 (not User2)
    // ==========================================
    console.log('\n--- Step 3: User1 creates Chat2 with User3 ---')

    await user1ChatPage.createChatWithParticipants(chat2Name, [user3Id], 'group')
    await user1ChatPage.waitForModalToClose()
    console.log(`Chat2 created: "${chat2Name}"`)

    // Get Chat2 ID
    await user1ChatPage.waitForChatInList(chat2Name, 10000)
    await user1ChatPage.clickChatByNameInList(chat2Name)
    await user1ChatPage.waitForChatRoom()
    const chat2Id = await getFirstChatIdByName(user1Driver, chat2Name)
    console.log(`Chat2 ID: ${chat2Id}`)

    // Wait for User3 to see Chat2
    await user3ChatPage.waitForChatInList(chat2Name, 30000)
    await user3ChatPage.clickChatByNameInList(chat2Name)
    await user3ChatPage.waitForChatRoom()
    console.log('User3 entered Chat2')

    // ==========================================
    // Step 4: User1 forwards message from Chat1 to Chat2
    // ==========================================
    console.log('\n--- Step 4: User1 forwards message to Chat2 ---')

    const forwardedMessage = await forwardMessage(user1Driver, messageId, chat2Id)
    console.log(`Forwarded message: ${JSON.stringify(forwardedMessage)}`)

    expect(forwardedMessage).to.not.be.null
    expect(forwardedMessage.forwarded_from_message_id).to.equal(messageId)

    // Wait for forwarded message to appear in Chat2
    await new Promise(resolve => setTimeout(resolve, 3000))

    // Get forwarded file link ID
    if (forwardedMessage.file_link_ids && forwardedMessage.file_link_ids.length > 0) {
      forwardedFileLinkId = forwardedMessage.file_link_ids[0]
    } else if (forwardedMessage.file_attachments && forwardedMessage.file_attachments.length > 0) {
      forwardedFileLinkId = forwardedMessage.file_attachments[0].link_id
    }

    console.log(`Forwarded file link ID: ${forwardedFileLinkId}`)
    expect(forwardedFileLinkId).to.not.be.undefined
    expect(forwardedFileLinkId).to.not.equal(originalFileLinkId, 'Forwarded link should be different from original')

    // ==========================================
    // Step 5: User3 (Chat2 participant) CAN access forwarded file
    // ==========================================
    console.log('\n--- Step 5: User3 can access forwarded file ---')

    const user3ForwardedAccess = await checkFileAccess(user3Driver, forwardedFileLinkId)
    console.log(`User3 forwarded file access: ${JSON.stringify(user3ForwardedAccess)}`)
    expect(user3ForwardedAccess.status).to.equal(200, 'User3 (Chat2 participant) should access forwarded file')

    // ==========================================
    // Step 6: User2 (NOT in Chat2) CANNOT access forwarded file
    // ==========================================
    console.log('\n--- Step 6: User2 cannot access forwarded file ---')

    const user2ForwardedAccess = await checkFileAccess(user2Driver, forwardedFileLinkId)
    console.log(`User2 forwarded file access: ${JSON.stringify(user2ForwardedAccess)}`)
    expect(user2ForwardedAccess.status).to.equal(403, 'User2 (NOT in Chat2) should be DENIED forwarded file')

    // ==========================================
    // Step 7: User2 CAN still access original file
    // ==========================================
    console.log('\n--- Step 7: User2 can still access original file ---')

    const user2OriginalAccessAgain = await checkFileAccess(user2Driver, originalFileLinkId)
    console.log(`User2 original file access (again): ${JSON.stringify(user2OriginalAccessAgain)}`)
    expect(user2OriginalAccessAgain.status).to.equal(200, 'User2 should still access original file in Chat1')

    // ==========================================
    // Step 8: User3 CANNOT access original file (not in Chat1)
    // ==========================================
    console.log('\n--- Step 8: User3 cannot access original file ---')

    const user3OriginalAccess = await checkFileAccess(user3Driver, originalFileLinkId)
    console.log(`User3 original file access: ${JSON.stringify(user3OriginalAccess)}`)
    expect(user3OriginalAccess.status).to.equal(403, 'User3 (NOT in Chat1) should be DENIED original file')

    console.log('\n=== Message Forward Test PASSED ===')
    console.log('Summary:')
    console.log(`- Original file link (Chat1): ${originalFileLinkId}`)
    console.log(`- Forwarded file link (Chat2): ${forwardedFileLinkId}`)
    console.log('- User2 (Chat1 only): Can access original, CANNOT access forwarded')
    console.log('- User3 (Chat2 only): Can access forwarded, CANNOT access original')
  })
})

// Helper functions

async function checkFileAccess(driver: WebDriver, fileLinkId: string): Promise<{ status: number; ok: boolean; error?: string }> {
  return driver.executeScript(`
    return new Promise(async (resolve) => {
      try {
        const token = localStorage.getItem('access_token');
        const response = await fetch('/api/files/${fileLinkId}', {
          headers: {
            'Authorization': 'Bearer ' + token
          }
        });
        resolve({
          status: response.status,
          ok: response.ok
        });
      } catch (e) {
        resolve({ error: e.message, status: 0, ok: false });
      }
    });
  `) as Promise<{ status: number; ok: boolean; error?: string }>
}

async function getFirstChatId(driver: WebDriver): Promise<string> {
  return driver.executeScript(`
    return new Promise(async (resolve) => {
      try {
        const token = localStorage.getItem('access_token');
        const response = await fetch('/api/chats', {
          headers: {
            'Authorization': 'Bearer ' + token
          }
        });
        const data = await response.json();
        if (data.chats && data.chats.length > 0) {
          resolve(data.chats[0].id);
        } else {
          resolve(null);
        }
      } catch (e) {
        resolve(null);
      }
    });
  `) as Promise<string>
}

async function getFirstChatIdByName(driver: WebDriver, chatName: string): Promise<string> {
  return driver.executeScript(`
    return new Promise(async (resolve) => {
      try {
        const token = localStorage.getItem('access_token');
        const response = await fetch('/api/chats', {
          headers: {
            'Authorization': 'Bearer ' + token
          }
        });
        const data = await response.json();
        if (data.chats) {
          const chat = data.chats.find(c => c.name.includes('${chatName}'));
          resolve(chat ? chat.id : null);
        } else {
          resolve(null);
        }
      } catch (e) {
        resolve(null);
      }
    });
  `) as Promise<string>
}

async function getLastMessageId(driver: WebDriver, chatId: string): Promise<string> {
  return driver.executeScript(`
    return new Promise(async (resolve) => {
      try {
        const token = localStorage.getItem('access_token');
        const response = await fetch('/api/chats/${chatId}/messages', {
          headers: {
            'Authorization': 'Bearer ' + token
          }
        });
        const data = await response.json();
        if (data.messages && data.messages.length > 0) {
          // Messages are usually returned newest first or oldest first
          // Get the last message (which should have file attachments)
          const messageWithFile = data.messages.find(m =>
            (m.file_link_ids && m.file_link_ids.length > 0) ||
            (m.file_attachments && m.file_attachments.length > 0)
          );
          resolve(messageWithFile ? messageWithFile.id : data.messages[data.messages.length - 1].id);
        } else {
          resolve(null);
        }
      } catch (e) {
        console.error('Error getting messages:', e);
        resolve(null);
      }
    });
  `) as Promise<string>
}

async function forwardMessage(driver: WebDriver, messageId: string, targetChatId: string): Promise<any> {
  return driver.executeScript(`
    return new Promise(async (resolve) => {
      try {
        const token = localStorage.getItem('access_token');
        const response = await fetch('/api/chats/messages/${messageId}/forward', {
          method: 'POST',
          headers: {
            'Authorization': 'Bearer ' + token,
            'Content-Type': 'application/json'
          },
          body: JSON.stringify({
            target_chat_id: '${targetChatId}'
          })
        });
        if (!response.ok) {
          const error = await response.text();
          console.error('Forward failed:', response.status, error);
          resolve({ error: error, status: response.status });
        } else {
          const data = await response.json();
          resolve(data);
        }
      } catch (e) {
        console.error('Forward error:', e);
        resolve({ error: e.message });
      }
    });
  `) as Promise<any>
}
