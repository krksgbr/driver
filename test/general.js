/* eslint-env mocha */

const { wait, getLogs, startDriver } = require('./utils')
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

it('Get log entries from HTTP endpoint.', async () => {
  const logs = await getLogs()
  expect(logs).to.be.an('array')
  expect(logs[0]).to.include({level: 'info', msg: 'Dividat Driver starting'})
})
