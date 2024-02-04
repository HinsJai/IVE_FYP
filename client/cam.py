import base64
import contextlib
import sys
import threading
import time
from concurrent import futures
from dataclasses import dataclass, field
from typing import NoReturn, Self

import cv2
import grpc

# from numpy.typing import NDArray

sys.path.extend([".."])
import tomllib
from logging import Logger

from logger import get_logger
from protos.proto_pb2 import Image
from protos.proto_pb2_grpc import (AnalysisServicer,
                                   add_AnalysisServicer_to_server)

# Load configuration once and store it in constants
with open("config.toml", "rb") as config:
    config = tomllib.load(config)

CAM_URL = config["cam"]["url"]
CAM_PORT = config["cam"]["port"]
CAMERA = config["cam"]["camera"]
del config

from queue import Queue

FRAME_QUEUE = Queue(maxsize=10)


@dataclass(slots=True, repr=False)
class CamServer:
    __cam_ip: str = field(init=False)
    __cam_port: int = field(init=False)
    __server: grpc.Server = field(init=False)
    __logger: Logger = field(init=False)
    __cap: cv2.VideoCapture = field(init=False)

    def __post_init__(self) -> None:
        self.__cam_ip = CAM_URL
        self.__cam_port = CAM_PORT
        self.__logger = get_logger("Cam Server")
        self.__server = grpc.server(
            futures.ThreadPoolExecutor(max_workers=4)
        )  # Limit the number of threads
        self.__cap = cv2.VideoCapture(CAMERA)
        # self.__cap = cv2.VideoCapture("../test1080p.mp4")

    def __enter__(self) -> Self:
        self.__setup_camera()
        self.__setup_server()
        return self

    def log(self, info: str) -> None:
        self.__logger.info(info)

    def start(self) -> None:
        with contextlib.suppress(KeyboardInterrupt):
            self.__server.wait_for_termination()

    def __exit__(self, _, __, ___) -> None:
        self.__cap.release()
        self.__server.stop(None)
        self.__logger.info("Server Closed")

    def __read_cam(self) -> NoReturn:
        while True:
            ret, frame = self.__cap.read()
            if ret:
                FRAME_QUEUE.put(frame)  # Use a queue to handle frames
            # time.sleep(0.016)

    def __setup_server(self) -> None:
        add_AnalysisServicer_to_server(AnalysisService(self), self.__server)
        self.__server.add_insecure_port(f"{self.__cam_ip}:{self.__cam_port}")
        self.__logger.info(f"Server started at {self.__cam_ip}:{self.__cam_port}")
        self.__server.start()

    def __setup_camera(self) -> None:
        cam_ok, _ = self.__cap.read()
        if cam_ok:
            threading.Thread(
                target=self.__read_cam, daemon=True
            ).start()  # Use threading for reading frames


class AnalysisService(AnalysisServicer):
    def __init__(self, cam_server: CamServer) -> None:
        super().__init__()
        self.__cam_server = cam_server

    def get_image(self, _, context):
        self.__cam_server.log(f"{context.peer()} connected")
        while True:
            frame = FRAME_QUEUE.get()  # Get frames from the queue
            _, buffer = cv2.imencode(".jpg", frame)
            yield Image(data=base64.b64encode(buffer))  # Send as base64 encoded
            # time.sleep(1 / 60)  # Control the frame rate
            time.sleep(1 / 24)


if __name__ == "__main__":
    with CamServer() as server:
        server.start()
