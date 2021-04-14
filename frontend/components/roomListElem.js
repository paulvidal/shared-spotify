import {Button, Card, Spinner} from 'react-bootstrap';
import axios from "axios";
import {showErrorToastWithError} from "./toast";
import {useRouter} from "next/router";
import {getUrl} from "../utils/urlUtils";
import styles from "../styles/rooms/Rooms.module.scss";
import moment from "moment";
import {FontAwesomeIcon} from "@fortawesome/react-fontawesome";
import {faTrash} from "@fortawesome/free-solid-svg-icons";
import {useState} from "react";
import setState from "../utils/stateUtils";

export default function RoomListElem(props) {
  const router = useRouter()
  const [room, setRoom] = useState({
    loading: false,
  });

  const axiosClient = axios.create({
    withCredentials: true
  })

  const addUserToRoom = () => {
    setState(setRoom, {loading: true})

    axiosClient.post(getUrl('/rooms/' + props.room.id + '/users'))
      .then(resp => {
        router.push('/rooms/' + props.room.id)
      })
      .catch(error => {
        showErrorToastWithError("Failed to join the room", error, router)
      }).finally(() => {
        setState(setRoom, {loading: false})
      })
  }

  let open;

  if (!props.room.locked) {
    open = (
      <div className={"float-right " + styles.lock}>🔓 Open</div>
    )

  } else {
    open = (
      <div className={"float-right " + styles.lock}>🔒 Locked</div>
    )
  }

  let button = (
    <Button variant="success" className="float-left" onClick={addUserToRoom}>
      Enter room  ➡️
    </Button>
  )

  if (room.loading) {
    button = (
      <Button variant="success" className="float-left" onClick={addUserToRoom}>
        Entering room <Spinner variant="light" animation="border" className="ml-2" size={"sm"}/>
      </Button>
    )
  }

  return (
    <Card className={"mt-2 col-11 col-md-5 " + styles.room_card}>
      <Card.Body>
        <Card.Title>
          {props.room.name}
          {open}
        </Card.Title>

        <p className="mb-0">Members: {props.room.users.map(user => user.name).join(", ")}</p>
        <p className={styles.creation_date}>Created on {moment(props.room.creation_time).format("MMMM Do YYYY")}</p>

        <div className="mt-3">
          {button}

          <div className={"float-right " + styles.trash_icon} onClick={() => props.showDeleteModal(props.room.id)}>
            <FontAwesomeIcon icon={faTrash} className={styles.trash_icon_size}/>
          </div>
        </div>
      </Card.Body>
    </Card>
  )
}