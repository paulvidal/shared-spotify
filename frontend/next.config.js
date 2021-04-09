const withPWA = require('next-pwa')

// https://github.com/shadowwalker/next-pwa#available-options
module.exports = withPWA({
  pwa: {
    dest: 'public',
    disable: false,
    register: true
  }
})