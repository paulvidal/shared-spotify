import AvatarGroup from "@material-ui/lab/AvatarGroup";
import {isEmpty} from "lodash";
import {Avatar} from "@material-ui/core";
import {getInitials} from "../utils/name";
import styles from "../styles/rooms/[roomId]/playlists/Playlist.module.scss";

export default function UserImageGrouped(props) {
  let users = props.users

  return (
    <AvatarGroup max={3} classes={{
      'avatar': styles.user_pic
    }}>
      {users.map(user => {
        return (
          <Avatar key={user.id} src={user.image}>
            {getInitials(user.name)}
          </Avatar>
        )
      })}
    </AvatarGroup>
  )
}