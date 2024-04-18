import base64
from concurrent import futures
import contextlib
from dataclasses import dataclass
from dataclasses import field
import sys
import threading
import time
from typing import NoReturn, Self

import cv2
import grpc

sys.path.extend([".."])
from logging import Logger
import tomllib

from common.logger import get_logger
from protos.proto_pb2 import Image
from protos.proto_pb2_grpc import add_AnalysisServicer_to_server
from protos.proto_pb2_grpc import AnalysisServicer

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
    frame_counter: int = field(init=False, default=0)

    def __post_init__(self) -> None:
        self.__cam_ip = CAM_URL
        self.__cam_port = CAM_PORT
        self.__logger = get_logger("Cam Server")
        self.__server = grpc.server(futures.ThreadPoolExecutor())
        self.__cap = cv2.VideoCapture(CAMERA)

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
            self.frame_counter += 1
            if self.frame_counter == self.__cap.get(cv2.CAP_PROP_FRAME_COUNT):
                self.frame_counter = 0
                self.__cap.set(cv2.CAP_PROP_POS_FRAMES, 0)

            if ret:
                FRAME_QUEUE.put(frame)  # Use a queue to handle frames

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
            yield Image(data=base64.b64encode(buffer))
            time.sleep(1 / 24)


if __name__ == "__main__":
    with CamServer() as server:
        server.start()
