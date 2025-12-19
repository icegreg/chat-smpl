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
 * File Security E2E Test
 *
 * Tests that file access permissions are correctly enforced:
 * 1. Chat participants can download files attached to messages
 * 2. Non-participants receive 403 Forbidden when trying to access files
 */
describe('File Security', function () {
  this.timeout(120000) // 2 minutes timeout

  let user1Driver: WebDriver
  let user2Driver: WebDriver
  let user3Driver: WebDriver // non-participant

  let user1ChatPage: ChatPage
  let user2ChatPage: ChatPage

  let user2Id: string

  let testFile: string
  let fileLinkId: string

  before(async function () {
    console.log('\n=== File Security Test Setup ===')

    // Create test file
    const tempDir = os.tmpdir()
    testFile = path.join(tempDir, `security-test-${Date.now()}.txt`)
    fs.writeFileSync(testFile, 'This is a security test file.')
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
    // user3 doesn't need ChatPage since they just try to access files directly

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

    const [, id2] = await Promise.all([
      registerUser(user1Driver, 1),
      registerUser(user2Driver, 2),
      registerUser(user3Driver, 3),
    ])

    user2Id = id2

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

  it('should allow chat participant to download file, deny non-participant', async function () {
    const chatName = `Security Test Chat ${Date.now()}`

    // Step 1: User 1 creates a chat with User 2 (but NOT User 3)
    console.log('\n--- Step 1: Create chat with participants ---')
    console.log(`User 1 creating chat "${chatName}" with User 2`)

    await user1ChatPage.createChatWithParticipants(chatName, [user2Id], 'group')
    await user1ChatPage.waitForModalToClose()
    console.log('Chat created')

    // User 1 enters the chat
    await user1ChatPage.waitForChatInList(chatName, 10000)
    await user1ChatPage.clickChatByNameInList(chatName)
    await user1ChatPage.waitForChatRoom()
    console.log('User 1 entered the chat')

    // Step 2: Wait for User 2 to see the chat
    console.log('\n--- Step 2: User 2 receives chat ---')
    // Refresh User 2's page to ensure they fetch chats from API (not just real-time)
    await user2ChatPage.refresh()
    await user2ChatPage.sleep(1000)
    await user2ChatPage.waitForChatInList(chatName, 15000)
    await user2ChatPage.clickChatByNameInList(chatName)
    await user2ChatPage.waitForChatRoom()
    console.log('User 2 entered the chat')

    // Step 3: User 1 sends message with file attachment
    console.log('\n--- Step 3: User 1 sends file in chat ---')
    await user1ChatPage.sendMessageWithFile('Here is a secure file', testFile)
    await user1ChatPage.waitForFileAttachment(15000)
    console.log('File sent')

    // Get the file link ID from the attachment href
    const hrefs = await user1ChatPage.getFileAttachmentHrefs()
    expect(hrefs.length).to.be.greaterThan(0, 'Should have file attachment')

    // Extract link ID from href like /api/files/uuid
    const hrefMatch = hrefs[0].match(/\/api\/files\/([a-f0-9-]+)/)
    expect(hrefMatch).to.not.be.null
    fileLinkId = hrefMatch![1]
    console.log(`File link ID: ${fileLinkId}`)

    // Step 4: User 2 should be able to download the file (permission granted via API)
    console.log('\n--- Step 4: User 2 can access file ---')
    // Give the system a moment to propagate permissions
    await new Promise(resolve => setTimeout(resolve, 2000))

    // Test file download for User 2 (participant)
    const user2DownloadResult = await user2Driver.executeScript(`
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
          resolve({ error: e.message });
        }
      });
    `) as { status: number; ok: boolean; error?: string }

    console.log(`User 2 download attempt: ${JSON.stringify(user2DownloadResult)}`)
    expect(user2DownloadResult.status).to.equal(200, 'User 2 (participant) should be able to download file')

    // Step 5: User 3 (non-participant) tries to download the file
    console.log('\n--- Step 5: User 3 (non-participant) denied access ---')

    const user3DownloadResult = await user3Driver.executeScript(`
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
          resolve({ error: e.message });
        }
      });
    `) as { status: number; ok: boolean; error?: string }

    console.log(`User 3 download attempt: ${JSON.stringify(user3DownloadResult)}`)
    expect(user3DownloadResult.status).to.equal(403, 'User 3 (non-participant) should be denied access')

    console.log('\n=== File Security Test PASSED ===')
    console.log('- Chat participant (User 2) can download file: 200 OK')
    console.log('- Non-participant (User 3) is denied: 403 Forbidden')
  })

  it('should allow uploader to always access their own file', async function () {
    // User 1 uploaded the file, they should be able to access it
    console.log('\n--- Verify uploader (User 1) can access own file ---')

    const user1DownloadResult = await user1Driver.executeScript(`
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
          resolve({ error: e.message });
        }
      });
    `) as { status: number; ok: boolean; error?: string }

    console.log(`User 1 (uploader) download attempt: ${JSON.stringify(user1DownloadResult)}`)
    expect(user1DownloadResult.status).to.equal(200, 'Uploader should always be able to download their own file')
  })

  it('should deny unauthenticated access', async function () {
    console.log('\n--- Verify unauthenticated access is denied ---')

    // Make request without auth token
    const unauthResult = await user1Driver.executeScript(`
      return new Promise(async (resolve) => {
        try {
          const response = await fetch('/api/files/${fileLinkId}');
          resolve({
            status: response.status,
            ok: response.ok
          });
        } catch (e) {
          resolve({ error: e.message });
        }
      });
    `) as { status: number; ok: boolean; error?: string }

    console.log(`Unauthenticated access attempt: ${JSON.stringify(unauthResult)}`)
    expect(unauthResult.status).to.equal(401, 'Unauthenticated request should be denied')
  })
})
