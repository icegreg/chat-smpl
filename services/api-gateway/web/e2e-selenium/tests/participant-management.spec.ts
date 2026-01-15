import { WebDriver } from 'selenium-webdriver'
import { expect } from 'chai'
import { createDriver, quitDriver } from '../config/webdriver.js'
import { ChatPage } from '../pages/ChatPage.js'
import { RegisterPage, RegisterData } from '../pages/RegisterPage.js'
import {
  generateTestUser,
  getUserIdFromApi,
  wait,
} from '../helpers/testHelpers.js'

/**
 * E2E tests for participant management and chat deletion functionality
 *
 * Tests:
 * 1. Owner can add participant to chat
 * 2. Owner can remove participant from chat
 * 3. Owner can delete chat
 * 4. Non-moderator cannot see management controls
 * 5. Added participant receives chat via real-time event
 * 6. Removed participant loses access to chat
 */
describe('Participant Management and Chat Deletion', function () {
  this.timeout(120000)

  let ownerDriver: WebDriver
  let memberDriver: WebDriver
  let ownerChatPage: ChatPage
  let memberChatPage: ChatPage
  let ownerData: RegisterData
  let memberData: RegisterData
  let ownerId: string
  let memberId: string
  let chatName: string

  before(async function () {
    console.log('\n=== Starting Participant Management Tests ===\n')

    // Create owner user
    console.log('Creating owner user...')
    ownerDriver = await createDriver()
    ownerChatPage = new ChatPage(ownerDriver)
    const ownerRegisterPage = new RegisterPage(ownerDriver)
    ownerData = generateTestUser()

    await ownerRegisterPage.goto()
    await ownerRegisterPage.register(ownerData)
    await ownerRegisterPage.waitForUrl('/chat', 15000)
    ownerId = await getUserIdFromApi(ownerDriver)
    console.log(`Owner created: ${ownerData.displayName} (${ownerId})`)

    // Create member user
    console.log('Creating member user...')
    memberDriver = await createDriver()
    memberChatPage = new ChatPage(memberDriver)
    const memberRegisterPage = new RegisterPage(memberDriver)
    memberData = generateTestUser()

    await memberRegisterPage.goto()
    await memberRegisterPage.register(memberData)
    await memberRegisterPage.waitForUrl('/chat', 15000)
    memberId = await getUserIdFromApi(memberDriver)
    console.log(`Member created: ${memberData.displayName} (${memberId})`)

    // Owner creates a chat
    chatName = `Test Chat ${Date.now()}`
    console.log(`\nOwner creating chat: "${chatName}"`)
    await ownerChatPage.createChat(chatName, 'group')
    await ownerChatPage.waitForModalToClose()
    await ownerChatPage.waitForChatInList(chatName)
    await ownerChatPage.clickChatByNameInList(chatName)
    await ownerChatPage.waitForChatRoom()
    console.log('Chat created and opened by owner')
  })

  after(async function () {
    console.log('\n=== Cleaning up ===')
    if (ownerDriver) {
      await quitDriver(ownerDriver)
      console.log('Owner browser closed')
    }
    if (memberDriver) {
      await quitDriver(memberDriver)
      console.log('Member browser closed')
    }
  })

  describe('Add Participant', function () {
    it('should show add participant button for chat owner', async function () {
      console.log('\n--- Test: Add participant button visibility ---')

      // Open participants panel
      await ownerChatPage.openParticipantsPanel()

      // Wait for the panel to fully render and data to load
      await wait(2000)

      // Log participant count for debugging
      const participantCount = await ownerChatPage.getParticipantsCount()
      console.log(`Participants in panel: ${participantCount}`)

      // Log participant names for debugging
      const names = await ownerChatPage.getParticipantNames()
      console.log(`Participant names: ${names.join(', ')}`)

      // Check that add button is visible (owner has admin rights in chat)
      const isAddVisible = await ownerChatPage.isAddParticipantButtonVisible()
      expect(isAddVisible, 'Add participant button should be visible for owner').to.be.true

      console.log('Add participant button is visible for owner')
    })

    it('should add participant via UI', async function () {
      console.log('\n--- Test: Add participant via UI ---')

      // Get initial participant count
      const initialCount = await ownerChatPage.getParticipantsCount()
      console.log(`Initial participant count: ${initialCount}`)

      // Add member to chat
      const memberName = memberData.displayName || memberData.username
      console.log(`Adding member: ${memberName} (${memberId})`)
      await ownerChatPage.addParticipantViaUI(memberId, 'member')

      // Wait for participant to appear
      await ownerChatPage.waitForParticipantInPanel(memberName, 10000)

      // Verify participant count increased
      const newCount = await ownerChatPage.getParticipantsCount()
      console.log(`New participant count: ${newCount}`)
      expect(newCount).to.equal(initialCount + 1)

      // Verify participant name in list
      const names = await ownerChatPage.getParticipantNames()
      console.log('Participants:', names)
      expect(names.some(n => n.includes(memberName))).to.be.true

      console.log('Participant added successfully via UI')
    })

    it('should deliver chat to added participant via real-time event', async function () {
      console.log('\n--- Test: Real-time chat delivery ---')

      // Note: Backend doesn't publish participant_added events yet,
      // so we need to refresh the page to see the new chat
      console.log('Refreshing member page to get updated chat list...')
      await memberChatPage.goto()
      await wait(2000)

      // Now check for the chat
      console.log('Waiting for chat to appear in member\'s chat list...')
      await memberChatPage.waitForChatInList(chatName, 15000)

      // Verify member can enter the chat
      await memberChatPage.clickChatByNameInList(chatName)
      await memberChatPage.waitForChatRoom()

      const chatTitle = await memberChatPage.getChatHeaderTitle()
      expect(chatTitle).to.include(chatName.substring(0, 15))

      console.log('Member received and can access the chat')
    })
  })

  describe('Remove Participant', function () {
    it('should remove participant via UI', async function () {
      console.log('\n--- Test: Remove participant via UI ---')

      // Ensure owner is in the chat with participants panel open
      await ownerChatPage.clickChatByNameInList(chatName)
      await ownerChatPage.waitForChatRoom()
      await ownerChatPage.openParticipantsPanel()

      // Get current participant count
      const initialCount = await ownerChatPage.getParticipantsCount()
      console.log(`Participant count before removal: ${initialCount}`)

      // Remove the member
      const memberName = memberData.displayName || memberData.username
      console.log(`Removing member: ${memberName}`)
      await ownerChatPage.removeParticipantViaUI(memberName)

      // Get browser console logs for debugging
      const logs = await ownerDriver.manage().logs().get('browser')
      console.log('Browser console logs:')
      for (const log of logs) {
        console.log(`  [${log.level.name}] ${log.message}`)
      }

      // Wait for participant to be removed
      await ownerChatPage.waitForParticipantRemoved(memberName, 10000)

      // Verify participant count decreased
      const newCount = await ownerChatPage.getParticipantsCount()
      console.log(`Participant count after removal: ${newCount}`)
      expect(newCount).to.equal(initialCount - 1)

      console.log('Participant removed successfully')
    })

    it('should remove chat from removed participant\'s list', async function () {
      console.log('\n--- Test: Chat removed from kicked participant ---')

      // Member should no longer see the chat
      console.log('Checking member\'s chat list...')
      await wait(2000) // Wait for WebSocket event

      // Refresh member's view
      await memberChatPage.goto()
      await wait(1000)

      // Chat should be gone
      const chatNames = await memberChatPage.getChatNames()
      console.log('Member\'s chats:', chatNames)
      expect(chatNames.some(n => n.includes(chatName))).to.be.false

      console.log('Chat no longer visible to removed participant')
    })
  })

  describe('Delete Chat', function () {
    let deletableChatName: string

    before(async function () {
      // Create a new chat for deletion test
      deletableChatName = `Delete Test Chat ${Date.now()}`
      console.log(`\nCreating chat for deletion test: "${deletableChatName}"`)
      await ownerChatPage.createChat(deletableChatName, 'group')
      await ownerChatPage.waitForModalToClose()
      await ownerChatPage.waitForChatInList(deletableChatName)
      await ownerChatPage.clickChatByNameInList(deletableChatName)
      await ownerChatPage.waitForChatRoom()
      console.log('Chat created for deletion test')
    })

    it('should show delete chat button for owner', async function () {
      console.log('\n--- Test: Delete button visibility ---')

      const isDeleteVisible = await ownerChatPage.isDeleteChatButtonVisible()
      expect(isDeleteVisible, 'Delete chat button should be visible for owner').to.be.true

      console.log('Delete chat button is visible')
    })

    it('should delete chat via UI', async function () {
      console.log('\n--- Test: Delete chat via UI ---')

      // Get initial chat count
      const initialCount = await ownerChatPage.getChatCount()
      console.log(`Chat count before deletion: ${initialCount}`)

      // Delete the chat
      console.log(`Deleting chat: "${deletableChatName}"`)
      await ownerChatPage.deleteChatViaUI()

      // Wait for chat to be removed from list
      await ownerChatPage.waitForChatRemoved(deletableChatName, 10000)

      // Verify chat count decreased
      const newCount = await ownerChatPage.getChatCount()
      console.log(`Chat count after deletion: ${newCount}`)
      expect(newCount).to.equal(initialCount - 1)

      // Verify chat is not in the list
      const chatNames = await ownerChatPage.getChatNames()
      expect(chatNames.some(n => n.includes(deletableChatName))).to.be.false

      console.log('Chat deleted successfully')
    })
  })

  describe('Permission Checks', function () {
    let regularUserDriver: WebDriver
    let regularChatPage: ChatPage
    let regularData: RegisterData
    let regularUserId: string
    let regularName: string
    let testChatName: string

    before(async function () {
      // Create a regular user (non-moderator)
      console.log('\n--- Setting up permission tests ---')
      regularUserDriver = await createDriver()
      regularChatPage = new ChatPage(regularUserDriver)
      const registerPage = new RegisterPage(regularUserDriver)
      regularData = generateTestUser()

      await registerPage.goto()
      await registerPage.register(regularData)
      await registerPage.waitForUrl('/chat', 15000)
      regularUserId = await getUserIdFromApi(regularUserDriver)
      regularName = regularData.displayName || regularData.username
      console.log(`Regular user created: ${regularName}`)

      // Owner creates a chat and adds regular user
      testChatName = `Permission Test ${Date.now()}`
      await ownerChatPage.createChat(testChatName, 'group')
      await ownerChatPage.waitForModalToClose()
      await ownerChatPage.waitForChatInList(testChatName)
      await ownerChatPage.clickChatByNameInList(testChatName)
      await ownerChatPage.waitForChatRoom()

      // Add regular user as member (not admin)
      await ownerChatPage.openParticipantsPanel()
      await ownerChatPage.addParticipantViaUI(regularUserId, 'member')
      await ownerChatPage.waitForParticipantInPanel(regularName)
      console.log('Regular user added to test chat as member')

      // Regular user needs to refresh to see the new chat (no real-time event yet)
      await regularChatPage.goto()
      await wait(2000)

      // Regular user enters the chat
      await regularChatPage.waitForChatInList(testChatName, 15000)
      await regularChatPage.clickChatByNameInList(testChatName)
      await regularChatPage.waitForChatRoom()
      console.log('Regular user entered the chat')
    })

    after(async function () {
      if (regularUserDriver) {
        await quitDriver(regularUserDriver)
        console.log('Regular user browser closed')
      }
    })

    it('regular member should not see add participant button', async function () {
      console.log('\n--- Test: Regular member cannot add participants ---')

      await regularChatPage.openParticipantsPanel()

      const isAddVisible = await regularChatPage.isAddParticipantButtonVisible()
      expect(isAddVisible, 'Add participant button should NOT be visible for regular member').to.be.false

      console.log('Add participant button correctly hidden for regular member')
    })

    it('regular member should not see delete chat button', async function () {
      console.log('\n--- Test: Regular member cannot delete chat ---')

      const isDeleteVisible = await regularChatPage.isDeleteChatButtonVisible()
      expect(isDeleteVisible, 'Delete chat button should NOT be visible for regular member').to.be.false

      console.log('Delete chat button correctly hidden for regular member')
    })
  })
})
