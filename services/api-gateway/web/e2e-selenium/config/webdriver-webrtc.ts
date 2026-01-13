import { Builder, WebDriver, Browser, logging } from 'selenium-webdriver'
import chrome from 'selenium-webdriver/chrome.js'
import firefox from 'selenium-webdriver/firefox.js'

const BASE_URL = process.env.BASE_URL || 'http://localhost:3000'
// WebRTC tests require non-headless mode for media access
const HEADLESS = process.env.HEADLESS === 'true'

// Extract host for insecure origin treatment (for external addresses)
function getOriginFromUrl(url: string): string {
  try {
    const parsed = new URL(url)
    return `${parsed.protocol}//${parsed.host}`
  } catch {
    return url
  }
}

export { BASE_URL }

/**
 * Create a Chrome driver configured for WebRTC testing
 *
 * Key features:
 * - Uses fake media devices to avoid requiring real microphone/camera
 * - Non-headless by default (WebRTC works better with visible browser)
 * - Ignores SSL certificate errors (for self-signed certs on FreeSWITCH)
 * - Allows autoplay without user gesture
 */
export async function createWebRTCDriver(): Promise<WebDriver> {
  const options = new chrome.Options()

  if (HEADLESS) {
    options.addArguments('--headless=new')
  }

  // Get origin for insecure origin treatment (needed for external IP addresses)
  const origin = getOriginFromUrl(BASE_URL)

  options.addArguments(
    '--no-sandbox',
    '--disable-dev-shm-usage',
    '--window-size=1920,1080',
    '--disable-extensions',
    '--disable-infobars',

    // WebRTC-specific flags
    '--use-fake-device-for-media-stream',      // Use fake audio/video devices
    '--use-fake-ui-for-media-stream',          // Auto-accept media permissions
    '--autoplay-policy=no-user-gesture-required', // Allow audio/video autoplay
    '--enable-features=WebRtcHideLocalIpsWithMdns', // Enable WebRTC features
    '--disable-features=WebRtcHWDecoding,WebRtcHWEncoding', // Disable HW encoding for fake devices

    // SSL/Security flags for self-signed certs
    '--ignore-certificate-errors',
    '--ignore-ssl-errors',
    '--allow-insecure-localhost',
    '--allow-running-insecure-content',

    // Allow getUserMedia on insecure origins (for external IP addresses like http://10.x.x.x)
    `--unsafely-treat-insecure-origin-as-secure=${origin}`,

    // Disable GPU for stability with fake devices
    '--disable-gpu',
    '--disable-software-rasterizer',

    // Enable WebRTC logging
    '--enable-logging',
    '--v=1',

    // Force audio for fake device
    '--enable-audio-focus-ducking=0'
  )

  // Enable browser logging
  const prefs = new logging.Preferences()
  prefs.setLevel(logging.Type.BROWSER, logging.Level.ALL)
  prefs.setLevel(logging.Type.PERFORMANCE, logging.Level.ALL)
  options.setLoggingPrefs(prefs)

  // Enable performance logging for network analysis
  options.setUserPreferences({
    'profile.default_content_setting_values.media_stream_mic': 1,
    'profile.default_content_setting_values.media_stream_camera': 1,
    'profile.default_content_setting_values.geolocation': 1,
    'profile.default_content_setting_values.notifications': 1,
  })

  const driver = await new Builder()
    .forBrowser(Browser.CHROME)
    .setChromeOptions(options)
    .build()

  // Set longer timeouts for WebRTC operations
  await driver.manage().setTimeouts({
    implicit: 10000,
    pageLoad: 60000,
    script: 60000,
  })

  return driver
}

/**
 * Create a Firefox driver configured for WebRTC testing
 *
 * Key features:
 * - Uses fake media devices to avoid requiring real microphone/camera
 * - Non-headless by default (WebRTC works better with visible browser)
 * - Allows autoplay without user gesture
 */
export async function createFirefoxWebRTCDriver(): Promise<WebDriver> {
  const options = new firefox.Options()

  if (HEADLESS) {
    options.addArguments('-headless')
  }

  options.addArguments(
    '--width=1920',
    '--height=1080'
  )

  // Firefox preferences for WebRTC
  options.setPreference('media.navigator.streams.fake', true)  // Use fake media streams
  options.setPreference('media.navigator.permission.disabled', true)  // Auto-accept media permissions
  options.setPreference('media.autoplay.default', 0)  // Allow autoplay
  options.setPreference('media.autoplay.blocking_policy', 0)
  options.setPreference('media.autoplay.allow-extension-background-pages', true)
  options.setPreference('media.autoplay.block-webaudio', false)

  // SSL/Security preferences for self-signed certs
  options.setPreference('webdriver.acceptInsecureCerts', true)

  // WebRTC logging
  options.setPreference('media.peerconnection.sdp.parser', 'sipcc')  // Use SIPCC SDP parser
  options.setPreference('media.peerconnection.enabled', true)
  options.setPreference('media.navigator.video.enabled', true)
  options.setPreference('media.navigator.audio.enabled', true)

  // Enable browser console logging
  const prefs = new logging.Preferences()
  prefs.setLevel(logging.Type.BROWSER, logging.Level.ALL)
  options.setLoggingPrefs(prefs)

  const driver = await new Builder()
    .forBrowser(Browser.FIREFOX)
    .setFirefoxOptions(options)
    .build()

  // Set longer timeouts for WebRTC operations
  await driver.manage().setTimeouts({
    implicit: 10000,
    pageLoad: 60000,
    script: 60000,
  })

  return driver
}

