const DEFAULT_IMAGE_URL = "/user.png"

function getPictureUrl(user) {
  let imageUrl = user.user_infos.image

  if (!imageUrl) {
    imageUrl = DEFAULT_IMAGE_URL
  }

  return imageUrl
}

export {
  getPictureUrl
}