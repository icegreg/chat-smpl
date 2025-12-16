/**
 * API Test Client - прямые HTTP запросы к API с измерением времени
 *
 * Используется для имитации работы пользователей через API без браузера.
 * Все запросы измеряются и собирается статистика.
 */

// Пороговые значения в миллисекундах
export const TIMING_THRESHOLDS = {
  EXCELLENT: 100,   // < 100ms - отлично
  GOOD: 300,        // < 300ms - хорошо
  ACCEPTABLE: 1000, // < 1s - приемлемо
  SLOW: 2000,       // >= 2s - медленно
} as const

export type TimingRating = 'excellent' | 'good' | 'acceptable' | 'slow'

export interface RequestTiming {
  url: string
  method: string
  duration: number
  rating: TimingRating
  timestamp: Date
  status: number
  success: boolean
  error?: string
}

export interface TimingStats {
  total: number
  successful: number
  failed: number
  excellent: number
  good: number
  acceptable: number
  slow: number
  avgDuration: number
  maxDuration: number
  minDuration: number
  p50: number
  p90: number
  p95: number
  p99: number
  totalDuration: number
}

export interface TestReport {
  testName: string
  startTime: Date
  endTime: Date
  totalDuration: number
  stats: TimingStats
  slowRequests: RequestTiming[]
  failedRequests: RequestTiming[]
  allRequests: RequestTiming[]
}

export interface UserCredentials {
  username: string
  email: string
  password: string
}

export interface AuthTokens {
  accessToken: string
  refreshToken: string
  userId: string
}

/**
 * API Test Client с измерением времени
 */
export class ApiTestClient {
  private baseUrl: string
  private accessToken: string | null = null
  private refreshToken: string | null = null
  private userId: string | null = null
  private requests: RequestTiming[] = []
  private testStartTime: Date | null = null

  constructor(baseUrl: string = 'http://127.0.0.1:8888') {
    this.baseUrl = baseUrl
  }

  /**
   * Начать тест - очистить историю запросов
   */
  startTest(): void {
    this.requests = []
    this.testStartTime = new Date()
  }

  /**
   * Получить рейтинг времени запроса
   */
  private getRating(durationMs: number): TimingRating {
    if (durationMs < TIMING_THRESHOLDS.EXCELLENT) return 'excellent'
    if (durationMs < TIMING_THRESHOLDS.GOOD) return 'good'
    if (durationMs < TIMING_THRESHOLDS.ACCEPTABLE) return 'acceptable'
    return 'slow'
  }

  /**
   * Выполнить HTTP запрос с измерением времени
   */
  async request<T = any>(
    method: string,
    path: string,
    options: {
      body?: any
      headers?: Record<string, string>
      auth?: boolean
    } = {}
  ): Promise<{ data: T; timing: RequestTiming }> {
    const url = `${this.baseUrl}${path}`
    const startTime = Date.now()

    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
      ...options.headers,
    }

    if (options.auth !== false && this.accessToken) {
      headers['Authorization'] = `Bearer ${this.accessToken}`
    }

    let timing: RequestTiming
    let data: T

    try {
      const response = await fetch(url, {
        method,
        headers,
        body: options.body ? JSON.stringify(options.body) : undefined,
      })

      const duration = Date.now() - startTime
      timing = {
        url: path,
        method,
        duration,
        rating: this.getRating(duration),
        timestamp: new Date(),
        status: response.status,
        success: response.ok,
      }

      if (!response.ok) {
        const errorText = await response.text()
        timing.error = errorText
        timing.success = false
      } else {
        const text = await response.text()
        data = text ? JSON.parse(text) : null
      }
    } catch (error) {
      const duration = Date.now() - startTime
      timing = {
        url: path,
        method,
        duration,
        rating: this.getRating(duration),
        timestamp: new Date(),
        status: 0,
        success: false,
        error: error instanceof Error ? error.message : String(error),
      }
      throw error
    } finally {
      this.requests.push(timing!)
    }

