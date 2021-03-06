const DEFAULT_IMAGE_URL = "/user.png"

function setDefaultPictureOnError(error) {
  error.target.src = DEFAULT_IMAGE_URL
}

function getPictureUrl(user) {
  let imageUrl = user.image

  if (!imageUrl) {
    imageUrl = DEFAULT_IMAGE_URL
  }

  return imageUrl
}

export {
  setDefaultPictureOnError,
  getPictureUrl
}