/* eslint-env mocha */
const { wait, startDriver, expectEvent } = require('../utils')
const expect = require('chai').expect
const WebSocket = require('ws')
const Promise = require('bluebird')

const mock = require('./mock')
// HELPERS

// Connect to the Senso WS endpoint
function connectSensoWS () {
  return new Promise((resolve, reject) => {
    const ws = new WebSocket('ws://127.0.0.1:8382/senso')
    ws.on('open', () => {
      ws.removeAllListeners()
      resolve(ws)
    }).on('error', reject)
  })
}

// Returns a promise that is resolved with a new connection to a server
function getConnection (server) {
  return new Promise((resolve, reject) => {
    server.on('connection', (c) => resolve(c))
          .on('error', (e) => reject(e))
  })
}

// TESTS

describe('Basic functionality', () => {
  var driver
  var senso = {}

  beforeEach(async () => {
  // Start driver
    var code = 0
    driver = startDriver().on('exit', (c) => {
      code = c
    })
  // Give driver 500ms to start up
    await wait(500)
    expect(code).to.be.equal(0)
    driver.removeAllListeners()

  // start a mock Senso
    senso.data = mock.dataChannel()
    senso.control = mock.controlChannel()
  })

  afterEach(() => {
    driver.kill()

    senso.data.close()
    senso.control.close()
  })

// Sends a command to Driver (over WS) to connect with the mock senso
  async function connectWithMockSenso (ws) {
    const cmd = JSON.stringify({
      type: 'Connect',
      address: '127.0.0.1'
    })

    ws.send(cmd)

    // wait until mock senso has a connection
    await Promise.all([getConnection(senso.data), getConnection(senso.control)])

    return ws
  }

  it('Can connect to a mock Senso.', async function () {
    // It takes at least 1s to connect (as driver waits one sec between data and control connection)
    this.timeout(1500)

    return connectSensoWS()
    .then(connectWithMockSenso)
  })

  it('Can get connection status', async function () {
    this.timeout(500)

    const sensoWS = await connectSensoWS()
    .then((ws) => {
      const cmd = JSON.stringify({
        type: 'GetStatus'
      })
      ws.send(cmd)
      return ws
    })

    return expectEvent(sensoWS, 'message', (s) => {
      const msg = JSON.parse(s)
      expect(msg.type).to.be.equal('Status')
      return true
    })
  })

  it('Data is forwarded from Senso data channel to WS', async function () {
    const sensoWS = await connectSensoWS().then(connectWithMockSenso)

    const chunkSize = 64
    const n = 1000
    this.timeout(1000 + n * 4 + 500)

    const expectOnWS = new Promise((resolve, reject) => {
      var received = 0
      sensoWS.on('message', (msg) => {
        received = received + msg.length

        if (received >= chunkSize * n) {
          resolve()
        }
      })
    })

    const buffer = Buffer.from(new ArrayBuffer(chunkSize))
    for (var i = 0; i < n; i++) {
      senso.data.stream.write(buffer)
      // Give one ms time for forwarding
      await wait(1)
    }

    return expectOnWS
  })

  it('Data is forwarded from WS to control channel', async function () {
    const sensoWS = await connectSensoWS()
    sensoWS.send(JSON.stringify({
      type: 'Connect',
      address: '127.0.0.1'
    }))

    const controlConnection = await getConnection(senso.control)

    const chunkSize = 64
    const n = 1000
    this.timeout(1000 + n * 2 + 500)

    const expectData = new Promise((resolve, reject) => {
      var received = 0
      controlConnection.on('data', (data) => {
        received = received + data.length

        if (received >= chunkSize * n) {
          resolve()
        }
      })
    })

    const buffer = Buffer.from(new ArrayBuffer(chunkSize))
    for (var i = 0; i < n; i++) {
      sensoWS.send(buffer)
      // Give one ms time for forwarding
      await wait(1)
    }

    return expectData
  })

  it('Can discover mock Senso', async function () {
    this.timeout(6000)

    // connect with Senso WS
    const sensoWS = await connectSensoWS()

    // start fake mdns responder
    const bonjour = require('bonjour')()
    bonjour.publish({name: 'Senso data replayer', type: 'sensoControl', port: '55567', txt: {msg: 'Hello!'}})

    // Expect a Discovered message
    const expectDiscovered = expectEvent(sensoWS, 'message', (s) => {
      const msg = JSON.parse(s)
      return (msg.type === 'Discovered')
    })

    // Send Discover command
    const cmd = JSON.stringify({
      type: 'Discover',
      duration: 5
    })
    sensoWS.send(cmd)

    return expectDiscovered
  })
})
