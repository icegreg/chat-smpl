import { Builder, Browser, By, until } from 'selenium-webdriver';
import chrome from 'selenium-webdriver/chrome.js';

const BASE_URL = process.env.BASE_URL || 'https://10.99.22.46';
const HEADLESS = process.env.HEADLESS !== 'false';

async function createDriver(name) {
  const options = new chrome.Options();
  options.addArguments(
    '--no-sandbox',
    '--disable-dev-shm-usage',
    '--use-fake-device-for-media-stream',
    '--use-fake-ui-for-media-stream',
    '--autoplay-policy=no-user-gesture-required'
  );

  if (HEADLESS) {
    options.addArguments('--headless=new');
  }

  // Accept self-signed SSL certificates
  options.setAcceptInsecureCerts(true);

  const driver = await new Builder()
    .forBrowser(Browser.CHROME)
    .setChromeOptions(options)
    .build();

  console.log(`[${name}] Browser created`);
  return driver;
}

async function registerUser(driver, name, suffix) {
  const username = `verto_test_${Date.now()}_${suffix}`;

  const result = await driver.executeScript(`
    return new Promise(async (resolve) => {
      try {
        const regResponse = await fetch('/api/auth/register', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            email: '${username}@test.com',
            username: '${username}',
            password: 'test1234'
          })
        });
        const data = await regResponse.json();
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

  if (result.success) {
    console.log(`[${name}] Registered: ${result.user.username} (ext: ${result.user.extension})`);
  }
  return result;
}

async function createGroupChat(driver, name, token, chatName, participantIds) {
  const result = await driver.executeScript(`
    return new Promise(async (resolve) => {
      try {
        const response = await fetch('/api/chats', {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            'Authorization': 'Bearer ${token}'
          },
          body: JSON.stringify({
            name: '${chatName}',
            type: 'group',
            participant_ids: ${JSON.stringify(participantIds)}
          })
        });
        const data = await response.json();
        resolve({ success: response.ok, chat: data });
      } catch(e) {
        resolve({ success: false, error: e.message });
      }
    });
  `);

  if (result.success) {
    console.log(`[${name}] Chat created: ${result.chat.name} (${result.chat.id})`);
  }
  return result;
}

async function initVertoConnection(driver, name, token) {
  console.log(`[${name}] Initializing Verto connection...`);

  const result = await driver.executeScript(`
    return new Promise(async (resolve) => {
      try {
        // Get Verto credentials
        const credResponse = await fetch('/api/voice/credentials', {
          headers: { 'Authorization': 'Bearer ${token}' }
        });
        const creds = await credResponse.json();

        // Build dynamic WebSocket URL from current location (like production code does)
        const wsProtocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const dynamicWsUrl = wsProtocol + '//' + window.location.host + '/verto';
        console.log('[Verto] Got credentials:', creds.login, 'ws_from_api:', creds.ws_url, 'ws_dynamic:', dynamicWsUrl);

        // Wait for Verto.js to load
        let attempts = 0;
        while (!window.Verto && attempts < 30) {
          await new Promise(r => setTimeout(r, 500));
          attempts++;
        }

        if (!window.Verto) {
          resolve({ success: false, error: 'Verto.js not loaded' });
          return;
        }

        // Create Verto instance with dynamic WebSocket URL
        window.__vertoInstance = new window.Verto({
          login: creds.login,
          passwd: creds.password,
          socketUrl: dynamicWsUrl,
          tag: 'verto-audio',
          deviceParams: {
            useMic: 'any',
            useSpeak: 'any',
            useCamera: false
          }
        }, {
          onWSLogin: function(v, success) {
            console.log('[Verto] Login result:', success);
            if (success) {
              window.__vertoLoggedIn = true;
              resolve({ success: true, login: creds.login });
            } else {
              resolve({ success: false, error: 'Login failed' });
            }
          },
          onWSClose: function() {
            console.log('[Verto] WebSocket closed');
          },
          onDialogState: function(dialog) {
            console.log('[Verto] Dialog state:', dialog.state?.name, 'direction:', dialog.direction);

            // Track incoming calls
            if (dialog.direction === 'inbound' && dialog.state?.name === 'ringing') {
              console.log('[Verto] INCOMING CALL!', {
                callID: dialog.callID,
                callerIdName: dialog.callerIdName,
                callerIdNumber: dialog.callerIdNumber,
                destinationNumber: dialog.destinationNumber
              });
              window.__incomingCall = {
                dialog: dialog,
                callID: dialog.callID,
                callerIdName: dialog.callerIdName,
                callerIdNumber: dialog.callerIdNumber,
                destinationNumber: dialog.destinationNumber,
                receivedAt: Date.now()
              };
            }

            // Track call state changes
            if (dialog.callID === window.__incomingCall?.callID) {
              window.__incomingCall.lastState = dialog.state?.name;
            }
          }
        });

        window.__vertoInstance.login();

        // Timeout
        setTimeout(() => {
          if (!window.__vertoLoggedIn) {
            resolve({ success: false, error: 'Login timeout' });
          }
        }, 10000);

      } catch(e) {
        resolve({ success: false, error: e.message });
      }
    });
  `);

  if (result.success) {
    console.log(`[${name}] Verto connected: ${result.login}`);
  } else {
    console.log(`[${name}] Verto connection failed: ${result.error}`);
  }
  return result;
}

async function createAdHocConference(driver, name, token, chatId, participantIds) {
  console.log(`[${name}] Creating ad-hoc conference...`);
  console.log(`[${name}] Participant IDs: ${JSON.stringify(participantIds)}`);

  const result = await driver.executeScript(`
    return new Promise(async (resolve) => {
      try {
        const body = {
          chat_id: '${chatId}',
          participant_ids: ${JSON.stringify(participantIds)}
        };
        console.log('[API] Request body:', JSON.stringify(body));

        const response = await fetch('/api/voice/conferences/adhoc-chat', {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            'Authorization': 'Bearer ${token}'
          },
          body: JSON.stringify(body)
        });
        const data = await response.json();
        console.log('[API] Ad-hoc conference response:', data);
        resolve({ success: response.ok, conference: data });
      } catch(e) {
        resolve({ success: false, error: e.message });
      }
    });
  `);

  if (result.success) {
    console.log(`[${name}] Conference created: ${result.conference.name} (fs: ${result.conference.freeswitch_name})`);
  } else {
    console.log(`[${name}] Conference creation failed: ${result.error}`);
  }
  return result;
}

async function checkIncomingCall(driver, name, timeout = 10000) {
  console.log(`[${name}] Waiting for incoming call (${timeout}ms)...`);

  const startTime = Date.now();
  let result = null;

  while (Date.now() - startTime < timeout) {
    result = await driver.executeScript(`
      return window.__incomingCall || null;
    `);

    if (result) {
      console.log(`[${name}] INCOMING CALL RECEIVED!`);
      console.log(`[${name}]   CallerID Name: ${result.callerIdName}`);
      console.log(`[${name}]   CallerID Number: ${result.callerIdNumber}`);
      console.log(`[${name}]   Destination: ${result.destinationNumber}`);
      console.log(`[${name}]   State: ${result.lastState}`);
      return { success: true, incomingCall: result };
    }

    await driver.sleep(500);
  }

  console.log(`[${name}] No incoming call received within ${timeout}ms`);
  return { success: false, error: 'No incoming call' };
}

async function answerIncomingCall(driver, name) {
  console.log(`[${name}] Answering incoming call...`);

  const result = await driver.executeScript(`
    return new Promise((resolve) => {
      try {
        if (!window.__incomingCall?.dialog) {
          resolve({ success: false, error: 'No incoming call to answer' });
          return;
        }

        const dialog = window.__incomingCall.dialog;
        console.log('[Verto] Answering call:', dialog.callID);
        dialog.answer();

        setTimeout(() => {
          resolve({
            success: true,
            callID: dialog.callID,
            state: dialog.state?.name
          });
        }, 2000);

      } catch(e) {
        resolve({ success: false, error: e.message });
      }
    });
  `);

  if (result.success) {
    console.log(`[${name}] Call answered, state: ${result.state}`);
  } else {
    console.log(`[${name}] Answer failed: ${result.error}`);
  }
  return result;
}

async function getBrowserLogs(driver, name) {
  try {
    const logs = await driver.manage().logs().get('browser');
    const vertoLogs = logs.filter(log =>
      log.message.includes('[Verto]') ||
      log.message.includes('conference') ||
      log.message.includes('incoming') ||
      log.message.includes('INVITE') ||
      log.message.includes('error')
    );

    if (vertoLogs.length > 0) {
      console.log(`\n[${name}] === Browser Logs ===`);
      vertoLogs.slice(-15).forEach(log => {
        console.log(`  ${log.message.substring(0, 200)}`);
      });
    }
  } catch (e) {
    // Ignore log errors
  }
}

async function testVertoIncoming() {
  console.log('=== Verto Incoming Call Test ===');
  console.log(`Base URL: ${BASE_URL}`);
  console.log(`Headless: ${HEADLESS}`);

  let driver1, driver2;

  try {
    // Create browsers
    console.log('\n1. Creating browser instances...');
    driver1 = await createDriver('Caller');
    driver2 = await createDriver('Callee');

    // Load pages
    console.log('\n2. Loading pages...');
    await driver1.get(BASE_URL + '/login');
    await driver2.get(BASE_URL + '/login');
    await driver1.sleep(2000);
    await driver2.sleep(2000);

    // Register users
    console.log('\n3. Registering users...');
    const user1 = await registerUser(driver1, 'Caller', 'caller');
    const user2 = await registerUser(driver2, 'Callee', 'callee');

    if (!user1.success || !user2.success) {
      console.error('Registration failed');
      return;
    }

    // Navigate to main page to load Vue app and Verto.js
    console.log('\n4. Loading main app...');
    await driver1.get(BASE_URL);
    await driver2.get(BASE_URL);
    await driver1.sleep(3000);
    await driver2.sleep(3000);

    // Create group chat with both users
    console.log('\n5. Creating group chat...');
    const chatResult = await createGroupChat(
      driver1, 'Caller', user1.token,
      `VertoTest_${Date.now()}`,
      [user1.user.id, user2.user.id]
    );

    if (!chatResult.success) {
      console.error('Chat creation failed:', chatResult.error);
      return;
    }

    // Initialize Verto for callee (user2) FIRST
    console.log('\n6. Connecting callee to Verto...');
    const verto2 = await initVertoConnection(driver2, 'Callee', user2.token);

    if (!verto2.success) {
      console.error('Callee Verto connection failed');
      await getBrowserLogs(driver2, 'Callee');
      return;
    }

    // Wait a moment for Verto connection to stabilize
    await driver2.sleep(2000);

    // Create ad-hoc conference (this should trigger INVITE to callee)
    console.log('\n7. Creating ad-hoc conference (should send INVITE to callee)...');
    const confResult = await createAdHocConference(
      driver1, 'Caller', user1.token,
      chatResult.chat.id,
      [user1.user.id, user2.user.id]
    );

    if (!confResult.success) {
      console.error('Conference creation failed:', confResult.error);
      return;
    }

    // Check if callee received incoming call
    console.log('\n8. Checking for incoming call on callee...');
    const incomingResult = await checkIncomingCall(driver2, 'Callee', 15000);

    // Get browser logs for debugging
    await getBrowserLogs(driver2, 'Callee');

    // Results
    console.log('\n=== TEST RESULTS ===');
    if (incomingResult.success) {
      console.log('✅ INCOMING CALL RECEIVED!');
      console.log('   The Verto INVITE was successfully delivered to the callee.');

      // Try to answer the call
      console.log('\n9. Answering the call...');
      const answerResult = await answerIncomingCall(driver2, 'Callee');

      if (answerResult.success) {
        console.log('✅ CALL ANSWERED!');
        console.log('   Call state:', answerResult.state);
      } else {
        console.log('❌ Answer failed:', answerResult.error);
      }
    } else {
      console.log('❌ NO INCOMING CALL RECEIVED');
      console.log('   The Verto INVITE was not delivered to the callee.');
      console.log('   Possible reasons:');
      console.log('   - FreeSWITCH not connected to ESL');
      console.log('   - User extension not set in database');
      console.log('   - Verto WebSocket not properly connected');
    }

    // Wait for manual inspection if not headless
    if (!HEADLESS) {
      console.log('\n10. Waiting 30 seconds for inspection...');
      await driver1.sleep(30000);
    }

  } catch (e) {
    console.error('Test error:', e);
  } finally {
    if (driver1) await driver1.quit();
    if (driver2) await driver2.quit();
  }
}

testVertoIncoming().catch(e => {
  console.error('Test failed:', e.message);
  process.exit(1);
});
