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
    // Match both /chat/ and /chats/ patterns
    const match = url.match(/\/chats?\/([a-f0-9-]+)/)
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

  // ========== Incoming Call Overlay ==========
  private readonly incomingCallOverlay = By.css('.incoming-call-overlay')
  private readonly incomingCallAnswerBtn = By.css('.incoming-call-overlay .action-btn.answer')
  private readonly incomingCallRejectBtn = By.css('.incoming-call-overlay .action-btn.reject')
  private readonly incomingCallerName = By.css('.incoming-call-overlay .caller-name')

  /**
   * Check if incoming call overlay is visible
   */
  async isIncomingCallVisible(): Promise<boolean> {
    return this.isDisplayed(this.incomingCallOverlay)
  }

  /**
   * Wait for incoming call overlay to appear
   */
  async waitForIncomingCall(timeout: number = 10000): Promise<void> {
    await this.waitForElement(this.incomingCallOverlay, timeout)
  }

  /**
   * Get the caller name from incoming call overlay
   */
  async getIncomingCallerName(): Promise<string> {
    const element = await this.waitForElement(this.incomingCallerName)
    return element.getText()
  }

  /**
   * Answer the incoming call
   */
  async answerIncomingCall(): Promise<void> {
    await this.click(this.incomingCallAnswerBtn)
    await this.sleep(500)
  }

  /**
   * Reject the incoming call
   */
  async rejectIncomingCall(): Promise<void> {
    await this.click(this.incomingCallRejectBtn)
    await this.sleep(500)
  }

  // ========== Active Conference Indicators ==========
  private readonly callIndicatorEmoji = By.css('.call-indicator')
  private readonly joinCallButton = By.css('.adhoc-call-button .join-btn')
  private readonly hangupButton = By.css('.adhoc-call-button .hangup-btn')

  /**
   * Check if any chat in the sidebar has the active call indicator (📞)
   */
  async hasActiveCallIndicator(): Promise<boolean> {
    return this.isDisplayed(this.callIndicatorEmoji)
  }

  /**
   * Get list of chat names that have active call indicator
   */
  async getChatsWithActiveCall(): Promise<string[]> {
    const chatItems = await this.driver.findElements(this.chatListItem)
    const chatsWithCall: string[] = []

    for (const item of chatItems) {
      try {
        const indicators = await item.findElements(this.callIndicatorEmoji)
        if (indicators.length > 0) {
          const nameSpan = await item.findElement(By.css('span.font-medium'))
          const text = await nameSpan.getText()
          chatsWithCall.push(text.trim())
        }
      } catch {
        continue
      }
    }
    return chatsWithCall
  }

  /**
   * Check if the Join button is visible (shown when there's an active conference)
   */
  async isJoinCallButtonVisible(): Promise<boolean> {
    return this.isDisplayed(this.joinCallButton)
  }

  /**
   * Click the Join button to join an existing conference
   */
  async clickJoinCall(): Promise<void> {
    await this.click(this.joinCallButton)
    await this.sleep(500)
  }

  /**
   * Get text from Join button (e.g., "Join (2)")
   */
  async getJoinButtonText(): Promise<string> {
    try {
      return await this.getText(this.joinCallButton)
    } catch {
      return ''
    }
  }

  /**
   * Check if the Hangup button is visible
   */
  async isHangupButtonVisible(): Promise<boolean> {
    return this.isDisplayed(this.hangupButton)
  }

  /**
   * Click the Hangup button to leave conference
   */
  async clickHangup(): Promise<void> {
    await this.click(this.hangupButton)
    await this.sleep(500)
  }

  /**
   * Wait for the call indicator to appear on a specific chat
   */
  async waitForCallIndicator(chatName: string, timeout: number = 10000): Promise<void> {
    const start = Date.now()
    while (Date.now() - start < timeout) {
      const chatsWithCall = await this.getChatsWithActiveCall()
      if (chatsWithCall.some(name => name.includes(chatName))) {
        return
      }
      await this.sleep(300)
    }
    throw new Error(`Call indicator for chat "${chatName}" not found within timeout`)
  }

  /**
   * Wait for the call indicator to disappear from a specific chat
   */
  async waitForCallIndicatorToDisappear(chatName: string, timeout: number = 10000): Promise<void> {
    const start = Date.now()
    while (Date.now() - start < timeout) {
      const chatsWithCall = await this.getChatsWithActiveCall()
      if (!chatsWithCall.some(name => name.includes(chatName))) {
        return
      }
      await this.sleep(300)
    }
    throw new Error(`Call indicator for chat "${chatName}" still visible after timeout`)
  }

  /**
   * Wait for Join button to appear
   */
  async waitForJoinButton(timeout: number = 10000): Promise<void> {
    const start = Date.now()
    while (Date.now() - start < timeout) {
      if (await this.isJoinCallButtonVisible()) {
        return
      }
      await this.sleep(300)
    }
    throw new Error('Join button did not appear within timeout')
  }

  /**
   * Wait for Join button to disappear (after conference ends)
   */
  async waitForJoinButtonToDisappear(timeout: number = 10000): Promise<void> {
    const start = Date.now()
    while (Date.now() - start < timeout) {
      if (!(await this.isJoinCallButtonVisible())) {
        return
      }
      await this.sleep(300)
    }
    throw new Error('Join button still visible after timeout')
  }

  /**
   * Get active conferences from API
   */
  async getActiveConferencesViaApi(): Promise<{ id: string; chat_id: string; name: string; participant_count: number }[]> {
    const result = await this.driver.executeScript(`
      return new Promise(async (resolve, reject) => {
        try {
          const token = localStorage.getItem('access_token');
          if (!token) {
            reject('No access token');
            return;
          }
          const response = await fetch('/api/voice/conferences/active', {
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
          resolve(data.conferences || []);
        } catch (e) {
          reject(e.message);
        }
      });
    `) as { id: string; chat_id: string; name: string; participant_count: number }[]

    return result || []
  }

  /**
   * End a conference via API
   */
  async endConferenceViaApi(conferenceId: string): Promise<void> {
    await this.driver.executeScript(`
      return new Promise(async (resolve, reject) => {
        try {
          const token = localStorage.getItem('access_token');
          if (!token) {
            reject('No access token');
            return;
          }
          const response = await fetch('/api/voice/conferences/${conferenceId}/end', {
            method: 'POST',
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
  }

  /**
   * Leave a conference via API
   */
  async leaveConferenceViaApi(conferenceId: string): Promise<void> {
    await this.driver.executeScript(`
      return new Promise(async (resolve, reject) => {
        try {
          const token = localStorage.getItem('access_token');
          if (!token) {
            reject('No access token');
            return;
          }
          const response = await fetch('/api/voice/conferences/${conferenceId}/leave', {
            method: 'POST',
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

  // ========== Conference View Popup ==========

  private readonly conferenceView = By.css('.conference-view')
  private readonly conferenceName = By.css('.conference-name')
  private readonly participantCount = By.css('.participant-count')
  private readonly conferenceCloseBtn = By.css('.close-btn')

  /**
   * Check if Conference View popup is visible
   */
  async isConferenceViewVisible(): Promise<boolean> {
    return this.isDisplayed(this.conferenceView)
  }

  /**
   * Wait for Conference View popup to appear
   */
  async waitForConferenceView(timeout: number = 10000): Promise<void> {
    const start = Date.now()
    while (Date.now() - start < timeout) {
      if (await this.isConferenceViewVisible()) {
        return
      }
      await this.sleep(100)
    }
    throw new Error('Conference View did not appear within timeout')
  }

  /**
   * Get conference name from the popup
   */
  async getConferenceName(): Promise<string> {
    try {
      return await this.getText(this.conferenceName)
    } catch {
      return ''
    }
  }

  /**
   * Get participant count text (e.g., "2 participants")
   */
  async getConferenceParticipantCount(): Promise<string> {
    try {
      return await this.getText(this.participantCount)
    } catch {
      return ''
    }
  }

  /**
   * Close conference view popup by clicking close button
   */
  async closeConferenceView(): Promise<void> {
    await this.click(this.conferenceCloseBtn)
    await this.sleep(500)
  }

  /**
   * Wait for Conference View to disappear
   */
  async waitForConferenceViewToDisappear(timeout: number = 10000): Promise<void> {
    const start = Date.now()
    while (Date.now() - start < timeout) {
      if (!(await this.isConferenceViewVisible())) {
        return
      }
      await this.sleep(100)
    }
    throw new Error('Conference View did not disappear within timeout')
  }

  // ========== Conference Controls (Mute, etc.) ==========

  private readonly muteButton = By.css('.call-controls .control-btn[title*="Mute"], .call-controls .control-btn[title*="Unmute"]')

  /**
   * Check if user is currently muted (mute button has 'active' class)
   */
  async isMutedInConference(): Promise<boolean> {
    try {
      const button = await this.driver.findElement(this.muteButton)
      const classes = await button.getAttribute('class')
      return classes.includes('active')
    } catch {
      return false
    }
  }

  /**
   * Toggle mute button in conference
   */
  async toggleMuteInConference(): Promise<void> {
    await this.click(this.muteButton)
    await this.sleep(500)
  }

  /**
   * Mute microphone if not already muted
   */
  async muteInConference(): Promise<void> {
    const isMuted = await this.isMutedInConference()
    if (!isMuted) {
      await this.toggleMuteInConference()
    }
  }

  /**
   * Unmute microphone if currently muted
   */
  async unmuteInConference(): Promise<void> {
    const isMuted = await this.isMutedInConference()
    if (isMuted) {
      await this.toggleMuteInConference()
    }
  }

  /**
   * Get participant count from conference view
   */
  async getParticipantCountNumber(): Promise<number> {
    const text = await this.getConferenceParticipantCount()
    // Extract number from "2 participants" or "1 participant"
    const match = text.match(/(\d+)/)
    return match ? parseInt(match[1], 10) : 0
  }

  /**
   * Wait for participant count to reach expected value
   */
  async waitForParticipantCount(expectedCount: number, timeout: number = 15000): Promise<void> {
    const start = Date.now()
    while (Date.now() - start < timeout) {
      const count = await this.getParticipantCountNumber()
      if (count === expectedCount) {
        return
      }
      await this.sleep(500)
    }
    throw new Error(`Participant count did not reach ${expectedCount} within timeout`)
  }

  // ========== Active Events Section in Sidebar ==========

  private readonly activeEventsSection = By.xpath('//h3[contains(text(), "Активные мероприятия")]')
  private readonly activeEventsChatItems = By.xpath('//h3[contains(text(), "Активные мероприятия")]/ancestor::div[contains(@class, "border-b")]//div[contains(@class, "hover:bg-green-50")]')
  private readonly activeEventsJoinButton = By.xpath('//button[contains(text(), "Присоединиться")]')
  private readonly activeEventsParticipatingBadge = By.xpath('//span[contains(text(), "Вы участвуете")]')
  private readonly activeEventsInCallStatus = By.xpath('//span[contains(text(), "В звонке")]')
  private readonly activeEventsOnHoldStatus = By.xpath('//span[contains(text(), "На hold")]')
  private readonly activeEventsOngoingText = By.xpath('//span[contains(text(), "Идёт")]')
  private readonly regularChatsSection = By.xpath('//h3[contains(text(), "Чаты")]')

  /**
   * Check if "Активные мероприятия" section is visible in sidebar
   */
  async isActiveEventsSectionVisible(): Promise<boolean> {
    return this.isDisplayed(this.activeEventsSection)
  }

  /**
   * Wait for "Активные мероприятия" section to appear
   */
  async waitForActiveEventsSection(timeout: number = 15000): Promise<void> {
    const start = Date.now()
    while (Date.now() - start < timeout) {
      if (await this.isActiveEventsSectionVisible()) {
        return
      }
      await this.sleep(300)
    }
    throw new Error('Active Events section did not appear within timeout')
  }

  /**
   * Wait for "Активные мероприятия" section to disappear
   */
  async waitForActiveEventsSectionToDisappear(timeout: number = 15000): Promise<void> {
    const start = Date.now()
    while (Date.now() - start < timeout) {
      if (!(await this.isActiveEventsSectionVisible())) {
        return
      }
      await this.sleep(300)
    }
    throw new Error('Active Events section still visible after timeout')
  }

  /**
   * Get count of chats in "Активные мероприятия" section
   */
  async getActiveEventsChatCount(): Promise<number> {
    try {
      const elements = await this.driver.findElements(this.activeEventsChatItems)
      return elements.length
    } catch {
      return 0
    }
  }

  /**
   * Get names of chats in "Активные мероприятия" section
   */
  async getActiveEventsChatNames(): Promise<string[]> {
    const names: string[] = []
    try {
      const elements = await this.driver.findElements(this.activeEventsChatItems)
      for (const el of elements) {
        try {
          const nameSpan = await el.findElement(By.css('span.font-medium'))
          const text = await nameSpan.getText()
          names.push(text.trim())
        } catch {
          continue
        }
      }
    } catch {
      // Section might not exist
    }
    return names
  }

  /**
   * Check if a specific chat is in "Активные мероприятия" section
   */
  async isChatInActiveEventsSection(chatName: string): Promise<boolean> {
    const names = await this.getActiveEventsChatNames()
    return names.some(name => name.includes(chatName))
  }

  /**
   * Check if "Присоединиться" button is visible in active events section
   */
  async isJoinActiveEventButtonVisible(): Promise<boolean> {
    return this.isDisplayed(this.activeEventsJoinButton)
  }

  /**
   * Click "Присоединиться" button in active events section
   */
  async clickJoinActiveEvent(): Promise<void> {
    await this.click(this.activeEventsJoinButton)
    await this.sleep(500)
  }

  /**
   * Check if "Вы участвуете" badge is visible
   */
  async isParticipatingBadgeVisible(): Promise<boolean> {
    return this.isDisplayed(this.activeEventsParticipatingBadge)
  }

  /**
   * Check if "В звонке" status is visible (for direct chats)
   */
  async isInCallStatusVisible(): Promise<boolean> {
    return this.isDisplayed(this.activeEventsInCallStatus)
  }

  /**
   * Check if "На hold" status is visible (for direct chats when muted)
   */
  async isOnHoldStatusVisible(): Promise<boolean> {
    return this.isDisplayed(this.activeEventsOnHoldStatus)
  }

  /**
   * Check if "Идёт звонок/мероприятие" text is visible
   */
  async isOngoingEventTextVisible(): Promise<boolean> {
    return this.isDisplayed(this.activeEventsOngoingText)
  }

  /**
   * Check if "Чаты" section header is visible (appears when active events are shown)
   */
  async isRegularChatsSectionVisible(): Promise<boolean> {
    return this.isDisplayed(this.regularChatsSection)
  }

  /**
   * Click on a chat in "Активные мероприятия" section by name
   */
  async clickActiveEventChatByName(chatName: string): Promise<void> {
    const elements = await this.driver.findElements(this.activeEventsChatItems)
    for (const el of elements) {
      try {
        const nameSpan = await el.findElement(By.css('span.font-medium'))
        const text = await nameSpan.getText()
        if (text.includes(chatName)) {
          await el.click()
          await this.sleep(500)
          return
        }
      } catch {
        continue
      }
    }
    throw new Error(`Chat "${chatName}" not found in active events section`)
  }

  /**
   * Get participant count displayed for a chat in active events section
   */
  async getActiveEventParticipantCount(chatName: string): Promise<number> {
    const elements = await this.driver.findElements(this.activeEventsChatItems)
    for (const el of elements) {
      try {
        const nameSpan = await el.findElement(By.css('span.font-medium'))
        const name = await nameSpan.getText()
        if (name.includes(chatName)) {
          // Find participant count span (e.g., "2 уч.")
          const countSpan = await el.findElement(By.css('.text-green-600.font-medium'))
          const countText = await countSpan.getText()
          const match = countText.match(/(\d+)/)
          return match ? parseInt(match[1], 10) : 0
        }
      } catch {
        continue
      }
    }
    return 0
  }

  /**
   * Wait for a chat to appear in active events section
   */
  async waitForChatInActiveEvents(chatName: string, timeout: number = 15000): Promise<void> {
    const start = Date.now()
    while (Date.now() - start < timeout) {
      if (await this.isChatInActiveEventsSection(chatName)) {
        return
      }
      await this.sleep(300)
    }
    throw new Error(`Chat "${chatName}" did not appear in active events section within timeout`)
  }

  /**
   * Wait for a chat to disappear from active events section
   */
  async waitForChatToLeaveActiveEvents(chatName: string, timeout: number = 15000): Promise<void> {
    const start = Date.now()
    while (Date.now() - start < timeout) {
      if (!(await this.isChatInActiveEventsSection(chatName))) {
        return
      }
      await this.sleep(300)
    }
    throw new Error(`Chat "${chatName}" still in active events section after timeout`)
  }

  /**
   * Refresh active conferences in voice store
   */
  async refreshActiveConferences(): Promise<void> {
    await this.driver.executeScript(`
      if (window.__voiceStore?.loadActiveConferences) {
        return window.__voiceStore.loadActiveConferences()
      }
    `)
    await this.sleep(500)
  }

  // ========== Event History Panel ==========

  private readonly historyButton = By.css('[data-testid="history-button"]')
  private readonly historyPanelTitle = By.xpath('//h4[contains(text(), "History")]')
  private readonly historyEventsTab = By.xpath('//button[.//span[contains(text(), "Events")]]')
  private readonly historyFilesTab = By.xpath('//button[.//span[contains(text(), "Files")]]')
  private readonly historyPanelClose = By.css('.w-80.border-l button[title="Close"]')
  private readonly historyEventItems = By.css('.w-80.border-l ul li')
  private readonly historyFileItems = By.css('.w-80.border-l ul li')
  private readonly historyLoading = By.css('.w-80.border-l .animate-spin')
  private readonly historyEmptyEvents = By.xpath('//*[contains(text(), "No events yet")]')
  private readonly historyEmptyFiles = By.xpath('//*[contains(text(), "No files in this chat")]')

  // Event detail view
  private readonly historyDetailView = By.xpath('//button[@title="Back to list"]')
  private readonly historyBackButton = By.xpath('//button[@title="Back to list"]')
  private readonly historyParticipantsTab = By.xpath('//button[contains(text(), "Participants")]')
  private readonly historyMessagesTab = By.xpath('//button[contains(text(), "Messages")]')
  private readonly historyActionsTab = By.xpath('//button[contains(text(), "Actions")]')
  private readonly historyParticipantItems = By.css('.w-80.border-l .space-y-4 > div')
  private readonly historyActionItems = By.css('.w-80.border-l .border-l-2.border-orange-400')

  /**
   * Click History button to open history panel
   */
  async clickHistoryButton(): Promise<void> {
    await this.click(this.historyButton)
    await this.sleep(500)
  }

  /**
   * Check if History button is visible
   */
  async isHistoryButtonVisible(): Promise<boolean> {
    return this.isDisplayed(this.historyButton)
  }

  /**
   * Check if History panel is visible
   */
  async isHistoryPanelVisible(): Promise<boolean> {
    return this.isDisplayed(this.historyPanelTitle)
  }

  /**
   * Wait for History panel to appear
   */
  async waitForHistoryPanel(timeout: number = 10000): Promise<void> {
    const start = Date.now()
    while (Date.now() - start < timeout) {
      if (await this.isHistoryPanelVisible()) {
        return
      }
      await this.sleep(300)
    }
    throw new Error('History panel did not appear within timeout')
  }

  /**
   * Close History panel
   */
  async closeHistoryPanel(): Promise<void> {
    await this.click(this.historyPanelClose)
    await this.sleep(300)
  }

  /**
   * Click Events tab in history panel
   */
  async clickHistoryEventsTab(): Promise<void> {
    await this.click(this.historyEventsTab)
    await this.sleep(500)
  }

  /**
   * Click Files tab in history panel
   */
  async clickHistoryFilesTab(): Promise<void> {
    await this.click(this.historyFilesTab)
    await this.sleep(500)
  }

  /**
   * Check if Events tab is active
   */
  async isHistoryEventsTabActive(): Promise<boolean> {
    try {
      const element = await this.driver.findElement(this.historyEventsTab)
      const classes = await element.getAttribute('class')
      return classes.includes('text-indigo-600')
    } catch {
      return false
    }
  }

  /**
   * Check if Files tab is active
   */
  async isHistoryFilesTabActive(): Promise<boolean> {
    try {
      const element = await this.driver.findElement(this.historyFilesTab)
      const classes = await element.getAttribute('class')
      return classes.includes('text-indigo-600')
    } catch {
      return false
    }
  }

  /**
   * Wait for history loading to complete
   */
  async waitForHistoryLoaded(timeout: number = 10000): Promise<void> {
    const start = Date.now()
    // First wait for loading to start or content to appear
    await this.sleep(500)
    // Then wait for loading to finish
    while (Date.now() - start < timeout) {
      const loading = await this.isDisplayed(this.historyLoading)
      if (!loading) {
        return
      }
      await this.sleep(300)
    }
    throw new Error('History loading did not complete within timeout')
  }

  /**
   * Get count of events in history list
   */
  async getHistoryEventCount(): Promise<number> {
    try {
      const elements = await this.driver.findElements(this.historyEventItems)
      return elements.length
    } catch {
      return 0
    }
  }

  /**
   * Get names of events in history list
   */
  async getHistoryEventNames(): Promise<string[]> {
    const names: string[] = []
    try {
      const elements = await this.driver.findElements(this.historyEventItems)
      for (const el of elements) {
        try {
          const nameEl = await el.findElement(By.css('.font-medium'))
          const text = await nameEl.getText()
          names.push(text.trim())
        } catch {
          continue
        }
      }
    } catch {
      // Panel might not exist
    }
    return names
  }

  /**
   * Click on an event in history list by name
   */
  async clickHistoryEventByName(eventName: string): Promise<void> {
    const elements = await this.driver.findElements(this.historyEventItems)
    for (const el of elements) {
      try {
        const text = await el.getText()
        if (text.includes(eventName)) {
          await el.click()
          await this.sleep(500)
          return
        }
      } catch {
        continue
      }
    }
    throw new Error(`Event "${eventName}" not found in history`)
  }

  /**
   * Click on first event in history list
   */
  async clickFirstHistoryEvent(): Promise<void> {
    const elements = await this.driver.findElements(this.historyEventItems)
    if (elements.length > 0) {
      await elements[0].click()
      await this.sleep(500)
    } else {
      throw new Error('No events found in history')
    }
  }

  /**
   * Check if empty events message is visible
   */
  async isHistoryEmptyEventsVisible(): Promise<boolean> {
    return this.isDisplayed(this.historyEmptyEvents)
  }

  /**
   * Check if empty files message is visible
   */
  async isHistoryEmptyFilesVisible(): Promise<boolean> {
    return this.isDisplayed(this.historyEmptyFiles)
  }

  /**
   * Get count of files in history files tab
   */
  async getHistoryFileCount(): Promise<number> {
    try {
      const elements = await this.driver.findElements(this.historyFileItems)
      return elements.length
    } catch {
      return 0
    }
  }

  /**
   * Check if event detail view is showing (back button visible)
   */
  async isHistoryDetailViewVisible(): Promise<boolean> {
    return this.isDisplayed(this.historyDetailView)
  }

  /**
   * Click back button in history detail view
   */
  async clickHistoryBackButton(): Promise<void> {
    await this.click(this.historyBackButton)
    await this.sleep(300)
  }

  /**
   * Click Participants tab in event detail view
   */
  async clickHistoryParticipantsTab(): Promise<void> {
    await this.click(this.historyParticipantsTab)
    await this.sleep(300)
  }

  /**
   * Click Messages tab in event detail view
   */
  async clickHistoryMessagesTab(): Promise<void> {
    await this.click(this.historyMessagesTab)
    await this.sleep(300)
  }

  /**
   * Click Actions tab in event detail view (only visible for moderators)
   */
  async clickHistoryActionsTab(): Promise<void> {
    await this.click(this.historyActionsTab)
    await this.sleep(300)
  }

  /**
   * Check if Actions tab is visible (only for moderators)
   */
  async isHistoryActionsTabVisible(): Promise<boolean> {
    return this.isDisplayed(this.historyActionsTab)
  }

  /**
   * Get count of participants in event detail view
   */
  async getHistoryParticipantCount(): Promise<number> {
    try {
      const elements = await this.driver.findElements(this.historyParticipantItems)
      return elements.length
    } catch {
      return 0
    }
  }

  /**
   * Get count of moderator actions in event detail view
   */
  async getHistoryActionCount(): Promise<number> {
    try {
      const elements = await this.driver.findElements(this.historyActionItems)
      return elements.length
    } catch {
      return 0
    }
  }

  /**
   * Get conference history via API
   */
  async getConferenceHistoryViaApi(chatId: string): Promise<any[]> {
    const result = await this.driver.executeScript(`
      return new Promise(async (resolve, reject) => {
        try {
          const token = localStorage.getItem('access_token');
          if (!token) {
            reject('No access token');
            return;
          }
          const response = await fetch('/api/voice/chats/${chatId}/conferences/history', {
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
          resolve(data.conferences || []);
        } catch (e) {
          reject(e.message);
        }
      });
    `) as any[]
    return result || []
  }

  /**
   * Get chat files via API
   */
  async getChatFilesViaApi(chatId: string): Promise<any[]> {
    const result = await this.driver.executeScript(`
      return new Promise(async (resolve, reject) => {
        try {
          const token = localStorage.getItem('access_token');
          if (!token) {
            reject('No access token');
            return;
          }
          const response = await fetch('/api/files/chats/${chatId}/files', {
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
          resolve(data.files || []);
        } catch (e) {
          reject(e.message);
        }
      });
    `) as any[]
    return result || []
  }

  /**
   * Create a conference and end it (for testing history)
   */
  async createAndEndConferenceViaApi(chatId: string, name: string): Promise<string> {
    const conferenceId = await this.driver.executeScript(`
      return new Promise(async (resolve, reject) => {
        try {
          const token = localStorage.getItem('access_token');
          if (!token) {
            reject('No access token');
            return;
          }

          // Create conference
          const createResponse = await fetch('/api/voice/conferences', {
            method: 'POST',
            headers: {
              'Authorization': 'Bearer ' + token,
              'Content-Type': 'application/json'
            },
            body: JSON.stringify({
              name: '${name}',
              chat_id: '${chatId}',
              event_type: 'adhoc'
            })
          });

          if (!createResponse.ok) {
            const error = await createResponse.text();
            reject('Create error: ' + createResponse.status + ' ' + error);
            return;
          }

          const conference = await createResponse.json();
          const confId = conference.id;

          // Wait a bit
          await new Promise(r => setTimeout(r, 1000));

          // End conference
          const endResponse = await fetch('/api/voice/conferences/' + confId + '/end', {
            method: 'POST',
            headers: {
              'Authorization': 'Bearer ' + token
            }
          });

          if (!endResponse.ok) {
            // Conference may auto-end if no participants
            console.log('End conference returned:', endResponse.status);
          }

          resolve(confId);
        } catch (e) {
          reject(e.message);
        }
      });
    `) as string

    return conferenceId
  }

  /**
   * Get current user role via API
   */
  async getCurrentUserRoleViaApi(): Promise<string> {
    const result = await this.driver.executeScript(`
      return new Promise(async (resolve, reject) => {
        try {
          const token = localStorage.getItem('access_token');
          if (!token) {
            reject('No access token');
            return;
          }
          const response = await fetch('/api/auth/me', {
            headers: {
              'Authorization': 'Bearer ' + token
            }
          });
          if (!response.ok) {
            reject('API error: ' + response.status);
            return;
          }
          const user = await response.json();
          resolve(user.role || 'user');
        } catch (e) {
          reject(e.message);
        }
      });
    `) as string
    return result
  }

  /**
   * Set user role via CLI command (for testing as moderator)
   * Note: This requires admin access
   */
  async setUserRoleViaApi(userId: string, role: string): Promise<void> {
    await this.driver.executeScript(`
      return new Promise(async (resolve, reject) => {
        try {
          const token = localStorage.getItem('access_token');
          if (!token) {
            reject('No access token');
            return;
          }
          const response = await fetch('/api/users/${userId}/role', {
            method: 'PUT',
            headers: {
              'Authorization': 'Bearer ' + token,
              'Content-Type': 'application/json'
            },
            body: JSON.stringify({ role: '${role}' })
          });
          if (!response.ok) {
            const error = await response.text();
            reject('API error: ' + response.status + ' ' + error);
            return;
          }
          resolve(true);
        } catch (e) {
          reject(e.message);
        }
      });
    `)
  }

  // ========== Conference Chat Panel ==========

  private readonly conferenceChatToggle = By.css('.conference-view .header-btn[title="Toggle chat"]')
  private readonly conferenceChatSidebar = By.css('.conference-view .chat-sidebar')
  private readonly conferenceChatMessage = By.css('.conference-view .chat-message')
  private readonly conferenceChatOwnMessage = By.css('.conference-view .chat-message.own-message')
  private readonly conferenceChatSystemMessage = By.css('.conference-view .chat-message.system-message')
  private readonly conferenceChatMessageContent = By.css('.conference-view .chat-message .message-content')
  private readonly conferenceChatLoading = By.css('.conference-view .chat-messages .loading-messages')
  private readonly conferenceChatNoMessages = By.css('.conference-view .chat-messages .no-messages')
  private readonly conferenceChatInput = By.css('.conference-view .chat-input input')
  private readonly conferenceChatSendButton = By.css('.conference-view .chat-input button')
  private readonly conferenceChatUnreadBadge = By.css('.conference-view .header-btn[title="Toggle chat"] .badge')

  /**
   * Check if chat toggle button is visible in conference view
   */
  async isConferenceChatToggleVisible(): Promise<boolean> {
    return this.isDisplayed(this.conferenceChatToggle)
  }

  /**
   * Click chat toggle button in conference view
   */
  async clickConferenceChatToggle(): Promise<void> {
    await this.click(this.conferenceChatToggle)
    await this.sleep(500)
  }

  /**
   * Check if chat sidebar is visible in conference view
   */
  async isConferenceChatSidebarVisible(): Promise<boolean> {
    return this.isDisplayed(this.conferenceChatSidebar)
  }

  /**
   * Wait for chat sidebar to appear in conference view
   */
  async waitForConferenceChatSidebar(timeout: number = 10000): Promise<void> {
    const start = Date.now()
    while (Date.now() - start < timeout) {
      if (await this.isConferenceChatSidebarVisible()) {
        return
      }
      await this.sleep(300)
    }
    throw new Error('Conference chat sidebar did not appear within timeout')
  }

  /**
   * Wait for chat sidebar to disappear in conference view
   */
  async waitForConferenceChatSidebarToClose(timeout: number = 10000): Promise<void> {
    const start = Date.now()
    while (Date.now() - start < timeout) {
      if (!(await this.isConferenceChatSidebarVisible())) {
        return
      }
      await this.sleep(300)
    }
    throw new Error('Conference chat sidebar did not close within timeout')
  }

  /**
   * Check if chat is loading in conference view
   */
  async isConferenceChatLoading(): Promise<boolean> {
    return this.isDisplayed(this.conferenceChatLoading)
  }

  /**
   * Wait for chat messages to load in conference view
   */
  async waitForConferenceChatLoaded(timeout: number = 10000): Promise<void> {
    const start = Date.now()
    await this.sleep(300) // Brief wait for loading to start
    while (Date.now() - start < timeout) {
      if (!(await this.isConferenceChatLoading())) {
        return
      }
      await this.sleep(300)
    }
    throw new Error('Conference chat did not finish loading within timeout')
  }

  /**
   * Check if "No messages" state is visible in conference chat
   */
  async isConferenceChatEmpty(): Promise<boolean> {
    return this.isDisplayed(this.conferenceChatNoMessages)
  }

  /**
   * Get count of messages in conference chat
   */
  async getConferenceChatMessageCount(): Promise<number> {
    try {
      const elements = await this.driver.findElements(this.conferenceChatMessage)
      return elements.length
    } catch {
      return 0
    }
  }

  /**
   * Get message texts from conference chat
   */
  async getConferenceChatMessages(): Promise<string[]> {
    const messages: string[] = []
    try {
      const elements = await this.driver.findElements(this.conferenceChatMessageContent)
      for (const el of elements) {
        const text = await el.getText()
        messages.push(text.trim())
      }
    } catch {
      // ignore
    }
    return messages
  }

  /**
   * Get last message text from conference chat
   */
  async getConferenceChatLastMessage(): Promise<string> {
    const messages = await this.getConferenceChatMessages()
    return messages.length > 0 ? messages[messages.length - 1] : ''
  }

  /**
   * Type message in conference chat input
   */
  async typeConferenceChatMessage(message: string): Promise<void> {
    await this.type(this.conferenceChatInput, message)
  }

  /**
   * Click send button in conference chat
   */
  async clickConferenceChatSend(): Promise<void> {
    await this.click(this.conferenceChatSendButton)
    await this.sleep(500)
  }

  /**
   * Send a message from conference chat panel
   */
  async sendConferenceChatMessage(message: string): Promise<void> {
    await this.typeConferenceChatMessage(message)
    await this.clickConferenceChatSend()
  }

  /**
   * Check if send button is enabled in conference chat
   */
  async isConferenceChatSendEnabled(): Promise<boolean> {
    try {
      const element = await this.driver.findElement(this.conferenceChatSendButton)
      const disabled = await element.getAttribute('disabled')
      return disabled === null
    } catch {
      return false
    }
  }

  /**
   * Check if unread badge is visible on chat toggle button
   */
  async hasConferenceChatUnreadBadge(): Promise<boolean> {
    return this.isDisplayed(this.conferenceChatUnreadBadge)
  }

  /**
   * Get unread count from chat toggle badge
   */
  async getConferenceChatUnreadCount(): Promise<number> {
    try {
      const text = await this.getText(this.conferenceChatUnreadBadge)
      return parseInt(text, 10) || 0
    } catch {
      return 0
    }
  }

  /**
   * Wait for a specific message to appear in conference chat
   */
  async waitForConferenceChatMessage(expectedText: string, timeout: number = 10000): Promise<void> {
    const start = Date.now()
    while (Date.now() - start < timeout) {
      const messages = await this.getConferenceChatMessages()
      if (messages.some(m => m.includes(expectedText))) {
        return
      }
      await this.sleep(300)
    }
    throw new Error(`Message containing "${expectedText}" not found in conference chat within timeout`)
  }

  /**
   * Wait for message count in conference chat
   */
  async waitForConferenceChatMessageCount(expectedCount: number, timeout: number = 10000): Promise<void> {
    const start = Date.now()
    while (Date.now() - start < timeout) {
      const count = await this.getConferenceChatMessageCount()
      if (count >= expectedCount) {
        return
      }
      await this.sleep(300)
    }
    throw new Error(`Expected at least ${expectedCount} messages in conference chat, but found ${await this.getConferenceChatMessageCount()}`)
  }

  /**
   * Check if a system message exists in conference chat
   */
  async hasConferenceChatSystemMessage(): Promise<boolean> {
    return this.isDisplayed(this.conferenceChatSystemMessage)
  }

  /**
   * Get system messages from conference chat
   */
  async getConferenceChatSystemMessages(): Promise<string[]> {
    const messages: string[] = []
    try {
      const elements = await this.driver.findElements(this.conferenceChatSystemMessage)
      for (const el of elements) {
        const contentEl = await el.findElement(By.css('.message-content'))
        const text = await contentEl.getText()
        messages.push(text.trim())
      }
    } catch {
      // ignore
    }
    return messages
  }

  /**
   * Check if own message exists in conference chat
   */
  async hasConferenceChatOwnMessage(): Promise<boolean> {
    return this.isDisplayed(this.conferenceChatOwnMessage)
  }

  /**
   * Close conference chat sidebar
   */
  async closeConferenceChatSidebar(): Promise<void> {
    try {
      const closeBtn = By.css('.conference-view .chat-sidebar .sidebar-close')
      await this.forceClick(closeBtn)
      await this.sleep(500)
    } catch (e) {
      // If close button fails, try clicking the toggle button instead
      console.log('Close button failed, trying toggle button')
      await this.clickConferenceChatToggle()
    }
  }
}
