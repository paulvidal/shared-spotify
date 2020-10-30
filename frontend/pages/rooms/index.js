import styles from "../../styles/rooms/Rooms.module.scss";
import {useEffect, useState} from "react";
import axios from "axios";
import RoomListElem from "../../components/roomListElem";
import {Button} from 'react-bootstrap';

import {showErrorToastWithError, showSuccessToast, Toast} from "../../components/toast";
import {getUrl} from "../../utils/urlUtils";
import CustomHead from "../../components/Head";
import {useRouter} from "next/router";
import Header from "../../components/Header";
import LoaderScreen from "../../components/LoaderScreen";

export default function Rooms() {
  const router = useRouter()

  const axiosClient = axios.create({
    withCredentials: true
  })

  const [rooms, setRooms] = useState({
    rooms: [],
    loading: true
  });

  const refresh = () => {
    axiosClient.get(getUrl('/rooms'))
      .then(resp => setRooms(prevState => {
        return {
          ...prevState,
          rooms: Object.values(resp.data.rooms),
          loading: false
        }
      }))
      .catch(error => {
        showErrorToastWithError("Failed to get all rooms info", error)
        setRooms(prevState => {
          return {
            ...prevState,
            loading: false
          }
        })
      })
  }

  const createRoom = () => {
    axiosClient.post(getUrl('/rooms'))
      .then(resp => {
        const roomId = resp.data.room_id
        router.push('/rooms/' + roomId)
      })
      .catch(error => {
        showErrorToastWithError("Room failed to create ! Please try again", error)
      })
  }

  useEffect(refresh, [])

  // Use a loader screen if nothing is ready
  if (rooms.loading) {
    return (
      <LoaderScreen/>
    )
  }

  let roomsList;

  if (rooms.rooms.length === 0) {
    roomsList = (
      <p className="mt-4">No rooms at the moment...</p>
    )

  } else {
    roomsList = rooms.rooms.map(room => {
      console.log(rooms)
      return (
        <RoomListElem key={room.id} room={room}/>
      )
    });
  }

  return (
    <div className={styles.container}>
      <CustomHead />

      <Header />

      <main className={styles.main}>
        <h1>Rooms</h1>

        {roomsList}

        <Button variant="outline-success" size="lg" className="mt-4" onClick={createRoom}>
          Create a new room
        </Button>
      </main>

      <footer className={styles.footer}>
        Powered by{' '}
        <img src="/spotify.svg" alt="Spotify Logo" className={styles.logo} />
      </footer>

      <Toast/>
    </div>
  )
}