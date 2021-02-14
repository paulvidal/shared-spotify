function
setState(setStateFunction, newState) {
  setStateFunction(prevState => {
    return {
      ...prevState,
      ...newState
    }
  })
}

export default setState;