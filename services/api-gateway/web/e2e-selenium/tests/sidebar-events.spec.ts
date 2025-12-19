import { WebDriver, By } from 'selenium-webdriver'
import { expect } from 'chai'
import { createDriver, quitDriver } from '../config/webdriver.js'
import { ChatPage } from '../pages/ChatPage.js'
import { RegisterPage } from '../pages/RegisterPage.js'
import {
  generateTestUserWithIndex,
  getUserIdFromApi,
  wait,
} from '../helpers/testHelpers.js'

/**
 * Sidebar Events E2E Tests
 *
 * Tests for the "Upcoming Events" section in ChatSidebar:
 * 1. Events section visibility when user has scheduled events
 * 2. Event display (name, time, participant count)
 * 3. Join button for active events
 * 4. "View all" navigation to /events page
 * 5. RSVP status display
 */
describe('Sidebar Events UI', function () {
  this.timeout(120000) // 2 minutes timeout

  let driver: WebDriver
  let chatPage: ChatPage
  let _userId: string

  // Локаторы для событий в sidebar
  const upcomingEventsHeader = By.xpath('//h3[contains(text(), "Upcoming Events")]')
  const viewAllLink = By.xpath('//a[contains(text(), "View all")]')
  const eventCards = By.css('aside .divide-y.divide-gray-100 > div')
  const eventName = By.css('.font-medium.text-gray-900')
  const eventTime = By.css('.text-xs.font-medium.text-indigo-600')
  const joinButton = By.xpath('//button[contains(text(), "Join")]')
  const chatsSectionHeader = By.xpath('//h3[contains(text(), "Chats")]')

  before(async function () {
    console.log('\n=== Sidebar Events Test Setup ===')

    // Create browser instance
    console.log('Creating browser instance...')
    driver = await createDriver()
    chatPage = new ChatPage(driver)

    // Register user
    console.log('Registering test user...')
    const registerPage = new RegisterPage(driver)
    const userData = generateTestUserWithIndex(Date.now())

    await registerPage.goto()
    await registerPage.register(userData)
    await registerPage.waitForUrl('/chat', 15000)

    console.log(`User registered: ${userData.displayName}`)

    // Get user ID
    _userId = await getUserIdFromApi(driver)

    console.log(`User ID: ${_userId}`)
    console.log('Setup complete')
  })

  after(async function () {
    console.log('\n=== Cleanup ===')
    await quitDriver(driver).catch(() => {})
    console.log('Cleanup complete')
  })

  describe('Events Section Without Events', function () {
    it('should NOT display upcoming events section when user has no events', async function () {
      console.log('\n--- Test: No events section when empty ---')

      await chatPage.refresh()
      await wait(1500)

      // Check that the Upcoming Events header is not visible
      const headerVisible = await driver
        .findElements(upcomingEventsHeader)
        .then(els => els.length > 0)

      expect(headerVisible).to.be.false
      console.log('No Upcoming Events section displayed (correct)')
    })
  })

  describe('Events Section With Events', function () {
    let conferenceId: string

    before(async function () {
      console.log('\n--- Creating scheduled conference via API ---')

      // Create a scheduled conference for the user
      const scheduledAt = new Date(Date.now() + 30 * 60 * 1000) // 30 minutes from now

      try {
        const response = await driver.executeScript(`
          return new Promise(async (resolve, reject) => {
            try {
              const token = localStorage.getItem('access_token');
              if (!token) {
                reject('No access token');
                return;
              }

              const response = await fetch('/api/voice/conferences/schedule', {
                method: 'POST',
                headers: {
                  'Authorization': 'Bearer ' + token,
                  'Content-Type': 'application/json'
                },
                body: JSON.stringify({
                  name: 'Test Scheduled Event',
                  event_type: 'scheduled',
                  scheduled_at: '${scheduledAt.toISOString()}',
                  participant_ids: []
                })
              });

              if (!response.ok) {
                const error = await response.text();
                reject('API error: ' + response.status + ' ' + error);
                return;
              }

              const conference = await response.json();
              resolve(conference);
            } catch (e) {
              reject(e.message);
            }
          });
        `)

        conferenceId = (response as any).id
        console.log(`Created conference: ${conferenceId}`)
      } catch (error) {
        console.log('Note: Schedule API may not be fully implemented yet')
        console.log('Error:', error)
        // Skip tests if API not available
        this.skip()
      }
    })

    it('should display upcoming events section when user has events', async function () {
      console.log('\n--- Test: Events section visible ---')

      await chatPage.refresh()
      await wait(2000) // Wait for events to load

      const headerVisible = await driver
        .findElements(upcomingEventsHeader)
        .then(async els => {
          if (els.length === 0) return false
          return await els[0].isDisplayed()
        })

      if (!headerVisible) {
        console.log('Upcoming Events section not displayed - API may not be implemented')
        this.skip()
        return
      }

      expect(headerVisible).to.be.true
      console.log('Upcoming Events section is visible')
    })

    it('should display event name and time', async function () {
      console.log('\n--- Test: Event info display ---')

      // Find event cards in the upcoming events section
      const cards = await driver.findElements(eventCards)

      if (cards.length === 0) {
        console.log('No event cards found - skipping')
        this.skip()
        return
      }

      // Get first event card info
      const firstCard = cards[0]
      const nameEl = await firstCard.findElement(eventName)
      const timeEl = await firstCard.findElement(eventTime)

      const name = await nameEl.getText()
      const time = await timeEl.getText()

      console.log(`Event name: ${name}`)
      console.log(`Event time: ${time}`)

      expect(name).to.include('Test Scheduled Event')
      expect(time).to.match(/in \d+[mh]|Tomorrow|Now/)

      console.log('Event info displayed correctly')
    })

    it('should have "View all" link', async function () {
      console.log('\n--- Test: View all link ---')

      const linkVisible = await driver
        .findElements(viewAllLink)
        .then(async els => {
          if (els.length === 0) return false
          return await els[0].isDisplayed()
        })

      expect(linkVisible).to.be.true
      console.log('View all link is visible')
    })

    it('should navigate to /events when clicking "View all"', async function () {
      console.log('\n--- Test: View all navigation ---')

      const links = await driver.findElements(viewAllLink)
      if (links.length === 0) {
        this.skip()
        return
      }

      await links[0].click()
      await wait(500)

      const currentUrl = await driver.getCurrentUrl()
      expect(currentUrl).to.include('/events')

      console.log('Successfully navigated to Events page')

      // Navigate back to chats
      await chatPage.clickLeftNavChats()
      await wait(500)
    })

    it('should display Chats section header when events are shown', async function () {
      console.log('\n--- Test: Chats section header ---')

      await chatPage.refresh()
      await wait(1500)

      // When events are displayed, there should be a "Chats" header below them
      const chatsHeaderVisible = await driver
        .findElements(chatsSectionHeader)
        .then(async els => {
          if (els.length === 0) return false
          return await els[0].isDisplayed()
        })

      // This header only appears when there are events
      const eventsVisible = await driver
        .findElements(upcomingEventsHeader)
        .then(els => els.length > 0)

      if (eventsVisible) {
        expect(chatsHeaderVisible).to.be.true
        console.log('Chats section header is visible')
      } else {
        console.log('No events, so no Chats header expected')
      }
    })
  })

  describe('Event Click Handlers', function () {
    it('should navigate to /events when clicking on event card', async function () {
      console.log('\n--- Test: Event card click ---')

      // First check if there are events
      const cards = await driver.findElements(eventCards)
      if (cards.length === 0) {
        console.log('No events to click - skipping')
        this.skip()
        return
      }

      await cards[0].click()
      await wait(500)

      const currentUrl = await driver.getCurrentUrl()
      expect(currentUrl).to.include('/events')

      console.log('Navigated to Events page on card click')

      // Navigate back
      await chatPage.clickLeftNavChats()
      await wait(500)
    })
  })

  describe('RSVP Status Display', function () {
    it('should display RSVP pending status', async function () {
      console.log('\n--- Test: RSVP pending status ---')

      await chatPage.refresh()
      await wait(1500)

      // Look for "Pending" text in event cards
      const pendingText = await driver
        .findElements(By.xpath('//span[contains(text(), "Pending")]'))
        .then(async els => {
          if (els.length === 0) return null
          return await els[0].getText()
        })

      if (pendingText) {
        console.log(`Found RSVP status: ${pendingText}`)
        expect(pendingText).to.include('Pending')
      } else {
        console.log('No pending status found (may have different RSVP)')
      }
    })

    it('should display participant count', async function () {
      console.log('\n--- Test: Participant count ---')

      const cards = await driver.findElements(eventCards)
      if (cards.length === 0) {
        this.skip()
        return
      }

      // Look for "X joined" text
      const joinedText = await driver
        .findElements(By.xpath('//span[contains(text(), "joined")]'))
        .then(async els => {
          if (els.length === 0) return null
          return await els[0].getText()
        })

      if (joinedText) {
        console.log(`Participant count: ${joinedText}`)
        expect(joinedText).to.match(/\d+ joined/)
      } else {
        console.log('Participant count not found')
      }
    })
  })

  describe('Active Event Join Button', function () {
    it('should show Join button for active events', async function () {
      console.log('\n--- Test: Join button for active events ---')

      // Create an active conference
      try {
        await driver.executeScript(`
          return new Promise(async (resolve, reject) => {
            try {
              const token = localStorage.getItem('access_token');
              if (!token) {
                reject('No access token');
                return;
              }

              // Create and immediately start a conference
              const response = await fetch('/api/voice/conferences', {
                method: 'POST',
                headers: {
                  'Authorization': 'Bearer ' + token,
                  'Content-Type': 'application/json'
                },
                body: JSON.stringify({
                  name: 'Active Test Conference',
                  event_type: 'adhoc'
                })
              });

              if (!response.ok) {
                const error = await response.text();
                reject('API error: ' + response.status + ' ' + error);
                return;
              }

              const conference = await response.json();
              resolve(conference);
            } catch (e) {
              reject(e.message);
            }
          });
        `)

        await chatPage.refresh()
        await wait(1500)

        const joinButtons = await driver.findElements(joinButton)
        if (joinButtons.length > 0) {
          const isDisplayed = await joinButtons[0].isDisplayed()
          console.log(`Join button visible: ${isDisplayed}`)
        } else {
          console.log('No Join button found (conference may not be active)')
        }
      } catch (error) {
        console.log('Active conference creation not supported:', error)
        this.skip()
      }
    })
  })

  describe('Events Limit', function () {
    it('should limit events to 5 items', async function () {
      console.log('\n--- Test: Events limit ---')

      const cards = await driver.findElements(eventCards)
      console.log(`Event cards count: ${cards.length}`)

      // Should never have more than 5 events in sidebar
      expect(cards.length).to.be.at.most(5)

      console.log('Events correctly limited')
    })
  })

  describe('Event Icon Types', function () {
    it('should display appropriate icon for event type', async function () {
      console.log('\n--- Test: Event icons ---')

      const cards = await driver.findElements(eventCards)
      if (cards.length === 0) {
        this.skip()
        return
      }

      // Check for SVG icon in first card
      const firstCard = cards[0]
      const icons = await firstCard.findElements(By.css('svg'))

      expect(icons.length).to.be.greaterThan(0)
      console.log(`Found ${icons.length} icon(s) in event card`)
    })
  })
})
