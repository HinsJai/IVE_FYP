import asyncio
import base64
import contextlib
import sys
import threading
import time
import tomllib
from concurrent import futures
from math import ceil
from queue import Queue
from typing import Any, Self

import cv2
import grpc
import joblib
import numpy as np
from surrealdb import Surreal
from typing_extensions import override
from ultralytics import YOLO

sys.path.extend([".."])


from common.item_getter import FrameGetter
from common.logger import get_logger
from protos.proto_pb2 import Class, Image, LogRequest, NotificationRequest, Response
from protos.proto_pb2_grpc import (
    AnalysisServicer,
    DiscordLogStub,
    Violation_NotificationStub,
    add_AnalysisServicer_to_server,
)

with open("config.toml", "rb") as config:
    config = tomllib.load(config)

DB_URL = config["database"]["url"]
DB_USER = config["database"]["user"]
DB_PASS = config["database"]["pass"]
DB_NAME_SPACE = config["database"]["namespace"]
DB_DATABASE = config["database"]["database"]
CAM_URL = config["cam"]["url"]
CAM_PORT = config["cam"]["port"]
SERVER_URL = config["server"]["url"]
SERVER_PORT = config["server"]["port"]
DISCORD_LOGGER_URL = config["discord"]["ip"]
DISCORD_LOGGER_PORT = config["discord"]["port"]
GRPC_NOTIFICATION_PORT = config["grpc_notification"]["port"]
GRPC_NOTIFICATION_URL = config["grpc_notification"]["url"]

del config


def image_to_histogram(image, bins=32):
    histogram = []
    for i in range(3):  # Assuming the image is in BGR format
        hist = cv2.calcHist([image], [i], None, [bins], [0, 256])
        histogram.extend(hist.flatten())
    return histogram


