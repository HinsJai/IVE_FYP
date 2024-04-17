let boxes = [[], [], [], []];
let images = [[], [], [], []];
let person_count = [0, 0, 0, 0];
let server_availabilties = [true, true, true, true];
let helment_roles = {};
let showing_items = [];
const normal_timeout = 0;
const error_timeout = 250;

let canvas_array = [];

let setting_array = []

let json_result //= get_user_profile_setting();

// const notificaitonTypeName = {
//   2: "NO_HARDHAT",
//   3: "NO_MASK",
//   4: "NO_SAFETY_VEST",
// }

//let notificaitonProfileSetting = Object.values(notificaitonTypeName)

window.addEventListener("load", async function () {
 
  let data = await get_user_profile_setting();
  showing_items = data[0]
  //notificaitonProfileSetting = data[1].map(item => notificaitonTypeName[item])
  helment_roles = await get_helment_roles();
  for (const [key, value] of Object.entries(helment_roles)) {
    classEnum[key] = value;
  }
})

function getTimeNow() {
  const now = new Date();
  const hours = now.getHours();
  const minutes = now.getMinutes();

  const formattedHours = hours.toString().padStart(2, '0');
  const formattedMinutes = minutes.toString().padStart(2, '0');

  return `${formattedHours}:${formattedMinutes}`;
}
function get_notification(url) {
  const socket = new WebSocket(`ws://${url}`);
 
  socket.onmessage = function (message) {
    const data = JSON.parse(message.data);
    let { camID, workplace, classType } = data;


    const set1 = new Set(classType)
    const set2 = new Set(notificaitonProfileSetting)

    const intersection = new Set([...set1].filter(x => set2.has(x)));

    if(intersection.size == 0) {
      return;
    }

    classType = Array.from(intersection).join(', ')

    const notification = document.getElementById("notic-container");
    const newNotification = document.createElement("div");

    // newNotification.innerHTML = `<p class="text-red-500 font-semibold">CamID:<span class="text-slate-50 font-semibold"> ${camID}</span></p><br>
    //   <p class="text-red-500 font-semibold">Workplace:<span class="text-slate-50 font-semibold"> ${workplace}</span></p><br>
    //   <p class="text-red-500 font-semibold">Violation:<p class="text-slate-50 font-semibold"> ${classType}</p></p><br>
    //   <hr class="mb-2">`;

    newNotification.innerHTML =
      `<div
    class="relative flex w-full p-1 text-base text-gray-900 border border-white-900 rounded-lg font-regular mb-2"
    style="opacity: 1;">
    <div class="shrink-0 text-yellow-500">
        <i class="fa-solid fa-circle-exclamation"></i>
    </div>
    <div class="ml-3 p-1">
        <p class="block text-yellow-400 font-semibold text-xl antialiased leading-relaxed ">
            New violation detected
        </p>
        <ul class="mt-2 ml-2 list-disc list-inside">
        <li class="text-red-500 font-semibold">
        Time : <span class="text-slate-50 font-semibold">${getTimeNow()}</span>
        </li>
            <li class="text-red-500 font-semibold">
                CamID : <span class="text-slate-50 font-semibold">${camID}</span>
            </li>
            <li class="text-red-500 font-semibold">
                Workplace : <span class="text-slate-50 font-semibold">${workplace}</span>
            </li>
            <li class="text-red-500 font-semibold">
                Violation : <span class="text-slate-50 font-semibold">${classType}</span>
            </li>
        </ul>
    </div>
</div>`

    notification.insertBefore(newNotification, notification.firstChild);

    while (notification.children.length > 4) {
      notification.removeChild(notification.lastChild);
    }
  };
  socket.onerror = function (error) {
    console.log(`[error] ${error}`);
  };
}

function get_data(storage, stream_source, data_type, url) {
  const socket = new WebSocket(
    `ws://${url}/${data_type}?stream_source=${stream_source}`
  );
  socket.onmessage = function (response) {
    if (response.data == "No data available") {
      server_availabilties[stream_source] = false;
      throw new Error("No data available");
    }
    storage[stream_source] = response.data;
  };
  socket.onerror = function (error) {
    if (error.message == "No data available") {
      socket.close();
    }
  };
}

