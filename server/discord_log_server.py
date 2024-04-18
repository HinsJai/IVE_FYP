import io
import logging
import sys
import tomllib
from typing import Any, Self
import discord

sys.path.extend([".."])

import asyncio
from asyncio import AbstractEventLoop
from dataclasses import dataclass
from dataclasses import field

from discord import app_commands

from common.logger import get_logger
from common.logger import Logger
from common.logger import logger_handler
from protos.proto_pb2 import Empty

from protos.proto_pb2_grpc import add_DiscordLogServicer_to_server
from protos.proto_pb2_grpc import DiscordLogServicer

from surrealdb import Surreal
from threading import Timer

try:
    from typing import override, Self
except ImportError:
    from typing_extensions import Self, override

from concurrent import futures

logging.getLogger("discord").addHandler(logger_handler())
logging.getLogger("discord.client").addHandler(logger_handler())

import grpc

with open("config.toml", "rb") as config:
    config = tomllib.load(config)
    
GUILD_ID = config["discord"]["guild_id"]
LOG_CHANNEL = config["discord"]["log_channel"]
TOKEN = config["discord"]["token"]
DISCORD_LOGGER_URL = config["discord"]["ip"]
DISCORD_LOGGER_PORT = config["discord"]["port"]
INTERACTION_USER_ID = config["discord"]["interaction_user_id"]

del config


class MyClient(discord.Client):
    def __init__(self, *, intents: discord.Intents) -> None:
        super().__init__(intents=intents)
        self.tree = app_commands.CommandTree(self)
        self.__my_guild = discord.Object(id=GUILD_ID)

    async def setup_hook(self) -> None:
        self.tree.copy_global_to(guild=self.__my_guild)
        await self.tree.sync(guild=self.__my_guild)


@dataclass(slots=True, repr=False)
class DiscordBotServer:
    loop: AbstractEventLoop = field(init=False, default=None)
    __my_guild = discord.Object(id=GUILD_ID)
    __client: MyClient = field(init=False)
    __server: grpc.server = field(init=False)
    __loger: Logger = field(init=False)

    def __post_init__(self) -> None:
        intents = discord.Intents.default()
        intents.message_content = True
        self.__client = MyClient(intents=intents)
        self.__client.event(self.on_ready)
        self.__client.tree.command(guild=self.__my_guild)(self.say)
        self.__server = grpc.server(futures.ThreadPoolExecutor(max_workers=10), options = [
            ('grpc.max_send_message_length', 1024 * 1024 * 1024),
            ('grpc.max_receive_message_length', 1024 * 1024 * 1024)
        ])
        self.__loger = get_logger("DiscordBotServer")

    def start(self) -> None:
        self.__server.start()
        self.__client.run(TOKEN)

    def __enter__(self) -> Self:
        add_DiscordLogServicer_to_server(DiscordLogService(self), self.__server)
        self.__server.add_insecure_port(f"{DISCORD_LOGGER_URL}:{DISCORD_LOGGER_PORT}")
        return self

    def __exit__(self, _, __, ___) -> None:
        self.__server.stop(None)
        asyncio.run(self.__client.close())

    async def on_ready(self) -> None:
        self.loop = asyncio.get_running_loop()
        self.__loger.info(
            f"Logged in as {self.__client.user} (ID: {self.__client.user.id})"
        )

    async def say(self, interaction: discord.Interaction, text_to_send: str) -> None:
        if interaction.user.id == INTERACTION_USER_ID:
            if interaction.channel is not None:
                await interaction.channel.send(text_to_send.replace("\\n", "\n"))
                await interaction.response.send_message(
                    "success", ephemeral=True, delete_after=5
                )
        else:
            await interaction.response.send_message(
                "You are not my owner", ephemeral=True, delete_after=5
            )

    async def send_log(self, log: str, image: bytes) -> None:
        guild = self.__client.get_guild(GUILD_ID)
        channel = discord.utils.get(guild.channels, name=LOG_CHANNEL)
        data = io.BytesIO(image)
        await channel.send(log, file=discord.File(data, "frame.png"))


class DiscordLogService(DiscordLogServicer):
    def __init__(self, server: DiscordBotServer) -> None:
        super().__init__()
        self.__server = server

    @override
    def log(self, request, _) -> Empty:
        if self.__server.loop != None:
            asyncio.run_coroutine_threadsafe(
                self.__server.send_log(request.message, request.image.data),
                self.__server.loop,
            )
        return Empty()


if __name__ == "__main__":
    with DiscordBotServer() as server:
        server.start()
