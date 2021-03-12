import styles from "../styles/rooms/Rooms.module.scss";
import CustomHead from "./Head";
import Header from "./Header";
import {Toast} from "./toast";
import {Spinner} from "react-bootstrap";
import Footer from "./Footer";

export default function LoaderScreen(props) {
  return (
    <div className={styles.container}>
      <CustomHead />

      <Header />

      <main className={styles.main}>
        {props.children}

        <Spinner animation="border" variant="success"/>
      </main>

      <Footer/>

      <Toast/>
    </div>
  )
}