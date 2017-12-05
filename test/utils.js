const { spawn } = require('child_process')
const Promise = require('bluebird')
const rp = require('request-promise')

module.exports = {
  wait: function (t) {
    return new Promise((resolve, reject) => {
      setTimeout(resolve, t)
    })
  },

  startDriver: function () {
    return spawn('bin/dividat-driver')
  },

  getLogs: function () {
    return rp({uri: 'http://127.0.0.1:8382/log', json: true})
  },

  expectEvent: function (emitter, event, filter) {
    return new Promise((resolve, reject) => {
      // TODO: remove listener once resolved
      emitter.on(event, (a) => {
        try {
          if (filter(a)) {
            resolve(a)
          }
        } catch (e) {
          //
        }
      })
    })
  }
}
