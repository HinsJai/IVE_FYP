from surrealdb import Surreal
from typing import Any
from datetime import datetime


class Database:
    def __init__(
        self, url: str, user: str, password: str, namesapce: str, db: str
    ) -> None:
        self.__url = url
        self.__user = user
        self.__pasword = password
        self.__namespace = namesapce
        self.__db = db

    async def get_db_response(
        self, query: str, data: dict[str, Any] = None
    ) -> list[dict[str, Any]]:
        async with Surreal(self.__url) as db:
            await db.signin({"user": self.__user, "pass": self.__pasword})
            await db.use(self.__namespace, self.__db)
            response = await db.query(query, data)
            # print(response)
        return response[0]["result"]


async def main():
    db = Database("ws://127.0.0.1:8080/rpc", "root", "root", "fyp", "violation_record")
    # await db.get_db_response(
    #     """
    #     insert into violation_record {
    #         cameraID: $cameraID,
    #         workplace: $workplace,

    #         violation_type: $violation_type,
    #     };
    #     """,
    #     {
    #         "cameraID": "c001",
    #         "workplace": "TY-IVE",
    #         # "time": datetime.now().strftime("%Y-%m-%dT%H:%M:%S.%f"),
    #         "violation_type": ["no_hardhat", "haha", "999"],
    #     },
    # )
    # print(await db.get_db_response("select * from user"))
    print(
        await db.get_db_response(
            "select 1 from user where email=$email and password=$password",
            data={"email": "jason199794@gmail.com", "password": "Aa123456"},
        )
    )

    # await db.query("""
    # insert into person {
    #     user: 'me',
    #     pass: 'very_safe',
    #     tags: ['python', 'documentation']
    # };

    # """)
    # # print(await db.query("select * from person"))

    # print(await db.query("""
    # update person content {
    #     user: 'you',
    #     pass: 'more_safe',
    #     tags: ['awesome']
    # };

    # """))
    # print(await db.query("delete person"))


if __name__ == "__main__":
    import asyncio

    asyncio.run(main())
