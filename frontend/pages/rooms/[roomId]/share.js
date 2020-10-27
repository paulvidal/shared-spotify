import {useRouter} from 'next/router'
import styles from "../../../styles/rooms/Rooms.module.scss";
import {showErrorToastWithError, Toast} from "../../../components/toast";
import axios from "axios";
import {getUrl} from "../../../utils/urlUtils";
import {Button} from "react-bootstrap";
import {useEffect, useState} from "react";
import {isEmpty} from "lodash";
import CustomHead from "../../../components/Head";
import Header from "../../../components/Header";

export default function RoomShare() {
  const router = useRouter()
  const { roomId } = router.query

  const axiosClient = axios.create({
    withCredentials: true
  })

  const [user, setUser] = useState({});

  const addUserToRoom = () => {
    axiosClient.post(getUrl('/rooms/' + roomId + '/users'))
      .then(resp => {
        router.push('/rooms/' + roomId)
      })
      .catch(error => {
        showErrorToastWithError("Failed to join the room", error)
      })
  }

  const refresh = () => {
    axiosClient.get(getUrl('/user'))
      .then(resp => setUser(resp.data))
      .catch(error => {})
  }

  useEffect(refresh, [roomId])

  let button;

  if (isEmpty(user)) {
    button = (
      <Button href={getUrl('/login')} variant="outline-success" size="lg" className="mt-5">
        Connect spotify account
      </Button>
    )

  } else {
    button = (
      <Button variant="success" size="lg" className="mt-5" onClick={addUserToRoom}>
        Join room
      </Button>
    )
  }

  return (
    <div className={styles.container}>
      <CustomHead />

      <Header />

      <main className={styles.main}>
        <h1>You are invited</h1>
        <p>room #{roomId}</p>

        {button}
      </main>

      <footer className={styles.footer}>
        Powered by{' '}
        <img src="/spotify.svg" alt="Spotify Logo" className={styles.logo} />
      </footer>

      <Toast/>
    </div>
  )
}