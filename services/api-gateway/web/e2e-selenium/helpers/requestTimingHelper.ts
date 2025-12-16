/**
 * Request Timing Helper - мониторинг времени выполнения API запросов
 *
 * Критерии оценки:
 * - EXCELLENT: < 500ms
 * - GOOD: < 1000ms
 * - ACCEPTABLE: < 2000ms
 * - SLOW: >= 2000ms (считается плохим)
 */
import { WebDriver } from 'selenium-webdriver'

// Пороговые значения в миллисекундах
export const TIMING_THRESHOLDS = {
  EXCELLENT: 500,
  GOOD: 1000,
  SLOW: 2000,
} as const

export type TimingRating = 'excellent' | 'good' | 'acceptable' | 'slow'

export interface RequestTiming {
  url: string
  method: string
  duration: number
  rating: TimingRating
  timestamp: Date
  status?: number
}

export interface TimingStats {
  total: number
  excellent: number
  good: number
  acceptable: number
  slow: number
  avgDuration: number
  maxDuration: number
  minDuration: number
  p50: number
  p90: number
  p99: number
}

export interface TimingSummary {
  stats: TimingStats
  slowRequests: RequestTiming[]
  allRequests: RequestTiming[]
}

/**
 * RequestTimingHelper - мониторинг таймингов API запросов через CDP
 */
export class RequestTimingHelper {
  private driver: WebDriver
  private requests: Map<string, { startTime: number; url: string; method: string }> = new Map()
  private completedRequests: RequestTiming[] = []
  private isMonitoring = false
  private cdpConnection: any = null

  constructor(driver: WebDriver) {
    this.driver = driver
  }

  /**
   * Инициализация CDP соединения и начало мониторинга
   */
  async startMonitoring(): Promise<void> {
    if (this.isMonitoring) return

    try {
      // Очищаем предыдущие данные
      this.requests.clear()
      this.completedRequests = []

      // Создаем CDP соединение
      this.cdpConnection = await (this.driver as any).createCDPConnection('page')

      // Включаем Network domain
      await this.cdpConnection.execute('Network.enable', {})

      // Слушаем события requestWillBeSent и responseReceived через JS в браузере
      // CDP events через Selenium не всегда работают надежно, используем Performance API
      await this.injectPerformanceObserver()

      this.isMonitoring = true
      console.log('[RequestTimingHelper] Monitoring started')
    } catch (error) {
      console.warn('[RequestTimingHelper] CDP init failed, using fallback:', error)
      // Fallback: используем только Performance API
      await this.injectPerformanceObserver()
      this.isMonitoring = true
    }
  }

  /**
   * Внедряем PerformanceObserver для отслеживания запросов
   */
  private async injectPerformanceObserver(): Promise<void> {
    await this.driver.executeScript(`
      window.__requestTimings = window.__requestTimings || [];
      window.__timingObserver = window.__timingObserver || null;

      // Очищаем предыдущие данные
      window.__requestTimings = [];

      // Останавливаем предыдущий observer если есть
      if (window.__timingObserver) {
        window.__timingObserver.disconnect();
      }

      // Создаем новый observer для Resource Timing API
      window.__timingObserver = new PerformanceObserver((list) => {
        for (const entry of list.getEntries()) {
          if (entry.entryType === 'resource' &&
              (entry.initiatorType === 'fetch' || entry.initiatorType === 'xmlhttprequest')) {
            // Фильтруем только API запросы
            if (entry.name.includes('/api/')) {
              window.__requestTimings.push({
                url: entry.name,
                duration: entry.duration,
                startTime: entry.startTime,
                responseEnd: entry.responseEnd,
                transferSize: entry.transferSize || 0
              });
            }
          }
        }
      });

      window.__timingObserver.observe({ entryTypes: ['resource'] });

      // Также перехватываем fetch для получения метода и статуса
      if (!window.__fetchIntercepted) {
        window.__fetchIntercepted = true;
        const originalFetch = window.fetch;
        window.fetch = async function(...args) {
          const startTime = performance.now();
          const url = typeof args[0] === 'string' ? args[0] : args[0].url;
          const method = args[1]?.method || 'GET';

          try {
            const response = await originalFetch.apply(this, args);
            const duration = performance.now() - startTime;

            if (url.includes('/api/')) {
              window.__requestTimings.push({
                url: url,
                method: method,
                duration: duration,
                status: response.status,
                timestamp: Date.now()
              });
            }
            return response;
          } catch (error) {
            const duration = performance.now() - startTime;
            if (url.includes('/api/')) {
              window.__requestTimings.push({
                url: url,
                method: method,
                duration: duration,
                status: 0,
                error: true,
                timestamp: Date.now()
              });
            }
            throw error;
          }
        };
      }
    `)
  }

