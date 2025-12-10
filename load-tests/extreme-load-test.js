/**
 * Extreme Load Test - до 10000 клиентов в одном чате с WebSocket
 *
 * Использует предсозданных пользователей testuser_XXXXX@loadtest.local
 * (создаются скриптом create-test-users.js)
 *
 * Сценарий:
 * - Setup: логин всех пользователей, создание чата, добавление участников
 *   (токены сохраняются и передаются в тест - без bcrypt в тестовой фазе!)
 * - Test: VUs используют готовые токены и шлют сообщения
 * - Каждый VU подключается к Centrifugo через WebSocket
 * - Подписка на персональный канал user:{userId} (как фронтенд)
 * - Настраиваемый % сообщений с файлами
 * - Все в одном чате
 *
 * Запуск:
 *   k6 run -e BASE_URL=http://localhost:8888 -e VUS=1000 extreme-load-test.js
 *
 * Параметры:
 *   - VUS: количество виртуальных пользователей (default: 400, max: 10000)
 *   - TARGET_MPS: целевое количество сообщений в секунду (default: 10)
 *   - DURATION: длительность теста (default: 5m)
 *   - FILE_RATIO: доля сообщений с файлами (default: 0.2 = 20%)
 *   - USER_PREFIX: префикс пользователей (default: testuser)
 */

import http from 'k6/http'
import { check, sleep } from 'k6'
import { Counter, Rate, Trend } from 'k6/metrics'
import { WebSocket } from 'k6/experimental/websockets'

// Custom metrics
const messagesSent = new Counter('messages_sent')
const messagesReceived = new Counter('messages_received')
const messagesWithFiles = new Counter('messages_with_files')
const filesUploaded = new Counter('files_uploaded')
const wsConnections = new Counter('ws_connections')
const wsSubscriptions = new Counter('ws_subscriptions')
const wsErrors = new Counter('ws_errors')
const apiErrors = new Rate('api_errors')
const messageLatency = new Trend('message_latency')
const fileUploadLatency = new Trend('file_upload_latency')
const loginLatency = new Trend('login_latency')

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8888'
const WS_URL = __ENV.WS_URL || BASE_URL.replace('http://', 'ws://').replace('https://', 'wss://').replace(':8888', ':8000')
const TARGET_MPS = parseInt(__ENV.TARGET_MPS) || 10  // 10 messages per second total
const VUS = parseInt(__ENV.VUS) || 400
const DURATION = __ENV.DURATION || '5m'
const FILE_RATIO = parseFloat(__ENV.FILE_RATIO) || 0.2  // 20% with files
const USER_PREFIX = __ENV.USER_PREFIX || 'testuser'  // prefix for pre-created users

// Calculate delay per VU to achieve target MPS
const DELAY_PER_VU = VUS / TARGET_MPS

// Parse duration to milliseconds
function parseDuration(dur) {
  const match = dur.match(/^(\d+)(s|m|h)?$/)
  if (!match) return 300000
  const value = parseInt(match[1])
  const unit = match[2] || 's'
  switch (unit) {
    case 'h': return value * 3600000
    case 'm': return value * 60000
    case 's': return value * 1000
    default: return value * 1000
  }
}

const DURATION_MS = parseDuration(DURATION)

export const options = {
  setupTimeout: '10m',  // Allow up to 10 minutes for setup (adding users to chat)
  scenarios: {
    message_senders: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: '30s', target: Math.floor(VUS * 0.25) },  // Ramp up to 25%
        { duration: '30s', target: Math.floor(VUS * 0.5) },   // Ramp up to 50%
        { duration: '30s', target: Math.floor(VUS * 0.75) },  // Ramp up to 75%
        { duration: '30s', target: VUS },                      // Ramp up to 100%
        { duration: DURATION, target: VUS },                   // Hold at 100%
      ],
      gracefulRampDown: '30s',
    },
  },
  thresholds: {
    http_req_duration: ['p(95)<3000'],  // Relaxed to 3s for high load
    http_req_failed: ['rate<0.1'],
    api_errors: ['rate<0.15'],
  },
}

