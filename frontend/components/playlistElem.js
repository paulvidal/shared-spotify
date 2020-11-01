import {Card, Col, Container, Row} from 'react-bootstrap';
import styles from '../styles/rooms/[roomId]/playlists/Playlists.module.scss'
import Link from "next/link";
import {sum} from "lodash";
import {getTotalTrackCount} from "../utils/trackUtils";

export default function PlaylistElem(props) {
  const maxSongsInTotal = getTotalTrackCount(props.playlist)

  return (
    <Link href={"/rooms/" + props.roomId + "/playlists/" + props.playlist.id}>
      <Card className={"mt-1 col-11 col-md-5 p-3 mb-2 " + styles.playlist_elem_card}>
        <Container>
          <Row>
            <Col xs={12}>
                <h5 className={"text-center " + styles.playlist_elem}>
                  {props.playlist.name}
                </h5>
                <p className="text-center mb-0 mt-1">
                  <strong>{maxSongsInTotal}</strong> songs at most
                </p>
            </Col>
          </Row>
        </Container>
      </Card>
  </Link>
  )
}