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

  // Message Input Locators
  private readonly messageInput = By.css('textarea[placeholder="Type a message..."]')
  private readonly sendMessageButton = By.css('main button.bg-indigo-600')
  private readonly messageContents = By.css('main .overflow-y-auto p.whitespace-pre-wrap')
  private readonly messageItems = By.css('[data-testid="message-item"]')
  private readonly emptyMessagesState = By.xpath('//*[contains(text(), "No messages yet")]')
  private readonly typingIndicator = By.css('main .italic')

  // Chat List Locators
  private readonly chatListItem = By.css('aside .divide-y button')

  // File Upload Locators
  private readonly fileInput = By.css('input[type="file"]')
  private readonly pendingFilePreview = By.css('[data-testid="pending-file"]')
  private readonly pendingFileName = By.css('[data-testid="pending-file"] .truncate')
  private readonly pendingFileSpinner = By.css('[data-testid="file-uploading-spinner"]')
  private readonly pendingFileRemoveBtn = By.css('[data-testid="remove-pending-file"]')
  private readonly messageFileAttachments = By.css('main .overflow-y-auto a[href^="/api/files/"]')
  private readonly messageImageAttachments = By.css('main .overflow-y-auto a img.max-w-full')

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

  // Message methods
  async typeMessage(message: string): Promise<void> {
    await this.type(this.messageInput, message)
  }

  async clickSendMessage(): Promise<void> {
    await this.click(this.sendMessageButton)
  }

  async sendMessage(message: string): Promise<void> {
    await this.typeMessage(message)
    await this.clickSendMessage()
  }

  async getMessageCount(): Promise<number> {
    const elements = await this.driver.findElements(this.messageItems)
    return elements.length
  }

  async getMessageTexts(): Promise<string[]> {
    const elements = await this.driver.findElements(this.messageContents)
    const texts: string[] = []
    for (const el of elements) {
      const text = await el.getText()
      texts.push(text.trim())
    }
    return texts
  }

  async getLastMessageText(): Promise<string> {
    const texts = await this.getMessageTexts()
    return texts.length > 0 ? texts[texts.length - 1] : ''
  }

  async isEmptyMessagesStateVisible(): Promise<boolean> {
    return this.isDisplayed(this.emptyMessagesState)
  }

  async waitForMessageCount(expectedCount: number, timeout: number = 10000): Promise<void> {
    const start = Date.now()
    while (Date.now() - start < timeout) {
      const count = await this.getMessageCount()
      if (count >= expectedCount) {
        return
      }
      await this.sleep(300)
    }
    throw new Error(`Expected at least ${expectedCount} messages, but found ${await this.getMessageCount()}`)
  }

  async waitForMessageContaining(text: string, timeout: number = 10000): Promise<void> {
    const start = Date.now()
    while (Date.now() - start < timeout) {
      const messages = await this.getMessageTexts()
      if (messages.some(m => m.includes(text))) {
        return
      }
      await this.sleep(300)
    }
    throw new Error(`Message containing "${text}" not found within timeout`)
  }

  async isTypingIndicatorVisible(): Promise<boolean> {
    return this.isDisplayed(this.typingIndicator)
  }

  // Multi-user test helpers
  async createChatWithParticipants(
    name: string,
    participantIds: string[],
    type: 'group' | 'channel' = 'group',
    description?: string
  ): Promise<void> {
    await this.openCreateChatModal()
    await this.enterChatName(name)
    if (type === 'channel') {
      await this.selectChatTypeChannel()
    }
    if (description) {
      await this.enterChatDescription(description)
    }
    if (participantIds.length > 0) {
      await this.enterParticipantIds(participantIds.join(','))
    }
    await this.submitCreateChat()
    await this.sleep(1000)
  }

  async waitForChatInList(chatName: string, timeout: number = 30000): Promise<void> {
    const start = Date.now()
    while (Date.now() - start < timeout) {
      const names = await this.getChatNames()
      if (names.some(n => n.includes(chatName))) {
        return
      }
      await this.sleep(500)
    }
    throw new Error(`Chat "${chatName}" not found in list within timeout`)
  }

  async waitForChatCount(expectedCount: number, timeout: number = 30000): Promise<void> {
    const start = Date.now()
    while (Date.now() - start < timeout) {
      const count = await this.getChatCount()
      if (count >= expectedCount) {
        return
      }
      await this.sleep(500)
    }
    throw new Error(`Expected at least ${expectedCount} chats, but found ${await this.getChatCount()}`)
  }

  async clickChatByNameInList(name: string): Promise<void> {
    const chatItems = await this.driver.findElements(this.chatListItem)
    for (const item of chatItems) {
      try {
        const nameSpan = await item.findElement(By.css('span.font-medium'))
        const text = await nameSpan.getText()
        if (text.includes(name)) {
          await item.click()
          return
        }
      } catch {
        continue
      }
    }
    throw new Error(`Chat "${name}" not found in list`)
  }

  async waitForChatRoom(timeout: number = 10000): Promise<void> {
    await this.waitForElement(this.chatHeaderTitle, timeout)
  }

  async getTypingIndicatorText(): Promise<string | null> {
    try {
      if (await this.isTypingIndicatorVisible()) {
        return await this.getText(this.typingIndicator)
      }
    } catch {
      // ignore
    }
    return null
  }

  // File upload methods
  async attachFile(filePath: string): Promise<void> {
    // File input is hidden, send keys directly to it
    const fileInputEl = await this.driver.findElement(this.fileInput)
    await fileInputEl.sendKeys(filePath)
  }

  async isPendingFileVisible(): Promise<boolean> {
    return this.isDisplayed(this.pendingFilePreview)
  }

  async getPendingFileName(): Promise<string> {
    try {
      return await this.getText(this.pendingFileName)
    } catch {
      return ''
    }
  }

  async getPendingFilesCount(): Promise<number> {
    const elements = await this.driver.findElements(this.pendingFilePreview)
    return elements.length
  }

  async isFileUploading(): Promise<boolean> {
    return this.isDisplayed(this.pendingFileSpinner)
  }

  async waitForFileUploadComplete(timeout: number = 15000): Promise<void> {
    const start = Date.now()

    // First wait for pending file to appear
    while (Date.now() - start < timeout) {
      if (await this.isPendingFileVisible()) {
        break
      }
      await this.sleep(200)
    }

    if (!(await this.isPendingFileVisible())) {
      throw new Error('Pending file did not appear within timeout')
    }

    // Wait a bit for upload to start
    await this.sleep(300)

    // Then wait for spinner to disappear (upload complete)
    // If no spinner exists, upload already completed (fast upload)
    while (Date.now() - start < timeout) {
      const spinners = await this.driver.findElements(this.pendingFileSpinner)
      if (spinners.length === 0) {
        // No spinners means all uploads complete (or already completed)
        // Verify by checking that remove button is visible (shown when not uploading)
        const removeButtons = await this.driver.findElements(this.pendingFileRemoveBtn)
        if (removeButtons.length > 0) {
          return // Upload complete - remove button visible
        }
        // If neither spinner nor remove button, file might still be processing
        // Wait a bit and check again
      }
      await this.sleep(200)
    }
    throw new Error('File upload did not complete within timeout')
  }

  async removePendingFile(): Promise<void> {
    await this.click(this.pendingFileRemoveBtn)
  }

  async sendMessageWithFile(message: string, filePath: string): Promise<void> {
    await this.attachFile(filePath)
    await this.waitForFileUploadComplete()
    if (message) {
      await this.typeMessage(message)
    }
    await this.clickSendMessage()
  }

  async getFileAttachmentsCount(): Promise<number> {
    const elements = await this.driver.findElements(this.messageFileAttachments)
    return elements.length
  }

  async getImageAttachmentsCount(): Promise<number> {
    const elements = await this.driver.findElements(this.messageImageAttachments)
    return elements.length
  }

  async waitForFileAttachment(timeout: number = 10000): Promise<void> {
    const start = Date.now()
    while (Date.now() - start < timeout) {
      const count = await this.getFileAttachmentsCount()
      if (count > 0) {
        return
      }
      await this.sleep(300)
    }
    throw new Error('File attachment not found within timeout')
  }

  async waitForImageAttachment(timeout: number = 10000): Promise<void> {
    const start = Date.now()
    while (Date.now() - start < timeout) {
      const count = await this.getImageAttachmentsCount()
      if (count > 0) {
        return
      }
      await this.sleep(300)
    }
    throw new Error('Image attachment not found within timeout')
  }

  async getFileAttachmentHrefs(): Promise<string[]> {
    const elements = await this.driver.findElements(this.messageFileAttachments)
    const hrefs: string[] = []
    for (const el of elements) {
      const href = await el.getAttribute('href')
      if (href) {
        hrefs.push(href)
      }
    }
    return hrefs
  }

  // Message display info methods (sender name, avatar, time, date separators)
  private readonly messageSenderName = By.css('[data-testid="message-sender-name"]')
  private readonly messageTime = By.css('[data-testid="message-time"]')
  private readonly messageAvatar = By.css('[data-testid="message-avatar"]')
  private readonly messageAvatarPlaceholder = By.css('[data-testid="message-avatar-placeholder"]')
  private readonly dateSeparator = By.css('[data-testid="date-separator"]')
  private readonly dateLabel = By.css('[data-testid="date-label"]')

  async getMessageSenderNames(): Promise<string[]> {
    const elements = await this.driver.findElements(this.messageSenderName)
    const names: string[] = []
    for (const el of elements) {
      const text = await el.getText()
      names.push(text.trim())
    }
    return names
  }

  async getFirstMessageSenderName(): Promise<string> {
    const names = await this.getMessageSenderNames()
    return names.length > 0 ? names[0] : ''
  }

  async getMessageTimes(): Promise<string[]> {
    const elements = await this.driver.findElements(this.messageTime)
    const times: string[] = []
    for (const el of elements) {
      const text = await el.getText()
      times.push(text.trim())
    }
    return times
  }

  async getFirstMessageTime(): Promise<string> {
    const times = await this.getMessageTimes()
    return times.length > 0 ? times[0] : ''
  }

  async hasMessageAvatar(): Promise<boolean> {
    const avatars = await this.driver.findElements(this.messageAvatar)
    return avatars.length > 0
  }

  async hasMessageAvatarPlaceholder(): Promise<boolean> {
    const placeholders = await this.driver.findElements(this.messageAvatarPlaceholder)
    return placeholders.length > 0
  }

  async getAvatarPlaceholderText(): Promise<string> {
    try {
      const element = await this.driver.findElement(this.messageAvatarPlaceholder)
      return await element.getText()
    } catch {
      return ''
    }
  }

  async getAvatarSrc(): Promise<string | null> {
    try {
      const element = await this.driver.findElement(this.messageAvatar)
      return await element.getAttribute('src')
    } catch {
      return null
    }
  }

  async getDateSeparatorsCount(): Promise<number> {
    const elements = await this.driver.findElements(this.dateSeparator)
    return elements.length
  }

  async getDateLabels(): Promise<string[]> {
    const elements = await this.driver.findElements(this.dateLabel)
    const labels: string[] = []
    for (const el of elements) {
      const text = await el.getText()
      labels.push(text.trim())
    }
    return labels
  }

  async waitForDateSeparator(timeout: number = 10000): Promise<void> {
    const start = Date.now()
    while (Date.now() - start < timeout) {
      const count = await this.getDateSeparatorsCount()
      if (count > 0) {
        return
      }
      await this.sleep(300)
    }
    throw new Error('Date separator not found within timeout')
  }

  async waitForSenderName(expectedName: string, timeout: number = 10000): Promise<void> {
    const start = Date.now()
    while (Date.now() - start < timeout) {
      const names = await this.getMessageSenderNames()
      if (names.some(n => n.includes(expectedName))) {
        return
      }
      await this.sleep(300)
    }
    throw new Error(`Sender name "${expectedName}" not found within timeout`)
  }
}
