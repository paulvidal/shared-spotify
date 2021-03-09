import styles from "../styles/HeaderFooter.module.scss";
import Link from "next/link";
import {Image} from "react-bootstrap";
import {useRouter} from "next/router";
import Menu from "./Menu";
import {useState} from "react";
import setState from "../utils/stateUtils";

export default function Header(props) {
  const router = useRouter()
  const [menu, setMenu] = useState({
    openMenu: false
  });

  let open = () => {
    setState(setMenu, {openMenu: true})
  }

  let closeCallback = () => {
    setState(setMenu, {openMenu: false})
  }

  let menuDisplay = [
    <div className={styles.home_block} onClick={open} key={"home"}>
      <svg className={styles.home_icon} fill="#000000" xmlns="http://www.w3.org/2000/svg" data-name="Layer 1"
           viewBox="0 0 100 100" x="0px" y="0px"><title>Essential Icons</title>
        <path d="M80,28H20a3,3,0,0,1,0-6H80a3,3,0,0,1,0,6Z"/>
        <path d="M80,53H20a3,3,0,0,1,0-6H80a3,3,0,0,1,0,6Z"/>
        <path d="M80,78H20a3,3,0,0,1,0-6H80a3,3,0,0,1,0,6Z"/>
      </svg>
    </div>,

    <div className={styles.arrow_block} key={"back"}>
      <a onClick={() => router.back()}>
        <svg className={styles.arrow_icon} viewBox="0 0 206 152" version="1.1" xmlns="http://www.w3.org/2000/svg"
             xmlnsXlink="http://www.w3.org/1999/xlink">
          <g id="back-button" stroke="none" strokeWidth="1" fill="none" fillRule="evenodd">
            <g id="noun_back_3324482" fill="#000000" fillRule="nonzero">
              <path
                d="M129.190656,152 L13.8025526,152 C8.08806669,152 3.45556054,148.31174 3.45556054,143.762041 C3.45556054,139.212341 8.08806669,135.524081 13.8025526,135.524081 L129.190656,135.524081 C149.237819,135.524081 167.762162,127.009022 177.785744,113.186442 C187.809326,99.3638631 187.809326,82.333743 177.785744,68.5111638 C167.762162,54.6885845 149.237819,46.1735247 129.190656,46.173525 L35.2948072,46.173525 L55.044111,61.8973183 C57.7243922,63.9673329 58.795842,67.0265627 57.8476759,69.9021256 C56.8995098,72.7776885 54.0788438,75.0234168 50.4670966,75.7783177 C46.8553494,76.5332187 43.0129141,75.680163 40.4129469,73.5462049 L3.00081024,43.7602148 C2.84870946,43.6391168 2.74213544,43.4978358 2.60141634,43.3709712 C2.44103797,43.2272188 2.29411068,43.0826426 2.14666604,42.9273571 C1.60542506,42.3822861 1.15343076,41.784635 0.801557072,41.1487817 L0.756547656,41.0894684 L0.75189151,41.0734043 C0.415480648,40.3971537 0.193826401,39.6882089 0.0927881138,38.9653105 C0.0658859344,38.7956085 0.0451919502,38.6304375 0.0317408605,38.4603236 C-0.0380372391,37.7514144 0.00734674869,37.0383 0.166769107,36.3386371 L0.186428392,36.2863261 C0.372915943,35.60758 0.666878733,34.9504293 1.06126657,34.3306345 C1.1574936,34.1737014 1.25579002,34.0233586 1.36443344,33.8709564 C1.80802615,33.2287087 2.35806788,32.6368638 2.99977554,32.1113282 L40.4119122,2.32492627 C44.4696076,-0.808882484 50.9352868,-0.769735218 54.9326506,2.4128439 C58.9300144,5.59542302 58.9791839,10.7431995 55.0430763,13.9738129 L35.2948072,29.6996656 L129.190656,29.6996656 C171.590539,29.7375718 205.950961,57.0933705 206,90.8508625 C205.950105,124.607738 171.589767,151.962549 129.190656,152 Z"
                id="Path"/>
            </g>
          </g>
        </svg>
      </a>
    </div>,

    <Menu open={menu.openMenu} closeCallback={closeCallback} key={"menu"}/>
  ]

  if (props.hideMenu) {
    menuDisplay = null;
  }

  return (
    <header className={styles.header}>
      <div className={"text-center mt-3 " + styles.header_block}>
        <Link href="/">
          <a className="float-left">
            <Image src="/logo.png" className={styles.header_logo}/>
          </a>
        </Link>

        {menuDisplay}
      </div>
    </header>
  )
}