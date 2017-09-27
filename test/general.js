/* eslint-env mocha */

const { wait, connectWithLog, startDriver } = require('./utils')
const rp = require('request-promise')
const expect = require('chai').expect

var driver

beforeEach(async () => {
  var code = 0
  driver = startDriver().on('exit', (c) => {
    code = c
  })
  await wait(500)
  expect(code).to.be.equal(0)
  driver.removeAllListeners()
})

afterEach(() => {
  driver.kill()
})

it('Get message and version with HTTP get.', async () => {
  return rp({uri: 'http://127.0.0.1:8382/', json: true})
  .then((response) => {
    expect(response).to.have.property('message').equal('Dividat Driver')
    expect(response).to.have.property('version')
  })
})

it('Opening a second instance of the driver fails.', (done) => {
  // the beforeEach hook already started the first running instance for us
  startDriver().on('exit', (c) => {
    expect(c).to.be.equal(2)
    done()
  })
})

it('Connect to log WebSocket endpoint and receive log entry.', async () => {
  const log = await connectWithLog()

  const receiveLogEntry = new Promise((resolve, reject) => {
    log.on('message', (s) => {
      var msg = JSON.parse(s)
      if (msg.package === 'monitor') {
        expect(msg).to.have.property('routines')
        expect(msg).to.have.property('sysMem')
        resolve()
      }
    })
  })

  // Cause a log entry
  // TODO: check if this works on windows. SIGHUP is a POSIX signal and might not be implemented on Windows.
  driver.kill('SIGHUP')

  return receiveLogEntry
})
