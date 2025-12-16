/**
 * API Multi-User Simulation Tests
 *
 * Тесты имитации работы нескольких пользователей одновременно
 * через параллельные API запросы с измерением времени.
 *
 * Запуск: npx mocha --require ts-node/register tests/api-multiuser-simulation.spec.ts
 */

import { expect } from 'chai'
import {
  ApiTestClient,
  generateTestUser,
  sleep,
  TestReport,
} from '../helpers/apiTestClient'

const BASE_URL = process.env.API_URL || 'http://127.0.0.1:8888'

// Сборщик отчетов
const allReports: TestReport[] = []

/**
 * Создать и авторизовать пользователя
 */
async function createUser(prefix: string): Promise<ApiTestClient> {
  const client = new ApiTestClient(BASE_URL)
  const user = generateTestUser(prefix)
  await client.register(user)
  return client
}

/**
 * Собрать статистику со всех клиентов
 */
function aggregateReports(reports: TestReport[]): {
  totalRequests: number
  totalDuration: number
  avgLatency: number
  maxLatency: number
  slowRequests: number
  failedRequests: number
} {
  let totalRequests = 0
  let totalDuration = 0
  let maxLatency = 0
  let slowRequests = 0
  let failedRequests = 0
  let allDurations: number[] = []

  reports.forEach(report => {
    totalRequests += report.stats.total
    totalDuration += report.stats.totalDuration
    maxLatency = Math.max(maxLatency, report.stats.maxDuration)
    slowRequests += report.stats.slow
    failedRequests += report.stats.failed
    allDurations.push(...report.allRequests.map(r => r.duration))
  })

  return {
    totalRequests,
    totalDuration,
    avgLatency: Math.round(allDurations.reduce((a, b) => a + b, 0) / allDurations.length),
    maxLatency,
    slowRequests,
    failedRequests,
  }
}

