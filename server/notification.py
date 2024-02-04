import contextlib
import sys
import tomllib

sys.path.extend([".."])

import asyncio
from collections import deque
from concurrent import futures
from dataclasses import dataclass, field
from threading import Thread
from typing import Any, Self

import grpc
import websockets
from google.protobuf.json_format import MessageToJson

from logger import Logger, get_logger

try:
    from typing import Self, override
except ImportError:
    from typing_extensions import Self, override

from protos.proto_pb2 import Class, Empty
from protos.proto_pb2_grpc import (
    Violation_NotificationServicer,
    add_Violation_NotificationServicer_to_server,
)

with open("config.toml", "rb") as config:
    config = tomllib.load(config)

GRPC_NOTIFICATION_PORT = config["grpc_notification"]["port"]
GRPC_NOTIFICATION_URL = config["grpc_notification"]["url"]
WEBSOCKET_NOTIFICATION_PORT = config["websocket_notification"]["port"]
WEBSOCKET_NOTIFICATION_URL = config["websocket_notification"]["url"]


del config


@dataclass(slots=True, repr=False)
class NotificationServer:
    __server: grpc.server = field(init=False)
    __logger: Logger = field(init=False)
    __websocket_server_thread: Thread = field(init=False)

    def __post_init__(self) -> None:
        self.__logger = get_logger("Notification Server")
        self.__server = grpc.server(futures.ThreadPoolExecutor(max_workers=4))
        self.__websocket_server_thread = Thread(
            target=start_websocket_server,
            daemon=True,
            name="Notification Websocket Server",
        )

    def __enter__(self) -> Self:
        add_Violation_NotificationServicer_to_server(
            NotificationService(self), self.__server
        )
        self.__server.add_insecure_port(
            f"{GRPC_NOTIFICATION_URL}:{GRPC_NOTIFICATION_PORT}"
        )
        return self

    def log_to_console(self, info: str) -> None:
        self.__logger.info(info)

    def __exit__(self, _, __, ___) -> None:
        self.__server.stop(None)
        self.__logger.info("Server Closed")

    def start(self) -> None:
        self.__websocket_server_thread.start()
        self.log_to_console(
            f"Server started at {GRPC_NOTIFICATION_URL}:{GRPC_NOTIFICATION_PORT}"
        )
        self.__server.start()
        with contextlib.suppress(KeyboardInterrupt):
            self.__server.wait_for_termination()


class WebSocketNotificationServer:
    def __init__(self, host: str, port: int, with_history: bool = False) -> None:
        self.host = host
        self.port = port
        self.clients = set()
        self.logger = get_logger("Notification WebSocket Server")
        self.loop = asyncio.get_event_loop()
        self.history = (
            deque(maxlen=10) if with_history else None
        )  # if with_history is True, store last 10 messages and send to browswer

    async def register(self, websocket):
        self.clients.add(websocket)
        self.logger.info(f"Client {websocket.remote_address} connected")

    async def unregister(self, websocket) -> None:
        self.clients.remove(websocket)
        self.logger.info(f"Client {websocket.remote_address} disconnected")

    async def send_to_all(self, message) -> None:
        if self.history is not None:
            self.history.append(message)
        if len(self.clients) == 0:
            return
        async with asyncio.TaskGroup() as tg:
            for client in self.clients:
                tg.create_task(client.send(message))
        self.logger.info("Published message to all clients")

    async def handler(self, websocket, _) -> None:
        await self.register(websocket)
        if self.history is not None:
            for message in self.history:
                await websocket.send(message)
        try:
            async for message in websocket:
                self.logger.info(f"Received message: {message}")
        finally:
            await self.unregister(websocket)


class NotificationService(Violation_NotificationServicer):
    def __init__(self, server: NotificationServer) -> None:
        super().__init__()
        # self.__server = server

    @override
    def notification(self, request, context) -> Empty:
        # self.__server.log_to_console(
        #     f"{context.peer()} connected with inference server"
        # )
        asyncio.run_coroutine_threadsafe(
            ws_server.send_to_all(MessageToJson(request)), ws_server.loop
        )
        return Empty()


def start_websocket_server():
    global ws_server
    loop = asyncio.new_event_loop()
    asyncio.set_event_loop(loop)
    ws_server = server = WebSocketNotificationServer(
        WEBSOCKET_NOTIFICATION_URL, WEBSOCKET_NOTIFICATION_PORT, with_history=False
    )
    start_server = websockets.serve(server.handler, server.host, server.port)
    loop.run_until_complete(start_server)
    loop.run_forever()


if __name__ == "__main__":
    with NotificationServer() as server:
        server.start()
