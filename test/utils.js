module.exports = {
  wait: function (t) {
    return new Promise((resolve, reject) => {
      setTimeout(resolve, t)
    })
  }
}
