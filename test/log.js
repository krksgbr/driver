const WebSocket = require('ws')

function connect () {
  const ws = new WebSocket('ws://127.0.0.1:8382/log')

  ws.onmessage = (e) => {
    console.log(JSON.parse(e.data))
  }

  ws.onclose = (e) => {
    setTimeout(connect, 1000)
  }

  ws.on('open', () => {
    console.log('Connection opened!')
  })

  ws.onerror = (e) => {
    ws.close()
  }

  return ws
}

connect()
