import Head from "next/head";

export default function CustomHead(props) {
  return (
    <Head>
      <title>Shared Spotify</title>
      <link rel="icon" href="/logo.svg" />
      <meta property="og:title" content="Shared spotify" />
      <meta property="og:description" content="The best way to create playlist with songs multiple like!" />
      <meta name='description' content='he best way to create playlist with songs multiple like!'/>
      <meta property="og:image" content={process.env.NEXT_PUBLIC_URL + "/share.png"} />
      <meta charSet='utf-8'/>
      <meta name='viewport' content='minimum-scale=1, initial-scale=1, width=device-width, shrink-to-fit=no, user-scalable=no, viewport-fit=cover' />
      <link rel="manifest" href="/manifest.json"/>

      {/* icons */}
      <link rel="apple-touch-icon" sizes="57x57" href="/apple-icon-57x57.png"/>
      <link rel="apple-touch-icon" sizes="60x60" href="/apple-icon-60x60.png"/>
      <link rel="apple-touch-icon" sizes="72x72" href="/apple-icon-72x72.png"/>
      <link rel="apple-touch-icon" sizes="76x76" href="/apple-icon-76x76.png"/>
      <link rel="apple-touch-icon" sizes="114x114" href="/apple-icon-114x114.png"/>
      <link rel="apple-touch-icon" sizes="120x120" href="/apple-icon-120x120.png"/>
      <link rel="apple-touch-icon" sizes="144x144" href="/apple-icon-144x144.png"/>
      <link rel="apple-touch-icon" sizes="152x152" href="/apple-icon-152x152.png"/>
      <link rel="apple-touch-icon" sizes="180x180" href="/apple-icon-180x180.png"/>
      <link rel="icon" type="image/png" sizes="192x192"  href="/android-icon-192x192.png"/>
      <link rel="icon" type="image/png" sizes="32x32" href="/favicon-32x32.png"/>
      <link rel="icon" type="image/png" sizes="96x96" href="/favicon-96x96.png"/>
      <link rel="icon" type="image/png" sizes="16x16" href="/favicon-16x16.png"/>
      <link rel="manifest" href="/manifest.json"/>
      <meta name="msapplication-TileColor" content="#ffffff"/>
      <meta name="msapplication-TileImage" content="/ms-icon-144x144.png"/>
      <meta name="theme-color" content="#ffffff"/>

      {/* always allow referrer to exist */}
      <meta name="referrer" content="always" />

      {props.children}
    </Head>
  )
}