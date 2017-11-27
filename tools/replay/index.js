// Mock up a Senso data and control server

var argv = require('minimist')(process.argv.slice(2))

var recFile = argv['_'].pop() || 'rec/zero.dat'
var dataTimeout = 't' in argv
    ? argv['t']
    : 20

// start a replay data server
require('./replay')(recFile, dataTimeout)

// start a control server
require('./control')
