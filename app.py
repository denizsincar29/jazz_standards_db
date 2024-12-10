from typing import List
from fastapi import FastAPI, HTTPException, Depends
from fastapi.responses import HTMLResponse, RedirectResponse
from fastapi.templating import Jinja2Templates
from sqlalchemy.orm import Session
from db import get_db, crud, engine, models, init_db
import schemas

#models.Base.metadata.create_all(bind=engine)
init_db()

app = FastAPI()

templates = Jinja2Templates(directory="templates")
@app.get("/api/", response_model=schemas.Root)
def api_root(db: Session = Depends(get_db)):
    users = crud.get_users(db)
    jazz_standards = crud.get_jazz_standards(db)
    response = schemas.Root(users=users, jazz_standards=jazz_standards)
    return response

@app.post("/api/users/", response_model=schemas.User)
def create_user(user: schemas.UserCreate, db: Session = Depends(get_db)):
    r = crud.create_user(db, username=user.username, name=user.name, is_admin=user.is_admin)
    if r is None:
        raise HTTPException(status_code=400, detail="Username already exists")
    return r

@app.get("/api/users/{user_id}", response_model=schemas.User)
def read_user(user_id: int, db: Session = Depends(get_db)):
    db_user = crud.get_user(db, user_id=user_id)
    if db_user is None:
        raise HTTPException(status_code=404, detail="User not found")
    return db_user

@app.get("/api/users/", response_model=List[schemas.User])
def read_users(skip: int = 0, limit: int = 100, db: Session = Depends(get_db)):
    users = crud.get_users(db, skip=skip, limit=limit)
    return users

@app.delete("/api/users/{user_id}", response_model=schemas.User)
def delete_user(user_id: int, db: Session = Depends(get_db)):
    db_user = crud.get_user(db, user_id=user_id)
    if db_user is None:
        raise HTTPException(status_code=404, detail="User not found")
    crud.delete_user(db, user_id=user_id)
    return db_user

@app.post("/api/jazz_standards/", response_model=schemas.JazzStandard)
def create_jazz_standard(jazz_standard: schemas.JazzStandardCreate, db: Session = Depends(get_db)):
    r=crud.add_jazz_standard(db, title=jazz_standard.title, composer=jazz_standard.composer, style=jazz_standard.style)
    if r is None:
        raise HTTPException(status_code=400, detail="Jazz Standard already exists")

@app.get("/api/jazz_standards/{jazz_standard_id}", response_model=schemas.JazzStandard)
def read_jazz_standard(jazz_standard_id: int, db: Session = Depends(get_db)):
    db_jazz_standard = crud.get_jazz_standard(db, jazz_standard_id=jazz_standard_id)
    if db_jazz_standard is None:
        raise HTTPException(status_code=404, detail="Jazz Standard not found")
    return db_jazz_standard

@app.get("/api/jazz_standards/", response_model=List[schemas.JazzStandard])
def read_jazz_standards(skip: int = 0, limit: int = 100, db: Session = Depends(get_db)):
    jazz_standards = crud.get_jazz_standards(db, skip=skip, limit=limit)
    return jazz_standards

@app.delete("/api/jazz_standards/{jazz_standard_id}", response_model=schemas.JazzStandard)
def delete_jazz_standard(jazz_standard_id: int, db: Session = Depends(get_db)):
    db_jazz_standard = crud.get_jazz_standard(db, jazz_standard_id=jazz_standard_id)
    if db_jazz_standard is None:
        raise HTTPException(status_code=404, detail="Jazz Standard not found")
    crud.delete_jazz_standard(db, jazz_standard_id=jazz_standard_id)
    return db_jazz_standard

@app.post("/api/users/{user_id}/jazz_standards/{jazz_standard_id}", response_model=schemas.UserJazzStandard)
def add_standard_to_user(user_id: int, jazz_standard_id: int, db: Session = Depends(get_db)):
    r = crud.add_standard_to_user(db, user_id=user_id, jazz_standard_id=jazz_standard_id)
    if r is None:
        raise HTTPException(status_code=400, detail="User Jazz Standard already exists")

@app.get("/api/users/{user_id}/jazz_standards/", response_model=List[schemas.UserJazzStandard])
def get_user_standards(user_id: int, db: Session = Depends(get_db)):
    return crud.get_user_standards(db, user_id=user_id)

@app.delete("/api/users/{user_id}/jazz_standards/{jazz_standard_id}", response_model=schemas.UserJazzStandard)
def delete_user_standard(user_id: int, jazz_standard_id: int, db: Session = Depends(get_db)):
    db_user_standard = crud.user_knows_standard(db, user_id=user_id, jazz_standard_id=jazz_standard_id)
    if db_user_standard is None:
        raise HTTPException(status_code=404, detail="User Jazz Standard not found")
    crud.delete_user_standard(db, user_id=user_id, jazz_standard_id=jazz_standard_id)
    return db_user_standard


# web routes
@app.get("/", response_class=HTMLResponse)
def read_root(request: Request):
    # if there is no username cookie, redirect to templates/login.html
    if "username" not in request.cookies:
        return templates.TemplateResponse("login.html", {"request": request})
    # if there is a username cookie, redirect to templates/index.html with template variable username
    username = request.cookies["username"]
    return templates.TemplateResponse(request, "index.html", {"username": username})

@app.post("/login/", response_class=HTMLResponse)
def login(username: str, response: Response):
    # check if there is a user with the given username
    user = crud.get_user_by_username(username)
    if user is None:
        raise HTTPException(status_code=404, detail="User not found")
    response.set_cookie("username", username)
    return RedirectResponse(url="/")