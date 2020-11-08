import {Card, Col, Container, Image, Row} from 'react-bootstrap';
import styles from '../styles/rooms/[roomId]/playlists/Playlist.module.scss'
import {getAlbumCoverUrlFromTrack, getArtistsFromTrack} from "../utils/trackUtils";
import {useState} from "react";
import CustomModal from "./CustomModal";
import {getPictureUrl} from "../utils/pictureUtils";

const maxLikedFaceToShow = 3

export default function PlaylistListElem(props) {
  const [show, showModal] = useState(false)

  let artist = getArtistsFromTrack(props.track)
  let albumCover = getAlbumCoverUrlFromTrack(props.track)

  let usersForTrack = props.usersForTrack;

  let showUsersForSong = usersForTrack.slice(0, maxLikedFaceToShow).map(user => {
    return (
      <div key={user.id} className="float-right mr-1">
        <Image className={styles.user_pic} src={getPictureUrl(user)} roundedCircle/>
      </div>
    )
  })

  let modalUsersForSong = (
    <div>
      <div className="mb-3">
        <Row>
          <Col xs={3} className={styles.album_pic_container}>
            <Image src={albumCover} className={styles.album_pic} rounded/>
          </Col>
          <Col xs={9}>
            <p className={styles.track_name}>{props.track.name}</p>
            <p className={styles.artist_name}>{artist}</p>
          </Col>
        </Row>
      </div>

      <div className={styles.songs_users_divider}/>

      {
        usersForTrack.map(user => {
          return (
            <Row key={user.id} className="ml-1 mr-1">
              <Col xs={12}>
                <Image className={styles.user_pic} src={getPictureUrl(user)} roundedCircle/>
                <p className={styles.user_name}>{user.name}</p>
              </Col>
            </Row>
          )
        })
      }
    </div>
  )

  let otherPeopleForSong;

  if (usersForTrack.length > maxLikedFaceToShow) {
    otherPeopleForSong = (
      <div className="float-right mr-1">
        +{usersForTrack.length - maxLikedFaceToShow}
      </div>
    )
  }

  let musicButton;

  if (props.track.preview_url && props.songPlaying === props.track.preview_url) {
    musicButton = (
      <div className="text-center btn p-0">⏸️</div>
    )

  } else if (props.track.preview_url) {
    musicButton = (
      <div className="text-center btn p-0">▶️</div>
    )
  }

  const onClickMusic = () => {
    if (props.track.preview_url && props.songPlaying === props.track.preview_url) {
      props.updateSongCallback("")

    } else if (props.track.preview_url) {
      props.updateSongCallback(props.track.preview_url)
    }
  }

  return (
    <Card className="mt-1 col-11 col-md-5 p-1">
      <div onClick={onClickMusic} className={styles.playlist_item}>
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

          <Row>
            <Col xs={12}>
              <div className="float-right btn p-0 pt-1 pb-1" onClick={
                (e) => {
                  e.stopPropagation();
                  showModal(true)
                }
              }>
                {otherPeopleForSong}
                {showUsersForSong}
              </div>
            </Col>
          </Row>
        </Container>
      </div>

      <CustomModal
        show={show}
        body={
          <div>
            {modalUsersForSong}
          </div>
        }
        onHideAction={
          () => {showModal(false)}
        }
      />
    </Card>
  )
}