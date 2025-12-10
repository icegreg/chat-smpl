/**
 * k6 Load Test - Chat API
 *
 * Тестирование REST API под нагрузкой:
 * - Регистрация/логин пользователей
 * - Создание чатов
 * - Отправка сообщений
 * - Получение списка чатов и сообщений
 *
 * Запуск:
 *   k6 run --vus 10 --duration 30s api-load-test.js
 *   k6 run -e SCENARIO=stress api-load-test.js
 */

import http from 'k6/http'
import { check, sleep, group } from 'k6'
import { Counter, Rate, Trend } from 'k6/metrics'
import { config, randomString, randomMessage, getScenario } from './config.js'

// Custom metrics
const messagesSent = new Counter('messages_sent')
const messagesReceived = new Counter('messages_received')
const loginSuccess = new Rate('login_success')
const apiErrors = new Rate('api_errors')
const messageLatency = new Trend('message_latency')

// Scenario configuration
const scenario = __ENV.SCENARIO || 'smoke'
const scenarioConfig = getScenario(scenario)

export const options = {
  scenarios: {
    default: {
      executor: scenarioConfig.stages ? 'ramping-vus' : 'constant-vus',
      ...(scenarioConfig.stages
        ? { stages: scenarioConfig.stages }
        : { vus: scenarioConfig.vus, duration: scenarioConfig.duration }
      ),
    },
  },
  thresholds: config.thresholds,
}

const BASE_URL = config.baseUrl

// Shared state per VU
let authToken = null
let userId = null
let currentChatId = null

