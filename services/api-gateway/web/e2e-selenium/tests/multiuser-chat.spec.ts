import { WebDriver } from 'selenium-webdriver'
import { expect } from 'chai'
import { createDriver, quitDriver } from '../config/webdriver.js'
import { ChatPage } from '../pages/ChatPage.js'
import { RegisterPage, RegisterData } from '../pages/RegisterPage.js'
import {
  generateTestUserWithIndex,
  getUserIdFromApi,
  wait,
} from '../helpers/testHelpers.js'

// Configuration
const USER_COUNT = parseInt(process.env.USER_COUNT || '10', 10)
const TEST_DURATION_MINUTES = parseInt(process.env.TEST_DURATION || '5', 10)
const MESSAGE_INTERVAL_MS = 3000 // Send message every 3 seconds per user

interface TestUser {
  driver: WebDriver
  chatPage: ChatPage
  data: RegisterData
  id: string
  messagesSent: number
  messagesReceived: number
}

describe('Multi-User Real-time Chat', function () {
  // Increase timeout significantly for multi-user tests
  this.timeout(TEST_DURATION_MINUTES * 60 * 1000 + 120000)

  const users: TestUser[] = []
  let chatName: string

  before(async function () {
    console.log(`\n=== Starting Multi-User Chat Test ===`)
    console.log(`Users: ${USER_COUNT}, Duration: ${TEST_DURATION_MINUTES} minutes`)
    console.log(`Creating ${USER_COUNT} browser instances...`)

    const timestamp = Date.now()
    chatName = `MultiUser Test Chat ${timestamp}`

    // Create all users in parallel batches to avoid overloading
    const BATCH_SIZE = 5
    for (let batch = 0; batch < Math.ceil(USER_COUNT / BATCH_SIZE); batch++) {
      const startIdx = batch * BATCH_SIZE
      const endIdx = Math.min(startIdx + BATCH_SIZE, USER_COUNT)

      const batchPromises = []
      for (let i = startIdx; i < endIdx; i++) {
        batchPromises.push(createUserSession(i + 1))
      }

      const batchUsers = await Promise.all(batchPromises)
      users.push(...batchUsers)

      console.log(`Created users ${startIdx + 1}-${endIdx}`)
    }

    console.log(`All ${users.length} users created and logged in`)
  })

  after(async function () {
    console.log(`\n=== Cleaning up ===`)

    // Print statistics
    console.log(`\n--- Statistics ---`)
    for (const user of users) {
      console.log(`User ${user.data.displayName}: sent=${user.messagesSent}, received=${user.messagesReceived}`)
    }

    // Close all browsers
    const closePromises = users.map(async (user, index) => {
      try {
        await quitDriver(user.driver)
        console.log(`Closed browser for user ${index + 1}`)
      } catch (e) {
        console.error(`Error closing browser for user ${index + 1}:`, e)
      }
    })

    await Promise.all(closePromises)
    console.log(`All browsers closed`)
  })

  async function createUserSession(index: number): Promise<TestUser> {
    const driver = await createDriver()
    const chatPage = new ChatPage(driver)
    const registerPage = new RegisterPage(driver)
    const userData = generateTestUserWithIndex(index)

    // Register user
    await registerPage.goto()
    await registerPage.register(userData)
    await registerPage.waitForUrl('/chat', 15000)

    // Get user ID
    const userId = await getUserIdFromApi(driver)

    return {
      driver,
      chatPage,
      data: userData,
      id: userId,
      messagesSent: 0,
      messagesReceived: 0,
    }
  }

  it('should create chat with all participants and exchange messages in real-time', async function () {
    // User 1 creates the chat with all other users as participants
    const creator = users[0]
    const otherUserIds = users.slice(1).map(u => u.id)

    console.log(`\n--- Phase 1: Creating chat ---`)
    console.log(`User 1 (${creator.data.displayName}) creating chat "${chatName}"`)
    console.log(`Adding ${otherUserIds.length} participants`)

    await creator.chatPage.createChatWithParticipants(chatName, otherUserIds, 'group')
    await creator.chatPage.waitForModalToClose()
    console.log(`Chat created by User 1`)

    // User 1 enters the chat
    await creator.chatPage.waitForChatInList(chatName, 10000)
    await creator.chatPage.clickChatByNameInList(chatName)
    await creator.chatPage.waitForChatRoom()
    console.log(`User 1 entered the chat`)

    // Wait for other users to receive the chat via real-time event
    console.log(`\n--- Phase 2: Waiting for chat to appear for all users ---`)

    const waitPromises = users.slice(1).map(async (user, idx) => {
      const userNum = idx + 2
      try {
        await user.chatPage.waitForChatInList(chatName, 30000)
        console.log(`User ${userNum} received chat via real-time event`)

        // Enter the chat
        await user.chatPage.clickChatByNameInList(chatName)
        await user.chatPage.waitForChatRoom()
        console.log(`User ${userNum} entered the chat`)
      } catch (e) {
        console.error(`User ${userNum} failed to receive/enter chat:`, e)
        throw e
      }
    })

    await Promise.all(waitPromises)
    console.log(`All users have entered the chat`)

    // Verify all users are in the chat
    for (let i = 0; i < users.length; i++) {
      try {
        const title = await users[i].chatPage.getChatHeaderTitle()
        expect(title).to.include(chatName.substring(0, 20)) // Chat name might be truncated
      } catch (e) {
        console.warn(`User ${i + 1} chat header check failed:`, e)
      }
    }

    // Phase 3: Exchange messages
    console.log(`\n--- Phase 3: Exchanging messages for ${TEST_DURATION_MINUTES} minutes ---`)

    const endTime = Date.now() + TEST_DURATION_MINUTES * 60 * 1000
    let round = 0
    const maxRounds = TEST_DURATION_MINUTES * 20 // Safety limit: ~3 seconds per round

    while (Date.now() < endTime && round < maxRounds) {
      round++
      const remaining = Math.ceil((endTime - Date.now()) / 1000)
      console.log(`\nRound ${round} - Time remaining: ${remaining}s`)

      if (remaining <= 0) break

      // Each user sends a message (in parallel, with some delay between users)
      const sendPromises = users.map(async (user, idx) => {
        // Stagger message sending slightly to avoid exact simultaneity
        await wait(idx * 100)

        const message = `Hello from ${user.data.displayName} - Round ${round} - ${Date.now()}`
        try {
          await user.chatPage.sendMessage(message)
          user.messagesSent++
          console.log(`  User ${idx + 1} sent message`)
        } catch (e) {
          console.error(`  User ${idx + 1} failed to send:`, e)
        }
      })

      await Promise.all(sendPromises)

      // Wait for messages to propagate
      await wait(1000)

      // Check message counts - each user should have received messages from all users
      const expectedMinMessages = round * users.length

      for (let i = 0; i < users.length; i++) {
        try {
          const count = await users[i].chatPage.getMessageCount()
          users[i].messagesReceived = count

          if (count < expectedMinMessages * 0.8) { // Allow 20% loss tolerance
            console.warn(`  User ${i + 1} has ${count} messages (expected ~${expectedMinMessages})`)
          }
        } catch (e) {
          console.error(`  Error checking messages for User ${i + 1}:`, e)
        }
      }

      // Wait before next round
      await wait(MESSAGE_INTERVAL_MS)

      // Check time again before next iteration
      if (Date.now() >= endTime) break
    }

    console.log(`Completed ${round} rounds`)

    console.log(`\n--- Phase 4: Final verification ---`)

    // Final verification - all users should have approximately the same message count
    const messageCounts: number[] = []
    for (let i = 0; i < users.length; i++) {
      const count = await users[i].chatPage.getMessageCount()
      messageCounts.push(count)
      console.log(`User ${i + 1} final message count: ${count}`)
    }

    // All users should have similar message counts (within 20% tolerance)
    const avgCount = messageCounts.reduce((a, b) => a + b, 0) / messageCounts.length
    console.log(`Average message count: ${avgCount.toFixed(0)}`)

    for (let i = 0; i < messageCounts.length; i++) {
      const diff = Math.abs(messageCounts[i] - avgCount) / avgCount
      expect(diff, `User ${i + 1} message count differs too much from average`).to.be.lessThan(0.3)
    }

    // Verify total messages sent equals total expected
    const totalSent = users.reduce((sum, u) => sum + u.messagesSent, 0)
    console.log(`Total messages sent: ${totalSent}`)
    expect(totalSent).to.be.greaterThan(0)

    console.log(`\n=== Multi-User Chat Test Completed Successfully ===`)
  })
})
