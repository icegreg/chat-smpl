import { Builder, Browser } from 'selenium-webdriver';
import chrome from 'selenium-webdriver/chrome.js';

async function testVertoCall() {
  const options = new chrome.Options();
  options.addArguments(
    '--headless=new',
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
    console.log('Step 1: Loading page...');
    await driver.get('http://127.0.0.1:8888/login');
    await driver.sleep(2000);

    console.log('Step 2: Registering and logging in...');
    const username = 'verto_test_' + Date.now();
    const loginResult = await driver.executeScript(`
      return new Promise(async (resolve) => {
        try {
          // First register
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
            resolve({ success: true, user: data.user });
          } else {
            resolve({ success: false, error: 'No token received', data: data });
          }
        } catch(e) {
          resolve({ success: false, error: e.message });
        }
      });
    `);
    console.log('Login result:', JSON.stringify(loginResult, null, 2));

    if (!loginResult.success) {
      console.log('Login failed, exiting...');
      return;
    }

    console.log('Step 3: Starting chat call with WebSocket monitoring...');

    // Intercept WebSocket messages and start call
    const callResult = await driver.executeScript(`
      return new Promise(async (resolve) => {
        // Store WebSocket messages
        window.__wsMessages = [];
        const originalWS = WebSocket;
        WebSocket = function(url) {
          const ws = new originalWS(url);
          const originalSend = ws.send.bind(ws);
          ws.send = function(data) {
            window.__wsMessages.push({ type: 'send', data: data, time: Date.now() });
            console.log('[WS SEND]', data);
            return originalSend(data);
          };
          ws.addEventListener('message', function(event) {
            window.__wsMessages.push({ type: 'recv', data: event.data, time: Date.now() });
          });
          return ws;
        };
        WebSocket.prototype = originalWS.prototype;

        try {
          // Get Verto credentials
          const token = localStorage.getItem('access_token');
          const credResponse = await fetch('/api/voice/credentials', {
            headers: { 'Authorization': 'Bearer ' + token }
          });
          const creds = await credResponse.json();
          console.log('Credentials:', JSON.stringify(creds));

          // Wait for Verto to load
          let attempts = 0;
          while (!window.Verto && attempts < 20) {
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
              console.log('[Verto] Login success:', success);
              if (success) {
                // Make test call after login
                setTimeout(function() {
                  try {
                    console.log('[Verto] Making call to 9196 (echo test)...');
                    const dialog = v.newCall({
                      destination_number: '9196',
                      caller_id_name: 'Test User',
                      useVideo: false,
                      tag: 'verto-audio'
                    });
                    console.log('[Verto] Dialog created:', dialog ? 'yes' : 'no');
                    console.log('[Verto] Dialog callID:', dialog?.callID);

                    // Wait and collect results
                    setTimeout(function() {
                      resolve({
                        success: true,
                        loginSuccess: success,
                        dialogCreated: !!dialog,
                        dialogCallID: dialog?.callID,
                        dialogState: dialog?.state?.name,
                        hasPeerConnection: !!(dialog?.rtc?.peerConnection),
                        peerConnectionState: dialog?.rtc?.peerConnection?.connectionState,
                        wsMessages: window.__wsMessages
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
              console.log('[Verto] Dialog state changed:', dialog.state.name);
            },
            onWSClose: function() {
              console.log('[Verto] WebSocket closed');
            }
          });

          verto.login();

        } catch(e) {
          resolve({ success: false, error: e.message });
        }
      });
    `);

    console.log('\\nCall result:');
    console.log(JSON.stringify(callResult, null, 2));

    if (callResult.wsMessages) {
      console.log('\\n=== WebSocket Messages ===');
      for (const msg of callResult.wsMessages) {
        console.log(`[${msg.type}]`, msg.data);
      }
    }

  } finally {
    await driver.quit();
  }
}

testVertoCall().catch(e => {
  console.error('Test failed:', e.message);
  process.exit(1);
});