describe('Multi-User Simulation Tests', function () {
  this.timeout(300000) // 5 минут

  after(function () {
    console.log('\n')
    console.log('#'.repeat(70))
    console.log('  MULTI-USER SIMULATION SUMMARY')
    console.log('#'.repeat(70))

    allReports.forEach(report => {
      console.log(`\n  ${report.testName}:`)
      console.log(`    Requests: ${report.stats.total}, Avg: ${report.stats.avgDuration}ms`)
      console.log(`    Failed: ${report.stats.failed}, Slow: ${report.stats.slow}`)
    })
    console.log('#'.repeat(70) + '\n')
  })

  describe('Concurrent Users in Same Chat', function () {
    it('should handle 5 users chatting simultaneously', async function () {
      const userCount = 5
      const messagesPerUser = 10

      console.log(`\n  Setting up ${userCount} users...`)

      // Создать пользователей
      const clients: ApiTestClient[] = []
      for (let i = 0; i < userCount; i++) {
        const client = await createUser(`multi5_user${i}`)
        client.startTest()
        clients.push(client)
      }

      // Первый пользователь создает чат
      const chat = await clients[0].createChat('Multi-User Chat ' + Date.now())
      const chatId = chat.id

      // Добавить остальных участников
      for (let i = 1; i < userCount; i++) {
        await clients[0].addParticipant(chatId, clients[i].getUserId()!)
      }

      console.log(`  Chat created: ${chatId}`)
      console.log(`  Starting parallel messaging...`)

      // Параллельная отправка сообщений
      const sendPromises: Promise<any>[] = []

      for (let round = 0; round < messagesPerUser; round++) {
        for (let userIdx = 0; userIdx < userCount; userIdx++) {
          sendPromises.push(
            clients[userIdx].sendMessage(chatId, `User ${userIdx} - Message ${round + 1}`)
          )
        }
      }

      await Promise.all(sendPromises)

      // Все читают сообщения
      const readPromises = clients.map(client => client.getMessages(chatId, 1, 100))
      const results = await Promise.all(readPromises)

      // Проверить что все получили сообщения
      results.forEach((res, idx) => {
        expect(res.messages.length).to.be.at.least(userCount * messagesPerUser)
      })

      // Собрать отчеты
      const reports = clients.map((client, i) => client.finishTest(`User ${i}`))
      const aggregate = aggregateReports(reports)

      console.log('\n' + '='.repeat(70))
      console.log('  5 CONCURRENT USERS - AGGREGATE RESULTS')
      console.log('='.repeat(70))
      console.log(`  Total Requests: ${aggregate.totalRequests}`)
      console.log(`  Total Duration: ${aggregate.totalDuration}ms`)
      console.log(`  Average Latency: ${aggregate.avgLatency}ms`)
      console.log(`  Max Latency: ${aggregate.maxLatency}ms`)
      console.log(`  Slow Requests: ${aggregate.slowRequests}`)
      console.log(`  Failed Requests: ${aggregate.failedRequests}`)
      console.log('='.repeat(70) + '\n')

      // Добавить сводный отчет
      allReports.push({
        testName: '5 Concurrent Users',
        startTime: new Date(),
        endTime: new Date(),
        totalDuration: aggregate.totalDuration,
        stats: {
          total: aggregate.totalRequests,
          successful: aggregate.totalRequests - aggregate.failedRequests,
          failed: aggregate.failedRequests,
          excellent: 0,
          good: 0,
          acceptable: 0,
          slow: aggregate.slowRequests,
          avgDuration: aggregate.avgLatency,
          maxDuration: aggregate.maxLatency,
          minDuration: 0,
          p50: 0,
          p90: 0,
          p95: 0,
          p99: 0,
          totalDuration: aggregate.totalDuration,
        },
        slowRequests: [],
        failedRequests: [],
        allRequests: [],
      })

      expect(aggregate.failedRequests).to.equal(0)
    })
  })

  describe('Concurrent Users - Different Chats', function () {
    it('should handle 10 users in separate chats', async function () {
      const userCount = 10
      const messagesPerUser = 20

      console.log(`\n  Setting up ${userCount} users with separate chats...`)

      // Создать пользователей
      const clients: ApiTestClient[] = []
      const chatIds: string[] = []

      for (let i = 0; i < userCount; i++) {
        const client = await createUser(`multi10_user${i}`)
        client.startTest()
        clients.push(client)

        const chat = await client.createChat(`User ${i} Chat`)
        chatIds.push(chat.id)
      }

      console.log(`  Starting parallel activity...`)

      // Параллельная активность
      const activityPromises: Promise<any>[] = []

      for (let i = 0; i < userCount; i++) {
        // Каждый пользователь отправляет сообщения в свой чат
        for (let j = 0; j < messagesPerUser; j++) {
          activityPromises.push(
            clients[i].sendMessage(chatIds[i], `Message ${j + 1} from user ${i}`)
          )
        }
      }

      await Promise.all(activityPromises)

      // Все читают свои чаты
      const readPromises = clients.map((client, i) => client.getMessages(chatIds[i]))
      await Promise.all(readPromises)

      // Собрать отчеты
      const reports = clients.map((client, i) => client.finishTest(`User ${i}`))
      const aggregate = aggregateReports(reports)

      console.log('\n' + '='.repeat(70))
      console.log('  10 USERS IN SEPARATE CHATS - AGGREGATE RESULTS')
      console.log('='.repeat(70))
      console.log(`  Total Requests: ${aggregate.totalRequests}`)
      console.log(`  Average Latency: ${aggregate.avgLatency}ms`)
      console.log(`  Max Latency: ${aggregate.maxLatency}ms`)
      console.log(`  Failed Requests: ${aggregate.failedRequests}`)
      console.log('='.repeat(70) + '\n')

      allReports.push({
        testName: '10 Users Separate Chats',
        startTime: new Date(),
        endTime: new Date(),
        totalDuration: aggregate.totalDuration,
        stats: {
          total: aggregate.totalRequests,
          successful: aggregate.totalRequests - aggregate.failedRequests,
          failed: aggregate.failedRequests,
          excellent: 0,
          good: 0,
          acceptable: 0,
          slow: aggregate.slowRequests,
          avgDuration: aggregate.avgLatency,
          maxDuration: aggregate.maxLatency,
          minDuration: 0,
          p50: 0,
          p90: 0,
          p95: 0,
          p99: 0,
          totalDuration: aggregate.totalDuration,
        },
        slowRequests: [],
        failedRequests: [],
        allRequests: [],
      })

      expect(aggregate.failedRequests).to.equal(0)
    })
  })

  describe('Burst Traffic Simulation', function () {
    it('should handle burst of registrations', async function () {
      const burstSize = 20

      console.log(`\n  Simulating burst of ${burstSize} registrations...`)

      const clients: ApiTestClient[] = []
      const registrationPromises: Promise<ApiTestClient>[] = []

      // Создать burst регистраций
      for (let i = 0; i < burstSize; i++) {
        const promise = (async () => {
          const client = new ApiTestClient(BASE_URL)
          client.startTest()
          const user = generateTestUser(`burst_${i}`)
          await client.register(user)
          return client
        })()
        registrationPromises.push(promise)
      }

      const results = await Promise.allSettled(registrationPromises)

      const successful = results.filter(r => r.status === 'fulfilled').length
      const failed = results.filter(r => r.status === 'rejected').length

      results.forEach(r => {
        if (r.status === 'fulfilled') {
          clients.push(r.value)
        }
      })

      console.log(`  Successful: ${successful}, Failed: ${failed}`)

      // Собрать статистику регистраций
      const reports = clients.map((c, i) => c.finishTest(`Burst User ${i}`))
      const aggregate = aggregateReports(reports)

      console.log('\n' + '='.repeat(70))
      console.log('  BURST REGISTRATION - RESULTS')
      console.log('='.repeat(70))
      console.log(`  Attempted: ${burstSize}`)
      console.log(`  Successful: ${successful}`)
      console.log(`  Failed: ${failed}`)
      console.log(`  Average Latency: ${aggregate.avgLatency}ms`)
      console.log(`  Max Latency: ${aggregate.maxLatency}ms`)
      console.log('='.repeat(70) + '\n')

      allReports.push({
        testName: 'Burst Registration',
        startTime: new Date(),
        endTime: new Date(),
        totalDuration: aggregate.totalDuration,
        stats: {
          total: aggregate.totalRequests,
          successful: successful,
          failed: failed,
          excellent: 0,
          good: 0,
          acceptable: 0,
          slow: aggregate.slowRequests,
          avgDuration: aggregate.avgLatency,
          maxDuration: aggregate.maxLatency,
          minDuration: 0,
          p50: 0,
          p90: 0,
          p95: 0,
          p99: 0,
          totalDuration: aggregate.totalDuration,
        },
        slowRequests: [],
        failedRequests: [],
        allRequests: [],
      })

      expect(successful).to.be.at.least(burstSize * 0.9) // 90% должны успешно зарегистрироваться
    })
  })

  describe('Real-time Chat Simulation', function () {
    it('should simulate real-time conversation between users', async function () {
      const userCount = 3
      const conversationRounds = 15

      console.log(`\n  Setting up ${userCount}-person conversation...`)

      // Создать пользователей
      const clients: ApiTestClient[] = []
      for (let i = 0; i < userCount; i++) {
        const client = await createUser(`conv_user${i}`)
        client.startTest()
        clients.push(client)
      }

      // Создать общий чат
      const chat = await clients[0].createChat('Conversation Chat')
      const chatId = chat.id

      // Добавить участников
      for (let i = 1; i < userCount; i++) {
        await clients[0].addParticipant(chatId, clients[i].getUserId()!)
      }

      console.log(`  Starting conversation simulation...`)

      // Симуляция разговора - пользователи по очереди отправляют сообщения
      for (let round = 0; round < conversationRounds; round++) {
        const speaker = round % userCount

        // Текущий пользователь отправляет сообщение
        await clients[speaker].sendMessage(chatId, `[User ${speaker}] Round ${round + 1}: ${getRandomPhrase()}`)

        // Остальные читают (имитация получения через WebSocket, но через polling)
        const readPromises = clients
          .filter((_, idx) => idx !== speaker)
          .map(client => client.getMessages(chatId))

        await Promise.all(readPromises)

        // Небольшая пауза между "репликами"
        await sleep(100)
      }

      // Все читают финальное состояние
      await Promise.all(clients.map(c => c.getMessages(chatId)))

      // Один добавляет реакцию
      const messages = await clients[0].getMessages(chatId)
      if (messages.messages.length > 0) {
        await clients[1].addReaction(messages.messages[0].id, '❤️')
      }

      // Собрать отчеты
      const reports = clients.map((c, i) => c.finishTest(`Conv User ${i}`))
      const aggregate = aggregateReports(reports)

      console.log('\n' + '='.repeat(70))
      console.log('  REAL-TIME CONVERSATION SIMULATION - RESULTS')
      console.log('='.repeat(70))
      console.log(`  Users: ${userCount}`)
      console.log(`  Conversation rounds: ${conversationRounds}`)
      console.log(`  Total Requests: ${aggregate.totalRequests}`)
      console.log(`  Average Latency: ${aggregate.avgLatency}ms`)
      console.log(`  Max Latency: ${aggregate.maxLatency}ms`)
      console.log(`  Slow Requests: ${aggregate.slowRequests}`)
      console.log('='.repeat(70) + '\n')

      allReports.push({
        testName: 'Real-time Conversation',
        startTime: new Date(),
        endTime: new Date(),
        totalDuration: aggregate.totalDuration,
        stats: {
          total: aggregate.totalRequests,
          successful: aggregate.totalRequests - aggregate.failedRequests,
          failed: aggregate.failedRequests,
          excellent: 0,
          good: 0,
          acceptable: 0,
          slow: aggregate.slowRequests,
          avgDuration: aggregate.avgLatency,
          maxDuration: aggregate.maxLatency,
          minDuration: 0,
          p50: 0,
          p90: 0,
          p95: 0,
          p99: 0,
          totalDuration: aggregate.totalDuration,
        },
        slowRequests: [],
        failedRequests: [],
        allRequests: [],
      })

      expect(aggregate.failedRequests).to.equal(0)
    })
  })

  describe('Load Test - Sustained Traffic', function () {
    it('should handle sustained load over time', async function () {
      const userCount = 5
      const durationSeconds = 30
      const requestsPerSecond = 2 // per user

      console.log(`\n  Starting ${durationSeconds}s sustained load test...`)
      console.log(`  Users: ${userCount}, Target RPS per user: ${requestsPerSecond}`)

      // Создать пользователей
      const clients: ApiTestClient[] = []
      const chatIds: string[] = []

      for (let i = 0; i < userCount; i++) {
        const client = await createUser(`load_user${i}`)
        client.startTest()
        clients.push(client)

        const chat = await client.createChat(`Load Test Chat ${i}`)
        chatIds.push(chat.id)
      }

      const startTime = Date.now()
      const endTime = startTime + (durationSeconds * 1000)
      let requestCount = 0

      // Генерировать нагрузку
      while (Date.now() < endTime) {
        const promises: Promise<any>[] = []

        for (let i = 0; i < userCount; i++) {
          // Отправить сообщение
          promises.push(clients[i].sendMessage(chatIds[i], `Load test message ${requestCount}`))
          requestCount++
        }

        await Promise.all(promises)

        // Контроль RPS
        await sleep(1000 / requestsPerSecond)
      }

      const actualDuration = Date.now() - startTime
      const actualRPS = requestCount / (actualDuration / 1000)

      // Собрать отчеты
      const reports = clients.map((c, i) => c.finishTest(`Load User ${i}`))
      const aggregate = aggregateReports(reports)

      console.log('\n' + '='.repeat(70))
      console.log('  SUSTAINED LOAD TEST - RESULTS')
      console.log('='.repeat(70))
      console.log(`  Duration: ${(actualDuration / 1000).toFixed(1)}s`)
      console.log(`  Total Requests: ${aggregate.totalRequests}`)
      console.log(`  Actual RPS: ${actualRPS.toFixed(2)}`)
      console.log(`  Average Latency: ${aggregate.avgLatency}ms`)
      console.log(`  Max Latency: ${aggregate.maxLatency}ms`)
      console.log(`  Slow Requests: ${aggregate.slowRequests}`)
      console.log(`  Failed Requests: ${aggregate.failedRequests}`)
      console.log('='.repeat(70) + '\n')

      allReports.push({
        testName: `Sustained Load (${durationSeconds}s)`,
        startTime: new Date(startTime),
        endTime: new Date(),
        totalDuration: actualDuration,
        stats: {
          total: aggregate.totalRequests,
          successful: aggregate.totalRequests - aggregate.failedRequests,
          failed: aggregate.failedRequests,
          excellent: 0,
          good: 0,
          acceptable: 0,
          slow: aggregate.slowRequests,
          avgDuration: aggregate.avgLatency,
          maxDuration: aggregate.maxLatency,
          minDuration: 0,
          p50: 0,
          p90: 0,
          p95: 0,
          p99: 0,
          totalDuration: aggregate.totalDuration,
        },
        slowRequests: [],
        failedRequests: [],
        allRequests: [],
      })

      expect(aggregate.failedRequests).to.equal(0)
      expect(aggregate.avgLatency).to.be.lessThan(2000)
    })
  })
})

// Вспомогательная функция для генерации фраз
function getRandomPhrase(): string {
  const phrases = [
    'Hello everyone!',
    'How is the project going?',
    'I think we should discuss this further.',
    'Great point!',
    'Let me check and get back to you.',
    'Can someone review my PR?',
    'The build is passing now.',
    'I found an interesting bug.',
    'Meeting in 5 minutes.',
    'Thanks for the update!',
    'Working on the fix now.',
    'Code review completed.',
    'Tests are green.',
    'Deploying to staging.',
    'Release is scheduled.',
  ]
  return phrases[Math.floor(Math.random() * phrases.length)]
}
