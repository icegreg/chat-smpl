import { Builder, Browser } from 'selenium-webdriver';
import chrome from 'selenium-webdriver/chrome.js';

const BASE_URL = process.env.BASE_URL || 'http://127.0.0.1:8888';

async function createDriver() {
  const options = new chrome.Options();
  options.addArguments(
    '--headless=new',
    '--no-sandbox',
    '--disable-dev-shm-usage',
    '--use-fake-device-for-media-stream',
    '--use-fake-ui-for-media-stream',
    '--autoplay-policy=no-user-gesture-required'
  );

  return await new Builder()
    .forBrowser(Browser.CHROME)
    .setChromeOptions(options)
    .build();
}

async function registerUser(driver, suffix) {
  const username = `group_test_${Date.now()}_${suffix}`;

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

  return result;
}

async function createGroupChat(driver, token, chatName) {
  const result = await driver.executeScript(`
    return new Promise(async (resolve) => {
      try {
        const response = await fetch('/api/chats', {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            'Authorization': 'Bearer ' + '${token}'
          },
          body: JSON.stringify({
            name: '${chatName}',
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
  return result;
}

async function addParticipant(driver, token, chatId, userId) {
  const result = await driver.executeScript(`
    return new Promise(async (resolve) => {
      try {
        const response = await fetch('/api/chats/${chatId}/participants', {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            'Authorization': 'Bearer ' + '${token}'
          },
          body: JSON.stringify({
            user_id: '${userId}'
          })
        });
        const data = await response.json();
        resolve({ success: response.ok, data });
      } catch(e) {
        resolve({ success: false, error: e.message });
      }
    });
  `);
  return result;
}

async function joinConference(driver, token, conferenceName) {
  const result = await driver.executeScript(`
    return new Promise(async (resolve) => {
      try {
        // Get Verto credentials
        const credResponse = await fetch('/api/voice/credentials', {
          headers: { 'Authorization': 'Bearer ' + '${token}' }
        });
        const creds = await credResponse.json();
        console.log('[Test] Got credentials:', creds.login);

        // Wait for Verto to load
        let attempts = 0;
        while (!window.Verto && attempts < 30) {
          await new Promise(r => setTimeout(r, 500));
          attempts++;
        }

        if (!window.Verto) {
          resolve({ success: false, error: 'Verto not loaded' });
          return;
        }

        // Create Verto instance
        const verto = new window.Verto({
          login: creds.login,
          passwd: creds.password,
          socketUrl: creds.ws_url,
          tag: 'verto-audio',
          deviceParams: {
            useMic: 'any',
            useSpeak: 'any',
            useCamera: false
          }
        }, {
          onWSLogin: function(v, success) {
            console.log('[Verto] Login:', success);
            if (success) {
              setTimeout(function() {
                try {
                  console.log('[Verto] Joining conference: ${conferenceName}');
                  const dialog = v.newCall({
                    destination_number: 'conf_${conferenceName}',
                    caller_id_name: creds.login,
                    useVideo: false,
                    tag: 'verto-audio'
                  });

                  window.__vertoDialog = dialog;
                  console.log('[Verto] Dialog created:', dialog?.callID);

                  // Wait for connection
                  setTimeout(function() {
                    resolve({
                      success: true,
                      loginSuccess: success,
                      dialogCreated: !!dialog,
                      dialogCallID: dialog?.callID,
                      dialogState: dialog?.state?.name
                    });
                  }, 5000);
                } catch(e) {
                  resolve({ success: false, error: 'Call error: ' + e.message });
                }
              }, 1000);
            } else {
              resolve({ success: false, error: 'Verto login failed' });
            }
          },
          onDialogState: function(dialog) {
            console.log('[Verto] Dialog state:', dialog.state.name);
          }
        });

        verto.login();

      } catch(e) {
        resolve({ success: false, error: e.message });
      }
    });
  `);
  return result;
}

async function checkWebRTCStats(driver) {
  const stats = await driver.executeScript(`
    return new Promise(async (resolve) => {
      try {
        const pcs = window.__vertoDialog?.rtc?.peerConnection;
        if (!pcs) {
          resolve({ hasPeerConnection: false });
          return;
        }

        const stats = await pcs.getStats();
        let bytesReceived = 0;
        let bytesSent = 0;
        let packetsReceived = 0;
        let packetsSent = 0;

        stats.forEach(report => {
          if (report.type === 'inbound-rtp' && report.kind === 'audio') {
            bytesReceived += report.bytesReceived || 0;
            packetsReceived += report.packetsReceived || 0;
          }
          if (report.type === 'outbound-rtp' && report.kind === 'audio') {
            bytesSent += report.bytesSent || 0;
            packetsSent += report.packetsSent || 0;
          }
        });

        resolve({
          hasPeerConnection: true,
          connectionState: pcs.connectionState,
          iceConnectionState: pcs.iceConnectionState,
          bytesReceived,
          bytesSent,
          packetsReceived,
          packetsSent
        });
      } catch(e) {
        resolve({ hasPeerConnection: false, error: e.message });
      }
    });
  `);
  return stats;
}

async function testGroupCall() {
  console.log('=== Group Call E2E Test ===');
  console.log(`Base URL: ${BASE_URL}`);

  let driver1, driver2;

  try {
    console.log('\n1. Creating two browser instances...');
    driver1 = await createDriver();
    driver2 = await createDriver();

    console.log('2. Loading pages...');
    await driver1.get(BASE_URL + '/login');
    await driver2.get(BASE_URL + '/login');
    await driver1.sleep(2000);
    await driver2.sleep(2000);

    console.log('3. Registering users...');
    const user1 = await registerUser(driver1, 'user1');
    const user2 = await registerUser(driver2, 'user2');

    if (!user1.success || !user2.success) {
      console.error('User registration failed:', user1, user2);
      return;
    }

    console.log(`   User 1: ${user1.user.username} (${user1.user.id})`);
    console.log(`   User 2: ${user2.user.username} (${user2.user.id})`);

    console.log('4. Creating group chat...');
    const chatName = `GroupCall_${Date.now()}`;
    const chatResult = await createGroupChat(driver1, user1.token, chatName);

    if (!chatResult.success) {
      console.error('Chat creation failed:', chatResult);
      return;
    }

    const chatId = chatResult.chat.id;
    console.log(`   Chat created: ${chatName} (${chatId})`);

    console.log('5. Adding second user to chat...');
    const addResult = await addParticipant(driver1, user1.token, chatId, user2.user.id);
    console.log(`   Add participant result:`, addResult.success ? 'OK' : addResult.error);

    // Generate conference name from chat ID
    const conferenceName = chatId.replace(/-/g, '').substring(0, 16);
    console.log(`6. Conference name: conf_${conferenceName}`);

    console.log('7. User 1 joining conference...');
    const join1 = await joinConference(driver1, user1.token, conferenceName);
    console.log('   User 1 result:', JSON.stringify(join1, null, 2));

    console.log('8. User 2 joining conference...');
    const join2 = await joinConference(driver2, user2.token, conferenceName);
    console.log('   User 2 result:', JSON.stringify(join2, null, 2));

    // Wait for media to flow
    console.log('9. Waiting for media traffic...');
    await driver1.sleep(3000);

    console.log('10. Checking WebRTC stats...');
    const stats1 = await checkWebRTCStats(driver1);
    const stats2 = await checkWebRTCStats(driver2);

    console.log('\n=== Results ===');
    console.log('User 1 WebRTC Stats:', JSON.stringify(stats1, null, 2));
    console.log('User 2 WebRTC Stats:', JSON.stringify(stats2, null, 2));

    // Determine success
    const success = join1.success && join2.success &&
                    join1.dialogState === 'active' && join2.dialogState === 'active';

    console.log('\n=== Test Result ===');
    if (success) {
      console.log('✅ GROUP CALL TEST PASSED');
      console.log('   Both users joined the conference successfully');
    } else {
      console.log('❌ GROUP CALL TEST FAILED');
      console.log('   User 1 state:', join1.dialogState);
      console.log('   User 2 state:', join2.dialogState);
    }

  } catch (e) {
    console.error('Test error:', e);
  } finally {
    if (driver1) await driver1.quit();
    if (driver2) await driver2.quit();
  }
}

testGroupCall().catch(e => {
  console.error('Test failed:', e.message);
  process.exit(1);
});
