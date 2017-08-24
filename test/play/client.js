const WebSocket = require('ws')

// The basis for a module to mock a connection from Play

function connect () {
  const ws = new WebSocket('wss://localhost.dividat.com:8382/senso')

  ws.onmessage = (e) => {
    console.log(JSON.parse(e.data))
  }

  ws.onclose = (e) => {
    setTimeout(connect, 1000)
  }

  ws.onerror = (e) => {
    console.log(e)
    ws.close()
  }

  return ws
}

var conn = connect()

conn.onopen = () => {
  console.log('Connected!')

  conn.send(new ArrayBuffer(20))
}
