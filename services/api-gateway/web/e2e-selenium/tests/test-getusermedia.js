import { Builder, Browser } from 'selenium-webdriver';
import chrome from 'selenium-webdriver/chrome.js';

async function testGetUserMedia() {
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
    console.log('Navigating to page...');
    await driver.get('http://127.0.0.1:8888');

    console.log('Testing getUserMedia...');
    const result = await driver.executeScript(`
      return new Promise(async (resolve) => {
        try {
          const stream = await navigator.mediaDevices.getUserMedia({ audio: true });
          const tracks = stream.getAudioTracks();
          resolve({
            success: true,
            trackCount: tracks.length,
            trackLabel: tracks[0]?.label,
            trackSettings: tracks[0]?.getSettings()
          });
        } catch(e) {
          resolve({ success: false, error: e.message, name: e.name });
        }
      });
    `);

    console.log('getUserMedia result:', JSON.stringify(result, null, 2));

    if (result.success) {
      console.log('Testing RTCPeerConnection...');
      const pcResult = await driver.executeScript(`
        try {
          const pc = new RTCPeerConnection();
          const state = pc.connectionState;
          pc.close();
          return { success: true, state: state };
        } catch(e) {
          return { success: false, error: e.message };
        }
      `);
      console.log('RTCPeerConnection result:', JSON.stringify(pcResult, null, 2));
    }

  } finally {
    await driver.quit();
  }
}

testGetUserMedia().catch(e => {
  console.error('Test failed:', e.message);
  process.exit(1);
});