  /**
   * Получить рейтинг для времени запроса
   */
  getRating(durationMs: number): TimingRating {
    if (durationMs < TIMING_THRESHOLDS.EXCELLENT) return 'excellent'
    if (durationMs < TIMING_THRESHOLDS.GOOD) return 'good'
    if (durationMs < TIMING_THRESHOLDS.SLOW) return 'acceptable'
    return 'slow'
  }

  /**
   * Получить emoji для рейтинга
   */
  getRatingEmoji(rating: TimingRating): string {
    switch (rating) {
      case 'excellent': return '🚀'
      case 'good': return '✅'
      case 'acceptable': return '⚠️'
      case 'slow': return '🐌'
    }
  }

  /**
   * Получить цвет для рейтинга (для консоли)
   */
  getRatingColor(rating: TimingRating): string {
    switch (rating) {
      case 'excellent': return '\x1b[32m' // green
      case 'good': return '\x1b[36m' // cyan
      case 'acceptable': return '\x1b[33m' // yellow
      case 'slow': return '\x1b[31m' // red
    }
  }

  /**
   * Собрать тайминги из браузера
   */
  async collectTimings(): Promise<RequestTiming[]> {
    const rawTimings = await this.driver.executeScript(`
      return window.__requestTimings || [];
    `) as any[]

    const timings: RequestTiming[] = rawTimings.map(t => ({
      url: t.url,
      method: t.method || 'GET',
      duration: Math.round(t.duration),
      rating: this.getRating(t.duration),
      timestamp: new Date(t.timestamp || Date.now()),
      status: t.status
    }))

    // Добавляем к общему списку
    this.completedRequests.push(...timings)

    // Очищаем в браузере
    await this.driver.executeScript(`
      window.__requestTimings = [];
    `)

    return timings
  }

  /**
   * Получить статистику
   */
  calculateStats(timings: RequestTiming[]): TimingStats {
    if (timings.length === 0) {
      return {
        total: 0,
        excellent: 0,
        good: 0,
        acceptable: 0,
        slow: 0,
        avgDuration: 0,
        maxDuration: 0,
        minDuration: 0,
        p50: 0,
        p90: 0,
        p99: 0
      }
    }

    const durations = timings.map(t => t.duration).sort((a, b) => a - b)

    return {
      total: timings.length,
      excellent: timings.filter(t => t.rating === 'excellent').length,
      good: timings.filter(t => t.rating === 'good').length,
      acceptable: timings.filter(t => t.rating === 'acceptable').length,
      slow: timings.filter(t => t.rating === 'slow').length,
      avgDuration: Math.round(durations.reduce((a, b) => a + b, 0) / durations.length),
      maxDuration: durations[durations.length - 1],
      minDuration: durations[0],
      p50: durations[Math.floor(durations.length * 0.5)],
      p90: durations[Math.floor(durations.length * 0.9)],
      p99: durations[Math.floor(durations.length * 0.99)]
    }
  }

  /**
   * Остановить мониторинг и получить сводку
   */
  async stopMonitoring(): Promise<TimingSummary> {
    // Собираем оставшиеся тайминги
    await this.collectTimings()

    this.isMonitoring = false

    // Останавливаем observer в браузере
    await this.driver.executeScript(`
      if (window.__timingObserver) {
        window.__timingObserver.disconnect();
      }
    `)

    const stats = this.calculateStats(this.completedRequests)
    const slowRequests = this.completedRequests.filter(t => t.rating === 'slow')

    return {
      stats,
      slowRequests,
      allRequests: [...this.completedRequests]
    }
  }

