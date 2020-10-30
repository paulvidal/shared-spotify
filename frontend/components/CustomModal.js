import {Modal, Button} from "react-bootstrap";

export default function CustomModal(props) {
  let title;

  if (props.title) {
    title = (
      <Modal.Header closeButton>
        <Modal.Title>
          {props.title}
        </Modal.Title>
      </Modal.Header>
    )
  }

  return (
    <Modal
      show={props.show}
      onHide={props.secondaryAction}
      animation={true}
      aria-labelledby="contained-modal-title-vcenter"
      centered
    >

      {title}

      <Modal.Body>
        {props.body}
      </Modal.Body>

      <Modal.Footer>
        <Button variant="secondary" className="mr-1" onClick={props.secondaryAction}>
          {props.secondaryActionName}
        </Button>

        <Button variant="success" onClick={props.primaryAction}>
          {props.primaryActionName}
        </Button>
      </Modal.Footer>
    </Modal>
  )
}