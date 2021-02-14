import '../styles/globals.scss'
import { datadogRum } from "@datadog/browser-rum";
import { datadogLogs } from '@datadog/browser-logs';

let isDatadogInit = false;

function MyApp({ Component, pageProps }) {
  // Initialise the datadog rum client
  const applicationId = process.env.NEXT_PUBLIC_DATADOG_APPLICATION_ID
  const clientToken = process.env.NEXT_PUBLIC_DATADOG_CLIENT_TOKEN

  if (!isDatadogInit && applicationId && clientToken) {
    console.log("Datadog rum initialised")

    datadogRum.init({
      applicationId: applicationId,
      clientToken: clientToken,
      site: 'datadoghq.com',
      service: 'shared-spotify-frontend',
      sampleRate: 100,
      trackInteractions: true,
      silentMultipleInit: true
    });

    datadogLogs.init({
      clientToken: clientToken,
      site: 'datadoghq.com',
      service: 'shared-spotify-frontend',
      forwardErrorsToLogs: true,
      sampleRate: 100
    });

    isDatadogInit = true
  }

  return <Component {...pageProps} />
}

export default MyApp
