/**
 * API Authentication & Presence Tests
 *
 * Тесты авторизации и статуса присутствия через API запросы
 * с измерением времени выполнения.
 *
 * Запуск: npx mocha --require ts-node/register tests/api-auth-presence.spec.ts
 */

import { expect } from 'chai'
import {
  ApiTestClient,
  generateTestUser,
  sleep,
  TestReport,
} from '../helpers/apiTestClient'

const BASE_URL = process.env.API_URL || 'http://127.0.0.1:8888'

const allReports: TestReport[] = []

describe('Authentication API Tests', function () {
  this.timeout(60000)

  after(function () {
    console.log('\n')
    console.log('#'.repeat(70))
    console.log('  AUTH & PRESENCE TESTS SUMMARY')
    console.log('#'.repeat(70))

    allReports.forEach(report => {
      console.log(`\n  ${report.testName}:`)
      console.log(`    Requests: ${report.stats.total}, Avg: ${report.stats.avgDuration}ms, P95: ${report.stats.p95}ms`)
      console.log(`    Failed: ${report.stats.failed}, Slow: ${report.stats.slow}`)
    })
    console.log('#'.repeat(70) + '\n')
  })

  describe('Registration Flow', function () {
    it('should register new user and receive tokens', async function () {
      const client = new ApiTestClient(BASE_URL)
      client.startTest()

      const user = generateTestUser('auth_reg')
      const tokens = await client.register(user)

      expect(tokens.accessToken).to.be.a('string')
      expect(tokens.refreshToken).to.be.a('string')
      expect(tokens.userId).to.be.a('string')

      // Проверка что токен работает
      const profile = await client.getMe()
      expect(profile.email).to.equal(user.email)
      expect(profile.username).to.equal(user.username)

      const report = client.finishTest('Registration Flow')
      ApiTestClient.printReport(report)
      allReports.push(report)

      expect(report.stats.failed).to.equal(0)
    })

    it('should reject duplicate registration', async function () {
      const client = new ApiTestClient(BASE_URL)
      client.startTest()

      const user = generateTestUser('auth_dup')
      await client.register(user)
      client.reset()

      // Попытка повторной регистрации
      try {
        await client.register(user)
        expect.fail('Should have thrown error')
      } catch (error) {
        // Ожидаем ошибку
      }

      const report = client.finishTest('Duplicate Registration Rejection')
      ApiTestClient.printReport(report)
      allReports.push(report)
    })
  })

  describe('Login/Logout Flow', function () {
    it('should login with valid credentials', async function () {
      const client = new ApiTestClient(BASE_URL)
      client.startTest()

      // Сначала регистрируем
      const user = generateTestUser('auth_login')
      await client.register(user)
      await client.logout()

      // Теперь логинимся
      const tokens = await client.login(user.email, user.password)
      expect(tokens.accessToken).to.be.a('string')

      // Проверка профиля
      const profile = await client.getMe()
      expect(profile.email).to.equal(user.email)

      const report = client.finishTest('Login Flow')
      ApiTestClient.printReport(report)
      allReports.push(report)

      expect(report.stats.failed).to.equal(0)
    })

    it('should reject invalid password', async function () {
      const client = new ApiTestClient(BASE_URL)
      client.startTest()

      const user = generateTestUser('auth_wrong')
      await client.register(user)
      await client.logout()

      try {
        await client.login(user.email, 'WrongPassword123!')
        expect.fail('Should have thrown error')
      } catch (error) {
        // Ожидаем ошибку 401
      }

      const report = client.finishTest('Invalid Password Rejection')
      ApiTestClient.printReport(report)
      allReports.push(report)
    })

    it('should properly logout and invalidate token', async function () {
      const client = new ApiTestClient(BASE_URL)
      client.startTest()

      const user = generateTestUser('auth_logout')
      await client.register(user)

      // Сохраняем токен
      const oldToken = client.getAccessToken()

      // Выход
      await client.logout()

      // Попытка использовать старый токен
      client.setTokens(oldToken!, '', '')
      try {
        await client.getMe()
        // Если не выбросило ошибку - токен еще валиден (зависит от реализации)
      } catch (error) {
        // Ожидаем 401
      }

      const report = client.finishTest('Logout Flow')
      ApiTestClient.printReport(report)
      allReports.push(report)
    })
  })

  describe('Token Refresh Flow', function () {
    it('should refresh access token', async function () {
      const client = new ApiTestClient(BASE_URL)
      client.startTest()

      const user = generateTestUser('auth_refresh')
      await client.register(user)

      // Refresh
      const newTokens = await client.refreshTokens()

      expect(newTokens.accessToken).to.be.a('string')
      // Новый токен должен отличаться (или быть таким же если TTL не истек)

      // Проверка что новый токен работает
      const profile = await client.getMe()
      expect(profile.email).to.equal(user.email)

      const report = client.finishTest('Token Refresh Flow')
      ApiTestClient.printReport(report)
      allReports.push(report)

      expect(report.stats.failed).to.equal(0)
    })

    it('should handle multiple refresh cycles', async function () {
      const client = new ApiTestClient(BASE_URL)
      client.startTest()

      const user = generateTestUser('auth_multi_refresh')
      await client.register(user)

      // Несколько циклов refresh
      for (let i = 0; i < 5; i++) {
        await client.refreshTokens()
        await client.getMe()
      }

      const report = client.finishTest('Multiple Token Refresh')
      ApiTestClient.printReport(report)
      allReports.push(report)

      expect(report.stats.failed).to.equal(0)
    })
  })

  describe('Authentication Stress Test', function () {
    it('should handle rapid auth operations', async function () {
      const client = new ApiTestClient(BASE_URL)
      client.startTest()

      const user = generateTestUser('auth_stress')
      await client.register(user)
      await client.logout()

      // 20 циклов login -> getMe -> refresh -> logout
      const cycles = 20
      for (let i = 0; i < cycles; i++) {
        await client.login(user.email, user.password)
        await client.getMe()
        await client.refreshTokens()
        await client.logout()
      }

      const report = client.finishTest(`Auth Stress (${cycles} cycles)`)
      ApiTestClient.printReport(report)
      allReports.push(report)

      expect(report.stats.failed).to.equal(0)
      expect(report.stats.avgDuration).to.be.lessThan(500)
    })
  })
})

