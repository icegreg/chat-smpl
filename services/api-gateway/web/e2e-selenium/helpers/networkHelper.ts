/**
 * Network emulation helper using Chrome DevTools Protocol (CDP)
 * Позволяет эмулировать различные сетевые условия для e2e тестов
 */
import { WebDriver } from 'selenium-webdriver'

// Network condition presets
export const NetworkConditions = {
  // Полное отключение сети
  OFFLINE: {
    offline: true,
    latency: 0,
    downloadThroughput: 0,
    uploadThroughput: 0,
  },
  // Нормальное соединение
  ONLINE: {
    offline: false,
    latency: 0,
    downloadThroughput: -1, // No limit
    uploadThroughput: -1,
  },
  // Медленный 3G
  SLOW_3G: {
    offline: false,
    latency: 2000, // 2 секунды задержки
    downloadThroughput: 50 * 1024, // 50 KB/s
    uploadThroughput: 25 * 1024, // 25 KB/s
  },
  // Быстрый 3G
  FAST_3G: {
    offline: false,
    latency: 500, // 500ms задержки
    downloadThroughput: 200 * 1024, // 200 KB/s
    uploadThroughput: 100 * 1024, // 100 KB/s
  },
  // Нестабильное соединение (высокая латентность)
  UNSTABLE: {
    offline: false,
    latency: 1000, // 1 секунда
    downloadThroughput: 100 * 1024,
    uploadThroughput: 50 * 1024,
  },
  // Очень медленное соединение
  VERY_SLOW: {
    offline: false,
    latency: 3000, // 3 секунды
    downloadThroughput: 10 * 1024, // 10 KB/s
    uploadThroughput: 5 * 1024, // 5 KB/s
  },
} as const

export type NetworkConditionName = keyof typeof NetworkConditions

interface NetworkCondition {
  offline: boolean
  latency: number
  downloadThroughput: number
  uploadThroughput: number
}

/**
 * NetworkHelper - управление сетевыми условиями через CDP
 */
export class NetworkHelper {
  private driver: WebDriver
  private cdpConnection: any = null
  private isNetworkEmulationEnabled = false

  constructor(driver: WebDriver) {
    this.driver = driver
  }

  /**
   * Инициализация CDP соединения
   */
  async init(): Promise<void> {
    try {
      // Selenium 4 способ получения CDP
      this.cdpConnection = await (this.driver as any).createCDPConnection('page')
      console.log('[NetworkHelper] CDP connection established')
    } catch (error) {
      console.warn('[NetworkHelper] Could not create CDP connection:', error)
      // Fallback: попробуем через execute CDP command напрямую
      this.cdpConnection = null
    }
  }

  /**
   * Установить сетевые условия по имени пресета
   */
  async setNetworkCondition(conditionName: NetworkConditionName): Promise<void> {
    const condition = NetworkConditions[conditionName]
    await this.setCustomNetworkCondition(condition)
    console.log(`[NetworkHelper] Network set to: ${conditionName}`)
  }

  /**
   * Установить произвольные сетевые условия
   */
  async setCustomNetworkCondition(condition: NetworkCondition): Promise<void> {
    try {
      if (this.cdpConnection) {
        // Через CDP connection
        await this.cdpConnection.execute('Network.enable', {})
        await this.cdpConnection.execute('Network.emulateNetworkConditions', condition)
      } else {
        // Fallback через executeCdpCommand (Selenium 4)
        await (this.driver as any).executeCdpCommand('Network.enable', {})
        await (this.driver as any).executeCdpCommand('Network.emulateNetworkConditions', condition)
      }
      this.isNetworkEmulationEnabled = true
    } catch (error) {
      console.error('[NetworkHelper] Failed to set network condition:', error)
      throw error
    }
  }

  /**
   * Перейти в offline режим
   */
  async goOffline(): Promise<void> {
    await this.setNetworkCondition('OFFLINE')
    console.log('[NetworkHelper] Network is now OFFLINE')
  }

