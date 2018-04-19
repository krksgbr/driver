/* eslint-env mocha */
const { wait, startDriver, connectWS, getJSON, expectEvent } = require('../utils')
const expect = require('chai').expect
const Promise = require('bluebird')

// TESTS

describe('Basic functionality', () => {
  var driver
  var rfid = {}

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
  })

  afterEach(() => {
    driver.kill()
  })

  it('Can retrieve current reader list.', async function () {
    this.timeout(500)

    const response = await getJSON('http://127.0.0.1:8382/rfid/readers')
    expect(response.readers).to.be.an('array')
  })

  it('Can connect to the RFID endpoint.', async function () {
    this.timeout(500)

    await connectWS('ws://127.0.0.1:8382/rfid')
  })

})
