import {useRouter} from 'next/router'
import styles from "../../../styles/rooms/Rooms.module.scss";
import {showErrorToastWithError, Toast} from "../../../components/toast";
import axios from "axios";
import {encodeParams, getUrl} from "../../../utils/urlUtils";
import {Button} from "react-bootstrap";
import {useEffect, useState} from "react";
import {isEmpty} from "lodash";
import CustomHead from "../../../components/Head";
import Header from "../../../components/Header";
import LoaderScreen from "../../../components/LoaderScreen";
import setState from "../../../utils/stateUtils";
import Footer from "../../../components/Footer";

export default function RoomShare() {
  const router = useRouter()
  const { roomId } = router.query

  const axiosClient = axios.create({
    withCredentials: true
  })

  const [user, setUser] = useState({
    user: {},
    loading: true
  });

  const login = () => {
    const params = {
      redirect_uri: window.location.pathname
    }

    setState(setUser, {loading: true})
    router.push('/login?' + encodeParams(params))
  }

  const addUserToRoom = () => {
    // Do not do anything if no roomId exists
    if (!roomId) {
      return;
    }

    axiosClient.post(getUrl('/rooms/' + roomId + '/users'))
      .then(resp => {
        router.push('/rooms/' + roomId)
      })
      .catch(error => {
        setState(setUser, {loading: false})
        showErrorToastWithError("Cannot join the room", error, router)
      })
  }

  const refresh = () => {
    axiosClient.get(getUrl('/user'))
      .then(resp => {
        setState(setUser, {user: resp.data})
        addUserToRoom()
      })
      .catch(error => {
        setState(setUser, {loading: false})
      })
  }

  useEffect(refresh, [roomId])

  let button;

  if (isEmpty(user.user)) {
    button = (
      <Button variant="outline-success" size="lg" className="mt-5" onClick={login}>
        Connect music account
      </Button>
    )

  } else {
    button = (
      <Button variant="success" size="lg" className="mt-5" onClick={addUserToRoom}>
        Join room
      </Button>
    )
  }

  // Use a loader screen if nothing is ready
  if (user.loading) {
    return (
      <LoaderScreen/>
    )
  }

  return (
    <div className={styles.container}>
      <CustomHead />

      <Header />

      <main className={styles.main}>
        <h1 className="text-center p-4">You are invited to join a room</h1>

        {button}
      </main>

      <Footer/>

      <Toast/>
    </div>
  )
}