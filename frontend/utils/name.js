import {isEmpty} from "lodash";

function getInitials(name) {
  let initials = null

  if (!isEmpty(name)) {
    initials = name.split(" ")
      .map(w => w[0].toUpperCase())
      .filter(c => c !== "")
      .slice(0, 2)
  }

  return initials
}

export {
  getInitials
}