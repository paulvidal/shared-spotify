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
import {useRouter} from "next/router";
import {encodeParams, getUrl} from "../utils/urlUtils";

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