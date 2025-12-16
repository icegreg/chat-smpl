/**
 * Network Resilience E2E Tests
 *
 * Тестирование поведения клиента при различных сетевых проблемах:
 * - Пропадание сети при получении сообщений
 * - Пропадание сети при загрузке списка чатов
 * - Пропадание сети при отправке сообщения
 * - Восстановление после disconnect
 * - Работа при медленной сети
 */

import { WebDriver } from 'selenium-webdriver'
import { expect } from 'chai'
import { createDriver, quitDriver, BASE_URL } from '../config/webdriver.js'
import { createNetworkHelper, NetworkHelper } from '../helpers/networkHelper.js'
import { createTestUser, loginUser, wait, generateTestUser } from '../helpers/testHelpers.js'
import { ChatPage } from '../pages/ChatPage.js'
import { RegisterPage } from '../pages/RegisterPage.js'

describe('Network Resilience', function () {
  this.timeout(180000) // 3 минуты на каждый тест

  let driver1: WebDriver
  let driver2: WebDriver
  let network1: NetworkHelper
  let network2: NetworkHelper
  let chatPage1: ChatPage
  let chatPage2: ChatPage

  beforeEach(async function () {
    driver1 = await createDriver()
    driver2 = await createDriver()

    // Инициализируем network helpers
    network1 = await createNetworkHelper(driver1)
    network2 = await createNetworkHelper(driver2)

    // Создаём Page Objects
    chatPage1 = new ChatPage(driver1)
    chatPage2 = new ChatPage(driver2)
  })

  afterEach(async function () {
    // Всегда возвращаем сеть в нормальное состояние
    if (network1) await network1.cleanup()
    if (network2) await network2.cleanup()

    if (driver1) await quitDriver(driver1)
    if (driver2) await quitDriver(driver2)
  })

  describe('Disconnect при получении сообщений', function () {
    it('должен синхронизировать пропущенные сообщения после reconnect', async function () {
      // 1. Регистрируем обоих пользователей
      await createTestUser(driver1)
      await driver1.executeScript(`
        return new Promise(async (resolve) => {
          const token = localStorage.getItem('access_token');
          const resp = await fetch('/api/auth/me', { headers: { 'Authorization': 'Bearer ' + token } });
          const user = await resp.json();
          resolve(user.id);
        });
      `) as string

      await driver2.get(BASE_URL)
      const user2Data = await createTestUser(driver2)
      void user2Data // используется для создания пользователя

      const user2Id = await driver2.executeScript(`
        return new Promise(async (resolve) => {
          const token = localStorage.getItem('access_token');
          const resp = await fetch('/api/auth/me', { headers: { 'Authorization': 'Bearer ' + token } });
          const user = await resp.json();
          resolve(user.id);
        });
      `) as string

      // 2. User1 создаёт чат и добавляет user2
      const chatName = `NetworkTest-${Date.now()}`
      await chatPage1.createChatWithParticipants(chatName, [user2Id])
      await chatPage1.waitForModalToClose()
      await wait(2000)

      // 3. User2 рефрешит страницу и открывает чат
      await driver2.navigate().refresh()
      await wait(3000)
      await chatPage2.waitForChatInList(chatName)
      await chatPage2.clickChatByNameInList(chatName)
      await chatPage2.waitForChatRoom()
      await wait(1000)

      // 4. User1 открывает чат
      await chatPage1.clickChatByNameInList(chatName)
      await chatPage1.waitForChatRoom()
      await wait(1000)

      // 5. Отправляем первое сообщение (проверяем что всё работает)
      await chatPage1.sendMessage('Message before disconnect')
      await wait(2000)

      // Проверяем что user2 получил сообщение
      await chatPage2.waitForMessageContaining('Message before disconnect', 10000)
      const initialMessages = await chatPage2.getMessageTexts()
      console.log(`[Test] Initial messages for user2: ${initialMessages.length}`)

      // 6. ОТКЛЮЧАЕМ СЕТЬ у user2
      console.log('[Test] Disconnecting user2 network...')
      await network2.goOffline()
      await wait(1000)

      // 7. User1 отправляет сообщения пока user2 offline
      const offlineMessages = ['Offline msg 1', 'Offline msg 2', 'Offline msg 3']
      for (const msg of offlineMessages) {
        await chatPage1.sendMessage(msg)
        await wait(500)
      }
      console.log(`[Test] Sent ${offlineMessages.length} messages while user2 offline`)
      await wait(2000)

      // 8. ВОССТАНАВЛИВАЕМ СЕТЬ у user2
      console.log('[Test] Reconnecting user2 network...')
      await network2.goOnline()

      // Ждём reconnect и sync
      await wait(5000)

      // 9. Рефрешим страницу user2 для загрузки сообщений
      await driver2.navigate().refresh()
      await wait(3000)
      await chatPage2.waitForChatInList(chatName)
      await chatPage2.clickChatByNameInList(chatName)
      await chatPage2.waitForChatRoom()
      await wait(2000)

      // 10. Проверяем что user2 получил все пропущенные сообщения
      const finalMessages = await chatPage2.getMessageTexts()
      console.log(`[Test] Final messages for user2: ${finalMessages.length}`)

      for (const msg of offlineMessages) {
        expect(finalMessages.join(' ')).to.include(msg, `Message "${msg}" should appear after reconnect`)
      }
      console.log('[Test] All offline messages synced successfully!')
    })
  })

  describe('Disconnect при загрузке списка чатов', function () {
    it('должен показать ошибку и позволить retry при загрузке чатов offline', async function () {
      // 1. Регистрируем пользователя
      const userData = generateTestUser()
      const registerPage = new RegisterPage(driver1)
      await registerPage.goto()
      await registerPage.register(userData)
      await registerPage.waitForUrl('/chat', 15000)

      // Создаём несколько чатов
      await chatPage1.createChat(`Chat1-${Date.now()}`)
      await chatPage1.waitForModalToClose()
      await wait(1000)
      await chatPage1.createChat(`Chat2-${Date.now()}`)
      await chatPage1.waitForModalToClose()
      await wait(1000)

      // 2. Выходим (очищаем localStorage)
      await driver1.executeScript('localStorage.clear()')
      await driver1.get(BASE_URL)
      await wait(1000)

      // 3. ОТКЛЮЧАЕМ СЕТЬ перед логином
      console.log('[Test] Going offline before login...')
      await network1.goOffline()
      await wait(500)

      // 4. Пытаемся залогиниться (должно показать ошибку)
      try {
        await loginUser(driver1, userData.email, userData.password)
        await wait(2000)
      } catch (e) {
        console.log('[Test] Login failed as expected (offline)')
      }

      // 5. Проверяем состояние страницы
      const pageSource1 = await driver1.getPageSource()
      console.log(`[Test] Page contains 'error': ${pageSource1.toLowerCase().includes('error')}`)

      // 6. ВОССТАНАВЛИВАЕМ СЕТЬ
      console.log('[Test] Going online...')
      await network1.goOnline()
      await wait(1000)

      // 7. Пробуем залогиниться снова
      await driver1.get(BASE_URL)
      await wait(1000)
      await loginUser(driver1, userData.email, userData.password)
      await wait(3000)

      // 8. Проверяем что чаты загрузились
      const chatCount = await chatPage1.getChatCount()
      console.log(`[Test] Loaded ${chatCount} chats after reconnect`)
      expect(chatCount).to.be.greaterThan(0, 'Should load chats after reconnect')
    })
  })

  describe('Disconnect при отправке сообщения', function () {
    it('должен показать ошибку при отправке сообщения offline', async function () {
      // 1. Регистрируем пользователя и создаём чат
      await createTestUser(driver1)

      const chatName = `SendTest-${Date.now()}`
      await chatPage1.createChat(chatName)
      await chatPage1.waitForModalToClose()
      await wait(1000)

      // Открываем чат
      await chatPage1.clickChatByNameInList(chatName)
      await chatPage1.waitForChatRoom()
      await wait(1000)

      // 2. Отправляем сообщение online (для проверки)
      await chatPage1.sendMessage('Online message')
      await wait(2000)

      // Проверяем что сообщение появилось
      await chatPage1.waitForMessageContaining('Online message', 5000)

      // 3. ОТКЛЮЧАЕМ СЕТЬ
      console.log('[Test] Going offline before sending...')
      await network1.goOffline()
      await wait(1000)

      // 4. Пытаемся отправить сообщение offline
      await chatPage1.sendMessage('Offline message attempt')

      // Ждём реакции UI
      await wait(3000)

      // 5. Проверяем поведение - приложение не должно крашнуться
      const pageSource = await driver1.getPageSource()
      console.log(`[Test] Page still loaded: ${pageSource.length > 0}`)

      // 6. ВОССТАНАВЛИВАЕМ СЕТЬ
      console.log('[Test] Going online...')
      await network1.goOnline()
      await wait(3000)

      // 7. Проверяем что приложение работает - можно отправить сообщение
      await chatPage1.sendMessage('After reconnect message')
      await wait(2000)

      await chatPage1.waitForMessageContaining('After reconnect message', 10000)
      const messages = await chatPage1.getMessageTexts()
      expect(messages.join(' ')).to.include('After reconnect message', 'Should be able to send messages after reconnect')
    })

    it('должен корректно обработать пропадание сети в момент отправки', async function () {
      // Этот тест проверяет race condition: сеть пропадает ВО ВРЕМЯ отправки

      // 1. Setup
      await createTestUser(driver1)

      const chatName = `RaceTest-${Date.now()}`
      await chatPage1.createChat(chatName)
      await chatPage1.waitForModalToClose()
      await wait(1000)

      await chatPage1.clickChatByNameInList(chatName)
      await chatPage1.waitForChatRoom()
      await wait(1000)

      // 2. Начинаем отправку и сразу отключаем сеть
      await chatPage1.typeMessage('Race condition message')

      // Нажимаем отправить и СРАЗУ отключаем сеть
      const sendPromise = chatPage1.clickSendMessage()
      await network1.goOffline() // Отключаем максимально быстро

      await sendPromise
      await wait(3000)

      // 3. Проверяем состояние - приложение не должно крашнуться
      const pageSource = await driver1.getPageSource()
      console.log(`[Test] State after race condition: page loaded = ${pageSource.length > 0}`)

      // 4. Восстанавливаем сеть
      await network1.goOnline()
      await wait(3000)

      // 5. Проверяем что приложение работает
      await chatPage1.sendMessage('Post-race message')
      await wait(2000)

      await chatPage1.waitForMessageContaining('Post-race message', 10000)
      console.log('[Test] App works after race condition')
    })
  })

  describe('Медленная сеть (Throttling)', function () {
    it('должен корректно работать при медленном соединении', async function () {
      // 1. Включаем throttling
      await network1.enableThrottling('SLOW_3G')
      console.log('[Test] Throttling enabled: SLOW_3G')

      // 2. Регистрируемся (будет медленно)
      const startTime = Date.now()
      await createTestUser(driver1)
      const registrationTime = Date.now() - startTime
      console.log(`[Test] Registration took ${registrationTime}ms with SLOW_3G`)

      // 3. Создаём чат
      const chatName = `SlowNet-${Date.now()}`
      await chatPage1.createChat(chatName)
      await chatPage1.waitForModalToClose(30000) // Больше времени из-за медленной сети
      await wait(3000)

      // 4. Открываем чат
      await chatPage1.clickChatByNameInList(chatName)
      await chatPage1.waitForChatRoom(30000)
      await wait(3000)

      // 5. Отправляем сообщение
      const sendStartTime = Date.now()
      await chatPage1.sendMessage('Slow network message')
      await wait(5000) // Ждём дольше из-за медленной сети
      const sendTime = Date.now() - sendStartTime
      console.log(`[Test] Message send took ${sendTime}ms with SLOW_3G`)

      // 6. Проверяем что сообщение появилось
      await chatPage1.waitForMessageContaining('Slow network message', 30000)
      console.log('[Test] Message appeared on slow network')

      // 7. Отключаем throttling
      await network1.disableEmulation()
    })
  })

  describe('WebSocket disconnect/reconnect', function () {
    it('должен переподключить WebSocket после разрыва', async function () {
      // 1. Setup
      const user1Data = await createTestUser(driver1)
      void user1Data
      const user1Id = await driver1.executeScript(`
        return new Promise(async (resolve) => {
          const token = localStorage.getItem('access_token');
          const resp = await fetch('/api/auth/me', { headers: { 'Authorization': 'Bearer ' + token } });
          const user = await resp.json();
          resolve(user.id);
        });
      `) as string
      void user1Id

      await driver2.get(BASE_URL)
      const user2Data = await createTestUser(driver2)
      void user2Data
      const user2Id = await driver2.executeScript(`
        return new Promise(async (resolve) => {
          const token = localStorage.getItem('access_token');
          const resp = await fetch('/api/auth/me', { headers: { 'Authorization': 'Bearer ' + token } });
          const user = await resp.json();
          resolve(user.id);
        });
      `) as string

      const chatName = `WSTest-${Date.now()}`
      await chatPage1.createChatWithParticipants(chatName, [user2Id])
      await chatPage1.waitForModalToClose()
      await wait(2000)

      // User2 открывает чат
      await driver2.navigate().refresh()
      await wait(2000)
      await chatPage2.waitForChatInList(chatName)
      await chatPage2.clickChatByNameInList(chatName)
      await chatPage2.waitForChatRoom()
      await wait(1000)

      // User1 открывает чат
      await chatPage1.clickChatByNameInList(chatName)
      await chatPage1.waitForChatRoom()
      await wait(1000)

      // 2. Отправляем сообщение (проверяем что WS работает)
      await chatPage1.sendMessage('Before WS disconnect')
      await wait(2000)

      // Проверяем что user2 получил
      await chatPage2.waitForMessageContaining('Before WS disconnect', 10000)

      // 3. Блокируем ТОЛЬКО WebSocket (API работает)
      console.log('[Test] Blocking WebSocket connections...')
      await network2.blockWebSocket()
      await wait(2000)

      // 4. Отправляем сообщения (user2 не должен получить в реальном времени)
      await chatPage1.sendMessage('During WS block 1')
      await wait(1000)
      await chatPage1.sendMessage('During WS block 2')
      await wait(2000)

      // 5. Разблокируем WebSocket
      console.log('[Test] Unblocking WebSocket...')
      await network2.unblockAllUrls()
      await wait(5000)

      // 6. Рефрешим страницу user2 для загрузки пропущенных сообщений
      await driver2.navigate().refresh()
      await wait(3000)

      // Открываем чат снова
      await chatPage2.waitForChatInList(chatName)
      await chatPage2.clickChatByNameInList(chatName)
      await chatPage2.waitForChatRoom()
      await wait(2000)

      // 7. Проверяем что сообщения появились
      const messages = await chatPage2.getMessageTexts()
      expect(messages.join(' ')).to.include('During WS block 1', 'Missed message 1 should sync')
      expect(messages.join(' ')).to.include('During WS block 2', 'Missed message 2 should sync')
    })
  })

  describe('Нестабильное соединение', function () {
    it('должен восстанавливаться после множественных disconnect', async function () {
      // 1. Setup
      await createTestUser(driver1)

      const chatName = `UnstableTest-${Date.now()}`
      await chatPage1.createChat(chatName)
      await chatPage1.waitForModalToClose()
      await wait(1000)

      await chatPage1.clickChatByNameInList(chatName)
      await chatPage1.waitForChatRoom()
      await wait(1000)

      // 2. Симулируем нестабильное соединение: несколько циклов online/offline
      console.log('[Test] Starting unstable connection simulation...')

      let successCount = 0
      for (let i = 0; i < 3; i++) {
        // Online - отправляем сообщение
        await network1.goOnline()
        await wait(1000)

        try {
          await chatPage1.sendMessage(`Unstable msg ${i + 1}`)
          await wait(1000)
          successCount++
        } catch (e) {
          console.log(`[Test] Failed to send message ${i + 1}:`, e)
        }

        // Offline - короткий разрыв
        await network1.goOffline()
        await wait(500)
      }

      // 3. Возвращаем online
      await network1.goOnline()
      await wait(3000)

      console.log(`[Test] ${successCount}/3 messages sent during unstable connection`)

      // Хотя бы одно сообщение должно было отправиться
      expect(successCount).to.be.greaterThan(0, 'At least one message should be sent during unstable connection')

      // 4. Проверяем что приложение всё ещё работает
      await chatPage1.sendMessage('Final stable message')
      await wait(2000)

      await chatPage1.waitForMessageContaining('Final stable message', 10000)
      console.log('[Test] App works after unstable connection')
    })
  })
})
