const WebSocket = require('ws')
const { spawn } = require('child_process')

module.exports = {
  wait: function (t) {
    return new Promise((resolve, reject) => {
      setTimeout(resolve, t)
    })
  },

  startDriver: function () {
    return spawn('release/dividat-driver', ['start'])
  },

  connectWithLog: function () {
    return new WebSocket('wss://localhost.dividat.com:8382/log')
  }
}
