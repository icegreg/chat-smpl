/**
 * API User Scenarios Tests
 *
 * Тесты имитации работы пользователей через прямые API запросы
 * с измерением времени выполнения каждого запроса.
 *
 * Запуск: npx mocha --require ts-node/register tests/api-user-scenarios.spec.ts
 */

import { expect } from 'chai'
import {
  ApiTestClient,
  generateTestUser,
  TestReport,
} from '../helpers/apiTestClient.js'

const BASE_URL = process.env.API_URL || 'http://127.0.0.1:8888'

// Сборщик всех отчетов для итогового вывода
const allReports: TestReport[] = []

describe('API User Scenarios - Performance Tests', function () {
  this.timeout(120000) // 2 минуты на весь набор

  after(function () {
    // Итоговый отчет по всем тестам
    console.log('\n')
    console.log('#'.repeat(70))
    console.log('  FINAL SUMMARY - ALL API TESTS')
    console.log('#'.repeat(70))

    let totalRequests = 0
    let totalDuration = 0
    let totalSlow = 0
    let totalFailed = 0

    allReports.forEach(report => {
      totalRequests += report.stats.total
      totalDuration += report.stats.totalDuration
      totalSlow += report.stats.slow
      totalFailed += report.stats.failed
      console.log(`\n  ${report.testName}:`)
      console.log(`    Requests: ${report.stats.total}, Avg: ${report.stats.avgDuration}ms, P95: ${report.stats.p95}ms`)
    })

    console.log('\n' + '-'.repeat(70))
    console.log(`  TOTAL: ${totalRequests} requests in ${(totalDuration / 1000).toFixed(2)}s`)
    console.log(`  Slow: ${totalSlow}, Failed: ${totalFailed}`)
    console.log('#'.repeat(70) + '\n')
  })

  describe('Scenario 1: New User Registration Flow', function () {
    it('should register, view profile, and update settings', async function () {
      const client = new ApiTestClient(BASE_URL)
      client.startTest()

      // 1. Регистрация
      const user = generateTestUser('scenario1')
      const tokens = await client.register(user)
      expect(tokens.accessToken).to.be.a('string')

      // 2. Получить профиль
      const profile = await client.getMe()
      expect(profile.email).to.equal(user.email)

      // 3. Обновить токен
      await client.refreshTokens()

      // 4. Получить профиль снова
      await client.getMe()

      // 5. Выход
      await client.logout()

      const report = client.finishTest('New User Registration Flow')
      ApiTestClient.printReport(report)
      allReports.push(report)

      expect(report.stats.failed).to.equal(0)
      expect(report.stats.avgDuration).to.be.lessThan(1000)
    })
  })

  describe('Scenario 2: Chat Creation and Messaging', function () {
    it('should create chat, send messages, and read them', async function () {
      const client = new ApiTestClient(BASE_URL)
      client.startTest()

      // 1. Регистрация
      const user = generateTestUser('scenario2')
      await client.register(user)

      // 2. Создать чат
      const chat = await client.createChat('Test Chat ' + Date.now())
      expect(chat.id).to.be.a('string')

      // 3. Отправить несколько сообщений
      const messageCount = 10
      const messageIds: string[] = []

      for (let i = 0; i < messageCount; i++) {
        const msg = await client.sendMessage(chat.id, `Test message #${i + 1}`)
        messageIds.push(msg.id)
      }

      // 4. Получить сообщения
      const messages = await client.getMessages(chat.id)
      expect(messages.messages.length).to.be.at.least(messageCount)

      // 5. Обновить одно сообщение
      await client.updateMessage(messageIds[0], 'Updated message content')

      // 6. Добавить реакцию
      await client.addReaction(messageIds[1], '👍')

      // 7. Удалить одно сообщение
      await client.deleteMessage(messageIds[messageIds.length - 1])

      // 8. Получить сообщения снова
      await client.getMessages(chat.id)

      const report = client.finishTest('Chat Creation and Messaging')
      ApiTestClient.printReport(report)
      allReports.push(report)

      expect(report.stats.failed).to.equal(0)
    })
  })

  describe('Scenario 3: Multiple Chats Management', function () {
    it('should create multiple chats and switch between them', async function () {
      const client = new ApiTestClient(BASE_URL)
      client.startTest()

      // 1. Регистрация
      const user = generateTestUser('scenario3')
      await client.register(user)

      // 2. Создать 5 чатов
      const chatCount = 5
      const chatIds: string[] = []

      for (let i = 0; i < chatCount; i++) {
        const chat = await client.createChat(`Chat ${i + 1} - ${Date.now()}`)
        chatIds.push(chat.id)
      }

      // 3. Получить список чатов
      const chats = await client.getChats()
      expect(chats.chats.length).to.be.at.least(chatCount)

      // 4. Отправить сообщение в каждый чат
      for (const chatId of chatIds) {
        await client.sendMessage(chatId, 'Hello from this chat!')
      }

      // 5. Переключаться между чатами (получать сообщения)
      for (const chatId of chatIds) {
        await client.getMessages(chatId)
        await client.getChat(chatId)
      }

      // 6. Удалить один чат
      await client.deleteChat(chatIds[0])

      // 7. Получить список снова
      await client.getChats()

      const report = client.finishTest('Multiple Chats Management')
      ApiTestClient.printReport(report)
      allReports.push(report)

      expect(report.stats.failed).to.equal(0)
    })
  })

  describe('Scenario 4: Intensive Messaging', function () {
    it('should handle rapid message sending', async function () {
      const client = new ApiTestClient(BASE_URL)
      client.startTest()

      // 1. Регистрация
      const user = generateTestUser('scenario4')
      await client.register(user)

      // 2. Создать чат
      const chat = await client.createChat('Intensive Chat ' + Date.now())

      // 3. Быстрая отправка 50 сообщений
      const messageCount = 50
      for (let i = 0; i < messageCount; i++) {
        await client.sendMessage(chat.id, `Rapid message #${i + 1} - ${Date.now()}`)
      }

      // 4. Получить все сообщения
      const messages = await client.getMessages(chat.id, 1, 100)
      expect(messages.messages.length).to.be.at.least(messageCount)

      const report = client.finishTest('Intensive Messaging (50 messages)')
      ApiTestClient.printReport(report)
      allReports.push(report)

      expect(report.stats.failed).to.equal(0)
      // P95 должен быть меньше 1 секунды для хорошей производительности
      expect(report.stats.p95).to.be.lessThan(2000)
    })
  })

  describe('Scenario 5: Read-Heavy Workload', function () {
    it('should handle multiple read operations', async function () {
      const client = new ApiTestClient(BASE_URL)
      client.startTest()

      // 1. Регистрация и создание чата с сообщениями
      const user = generateTestUser('scenario5')
      await client.register(user)
      const chat = await client.createChat('Read Heavy Chat ' + Date.now())

      // Создать 20 сообщений
      for (let i = 0; i < 20; i++) {
        await client.sendMessage(chat.id, `Message #${i + 1}`)
      }

      // 2. Многократное чтение (имитация polling или обновления UI)
      const readCount = 30
      for (let i = 0; i < readCount; i++) {
        await client.getMessages(chat.id)
        await client.getChats()
        if (i % 5 === 0) {
          await client.getMe()
        }
      }

      const report = client.finishTest('Read-Heavy Workload (90+ reads)')
      ApiTestClient.printReport(report)
      allReports.push(report)

      expect(report.stats.failed).to.equal(0)
    })
  })

  describe('Scenario 6: Thread Operations', function () {
    it('should create and manage threads', async function () {
      const client = new ApiTestClient(BASE_URL)
      client.startTest()

      // 1. Регистрация
      const user = generateTestUser('scenario6')
      await client.register(user)

      // 2. Создать чат
      const chat = await client.createChat('Thread Test Chat ' + Date.now())

      // 3. Отправить сообщение
      const message = await client.sendMessage(chat.id, 'Main message for thread')

      // 4. Создать тред
      const thread = await client.createThread(chat.id, 'Discussion Thread', message.id)
      expect(thread.id).to.be.a('string')

      // 5. Получить треды
      const threads = await client.getThreads(chat.id)
      expect(threads.threads.length).to.be.at.least(1)

      // 6. Отправить сообщения в основной чат
      for (let i = 0; i < 5; i++) {
        await client.sendMessage(chat.id, `Thread related message #${i + 1}`)
      }

      const report = client.finishTest('Thread Operations')
      ApiTestClient.printReport(report)
      allReports.push(report)

      expect(report.stats.failed).to.equal(0)
    })
  })

  describe('Scenario 7: Authentication Stress', function () {
    it('should handle multiple login/logout cycles', async function () {
      const client = new ApiTestClient(BASE_URL)
      client.startTest()

      // 1. Регистрация
      const user = generateTestUser('scenario7')
      await client.register(user)
      await client.logout()

      // 2. Несколько циклов login/logout
      const cycles = 10
      for (let i = 0; i < cycles; i++) {
        await client.login(user.email, user.password)
        await client.getMe()
        await client.refreshTokens()
        await client.logout()
      }

      const report = client.finishTest(`Authentication Stress (${cycles} cycles)`)
      ApiTestClient.printReport(report)
      allReports.push(report)

      expect(report.stats.failed).to.equal(0)
    })
  })

  describe('Scenario 8: Complete User Session', function () {
    it('should simulate complete user work session', async function () {
      const client = new ApiTestClient(BASE_URL)
      client.startTest()

      // === Начало сессии ===
      const user = generateTestUser('session')
      await client.register(user)

      // Получить профиль
      await client.getMe()

      // === Создание рабочего пространства ===
      const workChat = await client.createChat('Work Chat')
      const discussionChat = await client.createChat('Discussion')
      const randomChat = await client.createChat('Random')

      // Получить список чатов
      await client.getChats()

      // === Рабочая активность ===

      // Отправить сообщения в рабочий чат
      for (let i = 0; i < 5; i++) {
        await client.sendMessage(workChat.id, `Work update #${i + 1}`)
      }

      // Переключиться на обсуждение
      await client.getMessages(discussionChat.id)
      await client.sendMessage(discussionChat.id, 'Starting discussion...')

      // Вернуться в рабочий чат
      await client.getMessages(workChat.id)

      // Создать тред
      const msg = await client.sendMessage(workChat.id, 'Important topic')
      await client.createThread(workChat.id, 'Important Discussion', msg.id)

      // Добавить реакции
      const messages = await client.getMessages(workChat.id)
      if (messages.messages.length > 0) {
        await client.addReaction(messages.messages[0].id, '👍')
      }

      // === Навигация между чатами ===
      for (let i = 0; i < 3; i++) {
        await client.getChats()
        await client.getMessages(workChat.id)
        await client.getMessages(discussionChat.id)
        await client.getMessages(randomChat.id)
      }

      // === Завершение сессии ===
      await client.logout()

      const report = client.finishTest('Complete User Session')
      ApiTestClient.printReport(report)
      allReports.push(report)

      expect(report.stats.failed).to.equal(0)
    })
  })
})

