import { Page, Locator, expect } from '@playwright/test'

export class ChatPage {
  readonly page: Page
  readonly header: Locator
  readonly logoutButton: Locator
  readonly userName: Locator
  readonly chatSidebar: Locator
  readonly createChatButton: Locator
  readonly chatList: Locator
  readonly emptyState: Locator

  constructor(page: Page) {
    this.page = page
    this.header = page.locator('header')
    this.logoutButton = page.locator('button:has-text("Logout")')
    this.userName = page.locator('header span.text-gray-600')
    this.chatSidebar = page.locator('aside')
    this.createChatButton = page.locator('aside button[title="Create new chat"]')
    this.chatList = page.locator('aside .divide-y')
    this.emptyState = page.locator('text=Select a chat')
  }

  async goto() {
    await this.page.goto('/chat')
  }

  async expectOnChatPage() {
    await expect(this.header).toBeVisible()
    await expect(this.chatSidebar).toBeVisible()
  }

  async expectUserName(name: string) {
    await expect(this.userName).toContainText(name)
  }

  async logout() {
    await this.logoutButton.click()
  }

  async expectEmptyState() {
    await expect(this.emptyState).toBeVisible()
  }

  async createChat() {
    await this.createChatButton.click()
  }
}
