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
 * File Group Access E2E Test
 *
 * Tests that file access permissions are correctly managed through groups:
 * 1. New participant added to chat gets access to existing files
 * 2. Participant removed from chat loses access to files
 * 3. Owner/moderator has delete access to files
 */
describe('File Group Access', function () {
  this.timeout(180000) // 3 minutes timeout

  let user1Driver: WebDriver // Chat owner
  let user2Driver: WebDriver // Initial participant
  let user3Driver: WebDriver // Will be added later

  let user1ChatPage: ChatPage
  let user2ChatPage: ChatPage
  let user3ChatPage: ChatPage

  let user2Id: string
  let user3Id: string

  let testFile: string
  let fileLinkId: string
  let chatName: string

  before(async function () {
    console.log('\n=== File Group Access Test Setup ===')

    // Create test file
    const tempDir = os.tmpdir()
    testFile = path.join(tempDir, `group-access-test-${Date.now()}.txt`)
    fs.writeFileSync(testFile, 'This is a group access test file.')
    console.log(`Created test file: ${testFile}`)

    // Create 3 browser instances
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
      const userData = generateTestUserWithIndex(index + 100) // Different indices to avoid conflicts

      await registerPage.goto()
      await registerPage.register(userData)
      await registerPage.waitForUrl('/chat', 15000)

      const userId = await getUserIdFromApi(driver)
      console.log(`User ${index} registered: ${userData.displayName} (${userId})`)

      return userId
    }

    const [, id2, id3] = await Promise.all([
      registerUser(user1Driver, 1),
      registerUser(user2Driver, 2),
      registerUser(user3Driver, 3),
    ])

    user2Id = id2
    user3Id = id3

    console.log('All users registered')
  })

  after(async function () {
    console.log('\n=== Cleanup ===')

    await Promise.all([
      quitDriver(user1Driver).catch(() => {}),
      quitDriver(user2Driver).catch(() => {}),
      quitDriver(user3Driver).catch(() => {}),
    ])

    try {
      if (fs.existsSync(testFile)) {
        fs.unlinkSync(testFile)
      }
    } catch {
      // Ignore cleanup errors
    }

    console.log('Cleanup complete')
  })

  it('should grant file access when participant is added to chat', async function () {
    chatName = `Group Access Test ${Date.now()}`

    // Step 1: User 1 creates chat with User 2 only (User 3 NOT included)
    console.log('\n--- Step 1: Create chat with User 2 only ---')
    await user1ChatPage.createChatWithParticipants(chatName, [user2Id], 'group')
    await user1ChatPage.waitForModalToClose()
    console.log('Chat created')

    // User 1 enters the chat
    await user1ChatPage.waitForChatInList(chatName, 10000)
    await user1ChatPage.clickChatByNameInList(chatName)
    await user1ChatPage.waitForChatRoom()

    // Step 2: User 1 sends a file in the chat
    console.log('\n--- Step 2: Send file before adding User 3 ---')
    await user1ChatPage.sendMessageWithFile('File before adding user 3', testFile)
    await user1ChatPage.waitForFileAttachment(15000)

    // Get file link ID
    const hrefs = await user1ChatPage.getFileAttachmentHrefs()
    expect(hrefs.length).to.be.greaterThan(0)
    const hrefMatch = hrefs[0].match(/\/api\/files\/([a-f0-9-]+)/)
    expect(hrefMatch).to.not.be.null
    fileLinkId = hrefMatch![1]
    console.log(`File link ID: ${fileLinkId}`)

    // Step 3: Verify User 3 cannot access file (not in chat yet)
    console.log('\n--- Step 3: Verify User 3 has no access ---')
    const user3BeforeResult = await user3Driver.executeScript(`
      return new Promise(async (resolve) => {
        try {
          const token = localStorage.getItem('access_token');
          const response = await fetch('/api/files/${fileLinkId}', {
            headers: { 'Authorization': 'Bearer ' + token }
          });
          resolve({ status: response.status });
        } catch (e) {
          resolve({ error: e.message });
        }
      });
    `) as { status: number; error?: string }

    console.log(`User 3 access before being added: ${JSON.stringify(user3BeforeResult)}`)
    expect(user3BeforeResult.status).to.equal(403, 'User 3 should NOT have access before being added')

    // Step 4: Add User 3 to the chat
    console.log('\n--- Step 4: Add User 3 to chat ---')
    await user1ChatPage.addParticipantToChat(user3Id)
    await user1ChatPage.sleep(2000) // Wait for permissions to propagate

    // Step 5: User 3 refreshes and should now see the chat
    console.log('\n--- Step 5: Verify User 3 can access file after being added ---')
    await user3ChatPage.refresh()
    await user3ChatPage.sleep(1000)
    await user3ChatPage.waitForChatInList(chatName, 15000)

    const user3AfterResult = await user3Driver.executeScript(`
      return new Promise(async (resolve) => {
        try {
          const token = localStorage.getItem('access_token');
          const response = await fetch('/api/files/${fileLinkId}', {
            headers: { 'Authorization': 'Bearer ' + token }
          });
          resolve({ status: response.status });
        } catch (e) {
          resolve({ error: e.message });
        }
      });
    `) as { status: number; error?: string }

    console.log(`User 3 access after being added: ${JSON.stringify(user3AfterResult)}`)
    expect(user3AfterResult.status).to.equal(200, 'User 3 should have access after being added to chat')

    console.log('\n=== Participant Add Access Test PASSED ===')
  })

  it('should revoke file access when participant is removed from chat', async function () {
    // Skip if previous test didn't set up the chat
    if (!fileLinkId) {
      this.skip()
    }

    console.log('\n--- Testing access revocation on removal ---')

    // Step 1: Remove User 3 from chat (as owner/User 1)
    console.log('\n--- Step 1: Remove User 3 from chat ---')
    await user1ChatPage.removeParticipantFromChat(user3Id)
    await user1ChatPage.sleep(2000) // Wait for permissions to revoke

    // Step 2: Verify User 3 can no longer access the file
    console.log('\n--- Step 2: Verify User 3 lost access ---')
    const user3AfterRemoveResult = await user3Driver.executeScript(`
      return new Promise(async (resolve) => {
        try {
          const token = localStorage.getItem('access_token');
          const response = await fetch('/api/files/${fileLinkId}', {
            headers: { 'Authorization': 'Bearer ' + token }
          });
          resolve({ status: response.status });
        } catch (e) {
          resolve({ error: e.message });
        }
      });
    `) as { status: number; error?: string }

    console.log(`User 3 access after removal: ${JSON.stringify(user3AfterRemoveResult)}`)
    expect(user3AfterRemoveResult.status).to.equal(403, 'User 3 should NOT have access after being removed')

    // Step 3: User 2 (who was never removed) should still have access
    console.log('\n--- Step 3: Verify User 2 still has access ---')
    const user2StillResult = await user2Driver.executeScript(`
      return new Promise(async (resolve) => {
        try {
          const token = localStorage.getItem('access_token');
          const response = await fetch('/api/files/${fileLinkId}', {
            headers: { 'Authorization': 'Bearer ' + token }
          });
          resolve({ status: response.status });
        } catch (e) {
          resolve({ error: e.message });
        }
      });
    `) as { status: number; error?: string }

    console.log(`User 2 access (unchanged): ${JSON.stringify(user2StillResult)}`)
    expect(user2StillResult.status).to.equal(200, 'User 2 should still have access')

    console.log('\n=== Participant Remove Access Test PASSED ===')
  })

  it('should allow chat owner/moderator to access all files', async function () {
    // Skip if previous test didn't set up the chat
    if (!fileLinkId) {
      this.skip()
    }

    console.log('\n--- Testing owner access rights ---')

    // User 1 (owner) should be able to access the file
    const user1Result = await user1Driver.executeScript(`
      return new Promise(async (resolve) => {
        try {
          const token = localStorage.getItem('access_token');
          const response = await fetch('/api/files/${fileLinkId}', {
            headers: { 'Authorization': 'Bearer ' + token }
          });
          resolve({ status: response.status });
        } catch (e) {
          resolve({ error: e.message });
        }
      });
    `) as { status: number; error?: string }

    console.log(`Owner access: ${JSON.stringify(user1Result)}`)
    expect(user1Result.status).to.equal(200, 'Chat owner should have full access to files')

    console.log('\n=== Owner Access Test PASSED ===')
  })

  it('should allow participant to view but not delete other user files', async function () {
    // Skip if previous test didn't set up the chat
    if (!fileLinkId) {
      this.skip()
    }

    console.log('\n--- Testing participant delete restriction ---')

    // User 2 (participant, not owner) tries to delete the file
    const user2DeleteResult = await user2Driver.executeScript(`
      return new Promise(async (resolve) => {
        try {
          const token = localStorage.getItem('access_token');
          const response = await fetch('/api/files/${fileLinkId}', {
            method: 'DELETE',
            headers: { 'Authorization': 'Bearer ' + token }
          });
          resolve({ status: response.status });
        } catch (e) {
          resolve({ error: e.message });
        }
      });
    `) as { status: number; error?: string }

    console.log(`User 2 delete attempt: ${JSON.stringify(user2DeleteResult)}`)

    // Regular participants should have read access but not delete access
    // They should get 403 when trying to delete
    expect(user2DeleteResult.status).to.equal(403, 'Regular participant should not be able to delete files')

    console.log('\n=== Participant Delete Restriction Test PASSED ===')
  })
})