// Setup - выполняется один раз перед тестом
export function setup() {
  console.log(`Starting ${scenario} test against ${BASE_URL}`)

  // Создаём тестовый чат, который будут использовать все VU
  const username = `loadtest_owner_${randomString(8)}`
  const email = `${username}@loadtest.local`

  const registerRes = http.post(`${BASE_URL}/api/auth/register`, JSON.stringify({
    username: username,
    email: email,
    password: 'TestPass123!',
    display_name: 'Load Test Owner',
  }), {
    headers: { 'Content-Type': 'application/json' },
  })

  if (registerRes.status !== 200 && registerRes.status !== 201) {
    console.error('Failed to create test owner:', registerRes.body)
    return { sharedChatId: null }
  }

  const ownerData = JSON.parse(registerRes.body)
  const ownerToken = ownerData.access_token

  // Создаём общий чат для нагрузочного тестирования
  const chatRes = http.post(`${BASE_URL}/api/chats`, JSON.stringify({
    type: 'group',
    name: `Load Test Chat ${Date.now()}`,
    description: 'Shared chat for load testing',
    participant_ids: [],
  }), {
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${ownerToken}`,
    },
  })

  if (chatRes.status !== 200 && chatRes.status !== 201) {
    console.error('Failed to create shared chat:', chatRes.body)
    return { sharedChatId: null, ownerToken: null }
  }

  const chatData = JSON.parse(chatRes.body)
  console.log(`Created shared chat: ${chatData.id}`)

  return {
    sharedChatId: chatData.id,
    ownerToken: ownerToken,
  }
}

export default function (data) {
  const vuId = __VU

  group('User Authentication', () => {
    // Каждый VU регистрируется как уникальный пользователь
    if (!authToken) {
      const username = `loadtest_vu${vuId}_${randomString(6)}`
      const email = `${username}@loadtest.local`

      const registerRes = http.post(`${BASE_URL}/api/auth/register`, JSON.stringify({
        username: username,
        email: email,
        password: 'TestPass123!',
        display_name: `Load Test User ${vuId}`,
      }), {
        headers: { 'Content-Type': 'application/json' },
      })

      const registerOk = check(registerRes, {
        'register status 200/201': (r) => r.status === 200 || r.status === 201,
      })

      if (registerOk) {
        const regData = JSON.parse(registerRes.body)
        authToken = regData.access_token

        // Получаем user info чтобы узнать ID
        const meRes = http.get(`${BASE_URL}/api/auth/me`, {
          headers: { 'Authorization': `Bearer ${authToken}` },
        })
        if (meRes.status === 200) {
          const meData = JSON.parse(meRes.body)
          userId = meData.id
        }
        loginSuccess.add(1)
      } else {
        // Попробуем залогиниться - возможно пользователь уже существует
        const loginRes = http.post(`${BASE_URL}/api/auth/login`, JSON.stringify({
          email: email,
          password: 'TestPass123!',
        }), {
          headers: { 'Content-Type': 'application/json' },
        })

        const loginOk = check(loginRes, {
          'login status 200': (r) => r.status === 200,
        })

        if (loginOk) {
          const loginData = JSON.parse(loginRes.body)
          authToken = loginData.access_token

          // Получаем user info
          const meRes = http.get(`${BASE_URL}/api/auth/me`, {
            headers: { 'Authorization': `Bearer ${authToken}` },
          })
          if (meRes.status === 200) {
            const meData = JSON.parse(meRes.body)
            userId = meData.id
          }
          loginSuccess.add(1)
        } else {
          loginSuccess.add(0)
          apiErrors.add(1)
          return
        }
      }
    }
  })

  if (!authToken) {
    sleep(1)
    return
  }

  const headers = {
    'Content-Type': 'application/json',
    'Authorization': `Bearer ${authToken}`,
  }

  group('Chat Operations', () => {
    // Присоединяемся к общему чату если ещё не присоединились
    if (data.sharedChatId && !currentChatId && userId) {
      // Добавляем себя в чат (через owner token)
      const addRes = http.post(
        `${BASE_URL}/api/chats/${data.sharedChatId}/participants`,
        JSON.stringify({ user_id: userId, role: 'member' }),
        {
          headers: {
            'Content-Type': 'application/json',
            'Authorization': `Bearer ${data.ownerToken}`,
          },
        }
      )

      // Проверяем успешность добавления (или уже участник)
      if (addRes.status === 200 || addRes.status === 201 || addRes.status === 409) {
        currentChatId = data.sharedChatId
      }
    }

    // Получаем список чатов
    const chatsRes = http.get(`${BASE_URL}/api/chats`, { headers })

    check(chatsRes, {
      'get chats status 200': (r) => r.status === 200,
      'get chats has data': (r) => {
        try {
          const body = JSON.parse(r.body)
          return body.chats !== undefined
        } catch {
          return false
        }
      },
    })

    if (chatsRes.status !== 200) {
      apiErrors.add(1)
    }
  })

  group('Message Operations', () => {
    if (!currentChatId) {
      return
    }

    // Получаем сообщения
    const messagesRes = http.get(`${BASE_URL}/api/chats/${currentChatId}/messages?limit=50`, { headers })

    const messagesOk = check(messagesRes, {
      'get messages status 200': (r) => r.status === 200,
    })

    if (messagesOk) {
      try {
        const body = JSON.parse(messagesRes.body)
        messagesReceived.add(body.messages?.length || 0)
      } catch {
        // ignore
      }
    }

    // Отправляем сообщение
    const startTime = Date.now()

    const sendRes = http.post(
      `${BASE_URL}/api/chats/${currentChatId}/messages`,
      JSON.stringify({ content: randomMessage() }),
      { headers }
    )

    const sendOk = check(sendRes, {
      'send message status 200/201': (r) => r.status === 200 || r.status === 201,
    })

    if (sendOk) {
      messagesSent.add(1)
      messageLatency.add(Date.now() - startTime)
    } else {
      apiErrors.add(1)
    }
  })

  // Пауза между итерациями (имитация реального поведения пользователя)
  sleep(Math.random() * 2 + 1) // 1-3 секунды
}

export function teardown(data) {
  console.log('Load test completed')

  // Можно удалить тестовый чат если нужно
  // if (data.sharedChatId && data.ownerToken) {
  //   http.del(`${BASE_URL}/api/chats/${data.sharedChatId}`, null, {
  //     headers: { 'Authorization': `Bearer ${data.ownerToken}` },
  //   })
  // }
}
