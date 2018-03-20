// Moch up the Senso control server.

var net = require('net')

var HOST = '0.0.0.0'
var PORT = 55567

// Firmware version to report in DEV_INFO
const VERSION_MAJOR = 1
const VERSION_MINOR = 2
const VERSION_FEATURE = 0
const VERSION_FIX = 0

// DATA_TYPES
const DATA_TYPE_GET_DEV_INFO = 0xD1
const DATA_TYPE_GET_VCC_INFO = 0xD2

// Create a server instance, and chain the listen function to it
// The function passed to net.createServer() becomes the event handler for the 'connection' event
// The sock object the callback function receives UNIQUE for each connection
module.exports = net.createServer(function (sock) {
    // We have a connection - a socket object is assigned to the connection automatically
  console.log('CONTROL - Connection: ' + sock.remoteAddress + ':' + sock.remotePort)

    // Add a 'data' event handler to this instance of socket
  sock.on('data', function (data) {
    console.log(data)
    const body = data.slice(8)
    const type = body.readUInt16LE(2)
    switch (type) {
      case DATA_TYPE_GET_DEV_INFO:
        sock.write(devInfo())
        break

      case DATA_TYPE_GET_VCC_INFO:
        sock.write(vccInfo())
        break

      default:
        sock.write(standardResponse(type))
    }
  })

    // Add a 'close' event handler to this instance of socket
  sock.on('close', function (data) {
    console.log('CONTROL - Closed: ' + sock.remoteAddress + ' ' + sock.remotePort)
  })
}).listen(PORT, HOST)

function devInfoItem () {
  var devInfoItem = Buffer.alloc(32)
  devInfoItem.writeUInt32LE(0, 0)
  devInfoItem.writeUInt32LE(0, 4)
  devInfoItem.writeInt8(VERSION_MAJOR, 11)
  devInfoItem.writeInt8(VERSION_MINOR, 10)
  devInfoItem.writeInt8(VERSION_FEATURE, 9)
  devInfoItem.writeInt8(VERSION_FIX, 8)
  return devInfoItem
}

function devInfo () {
  var header = Buffer.alloc(8)

  var lenType = Buffer.alloc(4)
  lenType.writeUInt16LE(32 * 6, 0)
  lenType.writeUInt16LE(DATA_TYPE_GET_DEV_INFO | 0x8000, 2)

  return Buffer.concat([header,
    lenType,
    devInfoItem(),
    devInfoItem(),
    devInfoItem(),
    devInfoItem(),
    devInfoItem(),
    devInfoItem()
  ])
}

function vccInfo () {
  var header = Buffer.alloc(8)

  var lenType = Buffer.alloc(4)
  lenType.writeUInt16LE(12 * 6, 0)
  lenType.writeUInt16LE(DATA_TYPE_GET_VCC_INFO | 0x8000, 2)

  var vccInfo = Buffer.alloc(12)
  // vcc_3_3
  vccInfo.writeUInt16LE(3300, 0)
  // vcc5
  vccInfo.writeUInt16LE(5000, 2)
  // vcc_12_mot
  vccInfo.writeUInt16LE(12000, 4)
  // vcc_12_led
  vccInfo.writeUInt16LE(12000, 6)
  // vcc_19_led
  vccInfo.writeUInt16LE(19000, 8)
  // temperature
  vccInfo.writeInt16LE(0, 10)

  return Buffer.concat([
    header,
    lenType,
    // controller
    vccInfo,
    // 5 LED boards
    vccInfo,
    vccInfo,
    vccInfo,
    vccInfo,
    vccInfo
  ])
}

function standardResponse (type) {
  var header = Buffer.alloc(8)

  var response = Buffer.alloc(2 + 2 + 4 + 4 + 4)
  // write type with bit 15 set to indicate a response
  response.writeUInt16LE(type | 0x8000, 2)

  return Buffer.concat([header, response])
}

console.log('CONTROL listening on ' + HOST + ':' + PORT)
