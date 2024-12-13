from sqlalchemy import String, Boolean, ForeignKey, Enum, LargeBinary
from sqlalchemy.orm import relationship, Mapped, mapped_column
import enum
from db.database import Base

# User's public data
class User(Base):
    __tablename__ = "users"

    id: Mapped[int] = mapped_column(primary_key=True)
    username: Mapped[str] = mapped_column(String, nullable=False, unique=True)
    name: Mapped[str] = mapped_column(String, nullable=False)
    is_admin: Mapped[bool] = mapped_column(Boolean, default=False)

    private: Mapped["UserPrivate"] = relationship(back_populates="user", cascade="all, delete-orphan")
    jazz_standards: Mapped[list["UserJazzStandard"]] = relationship(back_populates="user")

    def __repr__(self):
        if self.is_admin:
            return f"{self.name} #{self.username}"  # when sudo su, prompt changes to #, so this is a joke
        return f"{self.name} @{self.username}"

# User's private data
class UserPrivate(Base):
    __tablename__ = "users_private"

    user_id: Mapped[int] = mapped_column(ForeignKey("users.id"), primary_key=True)
    password_hash: Mapped[bytes] = mapped_column(LargeBinary, nullable=False)
    salt: Mapped[bytes] = mapped_column(LargeBinary, nullable=False)
    token: Mapped[str | None] = mapped_column(String, nullable=True)  # for cookie-based auth

    user: Mapped["User"] = relationship(back_populates="private")

class JazzStyle(enum.Enum):
    dixieland = "dixieland"
    ragtime = "ragtime"
    big_band = "big band"
    bossa_nova = "bossa nova"
    samba = "samba"
    latin = "latin"
    latin_swing = "latin swing"
    swing = "swing"
    waltz = "waltz"
    bebop = "bebop"
    modal = "modal"
    free = "free"
    fusion = "fusion"

class JazzStandard(Base):
    __tablename__ = "jazz_standards"

    id: Mapped[int] = mapped_column(primary_key=True)
    title: Mapped[str] = mapped_column(String, nullable=False, unique=True)
    composer: Mapped[str] = mapped_column(String, nullable=False)
    style: Mapped[JazzStyle] = mapped_column(Enum(JazzStyle), nullable=False)

    users: Mapped[list["UserJazzStandard"]] = relationship(back_populates="jazz_standard")

    def __repr__(self):
        return f"{self.title} by {self.composer}"

class UserJazzStandard(Base):
    __tablename__ = "user_jazz_standards"

    user_id: Mapped[int] = mapped_column(ForeignKey("users.id"), primary_key=True)
    jazz_standard_id: Mapped[int] = mapped_column(ForeignKey("jazz_standards.id"), primary_key=True)

    user: Mapped["User"] = relationship(back_populates="jazz_standards")
    jazz_standard: Mapped["JazzStandard"] = relationship(back_populates="users")

    def __repr__(self):
        return f"{self.user} plays {self.jazz_standard}"
