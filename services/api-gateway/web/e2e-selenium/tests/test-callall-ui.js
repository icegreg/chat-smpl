import { Builder, Browser, By, until } from 'selenium-webdriver';
import chrome from 'selenium-webdriver/chrome.js';

const BASE_URL = process.env.BASE_URL || 'http://127.0.0.1:8888';

async function testCallAllUI() {
  const options = new chrome.Options();
  options.addArguments(
    // NO headless - we want to see what's happening
    '--no-sandbox',
    '--disable-dev-shm-usage',
    '--use-fake-device-for-media-stream',
    '--use-fake-ui-for-media-stream',
    '--autoplay-policy=no-user-gesture-required'
  );
  // Для работы с self-signed SSL сертификатами
  options.setAcceptInsecureCerts(true);

  const driver = await new Builder()
    .forBrowser(Browser.CHROME)
    .setChromeOptions(options)
    .build();

  try {
    console.log('=== Call All UI Test ===');
    console.log('1. Loading login page...');
    await driver.get(BASE_URL + '/login');
    await driver.sleep(2000);

    console.log('2. Registering user...');
    const username = 'callall_ui_' + Date.now();
    const registerResult = await driver.executeScript(`
      return new Promise(async (resolve) => {
        try {
          const response = await fetch('/api/auth/register', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
              email: '${username}@test.com',
              username: '${username}',
              password: 'test1234'
            })
          });
          const data = await response.json();
          if (data.access_token) {
            localStorage.setItem('access_token', data.access_token);
            localStorage.setItem('refresh_token', data.refresh_token);
            resolve({ success: true, user: data.user });
          } else {
            resolve({ success: false, error: data });
          }
        } catch(e) {
          resolve({ success: false, error: e.message });
        }
      });
    `);

    if (!registerResult.success) {
      console.error('Registration failed:', registerResult.error);
      return;
    }
    console.log('   User:', registerResult.user.username);

    console.log('3. Navigating to main page...');
    await driver.get(BASE_URL);
    await driver.sleep(2000);

    console.log('4. Creating group chat...');
    // Click "New Chat" button
    const newChatBtn = await driver.findElement(By.css('.new-chat-btn, button[title*="chat"], [data-testid="new-chat"]'));
    await newChatBtn.click();
    await driver.sleep(500);

    // Fill chat name
    const chatNameInput = await driver.findElement(By.css('input[placeholder*="name"], input[name="name"]'));
    await chatNameInput.sendKeys('Test Call All ' + Date.now());

    // Select group type
    try {
      const groupRadio = await driver.findElement(By.css('input[value="group"], label:contains("Group")'));
      await groupRadio.click();
    } catch (e) {
      // Maybe already selected or different UI
    }

    // Click create
    const createBtn = await driver.findElement(By.css('button[type="submit"], .create-btn, button:contains("Create")'));
    await createBtn.click();
    await driver.sleep(1000);

    console.log('5. Selecting chat...');
    const chatItem = await driver.findElement(By.css('.chat-item, .chat-list-item'));
    await chatItem.click();
    await driver.sleep(1000);

    console.log('6. Looking for Call button...');
    // Wait for call button
    await driver.wait(until.elementLocated(By.css('.adhoc-call-button .main-btn')), 10000);
    const callBtn = await driver.findElement(By.css('.adhoc-call-button .main-btn'));
    console.log('   Found call button');

    console.log('7. Clicking call button...');
    await callBtn.click();
    await driver.sleep(500);

    // Wait for dropdown
    const dropdown = await driver.wait(until.elementLocated(By.css('.adhoc-call-button .dropdown')), 5000);
    console.log('   Dropdown appeared');

    console.log('8. Clicking Call All...');
    const callAllBtn = await dropdown.findElement(By.xpath(".//button[contains(text(), 'Call All')]"));
    await callAllBtn.click();

    console.log('9. Waiting for conference view...');
    await driver.sleep(3000);

    // Check browser logs
    const logs = await driver.manage().logs().get('browser');
    console.log('\n=== Browser Console Logs ===');
    const vertoLogs = logs.filter(log =>
      log.message.includes('[Verto]') ||
      log.message.includes('VoiceStore') ||
      log.message.includes('conference') ||
      log.message.includes('error')
    );
    vertoLogs.slice(-20).forEach(log => console.log(log.message));

    // Check if conference view appeared
    const conferenceView = await driver.findElements(By.css('.conference-view'));
    if (conferenceView.length > 0) {
      console.log('\n✅ ConferenceView is visible!');
    } else {
      console.log('\n❌ ConferenceView is NOT visible');

      // Debug: check store state
      const storeState = await driver.executeScript(`
        const voiceStore = window.__pinia?.state?.value?.voice;
        return {
          currentConference: voiceStore?.currentConference,
          isInCall: voiceStore?.isInCall,
          loading: voiceStore?.loading,
          error: voiceStore?.error
        };
      `);
      console.log('Voice store state:', JSON.stringify(storeState, null, 2));
    }

    // Keep browser open for 30 seconds for inspection
    console.log('\n10. Waiting 30 seconds for manual inspection...');
    await driver.sleep(30000);

  } catch (e) {
    console.error('Test error:', e);
  } finally {
    await driver.quit();
  }
}

testCallAllUI().catch(e => {
  console.error('Test failed:', e.message);
  process.exit(1);
});
