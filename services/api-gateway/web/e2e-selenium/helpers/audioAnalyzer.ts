import { WebDriver } from 'selenium-webdriver'

/**
 * Audio Analyzer Helper for E2E Tests
 *
 * Uses Web Audio API to analyze received audio stream:
 * - Detect dominant frequency (via FFT)
 * - Measure audio level
 * - Check for signal presence
 */

export interface AudioAnalysisResult {
  hasSignal: boolean
  dominantFrequency: number | null  // Hz
  audioLevel: number  // 0-1 normalized RMS
  peakLevel: number   // 0-1 peak amplitude
  frequencies: number[]  // Top 5 frequency peaks
  sampleRate: number
  error?: string
}

export interface AudioQualityMetrics {
  signalDetected: boolean
  expectedFrequency: number
  detectedFrequency: number | null
  frequencyError: number | null  // Hz difference
  frequencyErrorPercent: number | null  // % difference
  audioLevel: number
  qualityScore: number  // 0-100
  packetLoss: number  // from WebRTC stats
  jitter: number  // ms
  roundTripTime: number  // ms
}

/**
 * Inject audio analyzer into the browser and attach to incoming audio stream
 * Returns analysis results after specified duration
 */
export async function analyzeIncomingAudio(
  driver: WebDriver,
  durationMs: number = 3000
): Promise<AudioAnalysisResult> {
  const result = await driver.executeScript(`
    return new Promise(async (resolve) => {
      try {
        // Find RTCPeerConnection from exposed Verto dialog
        let pc = null;
        const dialog = window.__vertoActiveDialog;
        if (dialog && dialog.rtc && dialog.rtc.peerConnection) {
          pc = dialog.rtc.peerConnection;
        }

        if (!pc) {
          resolve({ error: 'No RTCPeerConnection found', hasSignal: false });
          return;
        }

        // Get incoming audio track
        const receivers = pc.getReceivers();
        const audioReceiver = receivers.find(r => r.track && r.track.kind === 'audio');

        if (!audioReceiver || !audioReceiver.track) {
          resolve({ error: 'No incoming audio track', hasSignal: false });
          return;
        }

        // Create audio context and analyzer
        const audioContext = new (window.AudioContext || window.webkitAudioContext)();
        const analyser = audioContext.createAnalyser();
        analyser.fftSize = 2048;
        analyser.smoothingTimeConstant = 0.8;

        // Create media stream from track
        const stream = new MediaStream([audioReceiver.track]);
        const source = audioContext.createMediaStreamSource(stream);
        source.connect(analyser);

        // Buffers for analysis
        const bufferLength = analyser.frequencyBinCount;
        const dataArray = new Uint8Array(bufferLength);
        const floatData = new Float32Array(bufferLength);

        // Collect samples over duration
        const samples = [];
        const startTime = Date.now();

        const collectSample = () => {
          analyser.getByteFrequencyData(dataArray);
          analyser.getFloatFrequencyData(floatData);

          // Calculate RMS level
          let sum = 0;
          let peak = 0;
          for (let i = 0; i < bufferLength; i++) {
            const value = dataArray[i] / 255;
            sum += value * value;
            if (value > peak) peak = value;
          }
          const rms = Math.sqrt(sum / bufferLength);

          // Find dominant frequency
          let maxIndex = 0;
          let maxValue = -Infinity;
          for (let i = 0; i < bufferLength; i++) {
            if (floatData[i] > maxValue) {
              maxValue = floatData[i];
              maxIndex = i;
            }
          }

          const frequency = maxIndex * audioContext.sampleRate / analyser.fftSize;

          samples.push({
            rms,
            peak,
            dominantFrequency: frequency,
            maxValue
          });
        };

        // Sample every 50ms
        const interval = setInterval(collectSample, 50);

        // Wait for duration
        setTimeout(() => {
          clearInterval(interval);

          // Aggregate results
          if (samples.length === 0) {
            resolve({ error: 'No samples collected', hasSignal: false });
            return;
          }

          // Average values
          const avgRms = samples.reduce((a, s) => a + s.rms, 0) / samples.length;
          const maxPeak = Math.max(...samples.map(s => s.peak));

          // Find most common dominant frequency (mode)
          const freqCounts = {};
          samples.forEach(s => {
            const freq = Math.round(s.dominantFrequency / 10) * 10; // Round to 10Hz
            freqCounts[freq] = (freqCounts[freq] || 0) + 1;
          });

          const sortedFreqs = Object.entries(freqCounts)
            .sort((a, b) => b[1] - a[1])
            .slice(0, 5)
            .map(([freq]) => parseInt(freq));

          const dominantFreq = sortedFreqs[0] || null;

          // Cleanup
          source.disconnect();
          audioContext.close();

          resolve({
            hasSignal: avgRms > 0.01,
            dominantFrequency: dominantFreq,
            audioLevel: avgRms,
            peakLevel: maxPeak,
            frequencies: sortedFreqs,
            sampleRate: audioContext.sampleRate,
            samplesCollected: samples.length
          });
        }, ${durationMs});

      } catch (e) {
        resolve({ error: e.message, hasSignal: false });
      }
    });
  `) as AudioAnalysisResult

  return result
}

/**
 * Generate a test tone (sine wave) in the browser
 * Useful for testing audio output
 */