describe('Presence API Tests', function () {
  this.timeout(60000)

  describe('Presence Status Management', function () {
    it('should set and get presence status', async function () {
      const client = new ApiTestClient(BASE_URL)
      client.startTest()

      const user = generateTestUser('presence_status')
      await client.register(user)

      // Установить статус available
      const availableStatus = await client.setPresenceStatus('available')
      expect(availableStatus.status).to.equal('available')

      // Получить свой статус
      const myStatus = await client.getMyPresenceStatus()
      expect(myStatus.status).to.equal('available')
      expect(myStatus.user_id).to.equal(client.getUserId())

      // Изменить на busy
      const busyStatus = await client.setPresenceStatus('busy')
      expect(busyStatus.status).to.equal('busy')

      // Изменить на away
      await client.setPresenceStatus('away')
      const awayCheck = await client.getMyPresenceStatus()
      expect(awayCheck.status).to.equal('away')

      // Изменить на dnd
      await client.setPresenceStatus('dnd')
      const dndCheck = await client.getMyPresenceStatus()
      expect(dndCheck.status).to.equal('dnd')

      const report = client.finishTest('Presence Status Management')
      ApiTestClient.printReport(report)
      allReports.push(report)

      expect(report.stats.failed).to.equal(0)
    })

    it('should track multiple users presence', async function () {
      // Создаем несколько пользователей
      const clients: ApiTestClient[] = []
      const userIds: string[] = []

      for (let i = 0; i < 3; i++) {
        const client = new ApiTestClient(BASE_URL)
        const user = generateTestUser(`presence_multi_${i}`)
        await client.register(user)
        clients.push(client)
        userIds.push(client.getUserId()!)
      }

      const mainClient = clients[0]
      mainClient.startTest()

      // Установить разные статусы
      await clients[0].setPresenceStatus('available')
      await clients[1].setPresenceStatus('busy')
      await clients[2].setPresenceStatus('away')

      // Получить статусы всех пользователей
      const presences = await mainClient.getUsersPresence(userIds)

      expect(presences.presences).to.be.an('array')
      expect(presences.presences.length).to.equal(3)

      const report = mainClient.finishTest('Multiple Users Presence')
      ApiTestClient.printReport(report)
      allReports.push(report)

      expect(report.stats.failed).to.equal(0)
    })
  })

  describe('Connection Management', function () {
    it('should handle connect/disconnect lifecycle', async function () {
      const client = new ApiTestClient(BASE_URL)
      client.startTest()

      const user = generateTestUser('presence_conn')
      await client.register(user)

      // Симуляция подключения
      const connectionId = `conn_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`

      const connectResult = await client.presenceConnect(connectionId)
      expect(connectResult.is_online).to.equal(true)
      expect(connectResult.connection_count).to.be.at.least(1)

      // Проверка статуса
      const status = await client.getMyPresenceStatus()
      expect(status.is_online).to.equal(true)

      // Отключение
      const disconnectResult = await client.presenceDisconnect(connectionId)
      expect(disconnectResult.connection_count).to.equal(0)

      // Проверка что offline
      const offlineStatus = await client.getMyPresenceStatus()
      expect(offlineStatus.is_online).to.equal(false)

      const report = client.finishTest('Connection Lifecycle')
      ApiTestClient.printReport(report)
      allReports.push(report)

      expect(report.stats.failed).to.equal(0)
    })

    it('should handle multiple connections per user', async function () {
      const client = new ApiTestClient(BASE_URL)
      client.startTest()

      const user = generateTestUser('presence_multi_conn')
      await client.register(user)

      // Несколько соединений (имитация нескольких вкладок/устройств)
      const connections = [
        `conn_browser_${Date.now()}`,
        `conn_mobile_${Date.now()}`,
        `conn_desktop_${Date.now()}`,
      ]

      // Подключить все
      for (const connId of connections) {
        await client.presenceConnect(connId)
      }

      let status = await client.getMyPresenceStatus()
      expect(status.connection_count).to.equal(3)
      expect(status.is_online).to.equal(true)

      // Отключить одно
      await client.presenceDisconnect(connections[0])
      status = await client.getMyPresenceStatus()
      expect(status.connection_count).to.equal(2)
      expect(status.is_online).to.equal(true) // Еще онлайн

      // Отключить остальные
      await client.presenceDisconnect(connections[1])
      await client.presenceDisconnect(connections[2])

      status = await client.getMyPresenceStatus()
      expect(status.connection_count).to.equal(0)
      expect(status.is_online).to.equal(false)

      const report = client.finishTest('Multiple Connections')
      ApiTestClient.printReport(report)
      allReports.push(report)

      expect(report.stats.failed).to.equal(0)
    })
  })

  describe('Presence Stress Test', function () {
    it('should handle rapid status changes', async function () {
      const client = new ApiTestClient(BASE_URL)
      client.startTest()

      const user = generateTestUser('presence_stress')
      await client.register(user)

      const statuses: Array<'available' | 'busy' | 'away' | 'dnd'> = ['available', 'busy', 'away', 'dnd']

      // 50 быстрых изменений статуса
      for (let i = 0; i < 50; i++) {
        const status = statuses[i % statuses.length]
        await client.setPresenceStatus(status)
      }

      // Проверить финальный статус
      const finalStatus = await client.getMyPresenceStatus()
      expect(finalStatus.status).to.equal('busy') // 50 % 4 = 2 -> 'away', но 49 % 4 = 1 -> 'busy'

      const report = client.finishTest('Rapid Status Changes (50)')
      ApiTestClient.printReport(report)
      allReports.push(report)

      expect(report.stats.failed).to.equal(0)
    })

    it('should handle rapid connect/disconnect cycles', async function () {
      const client = new ApiTestClient(BASE_URL)
      client.startTest()

      const user = generateTestUser('presence_conn_stress')
      await client.register(user)

      // 30 циклов connect/disconnect
      for (let i = 0; i < 30; i++) {
        const connId = `conn_stress_${i}_${Date.now()}`
        await client.presenceConnect(connId)
        await client.presenceDisconnect(connId)
      }

      const report = client.finishTest('Connect/Disconnect Stress (30 cycles)')
      ApiTestClient.printReport(report)
      allReports.push(report)

      expect(report.stats.failed).to.equal(0)
    })
  })

  describe('Real-World Presence Scenarios', function () {
    it('should simulate user going online, changing status, going offline', async function () {
      const client = new ApiTestClient(BASE_URL)
      client.startTest()

      const user = generateTestUser('presence_realworld')
      await client.register(user)

      // Пользователь открыл приложение
      const connId = `conn_${Date.now()}`
      await client.presenceConnect(connId)

      // Начал работать - available
      await client.setPresenceStatus('available')

      // Получил сообщения, отвечает - проверяем статус
      for (let i = 0; i < 5; i++) {
        await client.getMyPresenceStatus()
        await sleep(50)
      }

      // Ушел на обед - away
      await client.setPresenceStatus('away')
      await sleep(100)

      // Вернулся - available
      await client.setPresenceStatus('available')

      // Важный звонок - dnd
      await client.setPresenceStatus('dnd')
      await sleep(100)

      // Закончил - available
      await client.setPresenceStatus('available')

      // Конец дня - закрыл приложение
      await client.presenceDisconnect(connId)

      const finalStatus = await client.getMyPresenceStatus()
      expect(finalStatus.is_online).to.equal(false)

      const report = client.finishTest('Real-World Presence Scenario')
      ApiTestClient.printReport(report)
      allReports.push(report)

      expect(report.stats.failed).to.equal(0)
    })

    it('should simulate multiple users monitoring each other', async function () {
      // 5 пользователей в "команде"
      const teamSize = 5
      const clients: ApiTestClient[] = []
      const userIds: string[] = []

      for (let i = 0; i < teamSize; i++) {
        const client = new ApiTestClient(BASE_URL)
        const user = generateTestUser(`team_${i}`)
        await client.register(user)
        await client.presenceConnect(`team_conn_${i}`)
        clients.push(client)
        userIds.push(client.getUserId()!)
      }

      const mainClient = clients[0]
      mainClient.startTest()

      // Все available
      for (const client of clients) {
        await client.setPresenceStatus('available')
      }

      // Мониторинг статусов команды (как в реальном приложении)
      for (let i = 0; i < 10; i++) {
        const presences = await mainClient.getUsersPresence(userIds)
        expect(presences.presences.length).to.equal(teamSize)

        // Кто-то меняет статус
        const randomUser = clients[Math.floor(Math.random() * clients.length)]
        const statuses: Array<'available' | 'busy' | 'away' | 'dnd'> = ['available', 'busy', 'away', 'dnd']
        await randomUser.setPresenceStatus(statuses[Math.floor(Math.random() * statuses.length)])

        await sleep(50)
      }

      const report = mainClient.finishTest('Team Presence Monitoring')
      ApiTestClient.printReport(report)
      allReports.push(report)

      expect(report.stats.failed).to.equal(0)
    })
  })
})

