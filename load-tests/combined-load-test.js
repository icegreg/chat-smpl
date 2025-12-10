/**
 * k6 Combined Load Test
 *
 * Комбинированное тестирование:
 * - REST API нагрузка
 * - WebSocket подключения
 * - Имитация реального поведения пользователей
 *
 * Сценарии пользователей:
 * - reader: только читает сообщения
 * - active: активно отправляет сообщения
 * - casual: редко отправляет, больше читает
 * - lurker: только подключен, не пишет
 *
 * Запуск:
 *   k6 run -e SCENARIO=load combined-load-test.js
 */

import http from 'k6/http'
import ws from 'k6/ws'
import { check, sleep, group } from 'k6'
import { Counter, Rate, Trend } from 'k6/metrics'
import { config, randomString, randomMessage } from './config.js'

// Custom metrics
const messagesSent = new Counter('messages_sent')
const messagesReceived = new Counter('messages_received')
const wsConnections = new Counter('ws_connections')
const wsMessages = new Counter('ws_messages_received')
const apiErrors = new Rate('api_errors')
const messageLatency = new Trend('message_latency')

const scenario = __ENV.SCENARIO || 'smoke'

// Сценарии с разными типами пользователей
export const options = {
  scenarios: {
    // Активные пользователи (пишут много)
    active_users: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: scenario === 'stress'
        ? [
            { duration: '1m', target: 20 },
            { duration: '3m', target: 50 },
            { duration: '3m', target: 100 },
            { duration: '1m', target: 0 },
          ]
        : [
            { duration: '30s', target: 5 },
            { duration: '1m', target: 10 },
            { duration: '30s', target: 0 },
          ],
      exec: 'activeUser',
    },

    // Читатели (в основном читают)
    readers: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: scenario === 'stress'
        ? [
            { duration: '1m', target: 50 },
            { duration: '3m', target: 150 },
            { duration: '3m', target: 300 },
            { duration: '1m', target: 0 },
          ]
        : [
            { duration: '30s', target: 10 },
            { duration: '1m', target: 30 },
            { duration: '30s', target: 0 },
          ],
      exec: 'readerUser',
    },

    // WebSocket подключения (только слушают)
    websocket_listeners: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: scenario === 'stress'
        ? [
            { duration: '1m', target: 100 },
            { duration: '5m', target: 500 },
            { duration: '1m', target: 0 },
          ]
        : [
            { duration: '30s', target: 20 },
            { duration: '1m', target: 50 },
            { duration: '30s', target: 0 },
          ],
      exec: 'websocketListener',
    },
  },
  thresholds: {
    http_req_duration: ['p(95)<1000'],
    http_req_failed: ['rate<0.05'],
    api_errors: ['rate<0.05'],
    message_latency: ['p(95)<2000'],
  },
}

const BASE_URL = config.baseUrl

