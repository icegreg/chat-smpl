import { Builder, WebDriver, Browser } from 'selenium-webdriver'
import chrome from 'selenium-webdriver/chrome.js'

const BASE_URL = process.env.BASE_URL || 'http://localhost:3000'
const HEADLESS = process.env.HEADLESS !== 'false'

export { BASE_URL }

export async function createDriver(): Promise<WebDriver> {
  const options = new chrome.Options()

  if (HEADLESS) {
    options.addArguments('--headless=new')
  }

  options.addArguments(
    '--no-sandbox',
    '--disable-dev-shm-usage',
    '--disable-gpu',
    '--window-size=1920,1080',
    '--disable-extensions',
    '--disable-infobars'
  )

  const driver = await new Builder()
    .forBrowser(Browser.CHROME)
    .setChromeOptions(options)
    .build()

  // Set implicit wait
  await driver.manage().setTimeouts({
    implicit: 5000,
    pageLoad: 30000,
    script: 30000,
  })

  return driver
}

export async function quitDriver(driver: WebDriver): Promise<void> {
  if (driver) {
    await driver.quit()
  }
}
