from sqlalchemy import Column, Integer, String, Boolean, ForeignKey, DateTime, Enum
from sqlalchemy.orm import relationship, backref
from db.database import Base
import enum

class User(Base):
    __tablename__ = "users"
    id = Column(Integer, primary_key=True)
    username = Column(String, nullable=False, unique=True)
    name = Column(String, nullable=False)
    is_admin = Column(Boolean, default=False)

    def __repr__(self):
        if self.is_admin:
            return f"{self.name} #{self.username}"  # when sudo su, prompt changes to #, so this is a joke
        return f"{self.name} @{self.username}"


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
    id = Column(Integer, primary_key=True)
    title = Column(String, nullable=False, unique=True)
    composer = Column(String, nullable=False)
    style = Column(Enum(JazzStyle), nullable=False)

    def __repr__(self):
        return f"{self.title} by {self.composer}"
    

class UserJazzStandard(Base):
    __tablename__ = "user_jazz_standards"
    user_id = Column(Integer, ForeignKey('users.id'), primary_key=True)
    jazz_standard_id = Column(Integer, ForeignKey('jazz_standards.id'), primary_key=True)
    user = relationship("User", backref=backref("user_jazz_standards", cascade="all, delete-orphan"))
    jazz_standard = relationship("JazzStandard", backref=backref("user_jazz_standards", cascade="all, delete-orphan"))

    def __repr__(self):
        return f"{self.user} plays {self.jazz_standard}"

User.jazz_standards = relationship("UserJazzStandard", back_populates="user", overlaps="user_jazz_standards")
JazzStandard.users = relationship("UserJazzStandard", back_populates="jazz_standard", overlaps="user_jazz_standards")