class AnalysisServer:
    def __init__(self) -> None:
        self.__logger = get_logger("Model Server")
        self.__server = grpc.server(futures.ThreadPoolExecutor())
        self.__frame_getter = FrameGetter(CAM_URL, CAM_PORT)
        self.__model = YOLO("../model/best_ppe.pt")
        self.__helmet_model, self.__label_encoder = joblib.load(
            "../model/helmet_color_cls.pkl", mmap_mode="r"
        )
        self.__reset_counter()
        self.__logger_channel = grpc.insecure_channel(
            f"{DISCORD_LOGGER_URL}:{DISCORD_LOGGER_PORT}",
            options=[
                ("grpc.max_send_message_length", 1024*1024*1024),
                ("grpc.max_receive_message_length", 1024*1024*1024),
            ],
        )
        self.__notification_channel = grpc.insecure_channel(
            f"{GRPC_NOTIFICATION_URL}:{GRPC_NOTIFICATION_PORT}",
             options=[
                ("grpc.max_send_message_length", 1024*1024*1024),
                ("grpc.max_receive_message_length", 1024*1024*1024),
            ],
        )
        self.__notification_stub = Violation_NotificationStub(
            self.__notification_channel
        )
        self.__logger_stub = DiscordLogStub(self.__logger_channel)
        self.__notification_queue: Queue[tuple[str, list[str], str]] = Queue()
        self.__dc_queue: Queue[tuple[str, bytes]] = Queue()
        self.__db_queue: Queue[list[str]] = Queue()
        self.__dc_worker = threading.Thread(target=self.__dc_worker, daemon=True)
        self.__db_worker = threading.Thread(target=self.__db_worker, daemon=True)
        self.__get_db_notificaiton_worker = threading.Thread(
            target=self.__get_db_notificaiton_worker, daemon=True
        )
        self.__notificaiton_worker = threading.Thread(
            target=self.__notificaiton_worker, daemon=True
        )
        self.__notification_setting = []

    def __enter__(self) -> Self:
        add_AnalysisServicer_to_server(AnalysisService(self), self.__server)
        self.__server.add_insecure_port(f"{SERVER_URL}:{SERVER_PORT}")
        return self

    def log_to_console(self, info: str) -> None:
        self.__logger.info(info)

    def __exit__(self, _, __, ___) -> None:
        self.__server.stop(None)
        self.__logger_channel.close()
        self.__notification_channel.close()

    def start(self) -> None:
        self.log_to_console(f"Server started at {SERVER_URL}:{SERVER_PORT}")
        self.__server.start()
        self.__frame_getter.start()
        self.__dc_worker.start()
        self.__db_worker.start()
        self.__get_db_notificaiton_worker.start()
        self.__notificaiton_worker.start()
        with contextlib.suppress(KeyboardInterrupt):
            self.__server.wait_for_termination()

    def __reset_counter(self) -> None:
        self.__time_s = time.time()
        self.__count = 0
        self.__violation_counter = {"NO_HARDHAT": 0, "NO_MASK": 0, "NO_SAFETY_VEST": 0}

    @staticmethod
    async def __get_db_response(
        query: str, data: dict[str, Any] = None
    ) -> list[dict[str, Any]]:
        async with Surreal(DB_URL) as db:
            await db.signin({"user": DB_USER, "pass": DB_PASS})
            await db.use(DB_NAME_SPACE, DB_DATABASE)
            response = await db.query(query, data)
        return response[0]["result"]

    def __log_to_notification(
        self, camID: str, violation_types: list[str], workplace: str
    ) -> None:
        self.__notification_stub.notification(
            NotificationRequest(
                camID=camID, class_type=violation_types, workplace=workplace
            )
        )

    def __log_to_discord(self, message: str, image: bytes) -> None:
        self.__logger_stub.log(LogRequest(message=message, image=Image(data=image)))

    def __log_to_db(self, violation_types: list[str]) -> None:
        asyncio.run(
            AnalysisServer.__get_db_response(
                """
            insert into violation_record {
                cameraID: $cameraID,
                workplace: $workplace,
                violation_type: $violation_type,
            };
            """,
                data={
                    "cameraID": "c001",
                    "workplace": "TY-IVE",
                    "violation_type": violation_types,
                },
            )
        )

    # Listener
    def __db_worker(self) -> None:
        while violation_types := self.__db_queue.get():
            self.__log_to_db(violation_types)
            self.__db_queue.task_done()

    def __dc_worker(self) -> None:
        while True:
            message, image = self.__dc_queue.get()
            # set the image size in grpc

            self.__log_to_discord(message, image)
            self.__dc_queue.task_done()

    def __notificaiton_worker(self) -> None:
        while True:
            camID, violation_types, workplace = self.__notification_queue.get()
            self.__log_to_notification(camID, violation_types, workplace)
            self.__notification_queue.task_done()

    def __get_db_notificaiton_worker(self) -> None:
        while True:
            self.__notification_setting = asyncio.run(
                AnalysisServer.__get_db_response(
                    "SELECT notificaitonProfileSetting FROM setting where email = 'jason199794@gmail.com';"
                )
            )[0]["notificaitonProfileSetting"]
            time.sleep(10)

    # store the action of the worker in queue
    def __log(self) -> None:
        violation_types = [
            violation
            for violation, count in self.__violation_counter.items()
            if count >= 2
        ]
        notificaitonTypeName = {2: "NO_HARDHAT", 3: "NO_MASK", 4: "NO_SAFETY_VEST"}
        set1 = set(violation_types)
        set2 = set(map(lambda i: notificaitonTypeName[i], self.__notification_setting))

        intersection = set1.intersection(set2)

        if len(intersection) == 0:
            return

        violation_types = list(intersection)

        self.__db_queue.put(violation_types)
        dc_pair = (
            f"Violation detected on camera c001 at workplace TY-IVE {violation_types}",
            cv2.imencode(".png", self.frame)[1].tobytes(),
        )
        self.__dc_queue.put(dc_pair)
        self.__notification_queue.put(("c001", violation_types, "TY-IVE"))
        self.__reset_counter()

    def __colored_helmet(self, x1: int, y1: int, x2: int, y2: int) -> int:
        helmet_image = self.frame[y1:y2, x1:x2]
        # print(helmet_image)
        # if len(helmet_image) == 0:
        #     breakpoint()
        histogram = image_to_histogram(helmet_image)
        prediction = self.__helmet_model.predict([histogram])
        predicted_label = self.__label_encoder.inverse_transform(prediction)
        return int(10 + predicted_label[0])

    @property
    def boxes(self) -> list[dict[str, int]] | None:
        self.frame_data = base64.b64decode(self.__frame_getter.frame)
        self.np_data = np.frombuffer(self.frame_data, dtype=np.uint8)
        self.frame = cv2.imdecode(self.np_data, flags=cv2.IMREAD_COLOR)
        if not (result := self.__model.predict(self.frame, verbose=False)):
            return None

        # img = cv2.resize(img, (resize_width, resize_height))
        # results = model.predict(img, conf=conf, verbose=False)
        # names = results[0].names
        # class_detections_values = []
        # for k, v in names.items():
        #     class_detections_values.append(results[0].boxes.cls.tolist().count(k))
        # # create dictionary of objects detected per class
        # classes_detected = dict(zip(names.values(), class_detections_values))
        #  f"Total Person: {classes_detected['Person']}",
        # person_count = 0

        boxes = []
        for box in result[0].boxes:
            # if time.time() - self.__time_s >= 15 and int(box.cls[0]) in {2, 3, 4}:
            #     _type = Class.Name(int(box.cls[0]))
            #     self.__violation_counter[_type] += 1
            #     self.__count += 1

            if ceil((box.conf[0])) > 0.5:

                x1, y1, x2, y2 = tuple(map(int, box.xyxy[0]))

                class_type = int(box.cls[0])

                if time.time() - self.__time_s >= 15 and class_type in {2, 3, 4}:
                    _type = Class.Name(class_type)
                    self.__violation_counter[_type] += 1
                    self.__count += 1

                # if class_type == 0:
                #     helmet_image = self.frame[y1:y2][x1:x2]
                #     histogram = image_to_histogram(helmet_image)
                #     prediction = self.__helmet_model.predict([histogram])
                #     predicted_label = self.__label_encoder.inverse_transform(prediction)
                #     class_type = 10 + predicted_label

                boxes.append(
                    {
                        "x1": x1,
                        "y1": y1,
                        "x2": x2,
                        "y2": y2,
                        # "x1": int(box.xyxy[0][0]),
                        # "y1": int(box.xyxy[0][1]),
                        # "x2": int(box.xyxy[0][2]),
                        # "y2": int(box.xyxy[0][3]),
                        # "class_type": int(box.cls[0]),
                        "class_type": (
                            self.__colored_helmet(x1, y1, x2, y2)
                            if class_type == 0
                            else class_type
                        ),
                    }
                )

        if time.time() - self.__time_s >= 20:
            self.__reset_counter()

        if self.__count > 6 and max(self.__violation_counter.values()) >= 2:
            self.__log()
        return boxes


class AnalysisService(AnalysisServicer):
    def __init__(self, server: AnalysisServer) -> None:
        super().__init__()
        self.__server = server

    @override
    def analysis(self, _, context):
        self.__server.log_to_console(f"{context.peer()} connected")
        while True:
            if boxes := self.__server.boxes:
                yield Response(item=boxes)


if __name__ == "__main__":
    with AnalysisServer() as server:
        server.start()
