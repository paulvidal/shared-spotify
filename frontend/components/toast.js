import { toast, ToastContainer } from 'react-toastify';
import 'react-toastify/dist/ReactToastify.css';
import {getUrl} from "../utils/urlUtils";
import Link from 'next/link'

function Toast() {
  return (
    <ToastContainer
      position="top-right"
      autoClose={5000}
      hideProgressBar
      newestOnTop={false}
      closeOnClick
      rtl={false}
      pauseOnFocusLoss
      draggable
      pauseOnHover
    />
  )
}

function showSuccessToast(msg) {
  toast.success(msg, {
    position: "top-right",
    autoClose: 3000,
    hideProgressBar: true,
    closeOnClick: true,
    pauseOnHover: true,
    draggable: true,
    progress: undefined,
  });
}

function showErrorToastWithError(msg, error) {
  let errorMsg = error.message;
  let loginUrl;

  if (error.response) {
    if (error.response.data) {
      errorMsg = error.response.data;
    }

    // Unauthorised code
    if (error.response.status === 401) {
      const style = {
        textDecoration: "underline"
      }

      loginUrl = (
        <Link href={getUrl('/login')}>
          <div>
            ➡️ <a className="text-white" style={style}>Click here to login</a>
          </div>
        </Link>
      )
    }
  }

  showErrorToast(
    <div>
      <strong>{msg}</strong>
      <br/>
      {errorMsg}
      <br/>
      {loginUrl}
    </div>
  )
}

function showErrorToast(msg) {
  toast.error(msg, {
    position: "top-right",
    autoClose: 10000,
    hideProgressBar: true,
    closeOnClick: true,
    pauseOnHover: true,
    draggable: true,
    progress: undefined,
  });
}

export {
  Toast,
  showSuccessToast,
  showErrorToast,
  showErrorToastWithError
}