  /**
   * Вывести красивый отчет в консоль
   */
  printReport(summary: TimingSummary, testName?: string): void {
    const { stats, slowRequests } = summary
    const reset = '\x1b[0m'

    console.log('\n' + '═'.repeat(60))
    console.log(`📊 REQUEST TIMING REPORT${testName ? ` - ${testName}` : ''}`)
    console.log('═'.repeat(60))

    console.log(`\n📈 Statistics (${stats.total} requests):`)
    console.log(`   ${this.getRatingEmoji('excellent')} Excellent (<500ms): ${stats.excellent} (${this.percent(stats.excellent, stats.total)})`)
    console.log(`   ${this.getRatingEmoji('good')} Good (<1s):        ${stats.good} (${this.percent(stats.good, stats.total)})`)
    console.log(`   ${this.getRatingEmoji('acceptable')} Acceptable (<2s):   ${stats.acceptable} (${this.percent(stats.acceptable, stats.total)})`)
    console.log(`   ${this.getRatingEmoji('slow')} Slow (>=2s):        ${stats.slow} (${this.percent(stats.slow, stats.total)})`)

    console.log(`\n⏱️  Timing Metrics:`)
    console.log(`   Average: ${stats.avgDuration}ms`)
    console.log(`   Min:     ${stats.minDuration}ms`)
    console.log(`   Max:     ${stats.maxDuration}ms`)
    console.log(`   P50:     ${stats.p50}ms`)
    console.log(`   P90:     ${stats.p90}ms`)
    console.log(`   P99:     ${stats.p99}ms`)

    if (slowRequests.length > 0) {
      console.log(`\n🐌 Slow Requests (>= 2s):`)
      slowRequests.forEach((req, i) => {
        const urlPath = new URL(req.url, 'http://localhost').pathname
        console.log(`   ${i + 1}. ${req.method} ${urlPath} - ${req.duration}ms`)
      })
    }

    // Общий вердикт
    const slowPercent = (stats.slow / stats.total) * 100
    console.log('\n' + '─'.repeat(60))
    if (slowPercent === 0) {
      console.log('✅ VERDICT: All requests within acceptable limits')
    } else if (slowPercent < 5) {
      console.log(`⚠️  VERDICT: ${stats.slow} slow request(s) detected (${slowPercent.toFixed(1)}%)`)
    } else {
      console.log(`❌ VERDICT: Too many slow requests: ${stats.slow} (${slowPercent.toFixed(1)}%)`)
    }
    console.log('═'.repeat(60) + '\n')
  }

  private percent(value: number, total: number): string {
    if (total === 0) return '0%'
    return `${((value / total) * 100).toFixed(1)}%`
  }

  /**
   * Проверить, что нет медленных запросов (для assertions в тестах)
   */
  hasSlowRequests(summary: TimingSummary): boolean {
    return summary.stats.slow > 0
  }

  /**
   * Получить процент медленных запросов
   */
  getSlowRequestPercent(summary: TimingSummary): number {
    if (summary.stats.total === 0) return 0
    return (summary.stats.slow / summary.stats.total) * 100
  }

  /**
   * Cleanup
   */
  async cleanup(): Promise<void> {
    if (this.cdpConnection) {
      try {
        await this.cdpConnection.execute('Network.disable', {})
      } catch (e) {
        // Ignore
      }
    }

    await this.driver.executeScript(`
      if (window.__timingObserver) {
        window.__timingObserver.disconnect();
        window.__timingObserver = null;
      }
      window.__requestTimings = [];
    `)

    this.requests.clear()
    this.completedRequests = []
    this.isMonitoring = false
  }
}

/**
 * Factory function
 */
export async function createRequestTimingHelper(driver: WebDriver): Promise<RequestTimingHelper> {
  const helper = new RequestTimingHelper(driver)
  await helper.startMonitoring()
  return helper
}
