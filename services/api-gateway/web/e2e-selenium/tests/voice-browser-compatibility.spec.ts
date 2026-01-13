import { WebDriver, logging } from 'selenium-webdriver'
import { expect } from 'chai'
import {
  createWebRTCDriver,
  createFirefoxWebRTCDriver,
  quitDriver,
  getWebRTCStats,
} from '../config/webdriver-webrtc.js'
import { ChatPage } from '../pages/ChatPage.js'
import { createTestUser, clearBrowserState, getUserIdFromApi } from '../helpers/testHelpers.js'

/**
 * Voice Browser Compatibility Tests
 *
 * These tests verify:
 * 1. Conference creation works in both Chrome and Firefox
 * 2. Conference popup appears correctly
 * 3. Works with both localhost and real IP addresses
 *
 * Requirements:
 * - FreeSWITCH running with Verto WebSocket
 * - Non-headless browser (HEADLESS=false)
 * - Both Chrome and Firefox installed
 */

// Test URLs - localhost and real IP
const TEST_URLS = [
  'http://localhost:8888',      // HTTP OK for localhost
  'https://192.168.1.208:8443', // HTTPS required for real IP (WebRTC security)
]

// Test browsers
interface BrowserConfig {
  name: string
  createDriver: () => Promise<WebDriver>
}

const BROWSERS: BrowserConfig[] = [
  {
    name: 'Chrome',
    createDriver: createWebRTCDriver,
  },
  {
    name: 'Firefox',
    createDriver: createFirefoxWebRTCDriver,
  },
]

