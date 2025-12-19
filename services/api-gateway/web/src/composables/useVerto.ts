import { ref, computed, onUnmounted } from 'vue'
import type { VertoCredentials, IceServer } from '@/types'

// Verto.js is loaded via script tag in index.html
declare global {
  interface Window {
    jQuery: unknown
    $: unknown
    Verto: VertoClass
  }
}

interface VertoClass {
  new (params: VertoParams, callbacks: VertoCallbacks): VertoInstance
}

interface VertoParams {
  login: string
  passwd: string
  socketUrl: string
  ringFile?: string
  iceServers?: RTCIceServer[]
  tag?: string
  deviceParams?: {
    useMic?: string | boolean
    useSpeak?: string | boolean
    useCamera?: boolean
  }
}

interface VertoCallbacks {
  onWSLogin?: (verto: VertoInstance, success: boolean) => void
  onWSClose?: (verto: VertoInstance, success: boolean) => void
  onDialogState?: (dialog: VertoDialog) => void
  onMessage?: (verto: VertoInstance, dialog: VertoDialog | null, msg: unknown, data: unknown) => void
}

interface VertoInstance {
  login(): void
  logout(): void
  hangup(callId?: string): void
  newCall(params: VertoCallParams): VertoDialog
  answer(callId: string, params?: VertoAnswerParams): void
  dtmf(digit: string): void
  processReply(method: string, success: boolean, data: unknown): void
  purge(): void
  rpcClient: unknown
  options: VertoParams
}

interface VertoCallParams {
  destination_number: string
  caller_id_name?: string
  caller_id_number?: string
  useVideo?: boolean
  useStereo?: boolean
  useMic?: string | boolean
  useSpeak?: string | boolean
  tag?: string
  localTag?: string
  remoteTag?: string
}

interface VertoAnswerParams {
  useVideo?: boolean
  useStereo?: boolean
  tag?: string
}

interface VertoDialog {
  callID: string
  direction: 'inbound' | 'outbound'
  state: VertoDialogState
  cause?: string
  causeCode?: number
  callerIdName?: string
  callerIdNumber?: string
  destinationNumber?: string
  hangup(params?: { cause?: string }): void
  answer(params?: VertoAnswerParams): void
  dtmf(digit: string): void
  toggleMute(what?: 'mic' | 'speaker'): boolean
  setMute(what: 'mic' | 'speaker', mute: boolean): boolean
  getMute(what: 'mic' | 'speaker'): boolean
}

type VertoDialogState =
  | 'new'
  | 'requesting'
  | 'trying'
  | 'recovering'
  | 'ringing'
  | 'answering'
  | 'early'
  | 'active'
  | 'held'
  | 'hangup'
  | 'destroy'
  | 'purge'

export interface CallState {
  callId: string
  direction: 'inbound' | 'outbound'
  state: VertoDialogState
  remoteName?: string
  remoteNumber?: string
  isMuted: boolean
  startTime?: number
}

