import {Card, Col, Container, Row} from 'react-bootstrap';
import styles from '../styles/rooms/[roomId]/playlists/Playlists.module.scss'
import Link from "next/link";

export default function PlaylistListElem(props) {
  return (
    <Link href={"/rooms/" + props.roomId + "/playlists/" + props.playlist.id}>
      <Card className={"mt-1 col-11 col-md-5 p-3 " + styles.playlist_elem_card}>
        <Container>
          <Row>
            <Col xs={12}>
                <h5 className={"text-center " + styles.playlist_elem}>
                  {props.index}. {props.playlist.name}
                </h5>
                <p className="text-center mb-0 mt-1">
                  {props.playlist.tracks.length} songs
                </p>
            </Col>
          </Row>
        </Container>
      </Card>
  </Link>
  )
}