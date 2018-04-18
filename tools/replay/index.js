// Mock up a Senso data and control server

const argv = require('minimist')(process.argv.slice(2))
const fs = require('fs')
const split = require('binary-split')
const net = require('net')
const bonjour = require('bonjour')()
const EventEmitter = require('events')

const control = require('./control')

async function mockSenso (profile, data) {
  var socket = await listenForConnection('0.0.0.0', 55567)

  // Helper callback that can be removed
  function send (data) {
    socket.write(data)
  }

  data.on('data', send)
  socket.on('data', (incoming) => {
    // Mock a suitable response
    socket.write(control(profile, incoming))
  })

  socket.on('close', () => {
    console.log('Connection closed.')
    data.removeListener('data', send)
    mockSenso(profile, data)
  })

  socket.on('error', (err) => {
    console.log(err)
    data.removeListener('data', send)
    mockSenso(profile, data)
  })
}

function listenForConnection (host, port) {
  return new Promise((resolve, reject) => {
    console.log('Listening on ' + host + ':' + port)
    var server = net.createServer((socket) => {
      console.log('Connection: ' + socket.remoteAddress + ':' + socket.remotePort)

      // disable Nagle
      socket.setNoDelay()

      // Only allow one connection at a time
      server.close()
      resolve(socket)
    }).listen(port, host)
  })
}

// Create a never ending stream of data
function Replayer (recFile, timeout) {
  var emitter = new EventEmitter()

  function createStream () {
    var stream = new fs.createReadStream(recFile).pipe(split())

    stream.on('data', (data) => {
      stream.pause()

      var buf = Buffer.from(data.toString(), 'base64')
      emitter.emit('data', buf)

      setTimeout(() => {
        stream.resume()
      }, timeout)
    }).on('end', () => {
      createStream()
    })
  }
  createStream()
  return emitter
}

const profile = {
  serial_number: '31-00000000',
  board_serial_numbers: {
    controller: '30-00000000',
    led_boards: {
      'center': '30-00000001',
      'up': '30-00000002',
      'right': '30-00000003',
      'down': '30-00000004',
      'left': '30-00000005'
    }
  }
}
var recFile = argv['_'].pop() || 'rec/zero.dat'
var dataTimeout = 't' in argv
    ? argv['t']
    : 20

const dataStream = Replayer(recFile, dataTimeout)

// Advertise Senso via mDNS
bonjour.publish({
  name: 'Senso data replayer',
  txt: {ser_no: profile.serial_number},
  type: 'sensoControl',
  port: '55567'})

mockSenso(profile, dataStream)