function randomString(length) {
  const chars = 'abcdefghijklmnopqrstuvwxyz0123456789'
  let result = ''
  for (let i = 0; i < length; i++) {
    result += chars.charAt(Math.floor(Math.random() * chars.length))
  }
  return result
}

function generateUUID() {
  return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, function(c) {
    const r = Math.random() * 16 | 0
    const v = c === 'x' ? r : (r & 0x3 | 0x8)
    return v.toString(16)
  })
}

function randomMessage() {
  const messages = [
    'Hello everyone!',
    'Testing message delivery',
    'How is everyone doing?',
    'This is a load test message',
    `Random number: ${Math.floor(Math.random() * 10000)}`,
    `Timestamp: ${Date.now()}`,
    'Lorem ipsum dolor sit amet',
    'Quick brown fox jumps over lazy dog',
  ]
  return messages[Math.floor(Math.random() * messages.length)]
}

function generateFileContent() {
  const uuid = generateUUID()
  const content = `File UUID: ${uuid}\n` +
    `Generated at: ${new Date().toISOString()}\n` +
    `Random data: ${randomString(100)}\n` +
    `Test ID: ${generateUUID()}`
  return { content, uuid }
}

// Get email for VU index (1-based)
function getUserEmail(vuIndex) {
  const paddedIndex = String(vuIndex).padStart(5, '0')
  return `${USER_PREFIX}_${paddedIndex}@loadtest.local`
}

