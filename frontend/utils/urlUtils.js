function getUrl(endpoint) {
  let hostUrl = endpoint

  if (process.env.NEXT_PUBLIC_HOST_URL) {
    hostUrl = process.env.NEXT_PUBLIC_HOST_URL + endpoint
  }

  return hostUrl
}

export {
  getUrl
}