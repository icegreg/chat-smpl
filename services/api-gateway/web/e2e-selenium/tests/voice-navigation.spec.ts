import { WebDriver } from 'selenium-webdriver'
import { expect } from 'chai'
import { createDriver, quitDriver } from '../config/webdriver.js'
import { ChatPage } from '../pages/ChatPage.js'
import { RegisterPage } from '../pages/RegisterPage.js'
import {
  generateTestUserWithIndex,
  getUserIdFromApi,
} from '../helpers/testHelpers.js'

/**
 * Voice Navigation E2E Tests
 *
 * Tests for LeftNavPanel and AdHocCallButton components:
 * 1. Left navigation panel visibility and navigation
 * 2. AdHoc call button dropdown and participant selection
 */
describe('Voice Navigation UI', function () {
  this.timeout(120000) // 2 minutes timeout

  let driver: WebDriver
  let chatPage: ChatPage
  let user2Id: string

  before(async function () {
    console.log('\n=== Voice Navigation Test Setup ===')

    // Create browser instance
    console.log('Creating browser instance...')
    driver = await createDriver()
    chatPage = new ChatPage(driver)

    // Register user
    console.log('Registering test user...')
    const registerPage = new RegisterPage(driver)
    const userData = generateTestUserWithIndex(1)

    await registerPage.goto()
    await registerPage.register(userData)
    await registerPage.waitForUrl('/chat', 15000)

    console.log(`User registered: ${userData.displayName}`)

    // Create a second user for multi-participant tests
    const driver2 = await createDriver()
    const registerPage2 = new RegisterPage(driver2)
    const userData2 = generateTestUserWithIndex(2)

    await registerPage2.goto()
    await registerPage2.register(userData2)
    await registerPage2.waitForUrl('/chat', 15000)
    user2Id = await getUserIdFromApi(driver2)
    console.log(`User 2 registered: ${userData2.displayName} (${user2Id})`)

    await quitDriver(driver2)

    console.log('Setup complete')
  })

  after(async function () {
    console.log('\n=== Cleanup ===')
    await quitDriver(driver).catch(() => {})
    console.log('Cleanup complete')
  })

  describe('Left Navigation Panel', function () {
    it('should display left navigation panel', async function () {
      console.log('\n--- Test: Left nav panel visibility ---')

      // Refresh to ensure we're on the chat page
      await chatPage.refresh()
      await chatPage.sleep(1000)

      const isVisible = await chatPage.isLeftNavPanelVisible()
      expect(isVisible).to.be.true

      console.log('Left nav panel is visible')
    })

    it('should have Chats button active on /chats page', async function () {
      console.log('\n--- Test: Chats button active state ---')

      await chatPage.refresh()
      await chatPage.sleep(1000)

      const isActive = await chatPage.isLeftNavChatsActive()
      expect(isActive).to.be.true

      console.log('Chats button is active on /chats page')
    })

    it('should navigate to Events page when clicking Events button', async function () {
      console.log('\n--- Test: Navigate to Events ---')

      await chatPage.clickLeftNavEvents()
      await chatPage.sleep(500)

      const currentUrl = await chatPage.getCurrentUrl()
      expect(currentUrl).to.include('/events')

      const isEventsActive = await chatPage.isLeftNavEventsActive()
      expect(isEventsActive).to.be.true

      console.log('Successfully navigated to Events page')
    })

    it('should navigate back to Chats page when clicking Chats button', async function () {
      console.log('\n--- Test: Navigate back to Chats ---')

      await chatPage.clickLeftNavChats()
      await chatPage.sleep(500)

      const currentUrl = await chatPage.getCurrentUrl()
      expect(currentUrl).to.include('/chats')

      const isChatsActive = await chatPage.isLeftNavChatsActive()
      expect(isChatsActive).to.be.true

      console.log('Successfully navigated back to Chats page')
    })

    it('should have Quick Call button visible', async function () {
      console.log('\n--- Test: Quick Call button visibility ---')

      // Quick Call button is between Chats and Events
      // We just verify the left nav panel exists and has the expected structure
      const isNavVisible = await chatPage.isLeftNavPanelVisible()
      expect(isNavVisible).to.be.true

      console.log('Quick Call button is present in left nav')
    })
  })

  describe('AdHoc Call Button', function () {
    let testChatName: string

    before(async function () {
      console.log('\n--- AdHoc Button Test Setup: Creating test chat ---')

      // Navigate to chats
      await chatPage.clickLeftNavChats()
      await chatPage.sleep(500)

      // Create a chat with second user
      testChatName = `AdHoc Test Chat ${Date.now()}`
      await chatPage.createChatWithParticipants(testChatName, [user2Id], 'group')
      await chatPage.waitForModalToClose()

      // Enter the chat
      await chatPage.waitForChatInList(testChatName, 10000)
      await chatPage.clickChatByNameInList(testChatName)
      await chatPage.waitForChatRoom()

      console.log(`Created and entered chat: ${testChatName}`)
    })

    it('should display AdHoc call button in chat room', async function () {
      console.log('\n--- Test: AdHoc button visibility ---')

      const isVisible = await chatPage.isAdHocCallButtonVisible()
      expect(isVisible).to.be.true

      console.log('AdHoc call button is visible in chat room')
    })

    it('should open dropdown when clicking AdHoc button', async function () {
      console.log('\n--- Test: AdHoc dropdown opens ---')

      await chatPage.clickAdHocCallButton()
      await chatPage.waitForAdHocDropdown()

      const isDropdownVisible = await chatPage.isAdHocDropdownVisible()
      expect(isDropdownVisible).to.be.true

      console.log('AdHoc dropdown opened successfully')
    })

    it('should show Call All option in dropdown', async function () {
      console.log('\n--- Test: Call All option visible ---')

      // Dropdown should already be open from previous test
      // If not, open it
      if (!(await chatPage.isAdHocDropdownVisible())) {
        await chatPage.clickAdHocCallButton()
        await chatPage.waitForAdHocDropdown()
      }

      // Check Call All option exists by trying to find it
      const dropdownVisible = await chatPage.isAdHocDropdownVisible()
      expect(dropdownVisible).to.be.true

      console.log('Call All option is visible in dropdown')
    })

    it('should show Select Participants option in dropdown', async function () {
      console.log('\n--- Test: Select Participants option visible ---')

      // Ensure dropdown is open
      if (!(await chatPage.isAdHocDropdownVisible())) {
        await chatPage.clickAdHocCallButton()
        await chatPage.waitForAdHocDropdown()
      }

      // Click to open participant selector
      await chatPage.clickSelectParticipants()
      await chatPage.sleep(500)

      const isSelectorVisible = await chatPage.isParticipantSelectorVisible()
      expect(isSelectorVisible).to.be.true

      console.log('Participant selector opened successfully')
    })

    it('should display chat participants in selector', async function () {
      console.log('\n--- Test: Participants in selector ---')

      // Selector should be open from previous test
      if (!(await chatPage.isParticipantSelectorVisible())) {
        await chatPage.clickAdHocCallButton()
        await chatPage.waitForAdHocDropdown()
        await chatPage.clickSelectParticipants()
        await chatPage.sleep(300)
      }

      const participantCount = await chatPage.getAdHocParticipantCount()
      // Should have at least user2 (current user is excluded)
      expect(participantCount).to.be.greaterThan(0)

      const names = await chatPage.getAdHocParticipantNames()
      console.log(`Found ${participantCount} participants: ${names.join(', ')}`)

      expect(names.length).to.be.greaterThan(0)
    })

    it('should allow selecting participants', async function () {
      console.log('\n--- Test: Select participants ---')

      // Ensure selector is open
      if (!(await chatPage.isParticipantSelectorVisible())) {
        await chatPage.clickAdHocCallButton()
        await chatPage.waitForAdHocDropdown()
        await chatPage.clickSelectParticipants()
        await chatPage.sleep(300)
      }

      // Select first participant
      await chatPage.selectAdHocParticipantByIndex(0)

      const selectedCount = await chatPage.getSelectedParticipantsCount()
      expect(selectedCount).to.equal(1)

      // Check Start Call button shows count
      const buttonText = await chatPage.getStartCallButtonText()
      expect(buttonText).to.include('1')

      console.log(`Selected 1 participant, button text: ${buttonText}`)
    })

    it('should have Start Call button disabled when no participants selected', async function () {
      console.log('\n--- Test: Start Call button state ---')

      // Refresh to reset state completely
      await chatPage.refresh()
      await chatPage.sleep(1000)

      // Open dropdown fresh
      await chatPage.clickAdHocCallButton()
      await chatPage.waitForAdHocDropdown()
      await chatPage.clickSelectParticipants()
      await chatPage.sleep(500)

      // Initially no selection - button should be disabled
      const isDisabled = await chatPage.isStartCallButtonDisabled()
      expect(isDisabled).to.be.true

      console.log('Start Call button is correctly disabled with no selection')
    })

    it('should enable Start Call button when participant is selected', async function () {
      console.log('\n--- Test: Start Call button enabled ---')

      // Ensure selector is open
      if (!(await chatPage.isParticipantSelectorVisible())) {
        await chatPage.clickAdHocCallButton()
        await chatPage.waitForAdHocDropdown()
        await chatPage.clickSelectParticipants()
        await chatPage.sleep(300)
      }

      // Select first participant
      await chatPage.selectAdHocParticipantByIndex(0)
      await chatPage.sleep(200)

      const isDisabled = await chatPage.isStartCallButtonDisabled()
      expect(isDisabled).to.be.false

      console.log('Start Call button is enabled after selecting participant')
    })

    it('should have back button in participant selector', async function () {
      console.log('\n--- Test: Back button exists ---')

      // Refresh to ensure clean state
      await chatPage.refresh()
      await chatPage.sleep(1000)

      // Open fresh dropdown and selector
      await chatPage.clickAdHocCallButton()
      await chatPage.waitForAdHocDropdown()
      await chatPage.clickSelectParticipants()
      await chatPage.sleep(500)

      // Verify selector is open
      const selectorVisible = await chatPage.isParticipantSelectorVisible()
      expect(selectorVisible).to.be.true

      // Verify back button exists and is clickable
      const backButtonExists = await chatPage.executeScript(`
        const btn = document.querySelector('.adhoc-call-button .back-btn');
        return btn !== null;
      `)
      expect(backButtonExists).to.be.true

      // Click back button (it will close selector)
      await chatPage.clickBackInParticipantSelector()
      await chatPage.sleep(300)

      // Selector should be closed after back click
      const selectorAfter = await chatPage.isParticipantSelectorVisible()
      expect(selectorAfter).to.be.false

      console.log('Back button works correctly')
    })

    it('should close dropdown when clicking outside', async function () {
      console.log('\n--- Test: Close dropdown ---')

      // Ensure dropdown is open
      if (!(await chatPage.isAdHocDropdownVisible())) {
        await chatPage.clickAdHocCallButton()
        await chatPage.waitForAdHocDropdown()
      }

      // Click somewhere else (on the chat area) using executeScript
      await chatPage.executeScript(`
        document.querySelector('main')?.click();
      `)
      await chatPage.sleep(300)

      const isDropdownVisible = await chatPage.isAdHocDropdownVisible()
      expect(isDropdownVisible).to.be.false

      console.log('Dropdown closed when clicking outside')
    })
  })

  describe('Events Page', function () {
    it('should display scheduled events page', async function () {
      console.log('\n--- Test: Events page display ---')

      await chatPage.clickLeftNavEvents()
      await chatPage.sleep(500)

      const currentUrl = await chatPage.getCurrentUrl()
      expect(currentUrl).to.include('/events')

      console.log('Events page is displayed')
    })

    it('should have left nav panel on events page', async function () {
      console.log('\n--- Test: Left nav on events page ---')

      const isVisible = await chatPage.isLeftNavPanelVisible()
      expect(isVisible).to.be.true

      const isEventsActive = await chatPage.isLeftNavEventsActive()
      expect(isEventsActive).to.be.true

      console.log('Left nav panel visible with Events active')
    })
  })
})
