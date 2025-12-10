/**
 * Create Test Users - создаёт большое количество тестовых пользователей
 *
 * Использование:
 *   k6 run -e BASE_URL=http://localhost:8888 -e USERS=10000 create-test-users.js
 *
 * Параметры:
 *   - USERS: количество пользователей для создания (default: 10000)
 *   - BATCH_SIZE: сколько пользователей создавать параллельно (default: 10)
 *   - PREFIX: префикс для имён пользователей (default: "testuser")
 */

import http from 'k6/http'
import { check, sleep } from 'k6'
import { Counter, Rate } from 'k6/metrics'
import { SharedArray } from 'k6/data'

const usersCreated = new Counter('users_created')
const usersFailed = new Counter('users_failed')
const successRate = new Rate('success_rate')

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8888'
const TOTAL_USERS = parseInt(__ENV.USERS) || 10000
const BATCH_SIZE = parseInt(__ENV.BATCH_SIZE) || 10
const PREFIX = __ENV.PREFIX || 'testuser'

// Calculate iterations needed
const ITERATIONS = Math.ceil(TOTAL_USERS / BATCH_SIZE)

export const options = {
  scenarios: {
    create_users: {
      executor: 'shared-iterations',
      vus: BATCH_SIZE,
      iterations: TOTAL_USERS,
      maxDuration: '2h',
    },
  },
  thresholds: {
    success_rate: ['rate>0.95'],  // 95% success rate
  },
}

export function setup() {
  console.log(`=== Creating ${TOTAL_USERS} test users ===`)
  console.log(`Prefix: ${PREFIX}`)
  console.log(`Batch size: ${BATCH_SIZE} parallel VUs`)
  console.log(`Base URL: ${BASE_URL}`)
  console.log(`Expected time: ~${Math.ceil(TOTAL_USERS * 0.1 / 60)} minutes (assuming ~100ms per user with parallelism)`)
  console.log('')

  return {
    startTime: Date.now(),
  }
}

export default function(data) {
  // Each VU gets unique iteration number
  const userIndex = __ITER + 1

  if (userIndex > TOTAL_USERS) {
    return
  }

  // Create user with zero-padded index for easy sorting
  const paddedIndex = String(userIndex).padStart(5, '0')
  const username = `${PREFIX}_${paddedIndex}`
  const email = `${username}@loadtest.local`
  const password = 'TestPass123!'

  const res = http.post(`${BASE_URL}/api/auth/register`, JSON.stringify({
    username: username,
    email: email,
    password: password,
  }), {
    headers: { 'Content-Type': 'application/json' },
    tags: { name: 'register' },
  })

  const success = check(res, {
    'user created': (r) => r.status === 200 || r.status === 201,
  })

  if (success) {
    usersCreated.add(1)
    successRate.add(1)
  } else {
    usersFailed.add(1)
    successRate.add(0)

    // Log failures (but not too many)
    if (res.status !== 409) { // 409 = already exists, that's ok
      console.log(`Failed to create user ${username}: ${res.status} - ${res.body}`)
    }
  }

  // Log progress every 500 users
  if (userIndex % 500 === 0) {
    console.log(`Progress: ${userIndex}/${TOTAL_USERS} users processed`)
  }
}

export function teardown(data) {
  const elapsed = (Date.now() - data.startTime) / 1000
  console.log('')
  console.log(`=== User creation completed ===`)
  console.log(`Total time: ${elapsed.toFixed(1)} seconds`)
  console.log(`Rate: ${(TOTAL_USERS / elapsed).toFixed(1)} users/second`)
  console.log('')
  console.log('Users can now login with:')
  console.log(`  Email: ${PREFIX}_00001@loadtest.local ... ${PREFIX}_${String(TOTAL_USERS).padStart(5, '0')}@loadtest.local`)
  console.log(`  Password: TestPass123!`)
}
