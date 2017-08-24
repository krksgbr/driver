/* eslint-env mocha */

const { spawn } = require('child_process')
const { wait } = require('./utils')
const rp = require('request-promise')
const expect = require('chai').expect
const WebSocket = require('ws')

var driver

beforeEach(async () => {
  var code = 0
  driver = spawn('release/dividat-driver', ['start']).on('exit', (c) => {
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
  return rp({uri: 'https://localhost.dividat.com:8382/', json: true})
  .then((response) => {
    expect(response).to.have.property('message').equal('Dividat Driver')
    expect(response).to.have.property('version')
  })
})

it('Opening a second instance of the driver fails.', (done) => {
  // the beforeEach hook already started the first running instance for us
  spawn('release/dividat-driver', ['start']).on('exit', (c) => {
    expect(c).to.be.equal(2)
    done()
  })
})

it('Connect to log WebSocket endpoint.', (done) => {
  new WebSocket('wss://localhost.dividat.com:8382/log')
    .on('error', () => {
    })
    .on('open', () => {
      done()
    })
})
