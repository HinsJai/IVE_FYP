import base64
import sys
import tomllib
from dataclasses import dataclass, field
from typing import Self

from fastapi import FastAPI, Request, Response
from fastapi.middleware.cors import CORSMiddleware
from fastapi.responses import HTMLResponse, JSONResponse
from fastapi.templating import Jinja2Templates

sys.path.extend([".."])
from logging import Logger

from item_getter import BoxGetter, FrameGetter
from logger import get_logger

with open("config.toml", "rb") as config:
    config = tomllib.load(config)

WEB_URL = config["web"]["url"]
WEB_PORT = config["web"]["port"]
INDEX = config["web"]["index"]
CAM_URL = config["cam"]["url"]
CAM_PORT = config["cam"]["port"]
SERVER_URL = config["server"]["url"]
SERVER_PORT = config["server"]["port"]

del config

TEMPLATES = Jinja2Templates(directory=".")


@dataclass(slots=True, repr=False)
class InferenceData:
    __cam_ip: str = field(init=False)
    __cam_port: int = field(init=False)
    __inference_ip: str = field(init=False)
    __inference_port: int = field(init=False)
    __logger: Logger = field(init=False)
    __frame_getter: FrameGetter = field(init=False)
    __box_getter: BoxGetter = field(init=False)

    def __post_init__(
        self,
    ) -> None:
        self.__cam_ip = CAM_URL
        self.__cam_port = CAM_PORT
        self.__inference_ip = SERVER_URL
        self.__inference_port = SERVER_PORT
        self.__logger = get_logger("Web Server")
        self.__frame_getter = FrameGetter(self.__cam_ip, self.__cam_port)
        self.__box_getter = BoxGetter(self.__inference_ip, self.__inference_port)

    def __enter__(self) -> Self:
        return self

    def __exit__(self, _, __, ___) -> None:
        pass

    def start(self) -> None:
        self.__frame_getter.start()
        self.__box_getter.start()
        self.__logger.info(
            f"Server started | Cam port: {self.__cam_port} Inference port: {self.__inference_port} Web URL: {WEB_URL[0]} or {WEB_URL[1]}"
        )

    @property
    def box(self) -> list[dict[str, int]]:
        return self.__box_getter.box

    @property
    def frame(self) -> bytes:
        return self.__frame_getter.frame


app = FastAPI()
app.add_middleware(
    CORSMiddleware,
    allow_origins=WEB_URL,
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)


@app.get("/", response_class=HTMLResponse)
async def root(request: Request):
    return TEMPLATES.TemplateResponse(
        INDEX, {"request": request, "url": f"{WEB_URL}:{WEB_PORT}"}
    )


@app.get("/image")
async def image() -> Response:
    frame_data = base64.b64decode(server.frame)
    print(server.frame)
    # return Response(content=server.frame, media_type="text/plain")
    return Response(content=frame_data, media_type="image/jpeg")


@app.get("/box")
async def box() -> Response:
    box_data = [
        {
            "x1": box.x1,
            "y1": box.y1,
            "x2": box.x2,
            "y2": box.y2,
            "class_type": box.class_type,
        }
        for box in server.box
    ]
    return JSONResponse(content=box_data, media_type="application/json")


with InferenceData() as server:
    server.start()