// Setup - create shared chat and add pre-existing users
export function setup() {
  console.log(`=== Extreme Load Test with Pre-created Users ===`)
  console.log(`VUs: ${VUS}, Target MPS: ${TARGET_MPS}, File ratio: ${FILE_RATIO * 100}%`)
  console.log(`Delay per VU: ${DELAY_PER_VU} seconds`)
  console.log(`WebSocket URL: ${WS_URL}`)
  console.log(`Duration: ${DURATION} (${DURATION_MS}ms)`)
  console.log(`Using pre-created users: ${USER_PREFIX}_00001 ... ${USER_PREFIX}_${String(VUS).padStart(5, '0')}`)
  console.log('')

  // Login as first user to create the chat (they become owner)
  const ownerEmail = getUserEmail(1)
  const ownerPassword = 'TestPass123!'

  console.log(`Logging in as owner: ${ownerEmail}`)
  const ownerLoginRes = http.post(`${BASE_URL}/api/auth/login`, JSON.stringify({
    email: ownerEmail,
    password: ownerPassword,
  }), {
    headers: { 'Content-Type': 'application/json' },
  })

  if (ownerLoginRes.status !== 200) {
    console.error(`Failed to login as owner: ${ownerLoginRes.status} - ${ownerLoginRes.body}`)
    console.error('Make sure you ran create-test-users.js first!')
    return null
  }

  const ownerData = JSON.parse(ownerLoginRes.body)
  const ownerToken = ownerData.access_token

  // Get owner's user ID
  const ownerMeRes = http.get(`${BASE_URL}/api/auth/me`, {
    headers: { 'Authorization': `Bearer ${ownerToken}` },
  })
  const ownerId = ownerMeRes.status === 200 ? JSON.parse(ownerMeRes.body).id : null

  // Create shared chat
  const chatRes = http.post(`${BASE_URL}/api/chats`, JSON.stringify({
    type: 'group',
    name: `Load Test ${VUS} users - ${Date.now()}`,
    description: `Extreme load test with ${VUS} pre-created users`,
    participant_ids: [],
  }), {
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${ownerToken}`,
    },
  })

  if (chatRes.status !== 200 && chatRes.status !== 201) {
    console.error('Failed to create chat:', chatRes.body)
    return null
  }

  const chatData = JSON.parse(chatRes.body)
  console.log(`Created chat: ${chatData.id}`)

  // Login all users and add them to chat
  // This is fast because login only verifies bcrypt hash (already computed)
  const users = []
  let added = 0
  let failed = 0

  console.log(`Adding ${VUS} users to chat...`)

  for (let i = 1; i <= VUS; i++) {
    const email = getUserEmail(i)
    const password = 'TestPass123!'

    // Login user
    const loginRes = http.post(`${BASE_URL}/api/auth/login`, JSON.stringify({
      email: email,
      password: password,
    }), {
      headers: { 'Content-Type': 'application/json' },
    })

    if (loginRes.status === 200) {
      const loginData = JSON.parse(loginRes.body)
      const token = loginData.access_token

      // Get user ID
      const meRes = http.get(`${BASE_URL}/api/auth/me`, {
        headers: { 'Authorization': `Bearer ${token}` },
      })

      if (meRes.status === 200) {
        const userId = JSON.parse(meRes.body).id

        // Add user to chat (skip for owner - they're already in)
        if (i > 1) {
          http.post(
            `${BASE_URL}/api/chats/${chatData.id}/participants`,
            JSON.stringify({ user_id: userId, role: 'member' }),
            {
              headers: {
                'Content-Type': 'application/json',
                'Authorization': `Bearer ${ownerToken}`,
              },
            }
          )
        }

        users.push({
          vuId: i,
          email: email,
          password: password,
          userId: userId,
          token: token,  // Save token from setup to avoid re-login
        })
        added++
      } else {
        failed++
      }
    } else {
      failed++
      if (failed <= 5) {
        console.log(`Failed to login user ${i} (${email}): ${loginRes.status}`)
      }
    }

    // Log progress every 100 users
    if (i % 100 === 0) {
      console.log(`Progress: ${added}/${i} users added to chat (${failed} failed)`)
    }
  }

  console.log(`Setup complete: ${added} users added, ${failed} failed`)
  console.log(`Chat ID: ${chatData.id}`)
  console.log('')

  return {
    chatId: chatData.id,
    ownerToken: ownerToken,
    users: users,
  }
}

export default function(data) {
  if (!data || !data.chatId || !data.users) {
    console.error('No chat data or users available')
    sleep(5)
    return
  }

  const vuId = __VU
  const chatId = data.chatId

  // Find pre-created user for this VU
  const user = data.users.find(u => u.vuId === vuId)
  if (!user) {
    console.error(`No user found for VU ${vuId}`)
    apiErrors.add(1)
    sleep(5)
    return
  }

  // Use token from setup (avoids bcrypt during test phase)
  let authToken = user.token
  let userId = user.userId

  // If token expired or not available, login again
  if (!authToken) {
    const loginStart = Date.now()
    const loginRes = http.post(`${BASE_URL}/api/auth/login`, JSON.stringify({
      email: user.email,
      password: user.password,
    }), {
      headers: { 'Content-Type': 'application/json' },
    })

    if (loginRes.status === 200) {
      loginLatency.add(Date.now() - loginStart)
      const loginData = JSON.parse(loginRes.body)
      authToken = loginData.access_token
    } else {
      console.log(`Login failed for VU ${vuId}: ${loginRes.status}`)
      apiErrors.add(1)
      sleep(5)
      return
    }
  }

  if (!authToken || !userId) {
    apiErrors.add(1)
    sleep(5)
    return
  }

  // Get Centrifugo connection token
  const connTokenRes = http.get(`${BASE_URL}/api/centrifugo/connection-token`, {
    headers: { 'Authorization': `Bearer ${authToken}` },
  })

  if (connTokenRes.status !== 200) {
    console.log(`Failed to get connection token: ${connTokenRes.status}`)
    apiErrors.add(1)
    sleep(5)
    return
  }

  const connectionToken = JSON.parse(connTokenRes.body).token

  // Get subscription token for the user's personal channel
  const channel = `user:${userId}`
  const subTokenRes = http.post(
    `${BASE_URL}/api/centrifugo/subscription-token`,
    JSON.stringify({ channel: channel }),
    {
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${authToken}`,
      },
    }
  )

  if (subTokenRes.status !== 200) {
    console.log(`Failed to get subscription token: ${subTokenRes.status}`)
    apiErrors.add(1)
    sleep(5)
    return
  }

  const subscriptionToken = JSON.parse(subTokenRes.body).token

  // Connect to WebSocket
  const wsUrl = `${WS_URL}/connection/websocket?cf_ws_frame_ping_pong=true`

  let subscribed = false
  let messageIntervalId = null

  const ws = new WebSocket(wsUrl)

  ws.onopen = function() {
    wsConnections.add(1)

    // Send connect command with token
    const connectCmd = {
      connect: {
        token: connectionToken,
        name: 'k6-loadtest-client',
      },
      id: 1,
    }
    ws.send(JSON.stringify(connectCmd))
  }

  ws.onmessage = function(event) {
    try {
      const msg = JSON.parse(event.data)

      // Handle connect response
      if (msg.connect) {
        const subscribeCmd = {
          subscribe: {
            channel: channel,
            token: subscriptionToken,
          },
          id: 2,
        }
        ws.send(JSON.stringify(subscribeCmd))
      }

      // Handle subscription response
      if (msg.subscribe) {
        subscribed = true
        wsSubscriptions.add(1)

        // Start sending messages
        messageIntervalId = setInterval(function() {
          if (!subscribed) return

          // Decide if this message has a file
          const hasFile = Math.random() < FILE_RATIO
          let fileLinkIds = []

          if (hasFile) {
            const { content, uuid } = generateFileContent()
            const filename = `test_${uuid.substring(0, 8)}.txt`

            const boundary = '----k6boundary' + Date.now()
            const body = `--${boundary}\r\n` +
              `Content-Disposition: form-data; name="file"; filename="${filename}"\r\n` +
              `Content-Type: text/plain\r\n\r\n` +
              `${content}\r\n` +
              `--${boundary}--\r\n`

            const uploadStart = Date.now()
            const uploadRes = http.post(`${BASE_URL}/api/files/upload`, body, {
              headers: {
                'Content-Type': `multipart/form-data; boundary=${boundary}`,
                'Authorization': `Bearer ${authToken}`,
              },
            })

            if (uploadRes.status === 200 || uploadRes.status === 201) {
              fileUploadLatency.add(Date.now() - uploadStart)
              filesUploaded.add(1)

              try {
                const uploadData = JSON.parse(uploadRes.body)
                fileLinkIds = [uploadData.link_id]
              } catch (e) {
                // Ignore parse errors
              }
            }
          }

          // Send message
          const messageContent = hasFile && fileLinkIds.length > 0
            ? `[File attached] ${randomMessage()}`
            : randomMessage()

          const messageBody = {
            content: messageContent,
          }

          if (fileLinkIds.length > 0) {
            messageBody.file_link_ids = fileLinkIds
          }

          const sendStart = Date.now()
          const sendRes = http.post(
            `${BASE_URL}/api/chats/${chatId}/messages`,
            JSON.stringify(messageBody),
            {
              headers: {
                'Content-Type': 'application/json',
                'Authorization': `Bearer ${authToken}`,
              },
            }
          )

          const sendOk = check(sendRes, {
            'message sent': (r) => r.status === 200 || r.status === 201,
          })

          if (sendOk) {
            messagesSent.add(1)
            messageLatency.add(Date.now() - sendStart)

            if (fileLinkIds.length > 0) {
              messagesWithFiles.add(1)
            }
          } else {
            apiErrors.add(1)
            if (sendRes.status !== 429) {
              console.log(`Send failed (VU ${vuId}): ${sendRes.status}`)
            }
          }
        }, Math.floor(DELAY_PER_VU * 1000))

        // Schedule close after duration
        setTimeout(function() {
          ws.close()
        }, DURATION_MS)
      }

      // Handle push messages
      if (msg.push && msg.push.pub) {
        messagesReceived.add(1)
      }

      // Handle errors
      if (msg.error) {
        console.log(`WS error (VU ${vuId}): ${JSON.stringify(msg.error)}`)
        wsErrors.add(1)
      }
    } catch (e) {
      // Ignore parse errors
    }
  }

  ws.onerror = function(e) {
    wsErrors.add(1)
    console.log(`WS socket error (VU ${vuId}): ${e.error || e}`)
  }

  ws.onclose = function() {
    subscribed = false
  }
}

export function teardown(data) {
  console.log('')
  console.log('=== Extreme load test completed ===')
  console.log(`Target was ${TARGET_MPS} msg/sec with ${VUS} VUs`)
  console.log(`Duration: ${DURATION}`)
}
