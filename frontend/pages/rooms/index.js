import styles from "../../styles/rooms/Rooms.module.scss";
import {useEffect, useState} from "react";
import axios from "axios";
import RoomListElem from "../../components/roomListElem";
import {Button, FormControl, InputGroup} from 'react-bootstrap';

import {showErrorToastWithError, showSuccessToast, Toast} from "../../components/toast";
import {getUrl} from "../../utils/urlUtils";
import CustomHead from "../../components/Head";
import {useRouter} from "next/router";
import Header from "../../components/Header";
import LoaderScreen from "../../components/LoaderScreen";
import CustomModal from "../../components/CustomModal";
import setState from "../../utils/stateUtils";
import moment from "moment";

export default function Rooms() {
  const router = useRouter()

  const axiosClient = axios.create({
    withCredentials: true
  })

  const [rooms, setRooms] = useState({
    rooms: [],
    loading: true,
    newRomName: "",
    showCreateRoomModal: false
  });

  const refresh = () => {
    axiosClient.get(getUrl('/rooms'))
      .then(resp => {
        setState(setRooms, {
          rooms: Object.values(resp.data.rooms),
          loading: false
        })
      })
      .catch(error => {
        setState(setRooms, {loading: false})
        showErrorToastWithError("Failed to get all rooms info", error)
      })
  }

  const createRoom = () => {
    axiosClient.post(getUrl('/rooms'), {
      room_name: rooms.newRomName

    }).then(resp => {
      const roomId = resp.data.room_id
      router.push('/rooms/' + roomId)

    }).catch(error => {
      showErrorToastWithError("Room failed to create ! Please try again", error)

    }).finally(() => {
      setState(setRooms, {newRomName: ""})
    })
  }

  const showModal = () => {
    setState(setRooms, {showCreateRoomModal: true})
  }

  const hideModal = () => {
    setState(setRooms, {showCreateRoomModal: false, newRomName: ""})
  }

  useEffect(refresh, [])

  // Use a loader screen if nothing is ready
  if (rooms.loading) {
    return (
      <LoaderScreen/>
    )
  }

  let emptyRoomText;
  let roomsList;

  if (rooms.rooms.length === 0) {
    emptyRoomText = (
      <p className="mt-4">No rooms at the moment...</p>
    )

  } else {
    roomsList = rooms.rooms.sort((room1, room2) => {
      return moment(room2.creation_time) - moment(room1.creation_time)
    }).map(room => {
      return (
        <RoomListElem key={room.id} room={room}/>
      )
    });
  }

  let modalBody = (
    <InputGroup className="mb-3">
      <FormControl
        aria-label="Default"
        aria-describedby="inputGroup-sizing-default"
        placeholder="Room #4FTY"
        onChange={(e) => setState(setRooms, {newRomName: e.target.value})}
      />
    </InputGroup>
  )

  return (
    <div className={styles.container}>
      <CustomHead />

      <Header />

      <main className={styles.main}>
        <h1>Rooms</h1>

        {emptyRoomText}

        <Button variant="outline-success" size="lg" className="mt-4 mb-4" onClick={showModal}>
          Create a new room
        </Button>

        {roomsList}
      </main>

      <footer className={styles.footer}>
        Powered by{' '}
        <img src="/spotify.svg" alt="Spotify Logo" className={styles.logo} />
      </footer>

      <Toast/>

      <CustomModal
        show={rooms.showCreateRoomModal}
        title={"How would you like to call you room ?"}
        body={modalBody}
        secondaryActionName={"Cancel"}
        secondaryAction={hideModal}
        onHideAction={hideModal}
        primaryActionName={"Create room"}
        primaryAction={createRoom}
      />
    </div>
  )
}