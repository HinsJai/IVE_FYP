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
import numpy as np
from surrealdb import Surreal
from typing_extensions import override
from ultralytics import YOLO

sys.path.extend([".."])


from item_getter import FrameGetter
from logger import get_logger
from protos.proto_pb2 import Class, Image, LogResponse, Response
from protos.proto_pb2_grpc import (
    AnalysisServicer,
    DiscordLogStub,
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

del config


class AnalysisServer:
    def __init__(self) -> None:
        self.__logger = get_logger("Model Server")
        self.__server = grpc.server(futures.ThreadPoolExecutor())
        self.__frame_getter = FrameGetter(CAM_URL, CAM_PORT)
        self.__model = YOLO("../model/ppe.pt")
        self.__reset_counter()
        self.__logger_channel = grpc.insecure_channel(
            f"{DISCORD_LOGGER_URL}:{DISCORD_LOGGER_PORT}"
        )
        self.__logger_stub = DiscordLogStub(self.__logger_channel)
        self.__dc_queue: Queue[tuple[str, bytes]] = Queue()
        self.__db_queue: Queue[list[str]] = Queue()
        self.__dc_worker = threading.Thread(target=self.__dc_worker)
        self.__db_worker = threading.Thread(target=self.__db_worker)

    def __enter__(self) -> Self:
        add_AnalysisServicer_to_server(AnalysisService(self), self.__server)
        self.__server.add_insecure_port(f"{SERVER_URL}:{SERVER_PORT}")
        return self

    def log_to_console(self, info: str) -> None:
        self.__logger.info(info)

    def __exit__(self, _, __, ___) -> None:
        self.__server.stop(None)
        self.__logger_channel.close()

    def start(self) -> None:
        self.log_to_console(f"Server started at {SERVER_URL}:{SERVER_PORT}")
        self.__server.start()
        self.__frame_getter.start()
        self.__dc_worker.start()
        self.__db_worker.start()
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

    def __log_to_discord(self, message: str, image: bytes) -> None:
        self.__logger_stub.log(LogResponse(message=message, image=Image(data=image)))

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
            self.__log_to_discord(message, image)
            self.__dc_queue.task_done()

    # store the action of the worker in queue
    def __log(self):
        violation_types = [
            violation
            for violation, count in self.__violation_counter.items()
            if count >= 2
        ]
        self.__db_queue.put(violation_types)
        dc_pair = (
            f"Violation detected at workplace TY-IVE {violation_types}",
            cv2.imencode(".png", self.frame)[1].tobytes(),
        )
        self.__dc_queue.put(dc_pair)
        self.__reset_counter()

    @property
    def boxes(self) -> list[dict[str, int]] | None:
        self.frame_data = base64.b64decode(self.__frame_getter.frame)
        self.np_data = np.frombuffer(self.frame_data, dtype=np.uint8)
        self.frame = cv2.imdecode(self.np_data, flags=cv2.IMREAD_COLOR)
        if not (result := self.__model.predict(self.frame)):
            return None

        boxes = []
        for box in result[0].boxes:
            if time.time() - self.__time_s >= 15 and int(box.cls[0]) in {2, 3, 4}:
                _type = Class.Name(int(box.cls[0]))
                self.__violation_counter[_type] += 1
                self.__count += 1

            if ceil((box.conf[0] * 100)) > 0.5 * 100:
                boxes.append(
                    {
                        "x1": int(box.xyxy[0][0]),
                        "y1": int(box.xyxy[0][1]),
                        "x2": int(box.xyxy[0][2]),
                        "y2": int(box.xyxy[0][3]),
                        "class_type": int(box.cls[0]),
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
