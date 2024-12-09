from fastapi import FastAPI
from db import get_db, crud
from sqlalchemy.orm import Session

app = FastAPI()


@app.get("/")
def root(db: Session = Depends(get_db)):
    # lets get all the users and jazz standards and return them
    users = crud.get_users(db)
    jazz_standards = crud.get_jazz_standards(db)
    # we need to dictify the users and jazz_standards
    users = [user.__dict__() for user in users]
    jazz_standards = [jazz_standard.__dict__() for jazz_standard in jazz_standards]
    return {"users": users, "jazz_standards": jazz_standards}