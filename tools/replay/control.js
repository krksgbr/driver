// Moch up the Senso control server.

var net = require('net')

var HOST = '0.0.0.0'
var PORT = 55567

// Create a server instance, and chain the listen function to it
// The function passed to net.createServer() becomes the event handler for the 'connection' event
// The sock object the callback function receives UNIQUE for each connection
module.exports = net.createServer(function (sock) {
    // We have a connection - a socket object is assigned to the connection automatically
  console.log('CONTROL - Connection: ' + sock.remoteAddress + ':' + sock.remotePort)

    // Add a 'data' event handler to this instance of socket
  sock.on('data', function (data) {
        // console.log('DATA ' + sock.remoteAddress + ': ' + data);
    console.log(data)
        // Write the data back to the socket, the client will receive it as data from the server
        // sock.write(data);
  })

    // Add a 'close' event handler to this instance of socket
  sock.on('close', function (data) {
    console.log('CONTROL - Closed: ' + sock.remoteAddress + ' ' + sock.remotePort)
  })
}).listen(PORT, HOST)

console.log('CONTROL listening on ' + HOST + ':' + PORT)
