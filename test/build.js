/* eslint-env mocha */

// Test the build process

const { spawn } = require('child_process')

it('make succeeds', function (done) {
  // set timeout to 5s
  this.timeout(5 * 1000)
  const make = spawn('make')
  make.on('exit', (code) => {
    if (code === 0) {
      return done()
    } else {
      return done(new Error('make exited with non-zero code (' + code + ')'))
    }
  })
})
