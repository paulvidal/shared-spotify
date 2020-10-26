import {Card, Col, Container, Image, Row} from 'react-bootstrap';
import styles from '../styles/rooms/[roomId]/playlist/Playlist.module.scss'
import {getAlbumCoverUrlFromTrack, getArtistsFromTrack} from "../utils/trackUtils";

export default function PlaylistElem(props) {
  let artist = getArtistsFromTrack(props.track)
  let albumCover = getAlbumCoverUrlFromTrack(props.track)

  let musicButton;

  if (props.track.preview_url && props.songPlaying === props.track.preview_url) {
    musicButton = (
      <div onClick={() => props.updateSongCallback("")} className="text-center btn">⏸️</div>
    )

  } else if (props.track.preview_url) {
    musicButton = (
      <div onClick={() => props.updateSongCallback(props.track.preview_url)} className="text-center btn">▶️</div>
    )
  }

  return (
    <Card className="mt-1 col-11 col-md-5 p-1">
      <Container>
        <Row>
          <Col xs={3} className={styles.album_pic_container}>
            <Image src={albumCover} className={styles.album_pic} rounded/>
          </Col>
          <Col xs={7}>
            <p className={styles.track_name}>{props.track.name}</p>
            <p className={styles.artist_name}>{artist}</p>
          </Col>
          <Col xs={2}>
            {musicButton}
          </Col>
        </Row>
      </Container>
    </Card>
  )
}