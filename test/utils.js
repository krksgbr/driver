const WebSocket = require('ws')
const { spawn } = require('child_process')
const Promise = require('bluebird')

module.exports = {
  wait: function (t) {
    return new Promise((resolve, reject) => {
      setTimeout(resolve, t)
    })
  },

  startDriver: function () {
    return spawn('bin/dividat-driver')
  },

  connectWithLog: function () {
    return new Promise((resolve, reject) => {
      const ws = new WebSocket('ws://127.0.0.1:8382/log')
      ws.on('open', () => {
        ws.removeAllListeners()
        resolve(ws)
      }).on('error', reject)
    })
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
