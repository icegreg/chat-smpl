import { WebDriver, By, WebElement, until, Key } from 'selenium-webdriver'
import { BASE_URL } from '../config/webdriver.js'

export class BasePage {
  protected driver: WebDriver
  protected baseUrl: string

  constructor(driver: WebDriver) {
    this.driver = driver
    this.baseUrl = BASE_URL
  }

  async navigate(path: string): Promise<void> {
    await this.driver.get(`${this.baseUrl}${path}`)
  }

  async getCurrentUrl(): Promise<string> {
    return this.driver.getCurrentUrl()
  }

  async getTitle(): Promise<string> {
    return this.driver.getTitle()
  }

  // Element finders
  async findElement(locator: By): Promise<WebElement> {
    return this.driver.findElement(locator)
  }

  async findElements(locator: By): Promise<WebElement[]> {
    return this.driver.findElements(locator)
  }

  async waitForElement(locator: By, timeout = 10000): Promise<WebElement> {
    return this.driver.wait(until.elementLocated(locator), timeout)
  }

  async waitForElementVisible(locator: By, timeout = 10000): Promise<WebElement> {
    const element = await this.waitForElement(locator, timeout)
    await this.driver.wait(until.elementIsVisible(element), timeout)
    return element
  }

  async waitForElementClickable(locator: By, timeout = 10000): Promise<WebElement> {
    const element = await this.waitForElement(locator, timeout)
    await this.driver.wait(until.elementIsEnabled(element), timeout)
    return element
  }

  // Element interactions
  async click(locator: By): Promise<void> {
    const element = await this.waitForElementClickable(locator)
    await element.click()
  }

  async type(locator: By, text: string): Promise<void> {
    const element = await this.waitForElementVisible(locator)
    await element.clear()
    await element.sendKeys(text)
  }

  async getText(locator: By): Promise<string> {
    const element = await this.waitForElementVisible(locator)
    return element.getText()
  }

  async getAttribute(locator: By, attribute: string): Promise<string | null> {
    const element = await this.waitForElement(locator)
    return element.getAttribute(attribute)
  }

  async isDisplayed(locator: By): Promise<boolean> {
    try {
      const element = await this.driver.findElement(locator)
      return element.isDisplayed()
    } catch {
      return false
    }
  }

  async isEnabled(locator: By): Promise<boolean> {
    try {
      const element = await this.driver.findElement(locator)
      return element.isEnabled()
    } catch {
      return false
    }
  }

  // Wait utilities
  async waitForUrl(urlPattern: string | RegExp, timeout = 10000): Promise<void> {
    await this.driver.wait(async () => {
      const currentUrl = await this.getCurrentUrl()
      if (typeof urlPattern === 'string') {
        return currentUrl.includes(urlPattern)
      }
      return urlPattern.test(currentUrl)
    }, timeout)
  }

  async waitForUrlExact(url: string, timeout = 10000): Promise<void> {
    await this.driver.wait(until.urlIs(`${this.baseUrl}${url}`), timeout)
  }

  async waitForTextInElement(locator: By, text: string, timeout = 10000): Promise<void> {
    await this.driver.wait(async () => {
      try {
        const element = await this.driver.findElement(locator)
        const elementText = await element.getText()
        return elementText.includes(text)
      } catch {
        return false
      }
    }, timeout)
  }

  // Keyboard actions
  async pressEnter(locator: By): Promise<void> {
    const element = await this.findElement(locator)
    await element.sendKeys(Key.ENTER)
  }

  async pressTab(): Promise<void> {
    await this.driver.actions().sendKeys(Key.TAB).perform()
  }

  // JavaScript execution
  async executeScript<T>(script: string, ...args: unknown[]): Promise<T> {
    return this.driver.executeScript(script, ...args)
  }

  async clearLocalStorage(): Promise<void> {
    await this.executeScript('localStorage.clear()')
  }

  async getLocalStorageItem(key: string): Promise<string | null> {
    return this.executeScript(`return localStorage.getItem('${key}')`)
  }

  async setLocalStorageItem(key: string, value: string): Promise<void> {
    await this.executeScript(`localStorage.setItem('${key}', '${value}')`)
  }

  // Page refresh
  async refresh(): Promise<void> {
    await this.driver.navigate().refresh()
  }

  async goBack(): Promise<void> {
    await this.driver.navigate().back()
  }

  // Screenshots
  async takeScreenshot(): Promise<string> {
    return this.driver.takeScreenshot()
  }

  // Explicit sleep (use sparingly)
  async sleep(ms: number): Promise<void> {
    return new Promise((resolve) => setTimeout(resolve, ms))
  }
}