    if (!timing.success) {
      throw new Error(`Request failed: ${method} ${path} - ${timing.status} - ${timing.error}`)
    }

    return { data: data!, timing }
  }

  /**
   * Загрузить файл (multipart/form-data)
   */
  async uploadFile(
    filePath: string,
    fileName: string,
    content: string | Buffer
  ): Promise<{ data: any; timing: RequestTiming }> {
    const url = `${this.baseUrl}/api/files/upload`
    const startTime = Date.now()

    const boundary = `----FormBoundary${Date.now()}`
    const body = [
      `--${boundary}`,
      `Content-Disposition: form-data; name="file"; filename="${fileName}"`,
      'Content-Type: application/octet-stream',
      '',
      content.toString(),
      `--${boundary}--`,
      '',
    ].join('\r\n')

    let timing: RequestTiming
    let data: any

    try {
      const response = await fetch(url, {
        method: 'POST',
        headers: {
          'Content-Type': `multipart/form-data; boundary=${boundary}`,
          'Authorization': `Bearer ${this.accessToken}`,
        },
        body,
      })

      const duration = Date.now() - startTime
      timing = {
        url: '/api/files/upload',
        method: 'POST',
        duration,
        rating: this.getRating(duration),
        timestamp: new Date(),
        status: response.status,
        success: response.ok,
      }

      if (!response.ok) {
        const errorText = await response.text()
        timing.error = errorText
        timing.success = false
        throw new Error(`Upload failed: ${response.status} - ${errorText}`)
      }

      data = await response.json()
    } catch (error) {
      if (!timing!) {
        const duration = Date.now() - startTime
        timing = {
          url: '/api/files/upload',
          method: 'POST',
          duration,
          rating: this.getRating(duration),
          timestamp: new Date(),
          status: 0,
          success: false,
          error: error instanceof Error ? error.message : String(error),
        }
      }
      this.requests.push(timing)
      throw error
    }

    this.requests.push(timing)
    return { data, timing }
  }

  // ==================== Auth API ====================

  /**
   * Регистрация пользователя
   */
  async register(credentials: UserCredentials): Promise<AuthTokens> {
    const { data } = await this.request<any>('POST', '/api/auth/register', {
      body: credentials,
      auth: false,
    })

    this.accessToken = data.access_token
    this.refreshToken = data.refresh_token
    this.userId = data.user?.id

    return {
      accessToken: data.access_token,
      refreshToken: data.refresh_token,
      userId: data.user?.id,
    }
  }

  /**
   * Вход пользователя
   */
  async login(email: string, password: string): Promise<AuthTokens> {
    const { data } = await this.request<any>('POST', '/api/auth/login', {
      body: { email, password },
      auth: false,
    })

    this.accessToken = data.access_token
    this.refreshToken = data.refresh_token
    this.userId = data.user?.id

    return {
      accessToken: data.access_token,
      refreshToken: data.refresh_token,
      userId: data.user?.id,
    }
  }

  /**
   * Обновление токена
   */
  async refreshTokens(): Promise<AuthTokens> {
    const { data } = await this.request<any>('POST', '/api/auth/refresh', {
      body: { refresh_token: this.refreshToken },
      auth: false,
    })

    this.accessToken = data.access_token
    this.refreshToken = data.refresh_token

    return {
      accessToken: data.access_token,
      refreshToken: data.refresh_token,
      userId: this.userId!,
    }
  }

  /**
   * Выход
   */
  async logout(): Promise<void> {
    await this.request('POST', '/api/auth/logout', {
      body: { refresh_token: this.refreshToken },
    })
    this.accessToken = null
    this.refreshToken = null
    this.userId = null
  }

  /**
   * Получить текущего пользователя
   */
  async getMe(): Promise<any> {
    const { data } = await this.request('GET', '/api/auth/me')
    return data
  }

  // ==================== Chat API ====================

  /**
   * Создать чат
   */
  async createChat(name: string, type: 'private' | 'group' = 'group'): Promise<any> {
    const { data } = await this.request('POST', '/api/chats', {
      body: { name, type },
    })
    return data
  }

  /**
   * Получить список чатов
   */
  async getChats(page: number = 1, count: number = 20): Promise<any> {
    const { data } = await this.request('GET', `/api/chats?page=${page}&count=${count}`)
    return data
  }

  /**
   * Получить чат по ID
   */
  async getChat(chatId: string): Promise<any> {
    const { data } = await this.request('GET', `/api/chats/${chatId}`)
    return data
  }

  /**
   * Удалить чат
   */
  async deleteChat(chatId: string): Promise<void> {
    await this.request('DELETE', `/api/chats/${chatId}`)
  }

  // ==================== Messages API ====================

  /**
   * Отправить сообщение
   */
  async sendMessage(
    chatId: string,
    content: string,
    options: { fileLinkIds?: string[]; replyToId?: string } = {}
  ): Promise<any> {
    const { data } = await this.request('POST', `/api/chats/${chatId}/messages`, {
      body: {
        content,
        file_link_ids: options.fileLinkIds,
        reply_to_id: options.replyToId,
      },
    })
    return data
  }

  /**
   * Получить сообщения чата
   */
  async getMessages(chatId: string, page: number = 1, count: number = 50): Promise<any> {
    const { data } = await this.request('GET', `/api/chats/${chatId}/messages?page=${page}&count=${count}`)
    return data
  }

  /**
   * Обновить сообщение
   */
  async updateMessage(messageId: string, content: string): Promise<any> {
    const { data } = await this.request('PUT', `/api/chats/messages/${messageId}`, {
      body: { content },
    })
    return data
  }

  /**
   * Удалить сообщение
   */
  async deleteMessage(messageId: string): Promise<void> {
    await this.request('DELETE', `/api/chats/messages/${messageId}`)
  }

  /**
   * Добавить реакцию
   */
  async addReaction(messageId: string, emoji: string): Promise<void> {
    await this.request('POST', `/api/chats/messages/${messageId}/reactions`, {
      body: { emoji },
    })
  }

  /**
   * Удалить реакцию
   */
  async removeReaction(messageId: string, emoji: string): Promise<void> {
    await this.request('DELETE', `/api/chats/messages/${messageId}/reactions/${encodeURIComponent(emoji)}`)
  }

  // ==================== Participants API ====================

  /**
   * Добавить участника
   */
  async addParticipant(chatId: string, userId: string, role: string = 'user'): Promise<void> {
    await this.request('POST', `/api/chats/${chatId}/participants`, {
      body: { user_id: userId, role },
    })
  }

  /**
   * Получить участников чата
   */
  async getParticipants(chatId: string): Promise<any> {
    const { data } = await this.request('GET', `/api/chats/${chatId}/participants`)
    return data
  }

  // ==================== Threads API ====================

  /**
   * Создать тред
   */
  async createThread(chatId: string, name: string, messageId?: string): Promise<any> {
    const { data } = await this.request('POST', `/api/chats/${chatId}/threads`, {
      body: { name, message_id: messageId },
    })
    return data
  }

  /**
   * Получить треды чата
   */
  async getThreads(chatId: string): Promise<any> {
    const { data } = await this.request('GET', `/api/chats/${chatId}/threads`)
    return data
  }

  // ==================== Presence API ====================

  /**
   * Установить статус присутствия
   * @param status - available, busy, away, dnd
   */
  async setPresenceStatus(status: 'available' | 'busy' | 'away' | 'dnd'): Promise<any> {
    const { data } = await this.request('PUT', '/api/presence/status', {
      body: { status },
    })
    return data
  }

  /**
   * Получить свой статус присутствия
   */
  async getMyPresenceStatus(): Promise<any> {
    const { data } = await this.request('GET', '/api/presence/status')
    return data
  }

  /**
   * Получить статус нескольких пользователей
   */
  async getUsersPresence(userIds: string[]): Promise<any> {
    const { data } = await this.request('GET', `/api/presence/users?ids=${userIds.join(',')}`)
    return data
  }

  /**
   * Зарегистрировать соединение (имитация WebSocket connect)
   */
  async presenceConnect(connectionId: string): Promise<any> {
    const { data } = await this.request('POST', '/api/presence/connect', {
      body: { connection_id: connectionId },
    })
    return data
  }

  /**
   * Отключить соединение (имитация WebSocket disconnect)
   */
  async presenceDisconnect(connectionId: string): Promise<any> {
    const { data } = await this.request('POST', '/api/presence/disconnect', {
      body: { connection_id: connectionId },
    })
    return data
  }

  // ==================== Reporting ====================

  /**
   * Вычислить статистику запросов
   */
  calculateStats(): TimingStats {
    const timings = this.requests
    if (timings.length === 0) {
      return {
        total: 0,
        successful: 0,
        failed: 0,
        excellent: 0,
        good: 0,
        acceptable: 0,
        slow: 0,
        avgDuration: 0,
        maxDuration: 0,
        minDuration: 0,
        p50: 0,
        p90: 0,
        p95: 0,
        p99: 0,
        totalDuration: 0,
      }
    }

    const durations = timings.map(t => t.duration).sort((a, b) => a - b)
    const totalDuration = durations.reduce((a, b) => a + b, 0)

    return {
      total: timings.length,
      successful: timings.filter(t => t.success).length,
      failed: timings.filter(t => !t.success).length,
      excellent: timings.filter(t => t.rating === 'excellent').length,
      good: timings.filter(t => t.rating === 'good').length,
      acceptable: timings.filter(t => t.rating === 'acceptable').length,
      slow: timings.filter(t => t.rating === 'slow').length,
      avgDuration: Math.round(totalDuration / durations.length),
      maxDuration: durations[durations.length - 1],
      minDuration: durations[0],
      p50: durations[Math.floor(durations.length * 0.5)],
      p90: durations[Math.floor(durations.length * 0.9)],
      p95: durations[Math.floor(durations.length * 0.95)],
      p99: durations[Math.floor(durations.length * 0.99)] || durations[durations.length - 1],
      totalDuration,
    }
  }

  /**
   * Завершить тест и получить отчет
   */
  finishTest(testName: string): TestReport {
    const endTime = new Date()
    const stats = this.calculateStats()

    return {
      testName,
      startTime: this.testStartTime || endTime,
      endTime,
      totalDuration: endTime.getTime() - (this.testStartTime?.getTime() || endTime.getTime()),
      stats,
      slowRequests: this.requests.filter(r => r.rating === 'slow'),
      failedRequests: this.requests.filter(r => !r.success),
      allRequests: [...this.requests],
    }
  }

  /**
   * Вывести отчет в консоль
   */
  static printReport(report: TestReport): void {
    const { stats } = report

    console.log('\n' + '='.repeat(70))
    console.log(`  TEST REPORT: ${report.testName}`)
    console.log('='.repeat(70))

    console.log(`\n  Test Duration: ${(report.totalDuration / 1000).toFixed(2)}s`)
    console.log(`  Time: ${report.startTime.toISOString()} -> ${report.endTime.toISOString()}`)

    console.log(`\n  REQUEST STATISTICS (${stats.total} requests):`)
    console.log('  ' + '-'.repeat(50))
    console.log(`  Successful:   ${stats.successful} (${this.percent(stats.successful, stats.total)})`)
    console.log(`  Failed:       ${stats.failed} (${this.percent(stats.failed, stats.total)})`)
    console.log('')
    console.log(`  [EXCELLENT]   < 100ms:  ${stats.excellent} (${this.percent(stats.excellent, stats.total)})`)
    console.log(`  [GOOD]        < 300ms:  ${stats.good} (${this.percent(stats.good, stats.total)})`)
    console.log(`  [ACCEPTABLE]  < 1s:     ${stats.acceptable} (${this.percent(stats.acceptable, stats.total)})`)
    console.log(`  [SLOW]        >= 1s:    ${stats.slow} (${this.percent(stats.slow, stats.total)})`)

    console.log(`\n  TIMING METRICS:`)
    console.log('  ' + '-'.repeat(50))
    console.log(`  Average:  ${stats.avgDuration}ms`)
    console.log(`  Min:      ${stats.minDuration}ms`)
    console.log(`  Max:      ${stats.maxDuration}ms`)
    console.log(`  P50:      ${stats.p50}ms`)
    console.log(`  P90:      ${stats.p90}ms`)
    console.log(`  P95:      ${stats.p95}ms`)
    console.log(`  P99:      ${stats.p99}ms`)
    console.log(`  Total:    ${stats.totalDuration}ms`)

    if (report.slowRequests.length > 0) {
      console.log(`\n  SLOW REQUESTS (${report.slowRequests.length}):`)
      console.log('  ' + '-'.repeat(50))
      report.slowRequests.slice(0, 10).forEach((req, i) => {
        console.log(`  ${i + 1}. ${req.method.padEnd(6)} ${req.url.substring(0, 40).padEnd(42)} ${req.duration}ms`)
      })
      if (report.slowRequests.length > 10) {
        console.log(`  ... and ${report.slowRequests.length - 10} more`)
      }
    }

    if (report.failedRequests.length > 0) {
      console.log(`\n  FAILED REQUESTS (${report.failedRequests.length}):`)
      console.log('  ' + '-'.repeat(50))
      report.failedRequests.slice(0, 5).forEach((req, i) => {
        console.log(`  ${i + 1}. ${req.method} ${req.url} - ${req.status} - ${req.error?.substring(0, 50)}`)
      })
    }

    // Verdict
    console.log('\n' + '-'.repeat(70))
    const slowPercent = (stats.slow / stats.total) * 100
    const failPercent = (stats.failed / stats.total) * 100

    if (stats.failed > 0) {
      console.log(`  VERDICT: FAIL - ${stats.failed} failed request(s)`)
    } else if (slowPercent > 10) {
      console.log(`  VERDICT: WARNING - ${slowPercent.toFixed(1)}% slow requests`)
    } else if (slowPercent > 0) {
      console.log(`  VERDICT: OK - ${stats.slow} slow request(s) (${slowPercent.toFixed(1)}%)`)
    } else {
      console.log(`  VERDICT: EXCELLENT - All requests within limits`)
    }
    console.log('='.repeat(70) + '\n')
  }

  private static percent(value: number, total: number): string {
    if (total === 0) return '0.0%'
    return `${((value / total) * 100).toFixed(1)}%`
  }

  /**
   * Получить все запросы
   */
  getRequests(): RequestTiming[] {
    return [...this.requests]
  }

  /**
   * Очистить данные
   */
  reset(): void {
    this.requests = []
    this.accessToken = null
    this.refreshToken = null
    this.userId = null
    this.testStartTime = null
  }

  /**
   * Установить токены вручную
   */
  setTokens(accessToken: string, refreshToken: string, userId: string): void {
    this.accessToken = accessToken
    this.refreshToken = refreshToken
    this.userId = userId
  }

  /**
   * Получить текущий userId
   */
  getUserId(): string | null {
    return this.userId
  }

  /**
   * Получить accessToken
   */
  getAccessToken(): string | null {
    return this.accessToken
  }
}

/**
 * Генерация уникальных тестовых данных
 */
export function generateTestUser(prefix: string = 'apitest'): UserCredentials {
  const timestamp = Date.now()
  const random = Math.random().toString(36).substring(2, 8)
  return {
    username: `${prefix}_${timestamp}_${random}`,
    email: `${prefix}_${timestamp}_${random}@test.local`,
    password: 'TestPass123!',
  }
}

/**
 * Пауза
 */
export function sleep(ms: number): Promise<void> {
  return new Promise(resolve => setTimeout(resolve, ms))
}
