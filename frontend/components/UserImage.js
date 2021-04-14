import Avatar from 'react-avatar';

export default function UserImage(props) {
  let id = props.id
  let name = props.name
  let pictureUrl = props.pictureUrl
  let onClick = props.onClick
  let classes = props.classes
  let size = props.size

  return (
    <span key={id} className={classes} onClick={onClick}>
      <Avatar src={pictureUrl} color={"#969696"} name={name} round={true} size={size} />
    </span>
  )
}