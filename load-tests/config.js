// Load Test Configuration
export const config = {
  baseUrl: __ENV.BASE_URL || 'http://localhost:8888',

  // Test scenarios
  scenarios: {
    // Smoke test - минимальная нагрузка для проверки работоспособности
    smoke: {
      vus: 5,
      duration: '30s',
    },

    // Load test - нормальная ожидаемая нагрузка
    load: {
      vus: 50,
      duration: '5m',
      rampUp: '1m',
      rampDown: '30s',
    },

    // Stress test - выше нормальной нагрузки
    stress: {
      stages: [
        { duration: '2m', target: 50 },   // Разогрев
        { duration: '5m', target: 100 },  // Нормальная нагрузка
        { duration: '5m', target: 200 },  // Стресс
        { duration: '5m', target: 300 },  // Пиковая нагрузка
        { duration: '2m', target: 0 },    // Остывание
      ],
    },

    // Spike test - резкий всплеск нагрузки
    spike: {
      stages: [
        { duration: '1m', target: 50 },   // Нормальная нагрузка
        { duration: '10s', target: 500 }, // Резкий всплеск
        { duration: '1m', target: 500 },  // Держим пик
        { duration: '10s', target: 50 },  // Спад
        { duration: '1m', target: 50 },   // Восстановление
      ],
    },

    // Soak test - длительная нагрузка для выявления утечек
    soak: {
      vus: 100,
      duration: '30m',
    },
  },

  // Thresholds - критерии успешности
  thresholds: {
    http_req_duration: ['p(95)<500', 'p(99)<1000'],  // 95% < 500ms, 99% < 1s
    http_req_failed: ['rate<0.01'],                  // < 1% ошибок
  },

  // WebSocket specific thresholds
  wsThresholds: {
    ws_connecting: ['p(95)<1000'],                   // WS connect < 1s
    ws_errors: ['rate<0.05'],                        // < 5% WS ошибок
  },
}

// Helper functions
export function getScenario(name) {
  return config.scenarios[name] || config.scenarios.smoke
}

export function randomString(length) {
  const chars = 'abcdefghijklmnopqrstuvwxyz0123456789'
  let result = ''
  for (let i = 0; i < length; i++) {
    result += chars.charAt(Math.floor(Math.random() * chars.length))
  }
  return result
}

export function randomMessage() {
  const messages = [
    'Hello!',
    'How are you?',
    'This is a test message',
    'Lorem ipsum dolor sit amet',
    'Testing under load...',
    'Message #' + Math.floor(Math.random() * 10000),
  ]
  return messages[Math.floor(Math.random() * messages.length)]
}