export async function quitDriver(driver: WebDriver): Promise<void> {
  if (driver) {
    await driver.quit()
  }
}

/**
 * Helper to check WebRTC connection state
 */
export interface WebRTCStats {
  connectionState: string
  iceConnectionState: string
  bytesReceived: number
  bytesSent: number
  packetsReceived: number
  packetsSent: number
  audioLevel?: number
  hasActiveStream: boolean
}

/**
 * Execute script to get WebRTC statistics from the browser
 */
export async function getWebRTCStats(driver: WebDriver): Promise<WebRTCStats | null> {
  try {
    const result = await driver.executeScript(`
      return new Promise(async (resolve) => {
        // Find RTCPeerConnection
        const pcs = window.__webrtcPeerConnections || [];
        if (pcs.length === 0) {
          // Try to find from exposed Verto dialog
          const dialog = window.__vertoActiveDialog;
          if (dialog && dialog.rtc && dialog.rtc.peerConnection) {
            pcs.push(dialog.rtc.peerConnection);
          }
        }

        if (pcs.length === 0) {
          resolve(null);
          return;
        }

        const pc = pcs[0];
        const stats = {
          connectionState: pc.connectionState,
          iceConnectionState: pc.iceConnectionState,
          bytesReceived: 0,
          bytesSent: 0,
          packetsReceived: 0,
          packetsSent: 0,
          audioLevel: null,
          hasActiveStream: false
        };

        // Check if there are active streams
        const receivers = pc.getReceivers();
        const senders = pc.getSenders();
        stats.hasActiveStream = receivers.some(r => r.track && r.track.readyState === 'live') ||
                                senders.some(s => s.track && s.track.readyState === 'live');

        // Get detailed stats
        try {
          const rtcStats = await pc.getStats();
          rtcStats.forEach(report => {
            if (report.type === 'inbound-rtp') {
              stats.bytesReceived += report.bytesReceived || 0;
              stats.packetsReceived += report.packetsReceived || 0;
              if (report.audioLevel !== undefined) {
                stats.audioLevel = report.audioLevel;
              }
            }
            if (report.type === 'outbound-rtp') {
              stats.bytesSent += report.bytesSent || 0;
              stats.packetsSent += report.packetsSent || 0;
            }
          });
        } catch (e) {
          console.warn('Error getting RTC stats:', e);
        }

        resolve(stats);
      });
    `) as WebRTCStats | null

    return result
  } catch (e) {
    console.error('Error getting WebRTC stats:', e)
    return null
  }
}

/**
 * Wait for WebRTC connection to establish
 */
export async function waitForWebRTCConnection(
  driver: WebDriver,
  timeout: number = 30000
): Promise<boolean> {
  const start = Date.now()

  while (Date.now() - start < timeout) {
    const stats = await getWebRTCStats(driver)
    if (stats && stats.connectionState === 'connected') {
      return true
    }
    await new Promise(r => setTimeout(r, 500))
  }

  return false
}

/**
 * Wait for media traffic to flow
 */
export async function waitForMediaTraffic(
  driver: WebDriver,
  minBytes: number = 1000,
  timeout: number = 30000
): Promise<boolean> {
  const start = Date.now()

  while (Date.now() - start < timeout) {
    const stats = await getWebRTCStats(driver)
    if (stats && (stats.bytesReceived >= minBytes || stats.bytesSent >= minBytes)) {
      return true
    }
    await new Promise(r => setTimeout(r, 500))
  }

  return false
}

/**
 * Create a Chrome driver with fake audio from a specific WAV file
 * This is useful for testing with known audio content
 */