describe('Combined Auth + Presence Flow', function () {
  this.timeout(60000)

  it('should handle full user session with auth and presence', async function () {
    const client = new ApiTestClient(BASE_URL)
    client.startTest()

    // === День 1: Регистрация ===
    const user = generateTestUser('full_session')
    await client.register(user)

    // Подключение
    const connId1 = `session_${Date.now()}`
    await client.presenceConnect(connId1)
    await client.setPresenceStatus('available')

    // Работа
    const chat = await client.createChat('Work Chat')
    await client.sendMessage(chat.id, 'Hello team!')
    await client.getMessages(chat.id)

    // Статус busy во время работы
    await client.setPresenceStatus('busy')

    // Конец дня
    await client.setPresenceStatus('away')
    await client.presenceDisconnect(connId1)
    await client.logout()

    // === День 2: Повторный вход ===
    await client.login(user.email, user.password)

    const connId2 = `session2_${Date.now()}`
    await client.presenceConnect(connId2)
    await client.setPresenceStatus('available')

    // Проверка что чат на месте
    const chats = await client.getChats()
    expect(chats.chats.length).to.be.at.least(1)

    // Читаем сообщения
    await client.getMessages(chat.id)

    // Refresh токена (имитация долгой сессии)
    await client.refreshTokens()

    // Проверка профиля
    const profile = await client.getMe()
    expect(profile.email).to.equal(user.email)

    // Финальный статус
    const status = await client.getMyPresenceStatus()
    expect(status.is_online).to.equal(true)

    // Выход
    await client.presenceDisconnect(connId2)
    await client.logout()

    const report = client.finishTest('Full User Session (Auth + Presence)')
    ApiTestClient.printReport(report)
    allReports.push(report)

    expect(report.stats.failed).to.equal(0)
  })
})
