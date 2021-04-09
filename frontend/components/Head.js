import Head from "next/head";

export default function CustomHead(props) {
  return (
    <Head>
      <title>Shared Spotify</title>
      <meta property="og:title" content="Shared spotify" />
      <meta property="og:description" content="The best way to create playlist with songs multiple like!" />
      <meta name='description' content='he best way to create playlist with songs multiple like!'/>
      <meta property="og:image" content={process.env.NEXT_PUBLIC_URL + "/share.png"} />

      {/* progressive web app */}
      <meta charSet='utf-8'/>
      <meta name='viewport' content='minimum-scale=1, initial-scale=1, width=device-width, shrink-to-fit=no, user-scalable=no, viewport-fit=cover' />
      <link rel="manifest" href="manifest.json"/>
      <meta name="mobile-web-app-capable" content="yes"/>
      <meta name="apple-mobile-web-app-capable" content="yes"/>
      <meta name="application-name" content="Shared Spotify"/>
      <meta name="apple-mobile-web-app-title" content="Shared Spotify"/>
      <meta name="theme-color" content="#2ec651"/>
      <meta name="msapplication-navbutton-color" content="#2ec651"/>
      <meta name="apple-mobile-web-app-status-bar-style" content="black-translucent"/>
      <meta name="msapplication-starturl" content="/"/>
      <meta name="viewport" content="width=device-width, initial-scale=1, shrink-to-fit=no"/>
      <link rel="icon" href="/logo_no_name.png"/>
      <link rel="apple-touch-icon" href="/logo_no_name.png"/>

      {/* always allow referrer to exist */}
      <meta name="referrer" content="always" />

      {props.children}
    </Head>
  )
}