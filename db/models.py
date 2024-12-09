from sqlalchemy import Column, Integer, String, Boolean, ForeignKey, DateTime
from sqlalchemy.orm import relationship, backref
from sqlalchemy.schema import UniqueConstraint
from db.database import Base
from datetime import datetime

class User(Base):
    __tablename__ = "users"
    id = Column(Integer, primary_key=True)
    username = Column(String, nullable=False)
    name = Column(String, nullable=False)
    is_admin = Column(Boolean, default=False)

    def __repr__(self):
        if self.is_admin:
            return f"{self.name} #{self.username}"  # when sudo su, prompt changes to #, so this is a joke
        return f"{self.name} @{self.username}"

class JazzStandard(Base):
    __tablename__ = "jazz_standards"
    id = Column(Integer, primary_key=True)
    title = Column(String, nullable=False)
    composer = Column(String, nullable=False)

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