export async function createWebRTCDriverWithAudioFile(audioFilePath: string): Promise<WebDriver> {
  const options = new chrome.Options()

  if (HEADLESS) {
    options.addArguments('--headless=new')
  }

  options.addArguments(
    '--no-sandbox',
    '--disable-dev-shm-usage',
    '--window-size=1920,1080',
    '--disable-extensions',
    '--disable-infobars',

    // WebRTC-specific flags with audio file
    '--use-fake-device-for-media-stream',
    '--use-fake-ui-for-media-stream',
    `--use-file-for-fake-audio-capture=${audioFilePath}`,
    '--autoplay-policy=no-user-gesture-required',

    // SSL/Security flags
    '--ignore-certificate-errors',
    '--ignore-ssl-errors',
    '--allow-insecure-localhost',

    '--disable-gpu',
    '--enable-logging',
    '--v=1'
  )

  const prefs = new logging.Preferences()
  prefs.setLevel(logging.Type.BROWSER, logging.Level.ALL)
  options.setLoggingPrefs(prefs)

  options.setUserPreferences({
    'profile.default_content_setting_values.media_stream_mic': 1,
    'profile.default_content_setting_values.media_stream_camera': 1,
  })

  const driver = await new Builder()
    .forBrowser(Browser.CHROME)
    .setChromeOptions(options)
    .build()

  await driver.manage().setTimeouts({
    implicit: 10000,
    pageLoad: 60000,
    script: 60000,
  })

  return driver
}

/**
 * Get detailed audio statistics from WebRTC connection
 */
export interface DetailedAudioStats {
  connectionState: string
  iceConnectionState: string
  inboundAudio: {
    packetsReceived: number
    bytesReceived: number
    packetsLost: number
    jitter: number
    audioLevel?: number
    totalAudioEnergy?: number
    codec?: string
  } | null
  outboundAudio: {
    packetsSent: number
    bytesSent: number
    codec?: string
  } | null
  hasActiveAudioTrack: boolean
}

export async function getDetailedAudioStats(driver: WebDriver): Promise<DetailedAudioStats | null> {
  try {
    const result = await driver.executeScript(`
      return new Promise(async (resolve) => {
        // Find RTCPeerConnection from exposed Verto dialog
        let pc = null;
        const dialog = window.__vertoActiveDialog;
        if (dialog && dialog.rtc && dialog.rtc.peerConnection) {
          pc = dialog.rtc.peerConnection;
        }

        if (!pc) {
          resolve(null);
          return;
        }

        const result = {
          connectionState: pc.connectionState,
          iceConnectionState: pc.iceConnectionState,
          inboundAudio: null,
          outboundAudio: null,
          hasActiveAudioTrack: false
        };

        // Check for active audio tracks
        const receivers = pc.getReceivers();
        const senders = pc.getSenders();

        result.hasActiveAudioTrack =
          receivers.some(r => r.track && r.track.kind === 'audio' && r.track.readyState === 'live') ||
          senders.some(s => s.track && s.track.kind === 'audio' && s.track.readyState === 'live');

        try {
          const stats = await pc.getStats();

          stats.forEach(report => {
            if (report.type === 'inbound-rtp' && report.kind === 'audio') {
              result.inboundAudio = {
                packetsReceived: report.packetsReceived || 0,
                bytesReceived: report.bytesReceived || 0,
                packetsLost: report.packetsLost || 0,
                jitter: report.jitter || 0,
                audioLevel: report.audioLevel,
                totalAudioEnergy: report.totalAudioEnergy,
                codec: report.mimeType
              };
            }

            if (report.type === 'outbound-rtp' && report.kind === 'audio') {
              result.outboundAudio = {
                packetsSent: report.packetsSent || 0,
                bytesSent: report.bytesSent || 0,
                codec: report.mimeType
              };
            }
          });
        } catch (e) {
          console.warn('Error getting stats:', e);
        }

        resolve(result);
      });
    `) as DetailedAudioStats | null

    return result
  } catch (e) {
    console.error('Error getting detailed audio stats:', e)
    return null
  }
}

/**
 * Monitor audio levels over time
 * Returns array of audio level samples
 */
export async function monitorAudioLevels(
  driver: WebDriver,
  durationMs: number = 5000,
  sampleIntervalMs: number = 200
): Promise<number[]> {
  const samples: number[] = []
  const start = Date.now()

  while (Date.now() - start < durationMs) {
    const stats = await getDetailedAudioStats(driver)
    if (stats?.inboundAudio?.audioLevel !== undefined) {
      samples.push(stats.inboundAudio.audioLevel)
    }
    await new Promise(r => setTimeout(r, sampleIntervalMs))
  }

  return samples
}

/**
 * Check if audio is being transmitted (bytes increasing over time)
 */
export async function isAudioTransmitting(
  driver: WebDriver,
  checkDurationMs: number = 2000
): Promise<{ sending: boolean; receiving: boolean }> {
  const initialStats = await getDetailedAudioStats(driver)

  if (!initialStats) {
    return { sending: false, receiving: false }
  }

  await new Promise(r => setTimeout(r, checkDurationMs))

  const finalStats = await getDetailedAudioStats(driver)

  if (!finalStats) {
    return { sending: false, receiving: false }
  }

  const initialSent = initialStats.outboundAudio?.bytesSent || 0
  const finalSent = finalStats.outboundAudio?.bytesSent || 0
  const initialReceived = initialStats.inboundAudio?.bytesReceived || 0
  const finalReceived = finalStats.inboundAudio?.bytesReceived || 0

  return {
    sending: finalSent > initialSent,
    receiving: finalReceived > initialReceived
  }
}
