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
    // Use forceClick to bypass any overlay issues
    await this.forceClick(this.logoutButton)
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

  // Reply methods locators
  private readonly replyButton = By.css('[data-testid="reply-button"]')
  private readonly replyPreview = By.css('[data-testid="reply-preview"]')
  private readonly replyPreviewContent = By.css('[data-testid="reply-preview-content"]')
  private readonly replyPreviewSender = By.css('[data-testid="reply-preview-sender"]')
  private readonly replyPreviewCancel = By.css('[data-testid="reply-preview-cancel"]')
  private readonly messageQuote = By.css('[data-testid="message-quote"]')
  private readonly messageQuoteContent = By.css('[data-testid="message-quote-content"]')
  private readonly messageQuoteSender = By.css('[data-testid="message-quote-sender"]')

  // Thread methods
  private readonly threadsButton = By.css('[data-testid="threads-button"]')
  private readonly threadsPanelTitle = By.xpath('//h4[contains(text(), "Threads")]')
  private readonly threadsPanelCloseBtn = By.css('[data-testid="threads-panel-close"]')
  private readonly createThreadButton = By.css('[data-testid="create-thread-button"]')
  private readonly threadListItems = By.css('[data-testid="thread-item"]')
  private readonly threadView = By.css('[data-testid="thread-view"]')
  private readonly threadViewTitle = By.css('[data-testid="thread-view"] h4')
  private readonly threadViewBackBtn = By.css('[data-testid="thread-view-back"]')
  private readonly threadViewCloseBtn = By.css('[data-testid="thread-view-close"]')
  private readonly threadMessageInput = By.css('[data-testid="thread-view"] textarea')
  private readonly threadSendButton = By.css('[data-testid="thread-view"] button.bg-indigo-600')
  private readonly threadMessages = By.css('[data-testid="thread-view"] [data-testid="message-item"]')
  private readonly subthreadsButton = By.css('[data-testid="subthreads-button"]')
  private readonly threadDepthBadge = By.css('[data-testid="thread-depth-badge"]')
  private readonly subthreadIndicator = By.css('[data-testid="subthread-indicator"]')

  async openThreadsPanel(): Promise<void> {
    await this.click(this.threadsButton)
    await this.sleep(500)
  }

  async isThreadsPanelVisible(): Promise<boolean> {
    return this.isDisplayed(this.threadsPanelTitle)
  }

  async closeThreadsPanel(): Promise<void> {
    await this.click(this.threadsPanelCloseBtn)
    await this.sleep(300)
  }

  async clickCreateThread(): Promise<void> {
    await this.click(this.createThreadButton)
    await this.sleep(300)
  }

  async getThreadsCount(): Promise<number> {
    const elements = await this.driver.findElements(this.threadListItems)
    return elements.length
  }

  async getThreadTitles(): Promise<string[]> {
    const elements = await this.driver.findElements(this.threadListItems)
    const titles: string[] = []
    for (const el of elements) {
      try {
        const titleEl = await el.findElement(By.css('.font-medium, p.text-sm'))
        const text = await titleEl.getText()
        titles.push(text.trim())
      } catch {
        // ignore
      }
    }
    return titles
  }

  async clickThreadByTitle(title: string): Promise<void> {
    const elements = await this.driver.findElements(this.threadListItems)
    for (const el of elements) {
      try {
        const text = await el.getText()
        if (text.includes(title)) {
          await el.click()
          await this.sleep(500)
          return
        }
      } catch {
        continue
      }
    }
    throw new Error(`Thread "${title}" not found`)
  }

  async clickFirstThread(): Promise<void> {
    const elements = await this.driver.findElements(this.threadListItems)
    if (elements.length > 0) {
      await elements[0].click()
      await this.sleep(500)
    } else {
      throw new Error('No threads found')
    }
  }

  async isThreadViewVisible(): Promise<boolean> {
    return this.isDisplayed(this.threadView)
  }

  async getThreadViewTitle(): Promise<string> {
    try {
      return await this.getText(this.threadViewTitle)
    } catch {
      return ''
    }
  }

  async closeThreadView(): Promise<void> {
    await this.click(this.threadViewCloseBtn)
    await this.sleep(300)
  }

  async goBackFromThreadView(): Promise<void> {
    await this.click(this.threadViewBackBtn)
    await this.sleep(300)
  }

  async sendThreadMessage(message: string): Promise<void> {
    await this.type(this.threadMessageInput, message)
    await this.click(this.threadSendButton)
    await this.sleep(500)
  }

  async getThreadMessageCount(): Promise<number> {
    const elements = await this.driver.findElements(this.threadMessages)
    return elements.length
  }

  async waitForThreadMessage(expectedCount: number, timeout: number = 10000): Promise<void> {
    const start = Date.now()
    while (Date.now() - start < timeout) {
      const count = await this.getThreadMessageCount()
      if (count >= expectedCount) {
        return
      }
      await this.sleep(300)
    }
    throw new Error(`Expected at least ${expectedCount} thread messages, but found ${await this.getThreadMessageCount()}`)
  }

  async clickSubthreadsButton(): Promise<void> {
    await this.click(this.subthreadsButton)
    await this.sleep(500)
  }

  async isSubthreadsButtonVisible(): Promise<boolean> {
    return this.isDisplayed(this.subthreadsButton)
  }

  async getThreadDepth(): Promise<string> {
    try {
      return await this.getText(this.threadDepthBadge)
    } catch {
      return ''
    }
  }

  async hasSubthreadIndicator(): Promise<boolean> {
    return this.isDisplayed(this.subthreadIndicator)
  }

  async waitForThreadsPanel(timeout: number = 10000): Promise<void> {
    const start = Date.now()
    while (Date.now() - start < timeout) {
      if (await this.isThreadsPanelVisible()) {
        return
      }
      await this.sleep(300)
    }
    throw new Error('Threads panel did not appear within timeout')
  }

  async waitForThreadView(timeout: number = 10000): Promise<void> {
    const start = Date.now()
    while (Date.now() - start < timeout) {
      if (await this.isThreadViewVisible()) {
        return
      }
      await this.sleep(300)
    }
    throw new Error('Thread view did not appear within timeout')
  }

  // Create thread via API from browser context
  async createThreadViaApi(chatId: string, title: string): Promise<string> {
    const result = await this.driver.executeScript(`
      return new Promise(async (resolve, reject) => {
        try {
          const token = localStorage.getItem('access_token');
          if (!token) {
            reject('No access token');
            return;
          }
          const response = await fetch('/api/chats/${chatId}/threads', {
            method: 'POST',
            headers: {
              'Authorization': 'Bearer ' + token,
              'Content-Type': 'application/json'
            },
            body: JSON.stringify({
              title: '${title}',
              thread_type: 'user'
            })
          });
          if (!response.ok) {
            const error = await response.text();
            reject('API error: ' + response.status + ' ' + error);
            return;
          }
          const thread = await response.json();
          resolve(thread.id);
        } catch (e) {
          reject(e.message);
        }
      });
    `, chatId, title) as string

    return result
  }

  // Create subthread via API
  async createSubthreadViaApi(parentThreadId: string, title: string): Promise<string> {
    const result = await this.driver.executeScript(`
      return new Promise(async (resolve, reject) => {
        try {
          const token = localStorage.getItem('access_token');
          if (!token) {
            reject('No access token');
            return;
          }
          const response = await fetch('/api/chats/threads/${parentThreadId}/subthreads', {
            method: 'POST',
            headers: {
              'Authorization': 'Bearer ' + token,
              'Content-Type': 'application/json'
            },
            body: JSON.stringify({
              title: '${title}',
              thread_type: 'user'
            })
          });
          if (!response.ok) {
            const error = await response.text();
            reject('API error: ' + response.status + ' ' + error);
            return;
          }
          const thread = await response.json();
          resolve(thread.id);
        } catch (e) {
          reject(e.message);
        }
      });
    `, parentThreadId, title) as string

    return result
  }

  // Get chat ID from URL
  async getCurrentChatId(): Promise<string> {
    const url = await this.getCurrentUrl()
    const match = url.match(/\/chat\/([a-f0-9-]+)/)
    return match ? match[1] : ''
  }

  // List threads via API
  async listThreadsViaApi(chatId: string): Promise<{ id: string; title: string; depth: number }[]> {
    const result = await this.driver.executeScript(`
      return new Promise(async (resolve, reject) => {
        try {
          const token = localStorage.getItem('access_token');
          if (!token) {
            reject('No access token');
            return;
          }
          const response = await fetch('/api/chats/${chatId}/threads', {
            headers: {
              'Authorization': 'Bearer ' + token
            }
          });
          if (!response.ok) {
            const error = await response.text();
            reject('API error: ' + response.status + ' ' + error);
            return;
          }
          const data = await response.json();
          resolve(data.threads || []);
        } catch (e) {
          reject(e.message);
        }
      });
    `, chatId) as { id: string; title: string; depth: number }[]

    return result || []
  }

  // List subthreads via API
  async listSubthreadsViaApi(parentThreadId: string): Promise<{ id: string; title: string; depth: number }[]> {
    const result = await this.driver.executeScript(`
      return new Promise(async (resolve, reject) => {
        try {
          const token = localStorage.getItem('access_token');
          if (!token) {
            reject('No access token');
            return;
          }
          const response = await fetch('/api/chats/threads/${parentThreadId}/subthreads', {
            headers: {
              'Authorization': 'Bearer ' + token
            }
          });
          if (!response.ok) {
            const error = await response.text();
            reject('API error: ' + response.status + ' ' + error);
            return;
          }
          const data = await response.json();
          resolve(data.threads || []);
        } catch (e) {
          reject(e.message);
        }
      });
    `, parentThreadId) as { id: string; title: string; depth: number }[]

    return result || []
  }

  // Reply methods
  async hoverOverFirstMessage(): Promise<void> {
    const messages = await this.driver.findElements(this.messageItems)
    if (messages.length > 0) {
      const actions = this.driver.actions({ async: true })
      await actions.move({ origin: messages[0] }).perform()
    }
  }

  async hoverOverMessage(index: number): Promise<void> {
    const messages = await this.driver.findElements(this.messageItems)
    if (messages.length > index) {
      const actions = this.driver.actions({ async: true })
      await actions.move({ origin: messages[index] }).perform()
    }
  }

  async isReplyButtonVisible(): Promise<boolean> {
    return this.isDisplayed(this.replyButton)
  }

  async clickReplyButton(): Promise<void> {
    await this.click(this.replyButton)
  }

  async clickReplyOnFirstMessage(): Promise<void> {
    await this.hoverOverFirstMessage()
    await this.sleep(300)
    await this.clickReplyButton()
  }

  async clickReplyOnMessage(index: number): Promise<void> {
    await this.hoverOverMessage(index)
    await this.sleep(300)
    await this.clickReplyButton()
  }

  async isReplyPreviewVisible(): Promise<boolean> {
    return this.isDisplayed(this.replyPreview)
  }

  async getReplyPreviewContent(): Promise<string> {
    try {
      return await this.getText(this.replyPreviewContent)
    } catch {
      // Try fallback to the whole preview area
      try {
        return await this.getText(this.replyPreview)
      } catch {
        return ''
      }
    }
  }

  async getReplyPreviewSenderName(): Promise<string> {
    try {
      return await this.getText(this.replyPreviewSender)
    } catch {
      return ''
    }
  }

  async cancelReply(): Promise<void> {
    await this.click(this.replyPreviewCancel)
  }

  async getReplyPreviewText(): Promise<string> {
    try {
      return await this.getText(this.replyPreview)
    } catch {
      return ''
    }
  }

  async hasMessageWithQuote(): Promise<boolean> {
    return this.isDisplayed(this.messageQuote)
  }

  async getQuoteContent(): Promise<string> {
    try {
      return await this.getText(this.messageQuoteContent)
    } catch {
      // Try fallback to the whole quote area
      try {
        const quotes = await this.driver.findElements(this.messageQuote)
        if (quotes.length > 0) {
          return await quotes[quotes.length - 1].getText()
        }
      } catch {
        // ignore
      }
      return ''
    }
  }

  async getQuoteSenderName(): Promise<string> {
    try {
      return await this.getText(this.messageQuoteSender)
    } catch {
      return ''
    }
  }

  async waitForQuote(timeout: number = 10000): Promise<void> {
    const start = Date.now()
    while (Date.now() - start < timeout) {
      if (await this.hasMessageWithQuote()) {
        return
      }
      await this.sleep(300)
    }
    throw new Error('Quote not found within timeout')
  }

  // Forward message methods
  private readonly forwardButton = By.css('[data-testid="forward-button"]')
  private readonly forwardModal = By.css('[data-testid="forward-modal"]')
  private readonly forwardSearchInput = By.css('[data-testid="forward-search-input"]')
  private readonly forwardChatItems = By.css('[data-testid="forward-chat-item"]')
  private readonly forwardCommentInput = By.css('[data-testid="forward-comment-input"]')
  private readonly forwardSubmitButton = By.css('[data-testid="forward-submit-button"]')
  private readonly messageForwarded = By.css('[data-testid="message-forwarded"]')

  async isForwardButtonVisible(): Promise<boolean> {
    return this.isDisplayed(this.forwardButton)
  }

  async clickForwardButton(): Promise<void> {
    await this.click(this.forwardButton)
  }

  async clickForwardOnFirstMessage(): Promise<void> {
    await this.hoverOverFirstMessage()
    await this.sleep(300)
    await this.clickForwardButton()
  }

  async clickForwardOnMessage(index: number): Promise<void> {
    await this.hoverOverMessage(index)
    await this.sleep(300)
    await this.clickForwardButton()
  }

  async isForwardModalVisible(): Promise<boolean> {
    return this.isDisplayed(this.forwardModal)
  }

  async waitForForwardModal(timeout: number = 10000): Promise<void> {
    const start = Date.now()
    while (Date.now() - start < timeout) {
      if (await this.isForwardModalVisible()) {
        return
      }
      await this.sleep(300)
    }
    throw new Error('Forward modal did not appear within timeout')
  }

  async searchChatInForwardModal(query: string): Promise<void> {
    await this.type(this.forwardSearchInput, query)
    await this.sleep(300)
  }

  async getForwardChatItemsCount(): Promise<number> {
    const elements = await this.driver.findElements(this.forwardChatItems)
    return elements.length
  }

  async getForwardChatNames(): Promise<string[]> {
    const elements = await this.driver.findElements(this.forwardChatItems)
    const names: string[] = []
    for (const el of elements) {
      const text = await el.getText()
      names.push(text.trim())
    }
    return names
  }

  async selectForwardChatByIndex(index: number): Promise<void> {
    const elements = await this.driver.findElements(this.forwardChatItems)
    if (elements.length > index) {
      await elements[index].click()
      await this.sleep(200)
    } else {
      throw new Error(`Chat at index ${index} not found`)
    }
  }

  async selectForwardChatByName(name: string): Promise<void> {
    const elements = await this.driver.findElements(this.forwardChatItems)
    for (const el of elements) {
      const text = await el.getText()
      if (text.includes(name)) {
        await el.click()
        await this.sleep(200)
        return
      }
    }
    throw new Error(`Chat "${name}" not found in forward modal`)
  }

  async enterForwardComment(comment: string): Promise<void> {
    await this.type(this.forwardCommentInput, comment)
  }

  async submitForward(): Promise<void> {
    await this.click(this.forwardSubmitButton)
    await this.sleep(500)
  }

  async closeForwardModal(): Promise<void> {
    // Click outside modal to close
    await this.driver.executeScript(`
      const modal = document.querySelector('[data-testid="forward-modal"]');
      if (modal && modal.parentElement) {
        modal.parentElement.click();
      }
    `)
    await this.sleep(300)
  }

  async forwardMessageToChat(messageIndex: number, targetChatName: string, comment?: string): Promise<void> {
    await this.clickForwardOnMessage(messageIndex)
    await this.waitForForwardModal()
    await this.selectForwardChatByName(targetChatName)
    if (comment) {
      await this.enterForwardComment(comment)
    }
    await this.submitForward()
  }

  async hasForwardedMessage(): Promise<boolean> {
    return this.isDisplayed(this.messageForwarded)
  }

  async getForwardedIndicatorText(): Promise<string> {
    try {
      const element = await this.driver.findElement(this.messageForwarded)
      return await element.getText()
    } catch {
      return ''
    }
  }

  async waitForForwardedMessage(timeout: number = 10000): Promise<void> {
    const start = Date.now()
    while (Date.now() - start < timeout) {
      if (await this.hasForwardedMessage()) {
        return
      }
      await this.sleep(300)
    }
    throw new Error('Forwarded message indicator not found within timeout')
  }

  // Quote time display (improved reply)
  async getQuoteTime(): Promise<string> {
    try {
      const quoteEl = await this.driver.findElement(this.messageQuote)
      const text = await quoteEl.getText()
      // Extract time pattern (HH:MM)
      const match = text.match(/\d{1,2}:\d{2}/)
      return match ? match[0] : ''
    } catch {
      return ''
    }
  }

  async hasQuoteAttachmentInfo(): Promise<boolean> {
    try {
      const quoteEl = await this.driver.findElement(this.messageQuote)
      const text = await quoteEl.getText()
      return text.includes('📷') || text.includes('📎')
    } catch {
      return false
    }
  }

  // ========== Left Navigation Panel ==========
  private readonly leftNavPanel = By.css('nav.left-nav-panel')
  private readonly leftNavChatsButton = By.css('nav.left-nav-panel button[title="Chats"]')
  private readonly leftNavQuickCallButton = By.css('nav.left-nav-panel button[title="Quick Call"]')
  private readonly leftNavEventsButton = By.css('nav.left-nav-panel button[title="Events"]')
  private readonly leftNavEventsBadge = By.css('nav.left-nav-panel button[title="Events"] .badge')

  async isLeftNavPanelVisible(): Promise<boolean> {
    return this.isDisplayed(this.leftNavPanel)
  }

  async clickLeftNavChats(): Promise<void> {
    await this.dismissNetworkStatusBar()
    await this.click(this.leftNavChatsButton)
    await this.sleep(300)
  }

  /** Dismiss network status bar if visible (it can block clicks) */
  async dismissNetworkStatusBar(): Promise<void> {
    try {
      const statusBar = await this.driver.findElements(By.css('[data-testid="network-status-bar"]'))
      if (statusBar.length > 0 && await statusBar[0].isDisplayed()) {
        // Wait a moment for network to stabilize
        await this.sleep(500)
        // If still visible, use JavaScript to hide it
        const stillVisible = await statusBar[0].isDisplayed().catch(() => false)
        if (stillVisible) {
          await this.executeScript(`
            const bar = document.querySelector('[data-testid="network-status-bar"]');
            if (bar) bar.style.display = 'none';
          `)
        }
      }
    } catch {
      // Ignore - status bar might not exist
    }
  }

  async clickLeftNavQuickCall(): Promise<void> {
    await this.dismissNetworkStatusBar()
    await this.click(this.leftNavQuickCallButton)
    await this.sleep(500)
  }

  async clickLeftNavEvents(): Promise<void> {
    await this.dismissNetworkStatusBar()
    await this.click(this.leftNavEventsButton)
    await this.sleep(300)
  }

  async isLeftNavChatsActive(): Promise<boolean> {
    try {
      const element = await this.driver.findElement(this.leftNavChatsButton)
      const classes = await element.getAttribute('class')
      return classes.includes('active')
    } catch {
      return false
    }
  }

  async isLeftNavEventsActive(): Promise<boolean> {
    try {
      const element = await this.driver.findElement(this.leftNavEventsButton)
      const classes = await element.getAttribute('class')
      return classes.includes('active')
    } catch {
      return false
    }
  }

  async hasEventsBadge(): Promise<boolean> {
    return this.isDisplayed(this.leftNavEventsBadge)
  }

  async getEventsBadgeText(): Promise<string> {
    try {
      return await this.getText(this.leftNavEventsBadge)
    } catch {
      return ''
    }
  }

  async isQuickCallButtonLoading(): Promise<boolean> {
    try {
      const element = await this.driver.findElement(this.leftNavQuickCallButton)
      const classes = await element.getAttribute('class')
      return classes.includes('loading')
    } catch {
      return false
    }
  }

  // ========== AdHoc Call Button (in chat room) ==========
  private readonly adHocCallButton = By.css('.adhoc-call-button .main-btn')
  private readonly adHocCallDropdown = By.css('.adhoc-call-button .dropdown')
  private readonly adHocCallAllOption = By.xpath('//button[contains(text(), "Call All")]')
  // Select Participants is clicked via JS in clickSelectParticipants()
  private readonly adHocParticipantSelector = By.css('.adhoc-call-button .participant-selector')
  private readonly adHocParticipantItems = By.css('.adhoc-call-button .participant-item')
  private readonly adHocStartCallButton = By.css('.adhoc-call-button .start-call-btn')
  private readonly adHocBackButton = By.css('.adhoc-call-button .back-btn')

  async isAdHocCallButtonVisible(): Promise<boolean> {
    return this.isDisplayed(this.adHocCallButton)
  }

  async clickAdHocCallButton(): Promise<void> {
    await this.click(this.adHocCallButton)
    await this.sleep(300)
  }

  async isAdHocDropdownVisible(): Promise<boolean> {
    return this.isDisplayed(this.adHocCallDropdown)
  }

  async waitForAdHocDropdown(timeout: number = 5000): Promise<void> {
    const start = Date.now()
    while (Date.now() - start < timeout) {
      if (await this.isAdHocDropdownVisible()) {
        return
      }
      await this.sleep(200)
    }
    throw new Error('AdHoc dropdown did not appear within timeout')
  }

  async clickCallAll(): Promise<void> {
    await this.click(this.adHocCallAllOption)
    await this.sleep(500)
  }

  async clickSelectParticipants(): Promise<void> {
    // Use JavaScript click to ensure the Vue handler is triggered
    await this.executeScript(`
      const items = document.querySelectorAll('.adhoc-call-button .dropdown-options .dropdown-item');
      if (items.length >= 2) {
        items[1].click();
      }
    `)
    await this.sleep(500)
  }

  async isParticipantSelectorVisible(): Promise<boolean> {
    return this.isDisplayed(this.adHocParticipantSelector)
  }

  async getAdHocParticipantCount(): Promise<number> {
    const elements = await this.driver.findElements(this.adHocParticipantItems)
    return elements.length
  }

  async getAdHocParticipantNames(): Promise<string[]> {
    const elements = await this.driver.findElements(this.adHocParticipantItems)
    const names: string[] = []
    for (const el of elements) {
      try {
        const nameEl = await el.findElement(By.css('.participant-name'))
        const text = await nameEl.getText()
        names.push(text.trim())
      } catch {
        const text = await el.getText()
        names.push(text.trim())
      }
    }
    return names
  }

  async selectAdHocParticipantByIndex(index: number): Promise<void> {
    const elements = await this.driver.findElements(this.adHocParticipantItems)
    if (elements.length > index) {
      await elements[index].click()
      await this.sleep(200)
    } else {
      throw new Error(`Participant at index ${index} not found`)
    }
  }

  async selectAdHocParticipantByName(name: string): Promise<void> {
    const elements = await this.driver.findElements(this.adHocParticipantItems)
    for (const el of elements) {
      const text = await el.getText()
      if (text.includes(name)) {
        await el.click()
        await this.sleep(200)
        return
      }
    }
    throw new Error(`Participant "${name}" not found in selector`)
  }

  async getSelectedParticipantsCount(): Promise<number> {
    const elements = await this.driver.findElements(this.adHocParticipantItems)
    let count = 0
    for (const el of elements) {
      const classes = await el.getAttribute('class')
      if (classes.includes('selected')) {
        count++
      }
    }
    return count
  }

  async clickStartCallWithSelected(): Promise<void> {
    await this.click(this.adHocStartCallButton)
    await this.sleep(500)
  }

  async getStartCallButtonText(): Promise<string> {
    try {
      return await this.getText(this.adHocStartCallButton)
    } catch {
      return ''
    }
  }

  async isStartCallButtonDisabled(): Promise<boolean> {
    try {
      const element = await this.driver.findElement(this.adHocStartCallButton)
      const disabled = await element.getAttribute('disabled')
      return disabled === 'true' || disabled === ''
    } catch {
      return true
    }
  }

  async clickBackInParticipantSelector(): Promise<void> {
    await this.click(this.adHocBackButton)
    await this.sleep(200)
  }

  async isAdHocButtonInCallState(): Promise<boolean> {
    try {
      const element = await this.driver.findElement(this.adHocCallButton)
      const classes = await element.getAttribute('class')
      return classes.includes('active')
    } catch {
      return false
    }
  }

  // Aliases and convenience methods
  async getChatList(): Promise<string[]> {
    return this.getChatNames()
  }

  async selectChatByIndex(index: number): Promise<void> {
    const elements = await this.driver.findElements(this.chatListItem)
    if (elements.length > index) {
      await elements[index].click()
      await this.sleep(500)
    } else {
      throw new Error(`Chat at index ${index} not found`)
    }
  }

  async selectChatByName(name: string): Promise<void> {
    await this.clickChatByNameInList(name)
    await this.sleep(500)
  }

  async waitForChatList(timeout: number = 10000): Promise<void> {
    await this.waitForElement(this.chatSidebar, timeout)
    // Wait for at least one chat or empty state
    const start = Date.now()
    while (Date.now() - start < timeout) {
      const count = await this.getChatCount()
      if (count > 0) {
        return
      }
      await this.sleep(300)
    }
    // No chats found, but sidebar loaded - that's ok
  }

  async waitForMessagesArea(timeout: number = 10000): Promise<void> {
    await this.waitForChatRoom(timeout)
  }

  async getMessages(): Promise<string[]> {
    return this.getMessageTexts()
  }

  async hasReplyQuote(): Promise<boolean> {
    return this.hasMessageWithQuote()
  }

  // ========== Participant Management ==========

  /**
   * Add a participant to the current chat via API
   * @param userId - UUID of the user to add
   */
  async addParticipantToChat(userId: string): Promise<void> {
    const chatId = await this.getCurrentChatId()
    if (!chatId) {
      throw new Error('Cannot add participant - not in a chat room')
    }

    const result = await this.driver.executeScript(`
      return new Promise(async (resolve, reject) => {
        try {
          const token = localStorage.getItem('access_token');
          if (!token) {
            reject('No access token');
            return;
          }
          const response = await fetch('/api/chats/${chatId}/participants', {
            method: 'POST',
            headers: {
              'Authorization': 'Bearer ' + token,
              'Content-Type': 'application/json'
            },
            body: JSON.stringify({
              user_id: '${userId}',
              role: 'member'
            })
          });
          if (!response.ok) {
            const error = await response.text();
            reject('API error: ' + response.status + ' ' + error);
            return;
          }
          resolve({ success: true });
        } catch (e) {
          reject(e.message);
        }
      });
    `)

    if (!result) {
      throw new Error('Failed to add participant')
    }

    // Wait for the change to propagate
    await this.sleep(1000)
  }

  /**
   * Remove a participant from the current chat via API
   * @param userId - UUID of the user to remove
   */
  async removeParticipantFromChat(userId: string): Promise<void> {
    const chatId = await this.getCurrentChatId()
    if (!chatId) {
      throw new Error('Cannot remove participant - not in a chat room')
    }

    const result = await this.driver.executeScript(`
      return new Promise(async (resolve, reject) => {
        try {
          const token = localStorage.getItem('access_token');
          if (!token) {
            reject('No access token');
            return;
          }
          const response = await fetch('/api/chats/${chatId}/participants/${userId}', {
            method: 'DELETE',
            headers: {
              'Authorization': 'Bearer ' + token
            }
          });
          if (!response.ok) {
            const error = await response.text();
            reject('API error: ' + response.status + ' ' + error);
            return;
          }
          resolve({ success: true });
        } catch (e) {
          reject(e.message);
        }
      });
    `)

    if (!result) {
      throw new Error('Failed to remove participant')
    }

    // Wait for the change to propagate
    await this.sleep(1000)
  }
}
