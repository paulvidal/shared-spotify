import Avatar from "@material-ui/core/Avatar";
import {getInitials} from "../utils/name";

export default function UserImage(props) {
  let name = props.name
  let pictureUrl = props.pictureUrl
  let onClick = props.onClick
  let classes = props.classes

  return (
    <Avatar className={classes} src={pictureUrl} onClick={onClick}>
      {getInitials(name)}
    </Avatar>
  )
}