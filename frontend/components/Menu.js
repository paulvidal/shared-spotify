import React from 'react';
import clsx from 'clsx';
import {makeStyles} from '@material-ui/core/styles';
import List from '@material-ui/core/List';
import Divider from '@material-ui/core/Divider';
import ListItem from '@material-ui/core/ListItem';
import ListItemText from '@material-ui/core/ListItemText';
import ListItemIcon from "@material-ui/core/ListItemIcon";
import HomeIcon from '@material-ui/icons/Home';
import {MeetingRoom} from "@material-ui/icons";
import Drawer from "@material-ui/core/Drawer";
import GitHubIcon from '@material-ui/icons/GitHub';
import LocalCafeIcon from '@material-ui/icons/LocalCafe';
import {useRouter} from "next/router";
import {getUrl} from "../utils/urlUtils";

const useStyles = makeStyles({
  list: {
    width: 250,
  },
  fullList: {
    width: 'auto',
  },
});

export default function Menu(props) {
  const router = useRouter()
  const classes = useStyles();

  const closeDrawer = () => {
    props.closeCallback()
  };

  const list = () => (
    <div
      className={clsx(classes.list)}
      role="presentation"
      onClick={() => closeDrawer()}
    >
      <List>
        <ListItem button key={"home"} onClick={() => router.push("/")}>
          <ListItemIcon><HomeIcon/></ListItemIcon>
          <ListItemText primary={"Home"}/>
        </ListItem>
      </List>
      <Divider/>
      <List>
        <ListItem button key={"logout"} onClick={() => window.location.assign(getUrl('/logout'))}>
          <ListItemIcon><MeetingRoom/></ListItemIcon>
          <ListItemText primary={"Logout"}/>
        </ListItem>
      </List>
      <List style={{position: "absolute", bottom: 0, width: "100%"}}>
        <Divider/>
        <ListItem button key={"coffee"} onClick={() => window.location.assign("https://www.buymeacoffee.com/paulvidal")}>
          <ListItemIcon><LocalCafeIcon/></ListItemIcon>
          <ListItemText primary={"Buy me a coffee"}/>
        </ListItem>
        <ListItem button key={"github"} onClick={() => window.location.assign("https://github.com/paulvidal/shared-spotify")}>
          <ListItemIcon><GitHubIcon/></ListItemIcon>
          <ListItemText primary={"Github project"}/>
        </ListItem>
      </List>
    </div>
  );

  return (
    <div>
      <React.Fragment>
        {/*<Button>{anchor}</Button>*/}
        <Drawer
          anchor={'right'}
          open={props.open}
          onClose={closeDrawer}>
          {list()}
        </Drawer>
      </React.Fragment>
    </div>
  );
}