function create_resize_function(canvas) {
  let scale = window.devicePixelRatio;
  return function () {
    let canvas_width =
      (1003.97 / (2552 - 224 - 288)) * (innerWidth - 224 - 288);
    let canvas_height = (279.2 / 708) * innerHeight;
    //canvas_width *= 2;
    //canvas_height *= 2;
    canvas.style.width = canvas_width + "px";
    canvas.style.height = canvas_height + "px";
    canvas.width = canvas_width * scale;
    canvas.height = canvas_height * scale;
    let ctx = canvas.getContext("2d");
    if (canvas.firstTimeResize) {
      ctx.scale(scale, scale);
      canvas.firstTimeResize = false;
    }
  };
}

function load_canvas(id) {
  let canvas = document.getElementById(`canvas ${id}`);
  canvas.firstTimeResize = true;
  window.addEventListener("load", create_resize_function(canvas));
  window.addEventListener("resize", create_resize_function(canvas));
}

function setup_images_box(stream_source, url) {
  get_data(images, stream_source, "image", url);
  get_data(boxes, stream_source, "box", url);
  update_image(stream_source);
}

function update_image(stream_source) {
  if (server_availabilties[stream_source] == false) {
    document.getElementById(`cam placeholder ${stream_source}`).hidden = false;
    return;
  }
  let arrayBuffer = images[stream_source];
  let urlObject = URL.createObjectURL(new Blob([arrayBuffer]));
  let image = new Image();
  image.src = urlObject;
  let canvas = document.getElementById(`canvas ${stream_source}`);
  let ctx = canvas.getContext("2d");
  image.addEventListener("load", function () {
    ctx.drawImage(image, 0, 0, canvas.width, canvas.height);
    person_count[stream_source] = 0;
    let box = JSON.parse(boxes[stream_source]);
    for (let i = 0; i < box.length; ++i) {
      if (box[i].class_type == 5) {
        person_count[stream_source] += 1;
      }
    }
    for (let i = 0; i < box.length; ++i) {
      let class_type = box[i].class_type;
      if (!(showing_items.includes(class_type))) {
        continue
      }
      const scaledX1 = box[i].x1 * (canvas.width / image.width);
      const scaledY1 = box[i].y1 * (canvas.height / image.height);
      const scaledWidth =
        (box[i].x2 - box[i].x1) * (canvas.width / image.width);
      const scaledHeight =
        (box[i].y2 - box[i].y1) * (canvas.height / image.height);
      //class_type = box[i].class_type;
      box_color = getBoxColor(class_type);
      ctx.beginPath();
      ctx.strokeStyle = box_color;
      ctx.lineWidth = 2;
      ctx.font = "lighter 12px Arial";
      ctx.fillStyle = "red";
      ctx.fillText(getClass(class_type), scaledX1, scaledY1 - 5);
      ctx.rect(scaledX1, scaledY1, scaledWidth, scaledHeight);
      ctx.stroke();
    }



    ctx.beginPath();
    ctx.fillStyle = "green";
    ctx.font = "lighter 24px Arial";
    ctx.fillText(`Person Count: ${person_count[stream_source]}`, 10, 40);
    ctx.fill();
    ctx.stroke();

    setTimeout(update_image, normal_timeout, stream_source);
    URL.revokeObjectURL(urlObject);
  });

  image.addEventListener("error", function () {
    image.src = "../images/cam_unavailable.jpg";
    image.hidden = false;
    setTimeout(update_image, error_timeout, stream_source);
  });
}

function getBoxColor(class_type) {
  switch (class_type) {
    case 1:
    case 7:
    case 10:
    case 11:
    case 12:
    case 13:
      return "green";
    case 2:
    case 3:
    case 4:
      return "red";
    default:
      return "blue";
  }
}

let classEnum = {
  1: "Mask",
  2: "No Hardhat",
  3: "No Mask",
  4: "No Safety Vest",
  5: "Person",
  6: "Safety Cone",
  7: "Safety Vest",
  8: "Machinery",
  9: "Vehicle",
  10: "Technicaler",
  11: "Signaller",
  12: "Supervisor",
  13: "Worker",
};

function getClass(class_type) {
  return classEnum[class_type];
}
