# This file is not related to the project! It's an importable playground to test SQLAlchemy features.

from sqlalchemy import Column, Integer, String, create_engine
from sqlalchemy.ext.declarative import declarative_base
from sqlalchemy.orm import sessionmaker
from sqlalchemy import Select, Update, Delete

# Base for model definitions
Base = declarative_base()

# Define your first table
class Table1(Base):
    __tablename__ = 'table1'
    id = Column(Integer, primary_key=True, index=True)
    name = Column(String, index=True)

# Define your second table
class Table2(Base):
    __tablename__ = 'table2'
    id = Column(Integer, primary_key=True, index=True)
    description = Column(String, index=True)

# Database configuration
database_url = "sqlite:///test.db"
engine = create_engine(database_url, connect_args={"check_same_thread": False})

# Create tables
Base.metadata.create_all(bind=engine)

# SessionLocal for database operations
SessionLocal = sessionmaker(autocommit=False, autoflush=False, bind=engine)
db = SessionLocal()

# help strings:
# to get help, run `Help.query`, `Help.update`, or `Help.delete` in the console
class Help:
    query = """
# Query all rows from Table1
db.execute(Select(Table1)).scalars().all()
"""
    update = """
# Update a row in Table1
db.execute(Update(Table1).where(Table1.id == 1).values(name="New Name"))
"""
    delete = """
# Delete a row in Table1
db.execute(Delete(Table1).where(Table1.id == 1))
"""
# Export components for easy import
__all__ = ["Base", "Table1", "Table2", "db", "engine", "Help", "Select", "Update", "Delete"]
