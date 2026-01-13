import * as fs from 'fs'
import * as path from 'path'

/**
 * Generate a WAV file with a pure sine wave tone
 * This file can be used with Chrome's --use-file-for-fake-audio-capture flag
 * for controlled audio testing
 */

interface WavGeneratorOptions {
  frequency: number      // Hz
  duration: number       // seconds
  sampleRate: number     // samples per second
  amplitude: number      // 0.0 - 1.0
}

const DEFAULT_OPTIONS: WavGeneratorOptions = {
  frequency: 1000,       // 1kHz
  duration: 60,          // 60 seconds (enough for test duration)
  sampleRate: 48000,     // 48kHz (standard for WebRTC)
  amplitude: 0.8         // 80% volume
}

/**
 * Generate a WAV file buffer with a sine wave
 */
function generateSineWaveWav(options: Partial<WavGeneratorOptions> = {}): Buffer {
  const opts = { ...DEFAULT_OPTIONS, ...options }
  const { frequency, duration, sampleRate, amplitude } = opts

  const numSamples = Math.floor(sampleRate * duration)
  const numChannels = 1  // Mono
  const bitsPerSample = 16
  const bytesPerSample = bitsPerSample / 8
  const blockAlign = numChannels * bytesPerSample
  const byteRate = sampleRate * blockAlign
  const dataSize = numSamples * blockAlign

  // WAV file structure
  const buffer = Buffer.alloc(44 + dataSize)

  // RIFF header
  buffer.write('RIFF', 0)
  buffer.writeUInt32LE(36 + dataSize, 4)  // File size - 8
  buffer.write('WAVE', 8)

  // fmt chunk
  buffer.write('fmt ', 12)
  buffer.writeUInt32LE(16, 16)             // Chunk size
  buffer.writeUInt16LE(1, 20)              // Audio format (1 = PCM)
  buffer.writeUInt16LE(numChannels, 22)    // Number of channels
  buffer.writeUInt32LE(sampleRate, 24)     // Sample rate
  buffer.writeUInt32LE(byteRate, 28)       // Byte rate
  buffer.writeUInt16LE(blockAlign, 30)     // Block align
  buffer.writeUInt16LE(bitsPerSample, 32)  // Bits per sample

  // data chunk
  buffer.write('data', 36)
  buffer.writeUInt32LE(dataSize, 40)

  // Generate sine wave samples
  const maxAmplitude = 32767 * amplitude  // Max for 16-bit signed
  let offset = 44

  for (let i = 0; i < numSamples; i++) {
    const t = i / sampleRate
    const sample = Math.sin(2 * Math.PI * frequency * t)
    const value = Math.round(sample * maxAmplitude)
    buffer.writeInt16LE(value, offset)
    offset += 2
  }

  return buffer
}

/**
 * Save a test audio file with specified frequency
 */
export function saveTestAudioFile(
  outputPath: string,
  frequency: number = 1000,
  duration: number = 60
): void {
  const wavBuffer = generateSineWaveWav({ frequency, duration })
  fs.writeFileSync(outputPath, wavBuffer)
  console.log(`Generated ${frequency}Hz test audio: ${outputPath}`)
  console.log(`  Duration: ${duration}s, Size: ${wavBuffer.length} bytes`)
}

/**
 * Get the path to the test audio file, creating it if necessary
 */
export function getTestAudioPath(frequency: number = 1000): string {
  const testDataDir = path.join(__dirname, '..', 'test-data')

  // Create test-data directory if it doesn't exist
  if (!fs.existsSync(testDataDir)) {
    fs.mkdirSync(testDataDir, { recursive: true })
  }

  const audioPath = path.join(testDataDir, `test-tone-${frequency}hz.wav`)

  // Generate file if it doesn't exist
  if (!fs.existsSync(audioPath)) {
    saveTestAudioFile(audioPath, frequency, 60)
  }

  return audioPath
}

/**
 * Pre-defined test frequencies
 */
export const TEST_FREQUENCIES = {
  LOW: 440,        // A4 note - Chrome's default fake audio
  MEDIUM: 1000,    // 1kHz - Common test tone
  HIGH: 2000,      // 2kHz - Higher frequency test
  SPEECH: 300,     // 300Hz - Typical speech fundamental
  DTMF_1: 697,     // DTMF digit 1 low frequency
  DTMF_2: 770,     // DTMF digit 4 low frequency
}

// CLI entry point
if (require.main === module) {
  const args = process.argv.slice(2)
  const frequency = parseInt(args[0]) || 1000
  const duration = parseInt(args[1]) || 60
  const outputPath = args[2] || `./test-tone-${frequency}hz.wav`

  console.log('Generating test audio file...')
  saveTestAudioFile(outputPath, frequency, duration)
  console.log('Done!')
}
