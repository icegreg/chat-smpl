import { Builder, Browser, By, until } from 'selenium-webdriver';
import chrome from 'selenium-webdriver/chrome.js';

const BASE_URL = process.env.BASE_URL || 'http://127.0.0.1:8888';

async function testWebSocketDebug() {
  const options = new chrome.Options();
  options.addArguments(
    '--no-sandbox',
    '--disable-dev-shm-usage',
    '--use-fake-device-for-media-stream',
    '--use-fake-ui-for-media-stream',
    '--autoplay-policy=no-user-gesture-required'
  );

  const driver = await new Builder()
    .forBrowser(Browser.CHROME)
    .setChromeOptions(options)
    .build();

  try {
    console.log('=== WebSocket Debug Test ===');

    // Step 1: Load login page and check initial WebSocket connections
    console.log('1. Loading login page...');
    await driver.get(BASE_URL + '/login');
    await driver.sleep(3000);

    let logs = await driver.manage().logs().get('browser');
    console.log('\\n=== Initial page load logs ===');
    logs.forEach(log => console.log('  [' + log.level + '] ' + log.message));

    // Step 2: Register user
    console.log('\\n2. Registering user...');
    const username = 'ws_debug_' + Date.now();
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

    // Step 3: Create chat
    console.log('\\n3. Creating group chat...');
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
              name: 'WebSocket Debug ' + Date.now(),
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
    console.log('   Chat:', chatResult.chat.id);

    // Step 4: Navigate to chat page
    console.log('\\n4. Navigating to chat page...');
    await driver.get(BASE_URL + '/chats/' + chatResult.chat.id);
    await driver.sleep(5000);

    logs = await driver.manage().logs().get('browser');
    console.log('\\n=== Logs after chat navigation ===');
    logs.forEach(log => {
      if (log.message.includes('WebSocket') || log.message.includes('verto') || log.message.includes('Verto')) {
        console.log('  [' + log.level + '] ' + log.message);
      }
    });

    // Step 5: Click Call All directly via store
    console.log('\\n5. Calling startChatCall directly via store...');
    const callResult = await driver.executeScript(`
      return new Promise(async (resolve) => {
        try {
          // Check if Pinia is available
          console.log('[Test] Checking Pinia...');
          console.log('[Test] window.__pinia:', !!window.__pinia);

          // Get voice store from app instance
          const app = document.getElementById('app').__vue_app__;
          console.log('[Test] Vue app:', !!app);

          if (!app) {
            resolve({ success: false, error: 'Vue app not found' });
            return;
          }

          const pinia = app._context.provides;
          console.log('[Test] Pinia from app:', !!pinia);

          // Try to access voice store via API
          const token = localStorage.getItem('access_token');
          console.log('[Test] Calling startChatCall API...');

          const response = await fetch('/api/voice/chats/' + '${chatResult.chat.id}' + '/call', {
            method: 'POST',
            headers: {
              'Content-Type': 'application/json',
              'Authorization': 'Bearer ' + token
            }
          });

          const data = await response.json();
          console.log('[Test] API response:', JSON.stringify(data));

          if (!response.ok) {
            resolve({ success: false, error: data.error || 'API failed' });
            return;
          }

          // Now try to connect to Verto manually
          console.log('[Test] Got credentials, connecting to Verto...');
          console.log('[Test] ws_url:', data.credentials.ws_url);

          // Wait for Verto to load
          if (!window.Verto) {
            resolve({ success: false, error: 'Verto not loaded' });
            return;
          }

          console.log('[Test] Creating Verto instance...');
          const verto = new window.Verto({
            login: data.credentials.login,
            passwd: data.credentials.password,
            socketUrl: data.credentials.ws_url,
            tag: 'verto-audio',
            deviceParams: {
              useMic: 'any',
              useSpeak: 'any',
              useCamera: false
            }
          }, {
            onWSLogin: function(v, success) {
              console.log('[Test] onWSLogin:', success);
              if (success) {
                resolve({ success: true, loginSuccess: true, wsUrl: data.credentials.ws_url });
              } else {
                resolve({ success: false, error: 'Login failed' });
              }
            },
            onWSClose: function(v, success) {
              console.log('[Test] onWSClose');
            }
          });

          console.log('[Test] Verto instance created, calling login...');
          verto.login();
          console.log('[Test] login() called');

          // Timeout after 10 seconds
          setTimeout(() => {
            resolve({ success: false, error: 'Timeout waiting for login' });
          }, 10000);

        } catch(e) {
          console.error('[Test] Error:', e);
          resolve({ success: false, error: e.message });
        }
      });
    `);

    console.log('\\n6. Call result:', JSON.stringify(callResult, null, 2));

    // Get final logs
    await driver.sleep(2000);
    logs = await driver.manage().logs().get('browser');
    console.log('\\n=== Final browser logs ===');
    logs.forEach(log => console.log('  [' + log.level + '] ' + log.message));

    console.log('\\n7. Waiting 10 seconds for inspection...');
    await driver.sleep(10000);

  } catch (e) {
    console.error('Test error:', e);
  } finally {
    await driver.quit();
  }
}

testWebSocketDebug().catch(e => {
  console.error('Test failed:', e.message);
  process.exit(1);
});
