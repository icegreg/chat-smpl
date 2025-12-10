/**
 * k6 Load Test - WebSocket (Centrifugo)
 *
 * Тестирование WebSocket соединений под нагрузкой:
 * - Множество одновременных подключений
 * - Получение сообщений в реальном времени
 * - Стабильность при длительных соединениях
 *
 * Запуск:
 *   k6 run --vus 50 --duration 2m websocket-load-test.js
 */

import http from 'k6/http'
import ws from 'k6/ws'
import { check, sleep } from 'k6'
import { Counter, Rate, Trend } from 'k6/metrics'
import { config, randomString } from './config.js'

// Custom metrics
const wsConnections = new Counter('ws_connections')
const wsMessages = new Counter('ws_messages_received')
const wsErrors = new Rate('ws_errors')
const wsConnectTime = new Trend('ws_connect_time')
const wsMessageLatency = new Trend('ws_message_latency')

const scenario = __ENV.SCENARIO || 'smoke'

export const options = {
  scenarios: {
    websocket_test: {
      executor: 'constant-vus',
      vus: scenario === 'stress' ? 100 : scenario === 'load' ? 50 : 10,
      duration: scenario === 'stress' ? '5m' : scenario === 'load' ? '2m' : '30s',
    },
  },
  thresholds: {
    ws_errors: ['rate<0.05'],           // < 5% ошибок WS
    ws_connect_time: ['p(95)<2000'],    // 95% connect < 2s
    ws_connections: ['count>10'],        // Минимум 10 успешных подключений
  },
}

const BASE_URL = config.baseUrl

export function setup() {
  console.log(`Starting WebSocket ${scenario} test against ${BASE_URL}`)

  // Создаём пользователя-владельца и чат
  const username = `wstest_owner_${randomString(8)}`
  const email = `${username}@wstest.local`

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

  // Создаём чат
  const chatRes = http.post(`${BASE_URL}/api/chats`, JSON.stringify({
    type: 'group',
    name: `WS Load Test ${Date.now()}`,
    participant_ids: [],
  }), {
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${ownerData.access_token}`,
    },
  })

  if (chatRes.status !== 200 && chatRes.status !== 201) {
    console.error('Failed to create chat:', chatRes.body)
    return null
  }

  const chatData = JSON.parse(chatRes.body)

  return {
    chatId: chatData.id,
    ownerToken: ownerData.access_token,
  }
}

export default function (data) {
  if (!data) {
    sleep(1)
    return
  }

  const vuId = __VU

  // 1. Регистрируем пользователя
  const username = `wstest_vu${vuId}_${randomString(6)}`
  const email = `${username}@wstest.local`

  const registerRes = http.post(`${BASE_URL}/api/auth/register`, JSON.stringify({
    username: username,
    email: email,
    password: 'TestPass123!',
  }), {
    headers: { 'Content-Type': 'application/json' },
  })

  let token = null
  let userId = null

  if (registerRes.status === 200 || registerRes.status === 201) {
    const regData = JSON.parse(registerRes.body)
    token = regData.access_token
    userId = username
  } else {
    // Попробуем залогиниться
    const loginRes = http.post(`${BASE_URL}/api/auth/login`, JSON.stringify({
      email: email,
      password: 'TestPass123!',
    }), {
      headers: { 'Content-Type': 'application/json' },
    })

    if (loginRes.status === 200) {
      const loginData = JSON.parse(loginRes.body)
      token = loginData.access_token
      userId = username
    } else {
      wsErrors.add(1)
      sleep(1)
      return
    }
  }

  // 2. Получаем Centrifugo connection token
  const centrifugoRes = http.get(`${BASE_URL}/api/centrifugo/connection-token`, {
    headers: { 'Authorization': `Bearer ${token}` },
  })

  if (centrifugoRes.status !== 200) {
    wsErrors.add(1)
    sleep(1)
    return
  }

  const centrifugoData = JSON.parse(centrifugoRes.body)
  const wsToken = centrifugoData.token

  // 3. Подключаемся к WebSocket
  const wsUrl = BASE_URL.replace('http://', 'ws://').replace('https://', 'wss://')
  const wsEndpoint = `${wsUrl}/connection/websocket`

  const connectStart = Date.now()

  const res = ws.connect(wsEndpoint, {}, function (socket) {
    wsConnections.add(1)
    wsConnectTime.add(Date.now() - connectStart)

    let connected = false
    let subscribed = false
    let messageCount = 0

    socket.on('open', () => {
      // Отправляем connect command с токеном
      socket.send(JSON.stringify({
        connect: {
          token: wsToken,
          name: 'k6-load-test',
        },
        id: 1,
      }))
    })

    socket.on('message', (msg) => {
      try {
        const data = JSON.parse(msg)

        // Connect response
        if (data.id === 1 && data.connect) {
          connected = true

          // Подписываемся на user channel
          socket.send(JSON.stringify({
            subscribe: {
              channel: `user:${userId}`,
            },
            id: 2,
          }))
        }

        // Subscribe response
        if (data.id === 2 && data.subscribe) {
          subscribed = true
        }

        // Publication (новое сообщение)
        if (data.push && data.push.pub) {
          messageCount++
          wsMessages.add(1)
          wsMessageLatency.add(Date.now() - connectStart)
        }

      } catch (e) {
        // Ignore parse errors
      }
    })

    socket.on('error', (e) => {
      wsErrors.add(1)
      console.error(`VU ${vuId} WS error:`, e)
    })

    socket.on('close', () => {
      // Connection closed
    })

    // Держим соединение открытым
    const duration = scenario === 'stress' ? 60000 : scenario === 'load' ? 30000 : 10000
    sleep(duration / 1000)

    // Отправляем тестовое сообщение через REST (чтобы проверить доставку через WS)
    if (connected && data.chatId) {
      // Добавляем себя в чат
      http.post(
        `${BASE_URL}/api/chats/${data.chatId}/participants`,
        JSON.stringify({ user_id: userId, role: 'member' }),
        {
          headers: {
            'Content-Type': 'application/json',
            'Authorization': `Bearer ${data.ownerToken}`,
          },
        }
      )

      // Отправляем сообщение
      http.post(
        `${BASE_URL}/api/chats/${data.chatId}/messages`,
        JSON.stringify({ content: `WS test from VU ${vuId}` }),
        {
          headers: {
            'Content-Type': 'application/json',
            'Authorization': `Bearer ${token}`,
          },
        }
      )
    }

    // Ждём ещё немного для получения сообщений
    sleep(2)

    socket.close()
  })

  check(res, {
    'WebSocket connected': (r) => r && r.status === 101,
  })
}

export function teardown(data) {
  console.log('WebSocket load test completed')
}
