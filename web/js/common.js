window.onload = get_notification_url_api();

const notificaitonTypeName = {
  2: "NO_HARDHAT",
  3: "NO_MASK",
  4: "NO_SAFETY_VEST",
}

let notificaitonProfileSetting

window.addEventListener("load", async function () {
  let data = await get_user_profile_setting();
  notificaitonProfileSetting = data[1].map(item => notificaitonTypeName[item])

})

async function get_user_profile_setting() {
  response = await fetch(`/get_setting_profile_api`)
  if (response.status === 401) {
    window.location.replace("/?unauth=true")
    throw new Error("not login")
  }
  data = await response.json()
  return [data[0].result[0]["profileSetting"], data[0].result[0]["notificaitonProfileSetting"]]
}


$(function () {
  var includes = $("[data-include]");
  $.each(includes, function () {
    var file = $(this).data("include") + ".html";
    $(this).load(file);
  });
});

function logout() {
  Swal.fire({
    title: 'Do you confirm logout?', showCancelButton: true, confirmButtonText: 'Yes',
  }).then((result) => {
    if (result.isConfirmed) {
      window.location.replace("/logout");
    }
  })
}
function dropdown() {
  document.querySelector("#drop-down-area").classList.toggle("hidden");
  document.querySelector("#drop-down-icon").classList.toggle("rotate-180");
}


function updateDateTime() {
  $("#date-time-now").text(new Date().toLocaleString());
}
setInterval(updateDateTime, 1000);

async function get_notification_url_api() {

  response = await fetch(`/get_notification_url`)
  if (response.status === 401) {
    window.location.replace("/?unauth=true")
    throw new Error("not login")
  }
  url = await response.json()
  get_notification(url)
}

function get_notification(url) {
  const socket = new WebSocket(`ws://${url}`);

  socket.onmessage = function (message) {
    const data = JSON.parse(message.data);
    const { camID, workplace, classType } = data;

    const set1 = new Set(classType)
    const set2 = new Set(notificaitonProfileSetting)
    const intersection = new Set([...set1].filter(x => set2.has(x)));

    if (intersection.size == 0) {
      return;
    }

    const Toast = Swal.mixin({
      toast: true,
      position: "top-end",
      showConfirmButton: false,
      timer: 5000,
      timerProgressBar: true,
      didOpen: (toast) => {
        toast.onmouseenter = Swal.stopTimer;
        toast.onmouseleave = Swal.resumeTimer;
      }
    });
    Toast.fire({
      icon: "warning",
      title: `Violation detected at ${workplace} by camera ${camID}!`
    });


  };
  socket.onerror = function (error) {
    console.log(`[error] ${error}`);
  };
}

