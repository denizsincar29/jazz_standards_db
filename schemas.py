from pydantic import BaseModel
from db.models import JazzStyle  # that's not good

class User(BaseModel):
    id: int
    username: str
    name: str
    is_admin: bool

    class Config:
        from_attributes = True

class JazzStandard(BaseModel):
    id: int
    title: str
    composer: str
    style: JazzStyle

    class Config:
        from_attributes = True

class UserJazzStandard(BaseModel):
    user_id: int
    jazz_standard_id: int

    class Config:
        from_attributes = True

class Root(BaseModel):
    users: list[User]
    jazz_standards: list[JazzStandard]

    class Config:
        from_attributes = True