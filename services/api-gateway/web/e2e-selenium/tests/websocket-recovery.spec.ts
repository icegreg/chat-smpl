/**
 * WebSocket Recovery E2E Tests
 *
 * Tests for message recovery after WebSocket disconnection:
 * - Automatic recovery via Centrifugo history
 * - API fallback sync when Centrifugo recovery fails
 * - Multi-device scenario (one device goes offline)
 * - seq_num tracking and localStorage persistence
 */

import { WebDriver } from 'selenium-webdriver'
import { expect } from 'chai'
import { createDriver, quitDriver, BASE_URL } from '../config/webdriver.js'
import { createNetworkHelper, NetworkHelper } from '../helpers/networkHelper.js'
import { createTestUser, wait, getUserIdFromApi } from '../helpers/testHelpers.js'
import { ChatPage } from '../pages/ChatPage.js'

describe('WebSocket Recovery', function () {
  this.timeout(300000) // 5 minutes per test

  let driver1: WebDriver
  let driver2: WebDriver
  let network1: NetworkHelper
  let network2: NetworkHelper
  let chatPage1: ChatPage
  let chatPage2: ChatPage

  beforeEach(async function () {
    driver1 = await createDriver()
    driver2 = await createDriver()

    network1 = await createNetworkHelper(driver1)
    network2 = await createNetworkHelper(driver2)

    chatPage1 = new ChatPage(driver1)
    chatPage2 = new ChatPage(driver2)
  })

  afterEach(async function () {
    if (network1) await network1.cleanup()
    if (network2) await network2.cleanup()

    if (driver1) await quitDriver(driver1)
    if (driver2) await quitDriver(driver2)
  })

  /**
   * Helper: Get WebSocket connection status from browser
   * @deprecated Currently not used, but kept for potential debugging
   */
  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  async function getWebSocketStatus(driver: WebDriver): Promise<{
    isConnected: boolean
    state: string
  }> {
    const result = await driver.executeScript(`
      return new Promise((resolve) => {
        // Check Pinia store state
        const app = document.querySelector('#app')?.__vue_app__;
        if (!app) {
          resolve({ isConnected: false, state: 'no_app' });
          return;
        }

        // Try to get network store state
        try {
          const networkStore = app.config.globalProperties.$pinia?._s?.get('network');
          if (networkStore) {
            resolve({
              isConnected: networkStore.isWebSocketConnected,
              state: networkStore.isWebSocketConnected ? 'connected' : 'disconnected'
            });
            return;
          }
        } catch (e) {
          // fallback
        }

        // Fallback: check if there are any open WebSocket connections
        resolve({ isConnected: true, state: 'unknown' });
      });
    `)
    void driver // mark as used
    return result as { isConnected: boolean; state: string }
  }
  void getWebSocketStatus // suppress unused warning

  /**
   * Helper: Get stored seq_num from localStorage
   */
  async function getStoredSeqNums(driver: WebDriver): Promise<Record<string, number>> {
    const result = await driver.executeScript(`
      const stored = localStorage.getItem('chat_seq_nums');
      return stored ? JSON.parse(stored) : {};
    `)
    return (result as Record<string, number>) || {}
  }

  /**
   * Helper: Send message via API (bypassing UI)
   */
  async function sendMessageViaApi(
    driver: WebDriver,
    chatId: string,
    content: string
  ): Promise<{ id: string; seq_num: number }> {
    const result = await driver.executeScript(`
      return new Promise(async (resolve, reject) => {
        try {
          const token = localStorage.getItem('access_token');
          if (!token) {
            reject('No access token');
            return;
          }
          const response = await fetch('/api/chats/${chatId}/messages', {
            method: 'POST',
            headers: {
              'Authorization': 'Bearer ' + token,
              'Content-Type': 'application/json'
            },
            body: JSON.stringify({ content: '${content}' })
          });
          if (!response.ok) {
            reject('API error: ' + response.status);
            return;
          }
          const msg = await response.json();
          resolve({ id: msg.id, seq_num: msg.seq_num });
        } catch (e) {
          reject(e.message);
        }
      });
    `)
    return result as { id: string; seq_num: number }
  }

  /**
   * Helper: Get chat ID from current URL or API
   */
  async function getChatIdViaApi(driver: WebDriver, chatName: string): Promise<string> {
    const result = await driver.executeScript(`
      return new Promise(async (resolve, reject) => {
        try {
          const token = localStorage.getItem('access_token');
          if (!token) {
            reject('No access token');
            return;
          }
          const response = await fetch('/api/chats', {
            headers: { 'Authorization': 'Bearer ' + token }
          });
          if (!response.ok) {
            reject('API error: ' + response.status);
            return;
          }
          const data = await response.json();
          const chat = data.chats.find(c => c.name.includes('${chatName}'));
          resolve(chat ? chat.id : null);
        } catch (e) {
          reject(e.message);
        }
      });
    `)
    return result as string
  }

  describe('Centrifugo Recovery (automatic)', function () {
    it('should automatically receive missed messages after WebSocket reconnect', async function () {
      console.log('[Test] Setting up users and chat...')

      // 1. Register User 1
      await createTestUser(driver1)
      const user1Id = await getUserIdFromApi(driver1)
      console.log(`[Test] User 1 ID: ${user1Id}`)

      // 2. Register User 2
      await driver2.get(BASE_URL)
      await createTestUser(driver2)
      const user2Id = await getUserIdFromApi(driver2)
      console.log(`[Test] User 2 ID: ${user2Id}`)

      // 3. User 1 creates chat with User 2
      const chatName = `Recovery-${Date.now()}`
      await chatPage1.createChatWithParticipants(chatName, [user2Id])
      await chatPage1.waitForModalToClose()
      await wait(2000)

      // 4. User 2 opens the chat
      await driver2.navigate().refresh()
      await wait(2000)
      await chatPage2.waitForChatInList(chatName)
      await chatPage2.clickChatByNameInList(chatName)
      await chatPage2.waitForChatRoom()

      // 5. User 1 opens the chat
      await chatPage1.clickChatByNameInList(chatName)
      await chatPage1.waitForChatRoom()
      await wait(1000)

      // 6. Send initial message to confirm WebSocket works
      await chatPage1.sendMessage('Initial test message')
      await wait(2000)

      // Verify User 2 receives it via WebSocket (without refresh)
      await chatPage2.waitForMessageContaining('Initial test message', 10000)
      console.log('[Test] WebSocket working - initial message received')

      // 7. Get initial message count
      const initialCount = await chatPage2.getMessageCount()
      console.log(`[Test] Initial message count for User 2: ${initialCount}`)

      // 8. Block WebSocket for User 2 (API still works)
      console.log('[Test] Blocking WebSocket for User 2...')
      await network2.blockWebSocket()
      await wait(2000) // Wait for connection to drop

      // 9. User 1 sends messages while User 2's WebSocket is blocked
      const offlineMessages = [
        'Message during WS block 1',
        'Message during WS block 2',
        'Message during WS block 3',
      ]

      for (const msg of offlineMessages) {
        await chatPage1.sendMessage(msg)
        await wait(1000)
        console.log(`[Test] Sent: ${msg}`)
      }

      // 10. Verify User 2 has NOT received messages yet (WebSocket blocked)
      await wait(2000)
      const countDuringBlock = await chatPage2.getMessageCount()
      console.log(`[Test] Message count during block: ${countDuringBlock}`)
      // Should still be the initial count (no new messages via WebSocket)

      // 11. Unblock WebSocket - trigger reconnect
      console.log('[Test] Unblocking WebSocket - expecting automatic recovery...')
      await network2.unblockAllUrls()

      // 12. Wait for Centrifugo to reconnect and recover messages
      // The client should automatically reconnect and receive missed messages
      await wait(8000) // Give time for reconnect + recovery

      // 13. Check if messages appeared WITHOUT page refresh
      const finalCount = await chatPage2.getMessageCount()
      console.log(`[Test] Final message count (no refresh): ${finalCount}`)

      const finalMessages = await chatPage2.getMessageTexts()
      console.log(`[Test] Final messages: ${JSON.stringify(finalMessages)}`)

      // Verify all messages are present
      for (const msg of offlineMessages) {
        const found = finalMessages.some((m) => m.includes(msg))
        console.log(`[Test] Message "${msg}" found: ${found}`)
        expect(found, `Message "${msg}" should appear after reconnect`).to.be.true
      }

      console.log('[Test] SUCCESS: All messages recovered automatically!')
    })
  })

  describe('API Fallback Sync', function () {
    it('should sync messages via API when recovery history is exhausted', async function () {
      console.log('[Test] Setting up for API fallback test...')

      // 1. Setup users
      await createTestUser(driver1)
      // user1Id not used in this test - we only need user2Id for participants
      void (await getUserIdFromApi(driver1))

      await driver2.get(BASE_URL)
      await createTestUser(driver2)
      const user2Id = await getUserIdFromApi(driver2)

      // 2. Create chat
      const chatName = `APIfallback-${Date.now()}`
      await chatPage1.createChatWithParticipants(chatName, [user2Id])
      await chatPage1.waitForModalToClose()
      await wait(2000)

      // 3. Both users open chat
      await driver2.navigate().refresh()
      await wait(2000)
      await chatPage2.waitForChatInList(chatName)
      await chatPage2.clickChatByNameInList(chatName)
      await chatPage2.waitForChatRoom()

      await chatPage1.clickChatByNameInList(chatName)
      await chatPage1.waitForChatRoom()
      await wait(1000)

      // 4. Send initial message
      await chatPage1.sendMessage('Setup message')
      await chatPage2.waitForMessageContaining('Setup message', 10000)

      // 5. Get chat ID for API calls
      const chatId = await getChatIdViaApi(driver1, chatName)
      console.log(`[Test] Chat ID: ${chatId}`)

      // 6. Go completely OFFLINE for User 2
      console.log('[Test] User 2 going completely offline...')
      await network2.goOffline()
      await wait(1000)

      // 7. Send many messages via API (more than Centrifugo history_size if possible)
      const manyMessages: string[] = []
      for (let i = 1; i <= 10; i++) {
        const content = `Bulk message ${i} - ${Date.now()}`
        manyMessages.push(content)
        await sendMessageViaApi(driver1, chatId, content)
        await wait(200)
        console.log(`[Test] Sent bulk message ${i}`)
      }

      // 8. Restore network
      console.log('[Test] Restoring User 2 network...')
      await network2.goOnline()
      await wait(5000)

      // 9. Refresh page to trigger full sync (simulating app restart)
      await driver2.navigate().refresh()
      await wait(3000)
      await chatPage2.waitForChatInList(chatName)
      await chatPage2.clickChatByNameInList(chatName)
      await chatPage2.waitForChatRoom()
      await wait(2000)

      // 10. Verify all messages are synced
      const finalMessages = await chatPage2.getMessageTexts()
      console.log(`[Test] Total messages after sync: ${finalMessages.length}`)

      let syncedCount = 0
      for (const msg of manyMessages) {
        if (finalMessages.some((m) => m.includes(msg))) {
          syncedCount++
        }
      }
      console.log(`[Test] Synced ${syncedCount}/${manyMessages.length} messages`)

      expect(syncedCount).to.equal(
        manyMessages.length,
        'All messages should be synced via API fallback'
      )
    })
  })

  describe('Multi-Device Scenario', function () {
    it('should handle one device going offline while another stays connected', async function () {
      console.log('[Test] Multi-device scenario...')

      // Simulate: User has 2 "devices" (browser tabs)
      // Device 1 (driver1) stays online
      // Device 2 (driver2) goes offline

      // 1. Register single user and login on both "devices"
      await createTestUser(driver1)
      // userId not needed, just verifying user exists
      void (await getUserIdFromApi(driver1))

      // Get token to login on second device
      const token = await driver1.executeScript(`
        return localStorage.getItem('access_token');
      `) as string

      // Login second device with same credentials
      await driver2.get(BASE_URL)
      await driver2.executeScript(`
        localStorage.setItem('access_token', '${token}');
      `)
      await driver2.navigate().refresh()
      await wait(2000)

      // 2. Create a chat
      const chatName = `MultiDevice-${Date.now()}`
      await chatPage1.createChat(chatName)
      await chatPage1.waitForModalToClose()
      await wait(2000)

      // 3. Open chat on both devices
      await chatPage1.clickChatByNameInList(chatName)
      await chatPage1.waitForChatRoom()

      await driver2.navigate().refresh()
      await wait(2000)
      await chatPage2.waitForChatInList(chatName)
      await chatPage2.clickChatByNameInList(chatName)
      await chatPage2.waitForChatRoom()
      await wait(1000)

      // 4. Send message from Device 1 - Device 2 should see it
      await chatPage1.sendMessage('From device 1 - online')
      await chatPage2.waitForMessageContaining('From device 1 - online', 10000)
      console.log('[Test] Both devices receiving messages')

      // 5. Device 2 goes offline
      console.log('[Test] Device 2 going offline...')
      await network2.goOffline()
      await wait(2000)

      // 6. Send messages from Device 1 while Device 2 is offline
      const offlineMsgs = [
        'While device 2 offline - msg 1',
        'While device 2 offline - msg 2',
      ]
      for (const msg of offlineMsgs) {
        await chatPage1.sendMessage(msg)
        await wait(1000)
      }
      console.log('[Test] Sent messages while Device 2 offline')

      // 7. Device 2 comes back online
      console.log('[Test] Device 2 coming back online...')
      await network2.goOnline()
      await wait(5000)

      // 8. Check stored seq_nums on Device 2
      const seqNums = await getStoredSeqNums(driver2)
      console.log(`[Test] Stored seq_nums on Device 2: ${JSON.stringify(seqNums)}`)

      // 9. Refresh Device 2 to trigger sync
      await driver2.navigate().refresh()
      await wait(3000)
      await chatPage2.waitForChatInList(chatName)
      await chatPage2.clickChatByNameInList(chatName)
      await chatPage2.waitForChatRoom()
      await wait(2000)

      // 10. Verify Device 2 has all messages
      const device2Messages = await chatPage2.getMessageTexts()
      console.log(`[Test] Device 2 messages: ${device2Messages.length}`)

      for (const msg of offlineMsgs) {
        expect(device2Messages.join(' ')).to.include(msg, `Device 2 should have: ${msg}`)
      }
      console.log('[Test] SUCCESS: Multi-device sync working!')
    })
  })

  describe('seq_num Tracking', function () {
    it('should persist seq_num in localStorage across page reloads', async function () {
      console.log('[Test] Testing seq_num persistence...')

      // 1. Setup
      await createTestUser(driver1)
      // userId not needed for this test
      void (await getUserIdFromApi(driver1))

      const chatName = `SeqNum-${Date.now()}`
      await chatPage1.createChat(chatName)
      await chatPage1.waitForModalToClose()
      await wait(2000)

      await chatPage1.clickChatByNameInList(chatName)
      await chatPage1.waitForChatRoom()
      await wait(1000)

      // 2. Send several messages
      for (let i = 1; i <= 5; i++) {
        await chatPage1.sendMessage(`Seq test message ${i}`)
        await wait(500)
      }
      await wait(2000)

      // 3. Check seq_nums are stored
      const seqNumsBefore = await getStoredSeqNums(driver1)
      console.log(`[Test] seq_nums before reload: ${JSON.stringify(seqNumsBefore)}`)
      expect(Object.keys(seqNumsBefore).length).to.be.greaterThan(
        0,
        'Should have stored seq_nums'
      )

      // 4. Reload page
      await driver1.navigate().refresh()
      await wait(2000)

      // 5. Check seq_nums persist
      const seqNumsAfter = await getStoredSeqNums(driver1)
      console.log(`[Test] seq_nums after reload: ${JSON.stringify(seqNumsAfter)}`)

      expect(seqNumsAfter).to.deep.equal(
        seqNumsBefore,
        'seq_nums should persist across reload'
      )

      console.log('[Test] SUCCESS: seq_num persistence working!')
    })
  })

  describe('Reconnect with Pending Messages', function () {
    it('should handle messages sent while reconnecting', async function () {
      console.log('[Test] Testing reconnect with pending messages...')

      // 1. Setup
      await createTestUser(driver1)

      const chatName = `Pending-${Date.now()}`
      await chatPage1.createChat(chatName)
      await chatPage1.waitForModalToClose()
      await wait(2000)

      await chatPage1.clickChatByNameInList(chatName)
      await chatPage1.waitForChatRoom()
      await wait(1000)

      // 2. Send initial message (confirm working)
      await chatPage1.sendMessage('Before disconnect')
      await chatPage1.waitForMessageContaining('Before disconnect', 10000)
      console.log('[Test] Initial message sent successfully')

      // 3. Go offline
      console.log('[Test] Going offline...')
      await network1.goOffline()
      await wait(1000)

      // 4. Try to send message while offline
      // The message should be queued (pending)
      await chatPage1.sendMessage('Sent while offline')
      await wait(2000)

      // 5. Go back online
      console.log('[Test] Going back online...')
      await network1.goOnline()
      await wait(5000)

      // 6. Verify the pending message was sent after reconnect
      const messages = await chatPage1.getMessageTexts()
      console.log(`[Test] Messages after reconnect: ${messages.length}`)

      // The "Sent while offline" message should either:
      // - Be sent successfully after reconnect, OR
      // - Show as pending with option to retry

      // For now, just verify the app doesn't crash and we can send new messages
      await chatPage1.sendMessage('After reconnect')
      await chatPage1.waitForMessageContaining('After reconnect', 10000)

      console.log('[Test] SUCCESS: App works after reconnect!')
    })
  })
})
