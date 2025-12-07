import { WebDriver } from 'selenium-webdriver'
import { expect } from 'chai'
import * as path from 'path'
import * as fs from 'fs'
import * as os from 'os'
import { createDriver, quitDriver } from '../config/webdriver.js'
import { ChatPage } from '../pages/ChatPage.js'
import { createTestUser, clearBrowserState } from '../helpers/testHelpers.js'

describe('File Upload', function () {
  let driver: WebDriver
  let chatPage: ChatPage
  let testTextFile: string
  let testImageFile: string

  before(async function () {
    driver = await createDriver()
    chatPage = new ChatPage(driver)

    // Create test files
    const tempDir = os.tmpdir()

    // Create a test text file
    testTextFile = path.join(tempDir, `test-file-${Date.now()}.txt`)
    fs.writeFileSync(testTextFile, 'This is a test file content for E2E testing.')

    // Create a simple test image (1x1 PNG)
    testImageFile = path.join(tempDir, `test-image-${Date.now()}.png`)
    // Minimal valid 1x1 white PNG
    const pngBuffer = Buffer.from([
      0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, // PNG signature
      0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52, // IHDR chunk
      0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
      0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53,
      0xde, 0x00, 0x00, 0x00, 0x0c, 0x49, 0x44, 0x41, // IDAT chunk
      0x54, 0x08, 0xd7, 0x63, 0xf8, 0xff, 0xff, 0x3f,
      0x00, 0x05, 0xfe, 0x02, 0xfe, 0xdc, 0xcc, 0x59,
      0xe7, 0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4e, // IEND chunk
      0x44, 0xae, 0x42, 0x60, 0x82
    ])
    fs.writeFileSync(testImageFile, pngBuffer)
  })

  after(async function () {
    await quitDriver(driver)

    // Clean up test files
    try {
      if (fs.existsSync(testTextFile)) fs.unlinkSync(testTextFile)
      if (fs.existsSync(testImageFile)) fs.unlinkSync(testImageFile)
    } catch {
      // Ignore cleanup errors
    }
  })

  beforeEach(async function () {
    await clearBrowserState(driver)
  })

  it('should show attach file button in chat', async function () {
    await createTestUser(driver)
    await chatPage.waitForChatPage()

    // Create a chat
    const chatName = `File Upload Test ${Date.now()}`
    await chatPage.createChat(chatName, 'group')
    await chatPage.waitForModalToClose()
    await chatPage.sleep(500)

    // Select the chat
    await chatPage.selectFirstChat()
    await chatPage.sleep(500)

    // Verify attach button is visible
    const attachButton = await driver.findElements(
      { css: 'button[title="Attach file"]' }
    )
    expect(attachButton.length).to.be.greaterThan(0)
  })

  it('should show pending file after selecting a file', async function () {
    await createTestUser(driver)
    await chatPage.waitForChatPage()

    // Create a chat
    const chatName = `Pending File Test ${Date.now()}`
    await chatPage.createChat(chatName, 'group')
    await chatPage.waitForModalToClose()
    await chatPage.sleep(500)

    // Select the chat
    await chatPage.selectFirstChat()
    await chatPage.sleep(1000)

    // Debug: check if file input exists
    const fileInputs = await driver.findElements({ css: 'input[type="file"]' })
    console.log(`Found ${fileInputs.length} file inputs`)

    // Attach a file
    console.log(`Attaching file: ${testTextFile}`)
    await chatPage.attachFile(testTextFile)

    // Give it time
    await chatPage.sleep(2000)

    // Debug: check pending files
    const pendingFiles = await driver.findElements({ css: '[data-testid="pending-file"]' })
    console.log(`Found ${pendingFiles.length} pending files after attach`)

    // Debug: check spinners
    const spinners = await driver.findElements({ css: '[data-testid="file-uploading-spinner"]' })
    console.log(`Found ${spinners.length} spinners`)

    // Debug: check remove buttons
    const removeButtons = await driver.findElements({ css: '[data-testid="remove-pending-file"]' })
    console.log(`Found ${removeButtons.length} remove buttons`)

    // Debug: wait a bit and check again
    await chatPage.sleep(1000)
    const spinnersAfter = await driver.findElements({ css: '[data-testid="file-uploading-spinner"]' })
    const removeButtonsAfter = await driver.findElements({ css: '[data-testid="remove-pending-file"]' })
    console.log(`After 1s: ${spinnersAfter.length} spinners, ${removeButtonsAfter.length} remove buttons`)

    // Wait for file upload to complete
    await chatPage.waitForFileUploadComplete()

    // Verify pending file is shown
    expect(await chatPage.isPendingFileVisible()).to.be.true

    // Verify file name is displayed
    const fileName = await chatPage.getPendingFileName()
    expect(fileName).to.include('test-file')
  })

  it('should remove pending file when clicking remove button', async function () {
    await createTestUser(driver)
    await chatPage.waitForChatPage()

    // Create a chat
    const chatName = `Remove File Test ${Date.now()}`
    await chatPage.createChat(chatName, 'group')
    await chatPage.waitForModalToClose()
    await chatPage.sleep(500)

    // Select the chat
    await chatPage.selectFirstChat()
    await chatPage.sleep(500)

    // Attach a file
    await chatPage.attachFile(testTextFile)
    await chatPage.waitForFileUploadComplete()
    expect(await chatPage.isPendingFileVisible()).to.be.true

    // Remove the file
    await chatPage.removePendingFile()
    await chatPage.sleep(300)

    // Verify file is removed
    expect(await chatPage.getPendingFilesCount()).to.equal(0)
  })

  it('should send message with file attachment', async function () {
    await createTestUser(driver)
    await chatPage.waitForChatPage()

    // Create a chat
    const chatName = `Send File Test ${Date.now()}`
    await chatPage.createChat(chatName, 'group')
    await chatPage.waitForModalToClose()
    await chatPage.sleep(500)

    // Select the chat
    await chatPage.selectFirstChat()
    await chatPage.sleep(500)

    // Send message with file
    const messageText = `Message with file ${Date.now()}`
    await chatPage.sendMessageWithFile(messageText, testTextFile)

    // Wait for message to appear
    await chatPage.waitForMessageContaining(messageText, 10000)

    // Wait for file attachment to appear
    await chatPage.waitForFileAttachment(10000)

    // Verify file attachment is displayed
    const attachmentsCount = await chatPage.getFileAttachmentsCount()
    expect(attachmentsCount).to.be.greaterThan(0)
  })

  it('should send message with only file (no text)', async function () {
    await createTestUser(driver)
    await chatPage.waitForChatPage()

    // Create a chat
    const chatName = `File Only Test ${Date.now()}`
    await chatPage.createChat(chatName, 'group')
    await chatPage.waitForModalToClose()
    await chatPage.sleep(500)

    // Select the chat
    await chatPage.selectFirstChat()
    await chatPage.sleep(500)

    const initialCount = await chatPage.getMessageCount()

    // Send file without text
    await chatPage.sendMessageWithFile('', testTextFile)

    // Wait for message count to increase
    await chatPage.waitForMessageCount(initialCount + 1, 10000)

    // Wait for file attachment to appear
    await chatPage.waitForFileAttachment(10000)

    // Verify file attachment is displayed
    const attachmentsCount = await chatPage.getFileAttachmentsCount()
    expect(attachmentsCount).to.be.greaterThan(0)
  })

  it('should display image preview for image files', async function () {
    await createTestUser(driver)
    await chatPage.waitForChatPage()

    // Create a chat
    const chatName = `Image Preview Test ${Date.now()}`
    await chatPage.createChat(chatName, 'group')
    await chatPage.waitForModalToClose()
    await chatPage.sleep(500)

    // Select the chat
    await chatPage.selectFirstChat()
    await chatPage.sleep(500)

    // Send message with image
    const messageText = `Message with image ${Date.now()}`
    await chatPage.sendMessageWithFile(messageText, testImageFile)

    // Wait for message to appear
    await chatPage.waitForMessageContaining(messageText, 10000)

    // Wait for image attachment to appear
    await chatPage.waitForImageAttachment(10000)

    // Verify image is displayed inline
    const imagesCount = await chatPage.getImageAttachmentsCount()
    expect(imagesCount).to.be.greaterThan(0)
  })

  it('should have correct download link for file attachments', async function () {
    await createTestUser(driver)
    await chatPage.waitForChatPage()

    // Create a chat
    const chatName = `Download Link Test ${Date.now()}`
    await chatPage.createChat(chatName, 'group')
    await chatPage.waitForModalToClose()
    await chatPage.sleep(500)

    // Select the chat
    await chatPage.selectFirstChat()
    await chatPage.sleep(500)

    // Send message with file
    await chatPage.sendMessageWithFile('Test file download', testTextFile)

    // Wait for file attachment to appear
    await chatPage.waitForFileAttachment(10000)

    // Get file attachment hrefs
    const hrefs = await chatPage.getFileAttachmentHrefs()
    expect(hrefs.length).to.be.greaterThan(0)

    // Verify href format is correct
    expect(hrefs[0]).to.include('/api/files/')
  })

  it('should attach multiple files', async function () {
    await createTestUser(driver)
    await chatPage.waitForChatPage()

    // Create a chat
    const chatName = `Multiple Files Test ${Date.now()}`
    await chatPage.createChat(chatName, 'group')
    await chatPage.waitForModalToClose()
    await chatPage.sleep(500)

    // Select the chat
    await chatPage.selectFirstChat()
    await chatPage.sleep(500)

    // Attach first file
    await chatPage.attachFile(testTextFile)
    await chatPage.waitForFileUploadComplete()

    // Attach second file
    await chatPage.attachFile(testImageFile)
    await chatPage.sleep(2000) // Wait for second upload

    // Verify multiple pending files
    const pendingCount = await chatPage.getPendingFilesCount()
    expect(pendingCount).to.equal(2)
  })
})
