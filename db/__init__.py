from database import SessionLocal, engine, Base

Base.metadata.create_all(bind=engine)  # yep, this is done at import time