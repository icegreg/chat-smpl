import { WebDriver, By } from 'selenium-webdriver'
import { BasePage } from './BasePage.js'

export class ChatPage extends BasePage {
  // Locators
  private readonly header = By.css('header')
  private readonly logoutButton = By.xpath('//button[contains(text(), "Logout")]')
  private readonly userName = By.css('header span.text-gray-600')
  private readonly chatSidebar = By.css('aside')
  private readonly createChatButton = By.css('aside button[title="Create new chat"]')
  private readonly emptyState = By.xpath('//*[contains(text(), "Select a chat")]')

  // Create Chat Modal Locators
  private readonly createChatModal = By.css('.fixed.inset-0.bg-black')
  private readonly createChatModalTitle = By.xpath('//h2[contains(text(), "Create New Chat")]')
  private readonly chatNameInput = By.css('input#name')
  private readonly chatDescriptionInput = By.css('textarea#description')
  private readonly chatParticipantsInput = By.css('input#participants')
  private readonly chatTypeGroup = By.css('input[value="group"]')
  private readonly chatTypeChannel = By.css('input[value="channel"]')
  private readonly createChatSubmit = By.xpath('//button[.//span[contains(text(), "Create Chat")]]')
  private readonly createChatCancel = By.xpath('//button[contains(text(), "Cancel")]')
  private readonly createChatError = By.css('.bg-red-50.text-red-700')

  // Chat Room Locators
  private readonly chatHeaderTitle = By.css('main h3.font-semibold')
  private readonly chatHeaderClickable = By.css('[data-testid="chat-header-clickable"]')
  private readonly participantsPanel = By.css('main .w-64.border-l')
  private readonly participantsPanelTitle = By.xpath('//h4[contains(text(), "Participants")]')
  private readonly participantsPanelCloseBtn = By.css('main .w-64.border-l button')
  private readonly participantListItems = By.css('main .w-64.border-l ul li')

  // Chat List Locators
  private readonly chatListItem = By.css('aside .divide-y button')

  constructor(driver: WebDriver) {
    super(driver)
  }

  async goto(): Promise<void> {
    await this.navigate('/chat')
  }

  async isOnChatPage(): Promise<boolean> {
    const url = await this.getCurrentUrl()
    return url.includes('/chat')
  }

  async isHeaderVisible(): Promise<boolean> {
    return this.isDisplayed(this.header)
  }

  async isSidebarVisible(): Promise<boolean> {
    return this.isDisplayed(this.chatSidebar)
  }

  async getUserName(): Promise<string> {
    return this.getText(this.userName)
  }

  async expectUserName(name: string): Promise<boolean> {
    const displayedName = await this.getUserName()
    return displayedName.includes(name)
  }

  async isLogoutButtonVisible(): Promise<boolean> {
    return this.isDisplayed(this.logoutButton)
  }

  async logout(): Promise<void> {
    await this.click(this.logoutButton)
  }

  async isEmptyStateVisible(): Promise<boolean> {
    return this.isDisplayed(this.emptyState)
  }

  async clickCreateChat(): Promise<void> {
    await this.click(this.createChatButton)
  }

  async waitForChatPage(): Promise<void> {
    await this.waitForUrl('/chat')
    await this.waitForElement(this.header)
  }

  // Chat creation methods
  async openCreateChatModal(): Promise<void> {
    await this.click(this.createChatButton)
    await this.waitForElement(this.createChatModal)
  }

  async isCreateChatModalVisible(): Promise<boolean> {
    return this.isDisplayed(this.createChatModalTitle)
  }

  async enterChatName(name: string): Promise<void> {
    await this.type(this.chatNameInput, name)
  }

  async enterChatDescription(description: string): Promise<void> {
    await this.type(this.chatDescriptionInput, description)
  }

  async enterParticipantIds(ids: string): Promise<void> {
    await this.type(this.chatParticipantsInput, ids)
  }

  async selectChatTypeGroup(): Promise<void> {
    await this.click(this.chatTypeGroup)
  }

  async selectChatTypeChannel(): Promise<void> {
    await this.click(this.chatTypeChannel)
  }

  async submitCreateChat(): Promise<void> {
    await this.click(this.createChatSubmit)
  }

  async cancelCreateChat(): Promise<void> {
    await this.click(this.createChatCancel)
  }

  async createChat(name: string, type: 'group' | 'channel' = 'group', description?: string): Promise<void> {
    await this.openCreateChatModal()
    await this.enterChatName(name)
    if (type === 'channel') {
      await this.selectChatTypeChannel()
    }
    if (description) {
      await this.enterChatDescription(description)
    }
    await this.submitCreateChat()
    await this.sleep(1000) // Wait for chat creation
  }

  async getCreateChatError(): Promise<string> {
    try {
      return await this.getText(this.createChatError)
    } catch {
      return ''
    }
  }

  // Chat list methods
  async getChatCount(): Promise<number> {
    const elements = await this.driver.findElements(this.chatListItem)
    return elements.length
  }

  async getChatNames(): Promise<string[]> {
    const elements = await this.driver.findElements(this.chatListItem)
    const names: string[] = []
    for (const el of elements) {
      // Get the chat name from the span.font-medium element
      const nameSpan = await el.findElement(By.css('span.font-medium'))
      const text = await nameSpan.getText()
      names.push(text.trim())
    }
    return names
  }

  async clickChatByName(name: string): Promise<void> {
    const chatItem = By.xpath(`//aside//div[contains(text(), "${name}")]`)
    await this.click(chatItem)
  }

  async isModalClosed(): Promise<boolean> {
    return !(await this.isDisplayed(this.createChatModal))
  }

  async waitForModalToClose(timeout: number = 15000): Promise<void> {
    const start = Date.now()
    while (Date.now() - start < timeout) {
      if (await this.isModalClosed()) {
        return
      }
      await this.sleep(200)
    }
    throw new Error('Modal did not close within timeout')
  }

  // Chat room methods - participants panel
  async clickChatHeader(): Promise<void> {
    await this.click(this.chatHeaderClickable)
  }

  async getChatHeaderTitle(): Promise<string> {
    return this.getText(this.chatHeaderTitle)
  }

  async isParticipantsPanelVisible(): Promise<boolean> {
    return this.isDisplayed(this.participantsPanelTitle)
  }

  async closeParticipantsPanel(): Promise<void> {
    await this.click(this.participantsPanelCloseBtn)
  }

  async getParticipantsCount(): Promise<number> {
    const elements = await this.driver.findElements(this.participantListItems)
    return elements.length
  }

  async waitForParticipantsPanel(timeout: number = 5000): Promise<void> {
    await this.waitForElement(this.participantsPanel, timeout)
  }

  async selectFirstChat(): Promise<void> {
    const elements = await this.driver.findElements(this.chatListItem)
    if (elements.length > 0) {
      await elements[0].click()
    }
  }
}