  /**
   * Вернуться в online режим
   */
  async goOnline(): Promise<void> {
    await this.setNetworkCondition('ONLINE')
    console.log('[NetworkHelper] Network is now ONLINE')
  }

  /**
   * Симулировать временное пропадание сети
   * @param durationMs - длительность offline в миллисекундах
   */
  async simulateNetworkBlip(durationMs: number): Promise<void> {
    console.log(`[NetworkHelper] Simulating network blip for ${durationMs}ms`)
    await this.goOffline()
    await this.sleep(durationMs)
    await this.goOnline()
    console.log('[NetworkHelper] Network blip completed')
  }

  /**
   * Симулировать нестабильное соединение (периодические отключения)
   * @param cycles - количество циклов online/offline
   * @param onlineDurationMs - время в online режиме
   * @param offlineDurationMs - время в offline режиме
   */
  async simulateUnstableConnection(
    cycles: number,
    onlineDurationMs: number,
    offlineDurationMs: number
  ): Promise<void> {
    console.log(`[NetworkHelper] Simulating unstable connection: ${cycles} cycles`)
    for (let i = 0; i < cycles; i++) {
      await this.goOnline()
      await this.sleep(onlineDurationMs)
      await this.goOffline()
      await this.sleep(offlineDurationMs)
    }
    await this.goOnline()
    console.log('[NetworkHelper] Unstable connection simulation completed')
  }

  /**
   * Включить throttling (медленную сеть)
   */
  async enableThrottling(conditionName: 'SLOW_3G' | 'FAST_3G' | 'UNSTABLE' | 'VERY_SLOW'): Promise<void> {
    await this.setNetworkCondition(conditionName)
  }

  /**
   * Отключить эмуляцию сети (вернуть нормальные условия)
   */
  async disableEmulation(): Promise<void> {
    if (!this.isNetworkEmulationEnabled) return

    try {
      if (this.cdpConnection) {
        await this.cdpConnection.execute('Network.disable', {})
      } else {
        await (this.driver as any).executeCdpCommand('Network.disable', {})
      }
      this.isNetworkEmulationEnabled = false
      console.log('[NetworkHelper] Network emulation disabled')
    } catch (error) {
      console.warn('[NetworkHelper] Failed to disable network emulation:', error)
    }
  }

  /**
   * Блокировать определенные URL паттерны
   */
  async blockUrls(urlPatterns: string[]): Promise<void> {
    try {
      if (this.cdpConnection) {
        await this.cdpConnection.execute('Network.setBlockedURLs', { urls: urlPatterns })
      } else {
        await (this.driver as any).executeCdpCommand('Network.setBlockedURLs', { urls: urlPatterns })
      }
      console.log(`[NetworkHelper] Blocked URLs: ${urlPatterns.join(', ')}`)
    } catch (error) {
      console.error('[NetworkHelper] Failed to block URLs:', error)
    }
  }

  /**
   * Разблокировать все URL
   */
  async unblockAllUrls(): Promise<void> {
    await this.blockUrls([])
    console.log('[NetworkHelper] All URLs unblocked')
  }

  /**
   * Блокировать WebSocket соединения
   */
  async blockWebSocket(): Promise<void> {
    await this.blockUrls(['*/connection/websocket*'])
    console.log('[NetworkHelper] WebSocket connections blocked')
  }

  /**
   * Блокировать только API запросы (не WebSocket)
   */
  async blockApiOnly(): Promise<void> {
    await this.blockUrls(['*/api/*'])
    console.log('[NetworkHelper] API requests blocked (WebSocket still works)')
  }

  /**
   * Helper для ожидания
   */
  private sleep(ms: number): Promise<void> {
    return new Promise(resolve => setTimeout(resolve, ms))
  }

  /**
   * Cleanup при завершении теста
   */
  async cleanup(): Promise<void> {
    await this.disableEmulation()
    await this.unblockAllUrls()
  }
}

/**
 * Factory function для создания NetworkHelper
 */
export async function createNetworkHelper(driver: WebDriver): Promise<NetworkHelper> {
  const helper = new NetworkHelper(driver)
  await helper.init()
  return helper
}