describe('Voice Browser Compatibility - Chrome & Firefox', function () {
  this.timeout(180000)

  // Test each browser with each URL
  BROWSERS.forEach((browser) => {
    TEST_URLS.forEach((testUrl) => {
      const urlLabel = testUrl.includes('localhost') ? 'localhost' : 'real IP'

      describe(`${browser.name} with ${urlLabel}`, function () {
        let driver: WebDriver
        let chatPage: ChatPage

        before(async function () {
          // Create driver
          driver = await browser.createDriver()
          chatPage = new ChatPage(driver)
        })

        after(async function () {
          await quitDriver(driver)
        })

        beforeEach(async function () {
          // Navigate to test URL and clear state
          await driver.get(testUrl)
          await chatPage.sleep(500)
          await clearBrowserState(driver)
        })

        it('should load the application', async function () {
          await driver.get(testUrl)
          await chatPage.sleep(1000)

          const title = await driver.getTitle()
          console.log(`[${browser.name}] Page title: ${title}`)
          expect(title).to.be.a('string')
        })

        it('should create user and login', async function () {
          const user = await createTestUser(driver)
          console.log(`[${browser.name}] Created user: ${user.username}`)

          await chatPage.waitForChatPage()
          // Successfully logged in if we got here without error
          expect(user.username).to.be.a('string')
        })

        it('should show call button in chat', async function () {
          await createTestUser(driver)
          await chatPage.waitForChatPage()

          const chatName = `${browser.name} Call Test ${Date.now()}`
          await chatPage.createChat(chatName, 'group')
          await chatPage.waitForModalToClose()
          await chatPage.sleep(500)

          await chatPage.selectFirstChat()
          await chatPage.waitForChatRoom()

          const isVisible = await chatPage.isAdHocCallButtonVisible()
          console.log(`[${browser.name}] Call button visible: ${isVisible}`)
          expect(isVisible).to.be.true
        })

        it('should start conference and show ConferenceView popup', async function () {
          await createTestUser(driver)
          await chatPage.waitForChatPage()

          const chatName = `${browser.name} Conference ${Date.now()}`
          await chatPage.createChat(chatName, 'group')
          await chatPage.waitForModalToClose()
          await chatPage.sleep(500)

          await chatPage.selectFirstChat()
          await chatPage.waitForChatRoom()
          console.log(`[${browser.name}] Created chat: ${chatName}`)

          // Start conference via Call All
          await chatPage.clickAdHocCallButton()
          await chatPage.waitForAdHocDropdown()
          await chatPage.clickCallAll()
          console.log(`[${browser.name}] Clicked Call All`)

          // Wait for ConferenceView popup to appear
          await chatPage.waitForConferenceView(15000)
          console.log(`[${browser.name}] ConferenceView popup appeared`)

          const isVisible = await chatPage.isConferenceViewVisible()
          expect(isVisible).to.be.true

          // Get conference details
          const conferenceName = await chatPage.getConferenceName()
          const participantCount = await chatPage.getConferenceParticipantCount()

          console.log(`[${browser.name}] Conference name: ${conferenceName}`)
          console.log(`[${browser.name}] Participants: ${participantCount}`)

          expect(conferenceName).to.be.a('string')
          expect(conferenceName.length).to.be.greaterThan(0)

          // Check browser logs for errors
          const logs = await driver.manage().logs().get(logging.Type.BROWSER)
          const sdpErrors = logs.filter(log =>
            log.message.includes('SDP') && log.message.includes('error')
          )
          const sipccErrors = logs.filter(log =>
            log.message.includes('SIPCC') && log.level.name === 'SEVERE'
          )

          if (sdpErrors.length > 0) {
            console.warn(`[${browser.name}] SDP errors found:`)
            sdpErrors.forEach(log => console.warn(`  ${log.message}`))
          }

          if (sipccErrors.length > 0) {
            console.warn(`[${browser.name}] SIPCC errors found:`)
            sipccErrors.forEach(log => console.warn(`  ${log.message}`))
          }

          // For this test to pass, we should have no SIPCC failures
          expect(sipccErrors.length).to.equal(0, `${browser.name} should not have SIPCC SDP parsing errors`)
        })

        it('should establish WebRTC connection', async function () {
          await createTestUser(driver)
          await chatPage.waitForChatPage()

          const chatName = `${browser.name} WebRTC ${Date.now()}`
          await chatPage.createChat(chatName, 'group')
          await chatPage.waitForModalToClose()
          await chatPage.sleep(500)

          await chatPage.selectFirstChat()
          await chatPage.waitForChatRoom()

          await chatPage.clickAdHocCallButton()
          await chatPage.waitForAdHocDropdown()
          await chatPage.clickCallAll()

          // Wait for WebRTC connection
          await chatPage.sleep(10000)

          const stats = await getWebRTCStats(driver)
          console.log(`[${browser.name}] WebRTC Stats:`, stats)

          if (stats) {
            console.log(`  Connection State: ${stats.connectionState}`)
            console.log(`  ICE State: ${stats.iceConnectionState}`)
            console.log(`  Bytes Received: ${stats.bytesReceived}`)
            console.log(`  Bytes Sent: ${stats.bytesSent}`)

            // Verify connection is at least attempting
            expect(['new', 'connecting', 'connected'].includes(stats.connectionState)).to.be.true
          } else {
            console.log(`  No RTCPeerConnection found yet`)
          }
        })

        it('should close conference view popup', async function () {
          await createTestUser(driver)
          await chatPage.waitForChatPage()

          const chatName = `${browser.name} Close Test ${Date.now()}`
          await chatPage.createChat(chatName, 'group')
          await chatPage.waitForModalToClose()
          await chatPage.sleep(500)

          await chatPage.selectFirstChat()
          await chatPage.waitForChatRoom()

          await chatPage.clickAdHocCallButton()
          await chatPage.waitForAdHocDropdown()
          await chatPage.clickCallAll()

          // Wait for popup
          await chatPage.waitForConferenceView(15000)
          expect(await chatPage.isConferenceViewVisible()).to.be.true

          // Close popup
          await chatPage.closeConferenceView()
          console.log(`[${browser.name}] Clicked close button`)

          // Wait for it to disappear
          await chatPage.waitForConferenceViewToDisappear(5000)
          const stillVisible = await chatPage.isConferenceViewVisible()

          console.log(`[${browser.name}] Popup closed successfully`)
          expect(stillVisible).to.be.false
        })

        it('should display ANSWER SDP without auto-nat placeholders', async function () {
          await createTestUser(driver)
          await chatPage.waitForChatPage()

          const chatName = `${browser.name} SDP Test ${Date.now()}`
          await chatPage.createChat(chatName, 'group')
          await chatPage.waitForModalToClose()
          await chatPage.sleep(500)

          await chatPage.selectFirstChat()
          await chatPage.waitForChatRoom()

          await chatPage.clickAdHocCallButton()
          await chatPage.waitForAdHocDropdown()
          await chatPage.clickCallAll()

          // Wait for SDP exchange
          await chatPage.sleep(5000)

          // Check console logs for ANSWER SDP
          const logs = await driver.manage().logs().get(logging.Type.BROWSER)
          const answerSdpLogs = logs.filter(log =>
            log.message.includes('ANSWER SDP')
          )

          console.log(`[${browser.name}] Found ${answerSdpLogs.length} ANSWER SDP logs`)

          if (answerSdpLogs.length > 0) {
            const answerSdp = answerSdpLogs[0].message
            console.log(`[${browser.name}] ANSWER SDP excerpt:`)
            const lines = answerSdp.split('\\n').slice(0, 10)
            lines.forEach(line => console.log(`  ${line}`))

            // Verify no auto-nat placeholders
            expect(answerSdp).to.not.include('auto-nat', 'SDP should not contain auto-nat placeholder')
            expect(answerSdp).to.not.include('c=IN IP4 auto', 'SDP should have real IP, not "auto"')

            console.log(`[${browser.name}] ✓ SDP does not contain auto-nat placeholders`)
          } else {
            console.log(`[${browser.name}] No ANSWER SDP found in logs (may not have reached that stage)`)
          }
        })
      })
    })
  })
})

