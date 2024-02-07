var boxes = [[], [], [], []];
var images = [[], [], [], []];
var server_availabilties = [true, true, true, true];
const normal_timeout = 0;
const error_timeout = 250;

var canvas_array = [];

function get_notification(url) {
  const socket = new WebSocket(`ws://${url}`);
  socket.onmessage = function (message) {
    const data = JSON.parse(message.data);
    const { camID, workplace, classType } = data;
    const notification = document.getElementById("notic-container");
    const newNotification = document.createElement("div");

    newNotification.innerHTML = `<p class="text-red-500 font-semibold">CamID:<span class="text-slate-50 font-semibold"> ${camID}</span></p><br>
        <p class="text-red-500 font-semibold">Workplace:<span class="text-slate-50 font-semibold"> ${workplace}</span></p><br>
        <p class="text-red-500 font-semibold">Violation:<span class="text-slate-50 font-semibold"> ${classType}</span></p><br>
        <hr class="mb-2">`;

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
    const canvas_width =
      (1003.97 / (2552 - 224 - 288)) * (innerWidth - 224 - 288);
    const canvas_height = (279.2 / 708) * innerHeight;
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
    let box = JSON.parse(boxes[stream_source]);
    for (let i = 0; i < box.length; ++i) {
      const scaledX1 = box[i].x1 * (canvas.width / image.width);
      const scaledY1 = box[i].y1 * (canvas.height / image.height);
      const scaledWidth =
        (box[i].x2 - box[i].x1) * (canvas.width / image.width);
      const scaledHeight =
        (box[i].y2 - box[i].y1) * (canvas.height / image.height);
      class_type = box[i].class_type;
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
    case 0:
    case 1:
    case 7:
      return "green";
    case 2:
    case 3:
    case 4:
      return "red";
    default:
      return "blue";
  }
}

const classEnum = {
  0: "Hardhat",
  1: "Mask",
  2: "No Hardhat",
  3: "No Mask",
  4: "No Safety Vest",
  5: "Person",
  6: "Safety Cone",
  7: "Safety Vest",
  8: "Machinery",
  9: "Vehicle",
};

function getClass(class_type) {
  return classEnum[class_type];
}