export async function playTestTone(
  driver: WebDriver,
  frequency: number = 1000,
  durationMs: number = 2000,
  volume: number = 0.3
): Promise<void> {
  await driver.executeScript(`
    return new Promise((resolve) => {
      const audioContext = new (window.AudioContext || window.webkitAudioContext)();
      const oscillator = audioContext.createOscillator();
      const gainNode = audioContext.createGain();

      oscillator.type = 'sine';
      oscillator.frequency.setValueAtTime(${frequency}, audioContext.currentTime);
      gainNode.gain.setValueAtTime(${volume}, audioContext.currentTime);

      oscillator.connect(gainNode);
      gainNode.connect(audioContext.destination);

      oscillator.start();

      setTimeout(() => {
        oscillator.stop();
        audioContext.close();
        resolve();
      }, ${durationMs});
    });
  `)
}

/**
 * Get comprehensive audio quality metrics
 * Combines WebRTC stats with audio analysis
 */
export async function getAudioQualityMetrics(
  driver: WebDriver,
  expectedFrequency: number = 1000,
  analysisDuration: number = 3000
): Promise<AudioQualityMetrics> {
  // Get WebRTC stats first
  const webrtcStats = await driver.executeScript(`
    return new Promise(async (resolve) => {
      try {
        let pc = null;
        const dialog = window.__vertoActiveDialog;
        if (dialog && dialog.rtc && dialog.rtc.peerConnection) {
          pc = dialog.rtc.peerConnection;
        }

        if (!pc) {
          resolve({ error: 'No RTCPeerConnection' });
          return;
        }

        const stats = await pc.getStats();
        const result = {
          packetLoss: 0,
          packetsReceived: 0,
          packetsLost: 0,
          jitter: 0,
          roundTripTime: 0
        };

        stats.forEach(report => {
          if (report.type === 'inbound-rtp' && report.kind === 'audio') {
            result.packetsReceived = report.packetsReceived || 0;
            result.packetsLost = report.packetsLost || 0;
            result.jitter = (report.jitter || 0) * 1000; // Convert to ms
          }
          if (report.type === 'candidate-pair' && report.state === 'succeeded') {
            result.roundTripTime = report.currentRoundTripTime * 1000 || 0; // Convert to ms
          }
        });

        if (result.packetsReceived > 0) {
          result.packetLoss = (result.packetsLost / (result.packetsReceived + result.packetsLost)) * 100;
        }

        resolve(result);
      } catch (e) {
        resolve({ error: e.message });
      }
    });
  `) as { packetLoss: number; jitter: number; roundTripTime: number; error?: string }

  // Get audio analysis
  const audioAnalysis = await analyzeIncomingAudio(driver, analysisDuration)

  // Calculate quality score
  let qualityScore = 100

  // Deduct for packet loss (each 1% = -10 points)
  if (webrtcStats.packetLoss) {
    qualityScore -= webrtcStats.packetLoss * 10
  }

  // Deduct for high jitter (>30ms starts deducting)
  if (webrtcStats.jitter > 30) {
    qualityScore -= Math.min(30, (webrtcStats.jitter - 30))
  }

  // Deduct for frequency mismatch
  let frequencyError: number | null = null
  let frequencyErrorPercent: number | null = null

  if (audioAnalysis.dominantFrequency && expectedFrequency) {
    frequencyError = Math.abs(audioAnalysis.dominantFrequency - expectedFrequency)
    frequencyErrorPercent = (frequencyError / expectedFrequency) * 100

    // Deduct if frequency is off by more than 5%
    if (frequencyErrorPercent > 5) {
      qualityScore -= Math.min(20, frequencyErrorPercent)
    }
  }

  // Deduct if no signal detected
  if (!audioAnalysis.hasSignal) {
    qualityScore -= 50
  }

  qualityScore = Math.max(0, Math.min(100, qualityScore))

  return {
    signalDetected: audioAnalysis.hasSignal,
    expectedFrequency,
    detectedFrequency: audioAnalysis.dominantFrequency,
    frequencyError,
    frequencyErrorPercent,
    audioLevel: audioAnalysis.audioLevel,
    qualityScore,
    packetLoss: webrtcStats.packetLoss || 0,
    jitter: webrtcStats.jitter || 0,
    roundTripTime: webrtcStats.roundTripTime || 0
  }
}

/**
 * Wait for audio signal to be detected
 */
export async function waitForAudioSignal(
  driver: WebDriver,
  timeout: number = 10000,
  minLevel: number = 0.01
): Promise<boolean> {
  const start = Date.now()

  while (Date.now() - start < timeout) {
    const analysis = await analyzeIncomingAudio(driver, 500)

    if (analysis.hasSignal && analysis.audioLevel >= minLevel) {
      return true
    }

    await new Promise(r => setTimeout(r, 500))
  }

  return false
}

/**
 * Verify that a specific frequency is being received
 * Useful for end-to-end audio verification
 */
export async function verifyFrequencyReceived(
  driver: WebDriver,
  expectedFrequency: number,
  toleranceHz: number = 50,
  analysisDuration: number = 3000
): Promise<{ verified: boolean; detectedFrequency: number | null; error: number | null }> {
  const analysis = await analyzeIncomingAudio(driver, analysisDuration)

  if (!analysis.hasSignal || !analysis.dominantFrequency) {
    return { verified: false, detectedFrequency: null, error: null }
  }

  const error = Math.abs(analysis.dominantFrequency - expectedFrequency)
  const verified = error <= toleranceHz

  return {
    verified,
    detectedFrequency: analysis.dominantFrequency,
    error
  }
}
