import { toast, ToastContainer } from 'react-toastify';
import 'react-toastify/dist/ReactToastify.css';

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
    autoClose: 5000,
    hideProgressBar: true,
    closeOnClick: true,
    pauseOnHover: true,
    draggable: true,
    progress: undefined,
  });
}

function showErrorToastWithError(msg, error) {
  let errorMsg;

  if (error.response) {
    errorMsg = error.response.data;

  } else {
    errorMsg = error.message;
  }

  showErrorToast(
    <div>
      <strong>{msg}</strong>
      <br/>Error: {errorMsg}
    </div>
  )
}

function showErrorToast(msg) {
  toast.error(msg, {
    position: "top-right",
    autoClose: 5000,
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