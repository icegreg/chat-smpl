import { ref, computed } from 'vue'
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

  // Disconnect callbacks
  const disconnectCallbacks: Array<() => void> = []

  // Register a callback to be called when WebSocket disconnects
  function onDisconnect(callback: () => void): void {
    disconnectCallbacks.push(callback)
  }

  // Notify all disconnect callbacks
  function notifyDisconnect(): void {
    for (const cb of disconnectCallbacks) {
      try {
        cb()
      } catch (e) {
        console.error('[Verto] Disconnect callback error:', e)
      }
    }
  }

  // Expose active dialog on window for E2E testing
  function exposeDialogForTesting(dialog: VertoDialog | null): void {
    // @ts-expect-error - exposing for E2E tests
    window.__vertoActiveDialog = dialog
  }

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

  // Generate dynamic WebSocket URL based on current host
  function getVertoWsUrl(): string {
    const wsProtocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    return `${wsProtocol}//${window.location.host}/verto`
  }

  async function connect(credentials: VertoCredentials): Promise<boolean> {
    // Use dynamic URL based on current host, ignore ws_url from backend
    const dynamicWsUrl = getVertoWsUrl()
    console.log('[Verto] connect() called with credentials:', {
      login: credentials.login,
      ws_url_from_backend: credentials.ws_url,
      ws_url_dynamic: dynamicWsUrl,
      ice_servers: credentials.ice_servers?.length
    })

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

    console.log('[Verto] Creating Verto instance...')
    isConnecting.value = true
    error.value = null

    // Small delay to ensure event loop is clear
    await new Promise(r => setTimeout(r, 100))

    return new Promise((resolve) => {
      try {
        console.log('[Verto] Calling new window.Verto() with params:', {
          login: credentials.login,
          socketUrl: dynamicWsUrl,
          hasIceServers: credentials.ice_servers?.length > 0
        })

        // Store resolve function for potential retry callback access
        // @ts-expect-error - exposing for debugging
        window.__vertoConnectResolve = resolve

        // Flag to track if onWSLogin callback was fired
        let loginCallbackFired = false

        const vertoInstance = new window.Verto(
          {
            login: credentials.login,
            passwd: credentials.password,
            socketUrl: dynamicWsUrl,
            iceServers: convertIceServers(credentials.ice_servers),
            tag: 'verto-audio',
            deviceParams: {
              useMic: 'any',  // "any" = auto-select device, true causes OverconstrainedError
              useSpeak: 'any',
              useCamera: false,
            },
          },
          {
            onWSLogin: (_v, success) => {
              console.log('[Verto] onWSLogin callback fired, success:', success)
              loginCallbackFired = true
              isConnecting.value = false
              isConnected.value = success
              if (!success) {
                error.value = 'WebSocket login failed'
                console.error('[Verto] Login failed!')
              }
              console.log('[Verto] WebSocket login:', success)
              resolve(success)
            },
            onWSClose: (_v, _success) => {
              console.warn('[Verto] onWSClose callback fired!')
              // Only update state if we were connected (not during initial connection with retries)
              const wasConnected = isConnected.value
              if (wasConnected) {
                isConnected.value = false
                isConnecting.value = false
                currentCall.value = null
                incomingCall.value = null
                activeDialog = null
                incomingDialog = null

                // Notify registered callbacks about disconnect
                // This allows voice store to clean up conferences
                notifyDisconnect()
              }
              console.log('[Verto] WebSocket closed, was connected:', wasConnected)
            },
            onDialogState: (dialog) => {
              handleDialogState(dialog)
            },
            onMessage: (_v, _dialog, msg, _data) => {
              console.log('[Verto] Message:', msg)
            },
          }
        )

        console.log('[Verto] Verto instance created, storing and calling login()...')
        verto.value = vertoInstance
        // @ts-expect-error - exposing for debugging
        window.__vertoInstance = vertoInstance
        vertoInstance.login()
        console.log('[Verto] login() called, waiting for onWSLogin callback...')

        // Poll for WebSocket connection and force resolve if onWSLogin doesn't fire
        // This is a workaround for Verto's internal retry mechanism not calling callbacks
        const pollConnection = setInterval(() => {
          // Try to find WebSocket in various possible locations
          // eslint-disable-next-line @typescript-eslint/no-explicit-any
          const rpcClient = (vertoInstance as any).rpcClient
          if (rpcClient) {
            // Try different property names
            const ws = rpcClient.ws || rpcClient._ws_socket || rpcClient.socket
            console.log('[Verto Poll] rpcClient keys:', Object.keys(rpcClient).slice(0, 8), 'isConnected:', isConnected.value)

            if (ws && ws.readyState === WebSocket.OPEN) {
              console.log('[Verto] WebSocket is OPEN! loginCallbackFired:', loginCallbackFired)
              if (!isConnected.value && !loginCallbackFired) {
                console.log('[Verto] WebSocket open but login not resolved - forcing success!')
                isConnecting.value = false
                isConnected.value = true
                clearInterval(pollConnection)
                resolve(true)
              } else {
                clearInterval(pollConnection)
              }
            }
          }
        }, 1000)
      } catch (e) {
        isConnecting.value = false
        error.value = e instanceof Error ? e.message : 'Unknown error'
        console.error('[Verto] Connect error:', e)
        resolve(false)
      }
    })
  }

  // Helper to extract state name from Verto dialog state (can be string or object)
  function getStateName(state: VertoDialogState | { name: string }): VertoDialogState {
    if (typeof state === 'object' && state && 'name' in state) {
      return state.name as VertoDialogState
    }
    return state as VertoDialogState
  }

  // Helper to extract direction name from Verto dialog direction
  function getDirectionName(direction: 'inbound' | 'outbound' | { name: string }): 'inbound' | 'outbound' {
    if (typeof direction === 'object' && direction && 'name' in direction) {
      return direction.name as 'inbound' | 'outbound'
    }
    return direction as 'inbound' | 'outbound'
  }

  function handleDialogState(dialog: VertoDialog) {
    // Extract actual state and direction (Verto may return objects with .name property)
    const stateName = getStateName(dialog.state)
    const directionName = getDirectionName(dialog.direction)

    console.warn('[Verto] handleDialogState:', stateName, 'direction:', directionName, 'callID:', dialog.callID)

    // Handle incoming call
    if (directionName === 'inbound' && stateName === 'ringing') {
      // Ignore incoming calls if we're already in an active call
      // This prevents duplicate/stale incoming call popups
      if (isInCall.value && currentCall.value) {
        console.warn('[Verto] Ignoring incoming call - already in active call:', currentCall.value.callId)
        return
      }

      // Log all caller info for debugging
      console.log('[Verto] Incoming call details:', {
        callID: dialog.callID,
        callerIdName: dialog.callerIdName,
        callerIdNumber: dialog.callerIdNumber,
        destinationNumber: dialog.destinationNumber,
      })

      // Get fsName from callerIdNumber or try to extract from callerIdName
      // FreeSWITCH sends both origination_caller_id_name and origination_caller_id_number
      let remoteNumber = dialog.callerIdNumber
      if (!remoteNumber && dialog.callerIdName) {
        // Check if callerIdName contains a conference name pattern
        const confMatch = dialog.callerIdName.match(/(adhoc_|conf_|scheduled_|private_)\S+/)
        if (confMatch) {
          remoteNumber = confMatch[0]
          console.log('[Verto] Extracted remoteNumber from callerIdName:', remoteNumber)
        }
      }

      incomingDialog = dialog
      incomingCall.value = {
        callId: dialog.callID,
        direction: 'inbound',
        state: stateName,
        remoteName: dialog.callerIdName,
        remoteNumber: remoteNumber,
        isMuted: false,
      }
      return
    }

    // Update call state
    const isIncoming = directionName === 'inbound'

    // For incoming calls, get remoteNumber with fallback to extract from callerIdName
    let callRemoteNumber: string | undefined
    if (isIncoming) {
      callRemoteNumber = dialog.callerIdNumber
      if (!callRemoteNumber && dialog.callerIdName) {
        const confMatch = dialog.callerIdName.match(/(adhoc_|conf_|scheduled_|private_)\S+/)
        if (confMatch) {
          callRemoteNumber = confMatch[0]
          console.log('[Verto] callState: Extracted remoteNumber from callerIdName:', callRemoteNumber)
        }
      }
    } else {
      callRemoteNumber = dialog.destinationNumber
    }

    const callState: CallState = {
      callId: dialog.callID,
      direction: directionName,
      state: stateName,
      remoteName: isIncoming ? dialog.callerIdName : undefined,
      remoteNumber: callRemoteNumber,
      isMuted: dialog.getMute?.('mic') ?? false,
      startTime: stateName === 'active' ? Date.now() : currentCall.value?.startTime,
    }

    // Handle state transitions
    switch (stateName) {
      case 'trying':
      case 'early':
      case 'ringing':
      case 'answering':
      case 'active':
      case 'held':
        activeDialog = dialog
        exposeDialogForTesting(dialog)
        currentCall.value = callState
        console.warn('[Verto] currentCall updated to state:', stateName)
        // Clear incoming if this is the same call being answered
        if (incomingCall.value?.callId === dialog.callID) {
          incomingCall.value = null
          incomingDialog = null
        }
        break

      case 'hangup':
      case 'destroy':
        console.warn('[Verto] Call ended:', stateName, 'cause:', dialog.cause)
        if (currentCall.value?.callId === dialog.callID) {
          currentCall.value = null
          activeDialog = null
          exposeDialogForTesting(null)
        }
        if (incomingCall.value?.callId === dialog.callID) {
          incomingCall.value = null
          incomingDialog = null
        }
        break
    }
  }

  function makeCall(destination: string, options?: { callerName?: string }): boolean {
    console.warn('[Verto] makeCall called, destination:', destination, 'isConnected:', isConnected.value)

    if (!verto.value || !isConnected.value) {
      error.value = 'Not connected to Verto'
      console.error('[Verto] makeCall failed - not connected')
      return false
    }

    if (isInCall.value) {
      error.value = 'Already in a call'
      console.warn('[Verto] makeCall failed - already in call')
      return false
    }

    try {
      console.warn('[Verto] Calling verto.newCall...')
      const dialog = verto.value.newCall({
        destination_number: destination,
        caller_id_name: options?.callerName || 'WebRTC User',
        useVideo: false,
        useStereo: false,
        tag: 'verto-audio',
      })

      console.warn('[Verto] newCall returned dialog:', dialog?.callID)

      activeDialog = dialog
      exposeDialogForTesting(dialog)
      currentCall.value = {
        callId: dialog.callID,
        direction: 'outbound',
        state: 'new',
        remoteNumber: destination,
        isMuted: false,
      }

      console.warn('[Verto] currentCall set:', currentCall.value?.state)
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
      console.log('[Verto] answerCall: Answering dialog:', incomingDialog.callID)
      incomingDialog.answer({
        useVideo: false,
        tag: 'verto-audio',
      })

      // Clear incoming call state immediately after answering
      // The handleDialogState will also clear it, but we do it here
      // to ensure UI updates right away
      console.log('[Verto] answerCall: Clearing incomingCall state')
      incomingCall.value = null
      // Note: don't clear incomingDialog here - it's needed for state transitions

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

  // Note: Don't use onUnmounted here as this composable is used in Pinia stores
  // which don't have component lifecycle. Cleanup should be handled by the store.

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

    // Callbacks
    onDisconnect,
  }
}
