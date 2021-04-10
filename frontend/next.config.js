const withPWA = require('next-pwa')

// https://github.com/shadowwalker/next-pwa#available-options
module.exports = withPWA({
  pwa: {
    dest: 'public',
    disable: process.env.NODE_ENV === 'development',
    register: true
  }
})