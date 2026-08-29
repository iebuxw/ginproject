let ws = null
let reconnectTimer = null
let disconnectRequested = false
const handlers = {}

export function connectWS(token) {
  disconnectRequested = false
  if (ws && ws.readyState === WebSocket.OPEN) return

  const protocol = location.protocol === 'https:' ? 'wss:' : 'ws:'
  const url = `${protocol}//${location.host}/api/ws?token=${encodeURIComponent(token)}`

  ws = new WebSocket(url)

  ws.onopen = () => {}

  ws.onmessage = (event) => {
    try {
      const msg = JSON.parse(event.data)
      if (msg.type === 'heartbeat') {
        ws.send(JSON.stringify({ type: 'pong' }))
        return
      }
      const cbs = handlers[msg.type]
      if (cbs) cbs.forEach(cb => cb(msg))
    } catch (e) {
      // ignore
    }
  }

  ws.onclose = () => {
    ws = null
    if (!disconnectRequested) reconnectTimer = setTimeout(() => connectWS(token), 5000)
  }

  ws.onerror = () => {
    ws.close()
  }
}

export function disconnectWS() {
  disconnectRequested = true
  if (reconnectTimer) clearTimeout(reconnectTimer)
  if (ws) ws.close()
  ws = null
}

export function onWSMessage(type, callback) {
  if (!handlers[type]) handlers[type] = []
  handlers[type].push(callback)
}

export function offWSMessage(type, callback) {
  if (!handlers[type]) return
  handlers[type] = handlers[type].filter(cb => cb !== callback)
}
