import {Container, Row, Col, Card, Image} from 'react-bootstrap';
import styles from '../styles/rooms/[roomId]/Room.module.scss'
import UserImage from "./UserImage";

export default function UserRoomListElem(props) {
  return (
    <Card className="mt-2 p-3 col-11 col-md-5" key={props.user.id}>
        <Container>
          <Row>
            <Col xs={3} className={styles.profile_pic_container}>
              <UserImage pictureUrl={props.user.image} classes={styles.profile_pic} name={props.user.name} />
            </Col>
            <Col xs={9}>
              <h3 className={styles.profile_name}>{props.user.name}</h3>
            </Col>
          </Row>
        </Container>
    </Card>
  )
}