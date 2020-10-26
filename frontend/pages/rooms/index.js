import styles from "../../styles/rooms/Rooms.module.scss";
import {useEffect, useState} from "react";
import axios from "axios";
import RoomListElem from "../../components/roomListElem";
import {Button} from 'react-bootstrap';

import {showErrorToastWithError, showSuccessToast, Toast} from "../../components/toast";
import {getUrl} from "../../utils/urlUtils";
import CustomHead from "../../components/Head";

export default function Rooms() {
  const axiosClient = axios.create({
    withCredentials: true
  })

  const [rooms, setRooms] = useState([]);

  const refresh = () => {
    axiosClient.get(getUrl('/rooms'))
      .then(resp => setRooms(Object.values(resp.data.rooms)))
      .catch(error => {
        showErrorToastWithError("Failed to get all rooms info", error)
      })
  }

  const createRoom = () => {
    axiosClient.post(getUrl('/rooms'))
      .then(resp => {
        showSuccessToast("Room successfully created !")
        refresh()
      })
      .catch(error => {
        showErrorToastWithError("Room failed to create ! Please try again", error)
      })
  }

  useEffect(refresh, [])

  let roomsList;

  if (rooms.length === 0) {
    roomsList = (
      <p className="mt-4">No rooms at the moment...</p>
    )

  } else {
    roomsList = rooms.map(room => {
      return (
        <RoomListElem key={room.id} room={room}/>
      )
    });
  }

  return (
    <div className={styles.container}>
      <CustomHead />

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