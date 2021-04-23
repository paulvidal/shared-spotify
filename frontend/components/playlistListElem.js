import {Card, Col, Container, Image, Row} from 'react-bootstrap';
import styles from '../styles/rooms/[roomId]/playlists/Playlist.module.scss'
import {getAlbumCoverUrlFromTrack, getArtistsFromTrack} from "../utils/trackUtils";
import {useEffect, useRef, useState} from "react";
import CustomModal from "./CustomModal";
import {getPictureUrl, setDefaultPictureOnError} from "../utils/pictureUtils";
import setState from "../utils/stateUtils";
import LazyLoad from 'react-lazyload';
import UserImageGrouped from "./UserImageGrouped";
import UserImage from "./UserImage";

const maxLikedFaceToShow = 2

export default function PlaylistListElem(props) {
  const [item, setItem] = useState({
    showModal: false,
    likedFaceToShow: maxLikedFaceToShow,
    likedFaceComputed: false
  })

  let face = null
  let faceOtherCount = null
  let faceContainer = null

  // we compute here how many faces for the element we can show
  const compute = () => {
    if (!face || !faceContainer|| !faceOtherCount || item.likedFaceComputed) {
      return
    }

    let faceWidth = face.offsetWidth + 1;
    let faceCountWidth = faceOtherCount ? faceOtherCount.offsetWidth + 1 : 0;
    let faceContainerWidth = faceContainer.offsetWidth + 1;

    let style = getComputedStyle(faceContainer);
    let faceContainerPadding = parseInt(style.paddingRight) + parseInt(style.paddingLeft) + 1
    let faceCount = Math.floor((faceContainerWidth - faceContainerPadding - faceCountWidth) / faceWidth)

    setState(setItem, {likedFaceToShow: faceCount, likedFaceComputed: true})
  }

  let artist = getArtistsFromTrack(props.track)
  let albumCover = getAlbumCoverUrlFromTrack(props.track)

  let usersForTrack = props.usersForTrack

  let showUsersForSong = usersForTrack.slice(0, item.likedFaceToShow).map(user => {
    return (
      <div key={user.id} className="float-right pr-1" ref={(ref) => {
        face = ref
        compute()
      }}>
        <Image className={styles.user_pic} src={getPictureUrl(user)} roundedCircle onError={setDefaultPictureOnError}/>
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
                <UserImage pictureUrl={user.image} classes={styles.user_pic} name={user.name} />
                <p className={styles.user_name}>{user.name}</p>
              </Col>
            </Row>
          )
        })
      }
    </div>
  )

  let otherPeopleForSong;

  if (usersForTrack.length > item.likedFaceToShow) {
    otherPeopleForSong = (
      <div className="float-right pr-1" ref={(ref) => {
        faceOtherCount = ref
        compute()
      }}>
        +{usersForTrack.length - item.likedFaceToShow}
      </div>
    )
  }

  let musicButton;

  if (props.track.preview_url && props.songPlaying === props.track.preview_url) {
    musicButton = (
      <div className={"text-center btn p-0 position-absolute " + styles.play_button}>
        <img className={styles.play_icon} src="/pause.svg"/>
        ️</div>
    )

  } else if (props.track.preview_url) {
    musicButton = (
      <div className={"text-center btn p-0 position-absolute " + styles.play_button}>
        <img className={styles.play_icon} src="/play.svg"/>
      </div>
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
    <Card className="mt-1 col-11 col-md-5 p-1 pt-2 pb-2">
      {/* Allow lazy loading of huge lists of songs */}
      <LazyLoad height={100} offset={200}>
        <div onClick={() => setState(setItem, {showModal: true})} className={styles.playlist_item}>
          <Container>
            <Row>
              <Col xs={3} md={3} className={styles.album_pic_container} onClick={(e) => {
                e.stopPropagation();
                onClickMusic()
              }}>
                <div className="position-relative">
                  {musicButton}
                  <Image src={albumCover} className={styles.album_pic} rounded/>
                </div>
              </Col>
              <Col xs={6} md={7}>
                <p className={styles.track_name}>{props.track.name}</p>
                <p className={styles.artist_name}>{artist}</p>
              </Col>
              <Col xs={3} md={2} className="p-0 pr-2 overflow-hidden d-flex justify-content-end">
                <UserImageGrouped users={usersForTrack} />
                {/*<div className="w-100 btn p-0 pt-1 pb-1" ref={(ref) => {*/}
                {/*  faceContainer = ref*/}
                {/*  compute()*/}
                {/*}}>*/}
                {/*  {otherPeopleForSong}*/}
                {/*  {showUsersForSong}*/}
                {/*</div>*/}
              </Col>
            </Row>
          </Container>
        </div>

        <CustomModal
          show={item.showModal}
          body={
            <div>
              {modalUsersForSong}
            </div>
          }
          onHideAction={
            () => {
              setState(setItem, {showModal: false})
            }
          }
        />
      </LazyLoad>
    </Card>
  )
}