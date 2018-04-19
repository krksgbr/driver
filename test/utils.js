const { spawn } = require('child_process')
const WebSocket = require('ws')
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
    // useful for debugging:
    // return spawn('bin/dividat-driver', [], {stdio: 'inherit'})
  },

  connectWS: function (url) {
    return new Promise((resolve, reject) => {
      const ws = new WebSocket(url)
      ws.on('open', () => {
        ws.removeAllListeners()
        resolve(ws)
      }).on('error', reject)
    })
  },

  getJSON: function (uri) {
    return rp({uri: uri, json: true})
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
