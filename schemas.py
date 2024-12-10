from pydantic import BaseModel, ConfigDict
from db.models import JazzStyle  # that's not good


class User(BaseModel):
    id: int
    username: str
    name: str
    is_admin: bool
    model_config = ConfigDict(from_attributes=True)

class JazzStandard(BaseModel):
    id: int
    title: str
    composer: str
    style: JazzStyle
    model_config = ConfigDict(from_attributes=True)


class UserJazzStandard(BaseModel):
    user_id: int
    jazz_standard_id: int
    model_config = ConfigDict(from_attributes=True)

class Root(BaseModel):
    users: list[User]
    jazz_standards: list[JazzStandard]
    model_config = ConfigDict(from_attributes=True)

# make models without ids for creation
class UserCreate(BaseModel):
    username: str
    name: str
    is_admin: bool
    model_config = ConfigDict(from_attributes=True)

class JazzStandardCreate(BaseModel):
    title: str
    composer: str
    style: JazzStyle
    model_config = ConfigDict(from_attributes=True)

class UserJazzStandardCreate(BaseModel):
    user_id: int
    jazz_standard_id: int
    model_config = ConfigDict(from_attributes=True)