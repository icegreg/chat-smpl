import { Builder, Browser, By, until } from 'selenium-webdriver';
import chrome from 'selenium-webdriver/chrome.js';

const BASE_URL = process.env.BASE_URL || 'http://127.0.0.1:8888';

async function testCallAllSimple() {
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
    console.log('=== Call All Simple Test ===');
    console.log('1. Loading login page...');
    await driver.get(BASE_URL + '/login');
    await driver.sleep(2000);

    console.log('2. Registering user via API...');
    const username = 'callall_' + Date.now();
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
            resolve({ success: true, user: data.user, token: data.access_token });
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

    console.log('3. Creating group chat via API...');
    const chatResult = await driver.executeScript(`
      return new Promise(async (resolve) => {
        try {
          const token = localStorage.getItem('access_token');
          const response = await fetch('/api/chats', {
            method: 'POST',
            headers: {
              'Content-Type': 'application/json',
              'Authorization': 'Bearer ' + token
            },
            body: JSON.stringify({
              name: 'CallAll Test ' + Date.now(),
              type: 'group'
            })
          });
          const data = await response.json();
          resolve({ success: response.ok, chat: data });
        } catch(e) {
          resolve({ success: false, error: e.message });
        }
      });
    `);

    if (!chatResult.success) {
      console.error('Chat creation failed:', chatResult.error);
      return;
    }
    console.log('   Chat:', chatResult.chat.name, '(' + chatResult.chat.id + ')');

    console.log('4. Navigating to chat...');
    await driver.get(BASE_URL + '/chats/' + chatResult.chat.id);
    await driver.sleep(3000);

    console.log('5. Looking for Call button...');
    try {
      await driver.wait(until.elementLocated(By.css('.adhoc-call-button .main-btn')), 10000);
      const callBtn = await driver.findElement(By.css('.adhoc-call-button .main-btn'));
      console.log('   Found call button');

      console.log('6. Clicking call button to open dropdown...');
      await callBtn.click();
      await driver.sleep(500);

      // Wait for dropdown
      await driver.wait(until.elementLocated(By.css('.adhoc-call-button .dropdown')), 5000);
      console.log('   Dropdown appeared');

      console.log('7. Clicking Call All...');
      // Find Call All button by text content
      const dropdownItems = await driver.findElements(By.css('.adhoc-call-button .dropdown-item'));
      let callAllBtn = null;
      for (const item of dropdownItems) {
        const text = await item.getText();
        if (text.includes('Call All')) {
          callAllBtn = item;
          break;
        }
      }

      if (!callAllBtn) {
        console.error('   Call All button not found in dropdown');
        return;
      }

      // Try clicking via JavaScript to ensure the event fires
      await driver.executeScript(`
        console.log('[Test] About to click Call All button');
        arguments[0].click();
        console.log('[Test] Click dispatched');
      `, callAllBtn);
      console.log('   Clicked Call All (via JS)');

      // Get logs immediately after click
      await driver.sleep(500);
      const clickLogs = await driver.manage().logs().get('browser');
      console.log('\n=== Logs after click ===');
      clickLogs.forEach(log => console.log('  [' + log.level + '] ' + log.message));

      console.log('8. Monitoring store state...');

      // First, check Pinia structure
      const piniaCheck = await driver.executeScript(`
        return {
          hasPinia: !!window.__pinia,
          stateKeys: window.__pinia?.state?.value ? Object.keys(window.__pinia.state.value) : [],
          voiceExists: !!window.__pinia?.state?.value?.voice
        };
      `);
      console.log('   Pinia check:', JSON.stringify(piniaCheck));

      // Poll the store state multiple times (quick check)
      for (let i = 0; i < 5; i++) {
        await driver.sleep(1000);
        const state = await driver.executeScript(`
          // Check exposed debug variables
          const vertoInstance = window.__vertoInstance;

          // Debug: log internal structure
          const rpcClient = vertoInstance?.rpcClient;
          const allRpcClientKeys = rpcClient ? Object.keys(rpcClient) : [];

          // Find WebSocket - check all possible property names
          let ws = null;
          let wsPropertyName = 'none';
          for (const key of allRpcClientKeys) {
            const val = rpcClient[key];
            if (val && typeof val === 'object' && val.readyState !== undefined) {
              ws = val;
              wsPropertyName = key;
              break;
            }
          }

          const wsState = ws?.readyState;
          const wsStateStr = wsState === 0 ? 'CONNECTING' : wsState === 1 ? 'OPEN' : wsState === 2 ? 'CLOSING' : wsState === 3 ? 'CLOSED' : 'unknown(' + wsState + ')';

          // Try new exposed store first
          const voiceStore = window.__voiceStore;
          if (voiceStore) {
            return {
              source: 'exposedStore',
              hasVertoInstance: !!vertoInstance,
              wsProperty: wsPropertyName,
              wsState: wsStateStr,
              loading: voiceStore.loading,
              error: voiceStore.error,
              currentConference: voiceStore.currentConference?.id,
              isConnected: voiceStore.isConnected,
              isInCall: voiceStore.isInCall
            };
          }

          const voice = window.__pinia?.state?.value?.voice;
          if (!voice) return {
            error: 'voice store not found',
            hasVertoInstance: !!vertoInstance,
            wsProperty: wsPropertyName,
            wsState: wsStateStr
          };
          return {
            source: 'pinia',
            loading: voice.loading,
            error: voice.error,
            currentConference: voice.currentConference?.id,
            isConnected: voice.isConnected,
            hasVertoInstance: !!vertoInstance,
            wsState: wsStateStr
          };
        `);
        console.log('   [' + i + '] State:', JSON.stringify(state));

        // Early exit if ConferenceView appears
        const confView = await driver.findElements(By.css('.conference-view'));
        if (confView.length > 0) {
          console.log('   ✅ ConferenceView appeared at iteration', i);
          break;
        }
      }

      await driver.sleep(2000);

      // Check ALL browser logs - ALL of them, not just last 40
      const logs = await driver.manage().logs().get('browser');
      console.log('\n=== All Browser Console Logs (ALL) ===');
      logs.forEach(log => console.log('  [' + log.level + '] ' + log.message));

      // Check if conference view appeared
      const conferenceView = await driver.findElements(By.css('.conference-view'));
      if (conferenceView.length > 0) {
        console.log('\n✅ ConferenceView is VISIBLE!');

        // Check for participant tiles
        const participants = await driver.findElements(By.css('.conference-view .participant-tile'));
        console.log('   Participant tiles:', participants.length);
      } else {
        console.log('\n❌ ConferenceView is NOT visible');

        // Debug: check voice store state
        const storeState = await driver.executeScript(`
          try {
            const voiceStore = window.__pinia?.state?.value?.voice;
            return {
              currentConference: voiceStore?.currentConference,
              isInCall: voiceStore?.isInCall,
              loading: voiceStore?.loading,
              error: voiceStore?.error,
              callState: voiceStore?.callState
            };
          } catch(e) {
            return { error: e.message };
          }
        `);
        console.log('   Voice store state:', JSON.stringify(storeState, null, 2));
      }

      // Keep browser open for manual inspection
      console.log('\n9. Waiting 5 seconds for manual inspection...');
      await driver.sleep(5000);

    } catch (e) {
      console.error('Error during call test:', e.message);
    }

  } catch (e) {
    console.error('Test error:', e);
  } finally {
    await driver.quit();
  }
}

testCallAllSimple().catch(e => {
  console.error('Test failed:', e.message);
  process.exit(1);
});
