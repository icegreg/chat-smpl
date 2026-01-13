import { WebDriver } from 'selenium-webdriver'

/**
 * Patches getUserMedia to use simpler constraints that work with Chrome's fake audio device.
 *
 * Chrome's fake audio device (--use-fake-device-for-media-stream) doesn't support
 * all constraint options that real devices do. This patcher intercepts getUserMedia
 * calls and simplifies constraints to avoid OverconstrainedError.
 *
 * Must be called BEFORE the page loads any WebRTC code (like Verto).
 */
export async function patchGetUserMediaForFakeDevice(driver: WebDriver): Promise<void> {
  await driver.executeScript(`
    // Save original getUserMedia
    const originalGetUserMedia = navigator.mediaDevices.getUserMedia.bind(navigator.mediaDevices);

    // Override getUserMedia with relaxed constraints for fake devices
    navigator.mediaDevices.getUserMedia = async function(constraints) {
      console.log('[MediaPatcher] Original constraints:', JSON.stringify(constraints));

      // Simplify audio constraints for fake device
      if (constraints && constraints.audio) {
        if (typeof constraints.audio === 'object') {
          // Keep only basic audio constraints that fake device supports
          constraints.audio = {
            echoCancellation: false,
            noiseSuppression: false,
            autoGainControl: false
          };
        } else {
          constraints.audio = true;
        }
      }

      // Disable video for audio-only tests
      if (constraints && constraints.video === undefined) {
        constraints.video = false;
      }

      console.log('[MediaPatcher] Simplified constraints:', JSON.stringify(constraints));

      try {
        const stream = await originalGetUserMedia(constraints);
        console.log('[MediaPatcher] Got stream with tracks:', stream.getTracks().map(t => t.kind + ':' + t.label).join(', '));
        return stream;
      } catch (e) {
        console.error('[MediaPatcher] getUserMedia failed:', e.message);

        // Last resort: try with just { audio: true }
        if (constraints.audio) {
          console.log('[MediaPatcher] Retrying with minimal constraints');
          try {
            return await originalGetUserMedia({ audio: true, video: false });
          } catch (e2) {
            console.error('[MediaPatcher] Minimal constraints also failed:', e2.message);
            throw e2;
          }
        }
        throw e;
      }
    };

    console.log('[MediaPatcher] getUserMedia patched for fake device compatibility');
  `)
}

/**
 * Injects the patcher script that will run on every page navigation.
 * This creates a script element that persists across navigations.
 */
export async function injectMediaPatcherScript(driver: WebDriver): Promise<void> {
  await driver.executeScript(`
    // Create a script that runs on DOMContentLoaded
    const patcherScript = document.createElement('script');
    patcherScript.id = 'media-patcher-script';
    patcherScript.textContent = \`
      (function() {
        // Only run once
        if (window.__mediaPatched) return;
        window.__mediaPatched = true;

        const originalGetUserMedia = navigator.mediaDevices.getUserMedia.bind(navigator.mediaDevices);

        navigator.mediaDevices.getUserMedia = async function(constraints) {
          // Simplify audio constraints
          if (constraints && constraints.audio && typeof constraints.audio === 'object') {
            constraints.audio = {
              echoCancellation: false,
              noiseSuppression: false,
              autoGainControl: false
            };
          }

          try {
            return await originalGetUserMedia(constraints);
          } catch (e) {
            console.warn('[MediaPatcher] getUserMedia failed, trying minimal constraints');
            if (constraints.audio) {
              return await originalGetUserMedia({ audio: true, video: false });
            }
            throw e;
          }
        };

        console.log('[MediaPatcher] Installed via script tag');
      })();
    \`;

    document.head.insertBefore(patcherScript, document.head.firstChild);
  `)
}

/**
 * Verify that the media patcher is working by checking if we can get a stream.
 */
export async function verifyFakeAudioDevice(driver: WebDriver): Promise<boolean> {
  const result = await driver.executeScript(`
    return new Promise(async (resolve) => {
      try {
        const stream = await navigator.mediaDevices.getUserMedia({ audio: true, video: false });
        const tracks = stream.getTracks();
        const audioTrack = tracks.find(t => t.kind === 'audio');

        // Cleanup
        tracks.forEach(t => t.stop());

        if (audioTrack) {
          console.log('[MediaVerify] Fake audio device working:', audioTrack.label);
          resolve({ success: true, label: audioTrack.label });
        } else {
          resolve({ success: false, error: 'No audio track' });
        }
      } catch (e) {
        resolve({ success: false, error: e.message });
      }
    });
  `) as { success: boolean; label?: string; error?: string }

  if (result.success) {
    console.log('[MediaVerify] Fake audio device verified:', result.label)
  } else {
    console.error('[MediaVerify] Fake audio device failed:', result.error)
  }

  return result.success
}