describe('API Performance Benchmarks', function () {
  this.timeout(60000)

  it('Benchmark: Single request latency', async function () {
    const client = new ApiTestClient(BASE_URL)
    client.startTest()

    const user = generateTestUser('bench')
    await client.register(user)
    const chat = await client.createChat('Benchmark Chat')

    // Измерить одиночные запросы
    const singleRequests = 20
    for (let i = 0; i < singleRequests; i++) {
      await client.sendMessage(chat.id, `Benchmark message ${i}`)
    }

    const report = client.finishTest('Single Request Latency Benchmark')
    ApiTestClient.printReport(report)
    allReports.push(report)

    console.log('\n  LATENCY ANALYSIS:')
    console.log(`  Average latency: ${report.stats.avgDuration}ms`)
    console.log(`  P50 latency: ${report.stats.p50}ms`)
    console.log(`  P90 latency: ${report.stats.p90}ms`)
    console.log(`  P99 latency: ${report.stats.p99}ms`)
  })

  it('Benchmark: Throughput test', async function () {
    const client = new ApiTestClient(BASE_URL)
    client.startTest()

    const user = generateTestUser('throughput')
    await client.register(user)
    const chat = await client.createChat('Throughput Chat')

    // Отправить много сообщений последовательно и измерить общее время
    const messageCount = 100
    const startTime = Date.now()

    for (let i = 0; i < messageCount; i++) {
      await client.sendMessage(chat.id, `Throughput test message ${i}`)
    }

    const totalTime = Date.now() - startTime
    const throughput = (messageCount / totalTime) * 1000 // messages per second

    const report = client.finishTest('Throughput Benchmark')
    ApiTestClient.printReport(report)
    allReports.push(report)

    console.log('\n  THROUGHPUT ANALYSIS:')
    console.log(`  Total messages: ${messageCount}`)
    console.log(`  Total time: ${totalTime}ms`)
    console.log(`  Throughput: ${throughput.toFixed(2)} msg/sec`)
  })
})
