import {Container, Row, Col, Card, Button, Image} from 'react-bootstrap';
import styles from '../styles/rooms/[roomId]/Room.module.scss'
import {getPictureUrl} from "../utils/pictureUtils";

export default function UserRoomListElem(props) {
  return (
    <Card className="mt-2 p-3 col-11 col-md-5">
        <Container>
          <Row>
            <Col xs={3} className={styles.profile_pic_container}>
              <Image src={getPictureUrl(props.user)} className={styles.profile_pic} roundedCircle/>
            </Col>
            <Col xs={9}>
              <h3 className={styles.profile_name}>{props.user.user_infos.name}</h3>
            </Col>
          </Row>
        </Container>
    </Card>
  )
}