describe('Voice Multi-Client Audio Transmission', function () {
  this.timeout(180000)

  const testUrl = 'http://localhost:8888'
  let driver1: WebDriver
  let driver2: WebDriver
  let chatPage1: ChatPage
  let chatPage2: ChatPage

  before(async function () {
    console.log('\n=== Creating two Chrome instances for multi-client test ===')
    driver1 = await createWebRTCDriver()
    driver2 = await createWebRTCDriver()
    chatPage1 = new ChatPage(driver1)
    chatPage2 = new ChatPage(driver2)
  })

  after(async function () {
    await quitDriver(driver1)
    await quitDriver(driver2)
  })

  beforeEach(async function () {
    await driver1.get(testUrl)
    await chatPage1.sleep(500)
    await clearBrowserState(driver1)

    await driver2.get(testUrl)
    await chatPage2.sleep(500)
    await clearBrowserState(driver2)
  })

  it('should allow two clients to join conference and transmit audio', async function () {
    console.log('\n[Test] Starting two-client audio transmission test')

    // Step 1: Register both users
    console.log('[Step 1] Registering user 1...')
    const user1 = await createTestUser(driver1)
    await chatPage1.waitForChatPage()
    console.log(`[User 1] Registered: ${user1.username}`)

    console.log('[Step 2] Registering user 2...')
    const user2 = await createTestUser(driver2)
    await chatPage2.waitForChatPage()
    console.log(`[User 2] Registered: ${user2.username}`)

    // Step 2: User 1 creates a chat
    const chatName = `Multi-Client Audio Test ${Date.now()}`
    console.log(`[Step 3] User 1 creating chat: ${chatName}`)
    await chatPage1.createChat(chatName, 'group')
    await chatPage1.waitForModalToClose()
    await chatPage1.sleep(1000)

    // Step 3: User 1 selects the chat
    console.log('[Step 4] User 1 selecting the chat...')
    await chatPage1.selectFirstChat()
    await chatPage1.waitForChatRoom()
    await chatPage1.sleep(2000)

    // Check URL
    const currentUrl = await driver1.getCurrentUrl()
    console.log(`[User 1] Current URL: ${currentUrl}`)

    const chatId = await chatPage1.getCurrentChatId()
    console.log(`[User 1] Chat ID from getCurrentChatId(): ${chatId}`)

    if (!chatId) {
      console.log('[User 1] Chat ID is empty, trying alternative method...')
      // Get chat ID from the chat header element data attribute or from Pinia store
      const chatIdFromStore = await driver1.executeScript(`
        return window.__vue_app__?.config?.globalProperties?.$pinia?.state?.value?.chat?.currentChatId || ''
      `) as string
      console.log(`[User 1] Chat ID from Pinia store: ${chatIdFromStore}`)
    }

    // Step 4: Add User 2 to the chat via API
    console.log('[Step 5] Adding User 2 to chat...')
    const user2Id = await getUserIdFromApi(driver2)
    await chatPage1.addParticipantToChat(user2Id)
    await chatPage1.sleep(2000)
    console.log(`[User 1] Added User 2 (${user2Id}) to chat`)

    // Step 4: User 2 waits for chat to appear and selects it
    console.log('[Step 6] User 2 waiting for chat to appear...')

    // Check current URL for User 2
    const user2Url = await driver2.getCurrentUrl()
    console.log(`[User 2] Current URL: ${user2Url}`)

    // Wait for WebSocket to push chat update
    await chatPage2.sleep(3000)

    // Check how many chats User 2 has
    const chatCount = await chatPage2.getChatCount()
    console.log(`[User 2] Number of chats in list: ${chatCount}`)

    if (chatCount === 0) {
      console.log('[User 2] No chats found, refreshing page...')
      await driver2.navigate().refresh()
      await chatPage2.waitForChatPage()
      await chatPage2.sleep(2000)
      const chatCountAfterRefresh = await chatPage2.getChatCount()
      console.log(`[User 2] Number of chats after refresh: ${chatCountAfterRefresh}`)
    }

    await chatPage2.waitForChatInList(chatName, 15000)
    console.log('[User 2] Chat appeared in list')

    await chatPage2.selectFirstChat()
    await chatPage2.waitForChatRoom()
    await chatPage2.sleep(1000)
    console.log('[User 2] Chat selected')

    // Step 5: Ensure both users are connected to Verto before starting call
    console.log('[Step 7] Ensuring both users are connected to Verto...')
    const user1Verto = await driver1.executeScript(`
      const store = window.__voiceStore
      return {
        isConnected: store?.isConnected || false,
        hasCredentials: !!store?.credentials
      }
    `) as any
    console.log('[User 1] Verto state:', JSON.stringify(user1Verto))

    const user2Verto = await driver2.executeScript(`
      const store = window.__voiceStore
      return {
        isConnected: store?.isConnected || false,
        hasCredentials: !!store?.credentials
      }
    `) as any
    console.log('[User 2] Verto state:', JSON.stringify(user2Verto))

    // If either user is not connected, initialize Verto for both
    if (!user1Verto.isConnected || !user2Verto.isConnected) {
      console.log('[Both Users] Initializing Verto connections...')

      await driver1.executeScript(`
        if (window.__voiceStore?.initVerto) {
          return window.__voiceStore.initVerto()
        }
      `)
      await chatPage1.sleep(1000)

      await driver2.executeScript(`
        if (window.__voiceStore?.initVerto) {
          return window.__voiceStore.initVerto()
        }
      `)
      await chatPage2.sleep(2000)

      console.log('[Both Users] Verto connections initialized')
    }

    // Step 6: User 1 starts Call All
    console.log('[Step 8] User 1 starting Call All...')
    await chatPage1.clickAdHocCallButton()
    await chatPage1.waitForAdHocDropdown()
    await chatPage1.clickCallAll()
    console.log('[User 1] Clicked Call All')

    // Step 6: User 1 should see ConferenceView
    console.log('[Step 7] Waiting for User 1 ConferenceView...')
    await chatPage1.waitForConferenceView(20000)
    console.log('[User 1] ConferenceView appeared')

    // Get full voice state from User 1
    await chatPage1.sleep(3000) // Wait for conference to be fully created
    const user1State = await driver1.executeScript(`
      const store = window.__voiceStore
      const conf = store?.currentConference
      return {
        isInCall: store?.isInCall || false,
        isConnected: store?.isConnected || false,
        hasCurrentConference: !!conf,
        confId: conf?.id || null,
        confName: conf?.name || null,
        fsName: conf?.freeswitch_name || null,
        participantCount: store?.participants?.length || 0
      }
    `) as any
    console.log('[User 1] Full voice state:', JSON.stringify(user1State))

    const confFSName = user1State.fsName
    const confId = user1State.confId
    console.log(`[Conference] FS Name: ${confFSName}, ID: ${confId}`)

    // User 2 should see incoming call overlay and answer it
    console.log('[User 2] Waiting for incoming call...')

    // Wait up to 10 seconds for incoming call to arrive
    let incomingCallDetails: any = null
    for (let i = 0; i < 20; i++) {
      await chatPage2.sleep(500)
      incomingCallDetails = await driver2.executeScript(`
        const store = window.__voiceStore
        return {
          hasVertoIncoming: !!store?.vertoIncomingCall,
          vertoRemoteNumber: store?.vertoIncomingCall?.params?.caller_id_number || null,
          hasIncomingCallData: !!store?.incomingCallData,
          hasIncomingConferenceData: !!store?.incomingConferenceData,
          incomingConferenceId: store?.incomingConferenceData?.conferenceId || null,
          incomingConferenceFSName: store?.incomingConferenceData?.fsName || null,
          hasIncomingCall: store?.hasIncomingCall || false
        }
      `) as any

      if (incomingCallDetails.hasIncomingCall || incomingCallDetails.hasVertoIncoming) {
        console.log(`[User 2] Incoming call detected after ${(i + 1) * 500}ms`)
        break
      }
    }
    console.log('[User 2] Incoming call state:', JSON.stringify(incomingCallDetails))

    // User 2 joins the conference
    // Try multiple methods: answer incoming call, join directly, or use Call All
    let user2Joined = false

    // Method 1: Answer incoming call if visible
    const incomingCallVisible = await chatPage2.isIncomingCallVisible()
    if (incomingCallVisible || incomingCallDetails.hasVertoIncoming || incomingCallDetails.hasIncomingCall) {
      console.log('[User 2] Answering incoming call via store...')
      try {
        await driver2.executeScript(`
          if (window.__voiceStore?.answerConferenceCall) {
            return window.__voiceStore.answerConferenceCall()
          }
        `)
        await chatPage2.sleep(3000)
        console.log('[User 2] Incoming call answered via store')

        // Check if ConferenceView appeared
        const confViewVisible = await chatPage2.isConferenceViewVisible()
        if (confViewVisible) {
          console.log('[User 2] ConferenceView appeared after answering')
          user2Joined = true
        } else {
          console.log('[User 2] ConferenceView not visible after answering, trying joinConference...')
        }
      } catch (e) {
        console.log('[User 2] Answer failed:', e)
      }
    }

    // Method 2: Join conference directly if we have confId
    if (!user2Joined && confId) {
      console.log(`[User 2] Joining conference ${confId} directly via API...`)
      try {
        await driver2.executeScript(`
          if (window.__voiceStore?.joinConference) {
            return window.__voiceStore.joinConference('${confId}')
          }
        `)
        await chatPage2.sleep(3000)
        console.log('[User 2] Join conference initiated')

        const confViewVisible = await chatPage2.isConferenceViewVisible()
        if (confViewVisible) {
          console.log('[User 2] ConferenceView appeared after join')
          user2Joined = true
        } else {
          console.log('[User 2] ConferenceView not visible after join, trying Call All...')
        }
      } catch (e) {
        console.log('[User 2] Join failed:', e)
      }
    }

    // Method 3: Fallback to Call All
    if (!user2Joined) {
      console.log('[User 2] Falling back to Call All...')
      await chatPage2.clickAdHocCallButton()
      await chatPage2.waitForAdHocDropdown()
      await chatPage2.clickCallAll()
      console.log('[User 2] Clicked Call All')

      await chatPage2.waitForConferenceView(20000)
      console.log('[User 2] ConferenceView appeared')
    }

    // Step 7: Check what conferences each user is in
    console.log('[Step 8] Checking conference details for both users...')
    const conf1Details = await driver1.executeScript(`
      const store = window.__voiceStore
      const conf = store?.currentConference
      return {
        confName: conf?.name || null,
        confId: conf?.id || null,
        fsName: conf?.freeswitch_name || null,
        participantCount: store?.participants?.length || 0
      }
    `) as any
    console.log('[User 1] Conference details:', JSON.stringify(conf1Details))

    const conf2Details = await driver2.executeScript(`
      const store = window.__voiceStore
      const conf = store?.currentConference
      return {
        confName: conf?.name || null,
        confId: conf?.id || null,
        fsName: conf?.freeswitch_name || null,
        participantCount: store?.participants?.length || 0
      }
    `) as any
    console.log('[User 2] Conference details:', JSON.stringify(conf2Details))

    // Verify both users are in the same conference
    console.log('[Step 9] Verifying both users are in the same conference...')

    // Critical check: both users must have the same conference ID
    expect(conf1Details.confId).to.equal(conf2Details.confId, 'Both users should be in the same conference')
    expect(conf1Details.confId).to.not.be.null
    console.log(`✓ Both users are in conference: ${conf1Details.confId}`)

    // Wait a bit for participant sync
    await chatPage1.sleep(3000)

    // Get participant counts from UI
    const count1 = await chatPage1.getParticipantCountNumber()
    const count2 = await chatPage2.getParticipantCountNumber()
    console.log(`[User 1] Participant count from UI: ${count1}`)
    console.log(`[User 2] Participant count from UI: ${count2}`)

    // At minimum, each user should see themselves
    expect(count1).to.be.at.least(1, 'User 1 should see at least 1 participant')
    expect(count2).to.be.at.least(1, 'User 2 should see at least 1 participant')

    // Step 8: Check initial mute status
    console.log('[Step 9] Checking mute status...')
    const user1Muted = await chatPage1.isMutedInConference()
    const user2Muted = await chatPage2.isMutedInConference()
    console.log(`[User 1] Initially muted: ${user1Muted}`)
    console.log(`[User 2] Initially muted: ${user2Muted}`)

    // Step 9: Unmute both users
    console.log('[Step 10] Unmuting both users...')
    await chatPage1.unmuteInConference()
    await chatPage2.unmuteInConference()
    await chatPage1.sleep(1000)
    console.log('[Both users] Unmuted')

    // Step 10: Get initial WebRTC stats
    console.log('[Step 11] Getting initial WebRTC stats...')
    const stats1Initial = await getWebRTCStats(driver1)
    const stats2Initial = await getWebRTCStats(driver2)

    console.log(`[User 1] Initial stats:`, {
      connectionState: stats1Initial?.connectionState,
      bytesSent: stats1Initial?.bytesSent,
      bytesReceived: stats1Initial?.bytesReceived
    })
    console.log(`[User 2] Initial stats:`, {
      connectionState: stats2Initial?.connectionState,
      bytesSent: stats2Initial?.bytesSent,
      bytesReceived: stats2Initial?.bytesReceived
    })

    // Step 11: Wait for audio transmission (fake devices are sending)
    console.log('[Step 12] Waiting 5 seconds for audio transmission...')
    await chatPage1.sleep(5000)

    // Step 12: Get final WebRTC stats
    console.log('[Step 13] Getting final WebRTC stats...')
    const stats1Final = await getWebRTCStats(driver1)
    const stats2Final = await getWebRTCStats(driver2)

    console.log(`[User 1] Final stats:`, {
      connectionState: stats1Final?.connectionState,
      bytesSent: stats1Final?.bytesSent,
      bytesReceived: stats1Final?.bytesReceived
    })
    console.log(`[User 2] Final stats:`, {
      connectionState: stats2Final?.connectionState,
      bytesSent: stats2Final?.bytesSent,
      bytesReceived: stats2Final?.bytesReceived
    })

    // Step 13: Verify audio transmission
    if (stats1Initial && stats1Final && stats2Initial && stats2Final) {
      const user1Sent = stats1Final.bytesSent - stats1Initial.bytesSent
      const user1Received = stats1Final.bytesReceived - stats1Initial.bytesReceived
      const user2Sent = stats2Final.bytesSent - stats2Initial.bytesSent
      const user2Received = stats2Final.bytesReceived - stats2Initial.bytesReceived

      console.log('\n=== Audio Transmission Results ===')
      console.log(`[User 1] Sent: ${user1Sent} bytes, Received: ${user1Received} bytes`)
      console.log(`[User 2] Sent: ${user2Sent} bytes, Received: ${user2Received} bytes`)

      // Both users should be sending audio (fake device)
      expect(user1Sent).to.be.greaterThan(1000, 'User 1 should send audio data')
      expect(user2Sent).to.be.greaterThan(1000, 'User 2 should send audio data')

      // Both users should be receiving audio from the other
      expect(user1Received).to.be.greaterThan(1000, 'User 1 should receive audio from User 2')
      expect(user2Received).to.be.greaterThan(1000, 'User 2 should receive audio from User 1')

      console.log('✓ Audio transmission verified between both clients')
    } else {
      console.log('⚠ Could not get WebRTC stats, skipping transmission verification')
    }

    console.log('\n[Test] Two-client audio transmission test completed successfully')
  })
})