// Shared setup
export function setup() {
  console.log(`Starting combined ${scenario} test against ${BASE_URL}`)

  // Создаём владельца и чат
  const username = `combined_owner_${randomString(8)}`
  const email = `${username}@combined.local`

  const registerRes = http.post(`${BASE_URL}/api/auth/register`, JSON.stringify({
    username: username,
    email: email,
    password: 'TestPass123!',
  }), {
    headers: { 'Content-Type': 'application/json' },
  })

  if (registerRes.status !== 200 && registerRes.status !== 201) {
    console.error('Setup failed:', registerRes.body)
    return null
  }

  const ownerData = JSON.parse(registerRes.body)

  // Создаём несколько чатов для распределения нагрузки
  const chats = []
  for (let i = 0; i < 5; i++) {
    const chatRes = http.post(`${BASE_URL}/api/chats`, JSON.stringify({
      type: 'group',
      name: `Combined Test Chat ${i + 1}`,
      participant_ids: [],
    }), {
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${ownerData.access_token}`,
      },
    })

    if (chatRes.status === 200 || chatRes.status === 201) {
      chats.push(JSON.parse(chatRes.body).id)
    }
  }

  console.log(`Created ${chats.length} test chats`)

  return {
    chatIds: chats,
    ownerToken: ownerData.access_token,
  }
}

// Helper: регистрация/логин пользователя
function getAuthToken(prefix) {
  const username = `${prefix}_${__VU}_${randomString(6)}`
  const email = `${username}@combined.local`

  let registerRes = http.post(`${BASE_URL}/api/auth/register`, JSON.stringify({
    username: username,
    email: email,
    password: 'TestPass123!',
  }), {
    headers: { 'Content-Type': 'application/json' },
  })

  if (registerRes.status === 200 || registerRes.status === 201) {
    return { token: JSON.parse(registerRes.body).access_token, userId: username }
  }

  // Попытка логина
  const loginRes = http.post(`${BASE_URL}/api/auth/login`, JSON.stringify({
    email: email,
    password: 'TestPass123!',
  }), {
    headers: { 'Content-Type': 'application/json' },
  })

  if (loginRes.status === 200) {
    return { token: JSON.parse(loginRes.body).access_token, userId: username }
  }

  return null
}

// Helper: выбрать случайный чат
function getRandomChat(data) {
  if (!data || !data.chatIds || data.chatIds.length === 0) return null
  return data.chatIds[Math.floor(Math.random() * data.chatIds.length)]
}

// ====== SCENARIO: Active User ======
// Активно отправляет сообщения
export function activeUser(data) {
  const auth = getAuthToken('active')
  if (!auth) {
    apiErrors.add(1)
    sleep(1)
    return
  }

  const headers = {
    'Content-Type': 'application/json',
    'Authorization': `Bearer ${auth.token}`,
  }

  const chatId = getRandomChat(data)
  if (!chatId) {
    sleep(1)
    return
  }

  // Добавляемся в чат
  http.post(
    `${BASE_URL}/api/chats/${chatId}/participants`,
    JSON.stringify({ user_id: auth.userId, role: 'member' }),
    { headers: { ...headers, 'Authorization': `Bearer ${data.ownerToken}` } }
  )

  // Активная работа: отправляем несколько сообщений
  for (let i = 0; i < 5; i++) {
    const startTime = Date.now()

    const sendRes = http.post(
      `${BASE_URL}/api/chats/${chatId}/messages`,
      JSON.stringify({ content: randomMessage() }),
      { headers }
    )

    if (sendRes.status === 200 || sendRes.status === 201) {
      messagesSent.add(1)
      messageLatency.add(Date.now() - startTime)
    } else {
      apiErrors.add(1)
    }

    sleep(Math.random() * 2 + 0.5) // 0.5-2.5 секунды между сообщениями
  }

  // Читаем сообщения
  const messagesRes = http.get(`${BASE_URL}/api/chats/${chatId}/messages?limit=50`, { headers })
  if (messagesRes.status === 200) {
    try {
      const body = JSON.parse(messagesRes.body)
      messagesReceived.add(body.messages?.length || 0)
    } catch {}
  }

  sleep(Math.random() * 3 + 2) // Пауза перед следующей сессией
}

// ====== SCENARIO: Reader User ======
// В основном читает, редко пишет
export function readerUser(data) {
  const auth = getAuthToken('reader')
  if (!auth) {
    apiErrors.add(1)
    sleep(1)
    return
  }

  const headers = {
    'Content-Type': 'application/json',
    'Authorization': `Bearer ${auth.token}`,
  }

  // Получаем список чатов
  const chatsRes = http.get(`${BASE_URL}/api/chats`, { headers })
  check(chatsRes, { 'get chats ok': (r) => r.status === 200 })

  const chatId = getRandomChat(data)
  if (!chatId) {
    sleep(1)
    return
  }

  // Добавляемся в чат
  http.post(
    `${BASE_URL}/api/chats/${chatId}/participants`,
    JSON.stringify({ user_id: auth.userId, role: 'member' }),
    { headers: { ...headers, 'Authorization': `Bearer ${data.ownerToken}` } }
  )

  // Читаем сообщения несколько раз
  for (let i = 0; i < 10; i++) {
    const messagesRes = http.get(`${BASE_URL}/api/chats/${chatId}/messages?limit=50`, { headers })

    if (messagesRes.status === 200) {
      try {
        const body = JSON.parse(messagesRes.body)
        messagesReceived.add(body.messages?.length || 0)
      } catch {}
    }

    // Иногда отправляем сообщение (10% шанс)
    if (Math.random() < 0.1) {
      const sendRes = http.post(
        `${BASE_URL}/api/chats/${chatId}/messages`,
        JSON.stringify({ content: randomMessage() }),
        { headers }
      )
      if (sendRes.status === 200 || sendRes.status === 201) {
        messagesSent.add(1)
      }
    }

    sleep(Math.random() * 3 + 1) // 1-4 секунды между запросами
  }
}

// ====== SCENARIO: WebSocket Listener ======
// Подключается по WebSocket и слушает
export function websocketListener(data) {
  const auth = getAuthToken('wslistener')
  if (!auth) {
    apiErrors.add(1)
    sleep(1)
    return
  }

  // Получаем Centrifugo token
  const centrifugoRes = http.get(`${BASE_URL}/api/centrifugo/connection-token`, {
    headers: { 'Authorization': `Bearer ${auth.token}` },
  })

  if (centrifugoRes.status !== 200) {
    apiErrors.add(1)
    sleep(1)
    return
  }

  const wsToken = JSON.parse(centrifugoRes.body).token
  const wsUrl = BASE_URL.replace('http://', 'ws://').replace('https://', 'wss://')
  const wsEndpoint = `${wsUrl}/connection/websocket`

  ws.connect(wsEndpoint, {}, function (socket) {
    wsConnections.add(1)

    socket.on('open', () => {
      socket.send(JSON.stringify({
        connect: { token: wsToken },
        id: 1,
      }))
    })

    socket.on('message', (msg) => {
      try {
        const data = JSON.parse(msg)
        if (data.id === 1 && data.connect) {
          // Подписываемся на user channel
          socket.send(JSON.stringify({
            subscribe: { channel: `user:${auth.userId}` },
            id: 2,
          }))
        }
        if (data.push && data.push.pub) {
          wsMessages.add(1)
        }
      } catch {}
    })

    socket.on('error', () => {
      apiErrors.add(1)
    })

    // Держим соединение открытым
    const duration = scenario === 'stress' ? 120 : scenario === 'load' ? 60 : 20
    sleep(duration)

    socket.close()
  })
}

export function teardown(data) {
  console.log('Combined load test completed')
}
