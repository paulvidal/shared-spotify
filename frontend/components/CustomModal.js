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

  let footer;

  if (props.secondaryAction || props.primaryAction) {
    footer = (
      <Modal.Footer>
        <Button variant="secondary" className="mr-1" onClick={props.secondaryAction}>
          {props.secondaryActionName}
        </Button>

        <Button variant="success" onClick={props.primaryAction}>
          {props.primaryActionName}
        </Button>
      </Modal.Footer>
    )
  }

  return (
    <Modal
      show={props.show}
      onHide={props.onHideAction}
      animation={true}
      aria-labelledby="contained-modal-title-vcenter"
      centered
    >

      {title}

      <Modal.Body>
        {props.body}
      </Modal.Body>

      {footer}
    </Modal>
  )
}