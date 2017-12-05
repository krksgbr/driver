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

    // Wait a bit until connection is opened
    // TODO: check for logs when connection is opened
    await wait(500)

    return ws
  }

  it('Can connect to a mock Senso.', async function () {
  // disable mocha timeout
    this.timeout(0)

    const controlConnection = getConnection(senso.control).timeout(500)
    const dataConnection = getConnection(senso.data).timeout(500)

    await connectSensoWS()
    .then(connectWithMockSenso)

    return Promise.all([controlConnection, dataConnection])
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

  it('Data is forwarded from Senso data channel to WS.', async function () {
    const sensoWS = await connectSensoWS().then(connectWithMockSenso)

    const buffer = Buffer.from(new ArrayBuffer(10))

    const readBuffer = expectEvent(sensoWS, 'message', (incoming) => {
      return (buffer.compare(incoming) === 0)
    })

    senso.data.stream.write(buffer)

    return readBuffer
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
