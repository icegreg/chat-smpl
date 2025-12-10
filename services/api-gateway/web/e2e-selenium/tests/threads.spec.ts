import { WebDriver } from 'selenium-webdriver'
import { expect } from 'chai'
import { createDriver, quitDriver } from '../config/webdriver.js'
import { ChatPage } from '../pages/ChatPage.js'
import { createTestUser, clearBrowserState } from '../helpers/testHelpers.js'

describe('Threads and Subthreads', function () {
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

  describe('Thread Creation via API', function () {
    it('should create a thread via API and list it', async function () {
      // Create user and chat
      await createTestUser(driver)
      await chatPage.waitForChatPage()

      const chatName = `Thread Test Chat ${Date.now()}`
      await chatPage.createChat(chatName, 'group')
      await chatPage.waitForModalToClose()
      await chatPage.sleep(1000)

      // Select the chat
      await chatPage.selectFirstChat()
      await chatPage.sleep(1000)

      // Get chat ID from URL
      const chatId = await chatPage.getCurrentChatId()
      console.log('Chat ID:', chatId)
      expect(chatId).to.not.be.empty

      // Create thread via API
      const threadTitle = `Test Thread ${Date.now()}`
      const threadId = await chatPage.createThreadViaApi(chatId, threadTitle)
      console.log('Created thread ID:', threadId)
      expect(threadId).to.not.be.empty

      // List threads and verify
      const threads = await chatPage.listThreadsViaApi(chatId)
      console.log('Threads:', threads)
      expect(threads.length).to.be.at.least(1)

      // Find the created thread
      const createdThread = threads.find(t => t.id === threadId)
      expect(createdThread).to.exist
      expect(createdThread!.title).to.equal(threadTitle)
      // protobuf3 omits zero values, so depth may be undefined for level 0
      expect(createdThread!.depth ?? 0).to.equal(0) // Top-level thread
    })

    it('should create a subthread via API with correct depth', async function () {
      // Create user and chat
      await createTestUser(driver)
      await chatPage.waitForChatPage()

      const chatName = `Subthread Test Chat ${Date.now()}`
      await chatPage.createChat(chatName, 'group')
      await chatPage.waitForModalToClose()
      await chatPage.sleep(1000)

      // Select the chat
      await chatPage.selectFirstChat()
      await chatPage.sleep(1000)

      // Get chat ID
      const chatId = await chatPage.getCurrentChatId()

      // Create parent thread
      const parentTitle = `Parent Thread ${Date.now()}`
      const parentThreadId = await chatPage.createThreadViaApi(chatId, parentTitle)
      console.log('Created parent thread ID:', parentThreadId)

      // Create subthread
      const subthreadTitle = `Subthread ${Date.now()}`
      const subthreadId = await chatPage.createSubthreadViaApi(parentThreadId, subthreadTitle)
      console.log('Created subthread ID:', subthreadId)
      expect(subthreadId).to.not.be.empty

      // List subthreads
      const subthreads = await chatPage.listSubthreadsViaApi(parentThreadId)
      console.log('Subthreads:', subthreads)
      expect(subthreads.length).to.be.at.least(1)

      // Find the created subthread
      const createdSubthread = subthreads.find(t => t.id === subthreadId)
      expect(createdSubthread).to.exist
      expect(createdSubthread!.title).to.equal(subthreadTitle)
      expect(createdSubthread!.depth).to.equal(1) // Subthread depth
    })

    it('should create nested subthreads up to depth 5', async function () {
      // Create user and chat
      await createTestUser(driver)
      await chatPage.waitForChatPage()

      const chatName = `Nested Subthreads Test ${Date.now()}`
      await chatPage.createChat(chatName, 'group')
      await chatPage.waitForModalToClose()
      await chatPage.sleep(1000)

      // Select the chat
      await chatPage.selectFirstChat()
      await chatPage.sleep(1000)

      // Get chat ID
      const chatId = await chatPage.getCurrentChatId()

      // Create thread chain: thread -> subthread -> subsubthread...
      let parentId = await chatPage.createThreadViaApi(chatId, 'Level 0 Thread')
      console.log('Level 0 thread ID:', parentId)

      for (let level = 1; level <= 5; level++) {
        const subthreadId = await chatPage.createSubthreadViaApi(parentId, `Level ${level} Thread`)
        console.log(`Level ${level} thread ID:`, subthreadId)

        // Verify depth via listing subthreads
        const subthreads = await chatPage.listSubthreadsViaApi(parentId)
        const createdSubthread = subthreads.find(t => t.id === subthreadId)
        expect(createdSubthread).to.exist
        expect(createdSubthread!.depth).to.equal(level)

        parentId = subthreadId
      }

      console.log('Successfully created 5 levels of nested threads')
    })

    it('should not list subthreads in main threads list', async function () {
      // Create user and chat
      await createTestUser(driver)
      await chatPage.waitForChatPage()

      const chatName = `Subthread Isolation Test ${Date.now()}`
      await chatPage.createChat(chatName, 'group')
      await chatPage.waitForModalToClose()
      await chatPage.sleep(1000)

      // Select the chat
      await chatPage.selectFirstChat()
      await chatPage.sleep(1000)

      // Get chat ID
      const chatId = await chatPage.getCurrentChatId()

      // Create parent thread
      const parentTitle = `Parent Thread ${Date.now()}`
      const parentThreadId = await chatPage.createThreadViaApi(chatId, parentTitle)

      // Create subthread
      const subthreadTitle = `Hidden Subthread ${Date.now()}`
      await chatPage.createSubthreadViaApi(parentThreadId, subthreadTitle)

      // List main threads (should only show parent, not subthread)
      const threads = await chatPage.listThreadsViaApi(chatId)
      console.log('Main threads list:', threads)

      // Verify subthread is not in main list
      const subthreadInMainList = threads.find(t => t.title === subthreadTitle)
      expect(subthreadInMainList).to.be.undefined

      // But parent should be there
      const parentInList = threads.find(t => t.id === parentThreadId)
      expect(parentInList).to.exist
    })
  })

  describe('Cascading Permissions', function () {
    it('should inherit access from parent thread via API', async function () {
      // Create user and chat
      await createTestUser(driver)
      await chatPage.waitForChatPage()

      const chatName = `Permission Test Chat ${Date.now()}`
      await chatPage.createChat(chatName, 'group')
      await chatPage.waitForModalToClose()
      await chatPage.sleep(1000)

      // Select the chat
      await chatPage.selectFirstChat()
      await chatPage.sleep(1000)

      // Get chat ID
      const chatId = await chatPage.getCurrentChatId()

      // Create thread chain
      const parentId = await chatPage.createThreadViaApi(chatId, 'Parent Thread')
      const subthreadId = await chatPage.createSubthreadViaApi(parentId, 'Subthread')

      // User should have access to both (since they're a chat participant)
      const parentThreads = await chatPage.listThreadsViaApi(chatId)
      const subthreads = await chatPage.listSubthreadsViaApi(parentId)

      expect(parentThreads.find(t => t.id === parentId)).to.exist
      expect(subthreads.find(t => t.id === subthreadId)).to.exist

      console.log('User has access to parent and subthread as expected')
    })
  })

  describe('Thread Depth Validation', function () {
    it('should correctly report thread depth in API response', async function () {
      // Create user and chat
      await createTestUser(driver)
      await chatPage.waitForChatPage()

      const chatName = `Depth Validation Test ${Date.now()}`
      await chatPage.createChat(chatName, 'group')
      await chatPage.waitForModalToClose()
      await chatPage.sleep(1000)

      // Select the chat
      await chatPage.selectFirstChat()
      await chatPage.sleep(1000)

      // Get chat ID
      const chatId = await chatPage.getCurrentChatId()

      // Create threads at different depths
      const level0Id = await chatPage.createThreadViaApi(chatId, 'Level 0')
      const level1Id = await chatPage.createSubthreadViaApi(level0Id, 'Level 1')
      const level2Id = await chatPage.createSubthreadViaApi(level1Id, 'Level 2')

      // Verify depths
      const threads = await chatPage.listThreadsViaApi(chatId)
      const level0Thread = threads.find(t => t.id === level0Id)
      expect(level0Thread).to.exist
      // protobuf3 omits zero values, so depth may be undefined for level 0
      expect(level0Thread!.depth ?? 0).to.equal(0)

      const level1Subthreads = await chatPage.listSubthreadsViaApi(level0Id)
      const level1Thread = level1Subthreads.find(t => t.id === level1Id)
      expect(level1Thread).to.exist
      expect(level1Thread!.depth).to.equal(1)

      const level2Subthreads = await chatPage.listSubthreadsViaApi(level1Id)
      const level2Thread = level2Subthreads.find(t => t.id === level2Id)
      expect(level2Thread).to.exist
      expect(level2Thread!.depth).to.equal(2)

      console.log('All depth values are correct')
    })
  })
})