export function useVerto() {
  const verto = ref<VertoInstance | null>(null)
  const isConnected = ref(false)
  const isConnecting = ref(false)
  const currentCall = ref<CallState | null>(null)
  const incomingCall = ref<CallState | null>(null)
  const error = ref<string | null>(null)

  // Internal dialog reference
  let activeDialog: VertoDialog | null = null
  let incomingDialog: VertoDialog | null = null

  const isInCall = computed(() => {
    if (!currentCall.value) return false
    return ['ringing', 'early', 'active', 'held'].includes(currentCall.value.state)
  })

  const hasIncomingCall = computed(() => incomingCall.value !== null)

  function convertIceServers(servers: IceServer[]): RTCIceServer[] {
    return servers.map(s => ({
      urls: s.urls,
      username: s.username,
      credential: s.credential,
    }))
  }

  async function connect(credentials: VertoCredentials): Promise<boolean> {
    if (isConnected.value || isConnecting.value) {
      console.warn('[Verto] Already connected or connecting')
      return isConnected.value
    }

    // Check if Verto.js is loaded
    if (!window.Verto) {
      error.value = 'Verto.js is not loaded'
      console.error('[Verto]', error.value)
      return false
    }

    isConnecting.value = true
    error.value = null

    return new Promise((resolve) => {
      try {
        const vertoInstance = new window.Verto(
          {
            login: credentials.login,
            passwd: credentials.password,
            socketUrl: credentials.ws_url,
            iceServers: convertIceServers(credentials.ice_servers),
            tag: 'verto-audio',
            deviceParams: {
              useMic: true,
              useSpeak: true,
              useCamera: false,
            },
          },
          {
            onWSLogin: (_v, success) => {
              isConnecting.value = false
              isConnected.value = success
              if (!success) {
                error.value = 'WebSocket login failed'
              }
              console.log('[Verto] WebSocket login:', success)
              resolve(success)
            },
            onWSClose: (_v, _success) => {
              isConnected.value = false
              isConnecting.value = false
              currentCall.value = null
              incomingCall.value = null
              activeDialog = null
              incomingDialog = null
              console.log('[Verto] WebSocket closed')
            },
            onDialogState: (dialog) => {
              handleDialogState(dialog)
            },
            onMessage: (_v, _dialog, msg, _data) => {
              console.log('[Verto] Message:', msg)
            },
          }
        )

        verto.value = vertoInstance
        vertoInstance.login()
      } catch (e) {
        isConnecting.value = false
        error.value = e instanceof Error ? e.message : 'Unknown error'
        console.error('[Verto] Connect error:', e)
        resolve(false)
      }
    })
  }

  function handleDialogState(dialog: VertoDialog) {
    console.log('[Verto] Dialog state:', dialog.state, 'direction:', dialog.direction, 'callID:', dialog.callID)

    // Handle incoming call
    if (dialog.direction === 'inbound' && dialog.state === 'ringing') {
      incomingDialog = dialog
      incomingCall.value = {
        callId: dialog.callID,
        direction: 'inbound',
        state: dialog.state,
        remoteName: dialog.callerIdName,
        remoteNumber: dialog.callerIdNumber,
        isMuted: false,
      }
      return
    }

    // Update call state
    const isIncoming = dialog.direction === 'inbound'
    const callState: CallState = {
      callId: dialog.callID,
      direction: dialog.direction,
      state: dialog.state,
      remoteName: isIncoming ? dialog.callerIdName : undefined,
      remoteNumber: isIncoming ? dialog.callerIdNumber : dialog.destinationNumber,
      isMuted: dialog.getMute?.('mic') ?? false,
      startTime: dialog.state === 'active' ? Date.now() : currentCall.value?.startTime,
    }

    // Handle state transitions
    switch (dialog.state) {
      case 'trying':
      case 'early':
      case 'ringing':
      case 'answering':
      case 'active':
      case 'held':
        activeDialog = dialog
        currentCall.value = callState
        // Clear incoming if this is the same call being answered
        if (incomingCall.value?.callId === dialog.callID) {
          incomingCall.value = null
          incomingDialog = null
        }
        break

      case 'hangup':
      case 'destroy':
        if (currentCall.value?.callId === dialog.callID) {
          currentCall.value = null
          activeDialog = null
        }
        if (incomingCall.value?.callId === dialog.callID) {
          incomingCall.value = null
          incomingDialog = null
        }
        break
    }
  }

  function makeCall(destination: string, options?: { callerName?: string }): boolean {
    if (!verto.value || !isConnected.value) {
      error.value = 'Not connected to Verto'
      return false
    }

    if (isInCall.value) {
      error.value = 'Already in a call'
      return false
    }

    try {
      const dialog = verto.value.newCall({
        destination_number: destination,
        caller_id_name: options?.callerName || 'WebRTC User',
        useVideo: false,
        useStereo: false,
        tag: 'verto-audio',
      })

      activeDialog = dialog
      currentCall.value = {
        callId: dialog.callID,
        direction: 'outbound',
        state: 'new',
        remoteNumber: destination,
        isMuted: false,
      }

      return true
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to make call'
      console.error('[Verto] Make call error:', e)
      return false
    }
  }

  function answerCall(): boolean {
    if (!incomingDialog) {
      error.value = 'No incoming call to answer'
      return false
    }

    try {
      incomingDialog.answer({
        useVideo: false,
        tag: 'verto-audio',
      })
      return true
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to answer call'
      console.error('[Verto] Answer call error:', e)
      return false
    }
  }

  function rejectCall(): boolean {
    if (!incomingDialog) {
      error.value = 'No incoming call to reject'
      return false
    }

    try {
      incomingDialog.hangup({ cause: 'CALL_REJECTED' })
      incomingCall.value = null
      incomingDialog = null
      return true
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to reject call'
      console.error('[Verto] Reject call error:', e)
      return false
    }
  }

  function hangup(): boolean {
    if (!activeDialog) {
      error.value = 'No active call to hangup'
      return false
    }

    try {
      activeDialog.hangup()
      return true
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to hangup'
      console.error('[Verto] Hangup error:', e)
      return false
    }
  }

  function toggleMute(): boolean {
    if (!activeDialog) {
      return false
    }

    try {
      const newMuteState = activeDialog.toggleMute('mic')
      if (currentCall.value) {
        currentCall.value = { ...currentCall.value, isMuted: newMuteState }
      }
      return newMuteState
    } catch (e) {
      console.error('[Verto] Toggle mute error:', e)
      return currentCall.value?.isMuted ?? false
    }
  }

  function setMute(mute: boolean): boolean {
    if (!activeDialog) {
      return false
    }

    try {
      activeDialog.setMute('mic', mute)
      if (currentCall.value) {
        currentCall.value = { ...currentCall.value, isMuted: mute }
      }
      return true
    } catch (e) {
      console.error('[Verto] Set mute error:', e)
      return false
    }
  }

  function sendDTMF(digit: string): boolean {
    if (!activeDialog) {
      return false
    }

    try {
      activeDialog.dtmf(digit)
      return true
    } catch (e) {
      console.error('[Verto] DTMF error:', e)
      return false
    }
  }

  function disconnect(): void {
    if (verto.value) {
      try {
        if (activeDialog) {
          activeDialog.hangup()
        }
        verto.value.logout()
        verto.value.purge()
      } catch (e) {
        console.error('[Verto] Disconnect error:', e)
      }
    }

    verto.value = null
    isConnected.value = false
    isConnecting.value = false
    currentCall.value = null
    incomingCall.value = null
    activeDialog = null
    incomingDialog = null
  }

  // Cleanup on unmount
  onUnmounted(() => {
    disconnect()
  })

  return {
    // State
    isConnected,
    isConnecting,
    isInCall,
    hasIncomingCall,
    currentCall,
    incomingCall,
    error,

    // Methods
    connect,
    disconnect,
    makeCall,
    answerCall,
    rejectCall,
    hangup,
    toggleMute,
    setMute,
    sendDTMF,
  }
}
