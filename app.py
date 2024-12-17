from typing import List
from fastapi import FastAPI, HTTPException, Depends, Form
from fastapi.responses import HTMLResponse, RedirectResponse
from fastapi.requests import Request
from fastapi.templating import Jinja2Templates
from sqlalchemy.orm import Session
from db import get_db, crud, engine, models, init_db
import utils
import schemas
import base64
import bcrypt  # for hashing passwords
from typing import Annotated
from secrets import token_urlsafe



#models.Base.metadata.create_all(bind=engine)
init_db()

app = FastAPI()
templates = Jinja2Templates(directory="templates")


# a helper function that checks if header contains either username+password or cookie_token
def check_auth(request: Request, must_be_admin: bool = False) -> models.User:
    # check if there is an Authorization header
    if "Authorization" in request.headers:
        auth = request.headers["Authorization"]
        # check if it is Basic or Bearer
        if auth.startswith("Basic "):
            auth = auth[6:]
            username, password = base64.b64decode(auth).decode().split(":")
            user = crud.get_user_by_username(username)
            if user is None:
                raise HTTPException(status_code=404, detail="User not found")
            # lets get salt and hash the password
            salt = crud.get_salt(username)
            password_hash = bcrypt.hashpw(password.encode(), salt)
            if not crud.check_password(username, password_hash):
                raise HTTPException(status_code=401, detail="Incorrect password")
            if must_be_admin and not user.is_admin:
                raise HTTPException(status_code=401, detail="Unauthorized")
            return user
        if auth.startswith("Bearer "):
            token = auth[7:]
            user = crud.get_user_by_token(token)
            if user is None:
                raise HTTPException(status_code=404, detail="User not found")
            if must_be_admin and not user.is_admin:
                raise HTTPException(status_code=401, detail="Unauthorized")
            return user
    if "cookie_token" in request.cookies:
        token = request.cookies["cookie_token"]
        user = crud.get_user_by_token(token)
        if user is None:
            raise HTTPException(status_code=404, detail="User not found")
        if must_be_admin and not user.is_admin:
            raise HTTPException(status_code=401, detail="Unauthorized")
        return user
    raise HTTPException(status_code=401, detail="Unauthorized")

@app.get("/api/", response_model=schemas.Root)
def api_root(request: Request, db: Session = Depends(get_db)):
    # check if the user is authorized
    user = check_auth(request)
    if user.is_admin:
        users = crud.get_users(db)
        jazz_standards = crud.get_jazz_standards(db)
        response = schemas.Root(users=users, jazz_standards=jazz_standards)
        return response
    else:
        # lets keep json same to not break the frontend
        user_jazz_standards = crud.get_user_standards(db, user_id=user.id)
        response = schemas.Root(users=[], jazz_standards=user_jazz_standards)
        return response

@app.post("/api/users/", response_model=schemas.User)
def create_user(user: schemas.UserCreate, db: Session = Depends(get_db)):
    # this can be done by whoever
    salt = bcrypt.gensalt()
    password_hash = bcrypt.hashpw(user.password.encode(), salt)
    r = crud.create_user(db, username=user.username, name=user.name, is_admin=user.is_admin, password_hash=password_hash, salt=salt)
    if r is None:
        raise HTTPException(status_code=400, detail="Username already exists")
    return r

@app.get("/api/users/{user}", response_model=schemas.User)
def read_user(request: Request, user: str, db: Session = Depends(get_db)):
    # check if the user is authorized, but adminity is not necessary
    check_auth(request)
    db_user = crud.get_user(db, **utils.userstr(user))
    if db_user is None:
        raise HTTPException(status_code=404, detail="User not found")
    return db_user

@app.get("/api/users/", response_model=List[schemas.User])
def read_users(request: Request, skip: int = 0, limit: int = 100, db: Session = Depends(get_db)):
    # only admin can see all users
    user = check_auth(request)
    if not user.is_admin:
        raise HTTPException(status_code=401, detail="Unauthorized")  # You're not admin, go away
    users = crud.get_users(db, skip=skip, limit=limit)
    return users

@app.delete("/api/users/{user}", response_model=schemas.User)
def delete_user(request: Request, user: str, db: Session = Depends(get_db)):
    current_user = check_auth(request)
    if not current_user.is_admin:
        raise HTTPException(status_code=401, detail="Unauthorized")
    db_user = crud.get_user(db, **utils.userstr(user))
    if db_user is None:
        raise HTTPException(status_code=404, detail="User not found")
    crud.delete_user(db, user_id=user_id)
    return db_user

@app.post("/api/jazz_standards/", response_model=schemas.JazzStandard)
def create_jazz_standard(request: Request, jazz_standard: schemas.JazzStandardCreate, db: Session = Depends(get_db)):
    check_auth(request)
    r=crud.add_jazz_standard(db, title=jazz_standard.title, composer=jazz_standard.composer, style=jazz_standard.style)
    if r is None:
        raise HTTPException(status_code=400, detail="Jazz Standard already exists")

@app.get("/api/jazz_standards/{jazz_standard}", response_model=schemas.JazzStandard)
def read_jazz_standard(request: Request, jazz_standard: str, db: Session = Depends(get_db)):
    check_auth(request)
    db_jazz_standard = crud.get_jazz_standard(db, **utils.jazz_standardstr(jazz_standard))
    if db_jazz_standard is None:
        raise HTTPException(status_code=404, detail="Jazz Standard not found")
    return db_jazz_standard

@app.get("/api/jazz_standards/", response_model=List[schemas.JazzStandard])
def read_jazz_standards(request: Request, skip: int = 0, limit: int = 100, db: Session = Depends(get_db)):
    current_user = check_auth(request)
    if not current_user.is_admin:
        raise HTTPException(status_code=401, detail="Unauthorized")
    jazz_standards = crud.get_jazz_standards(db, skip=skip, limit=limit)
    return jazz_standards

@app.delete("/api/jazz_standards/{jazz_standard}", response_model=schemas.JazzStandard)
def delete_jazz_standard(request: Request, jazz_standard: str, db: Session = Depends(get_db)):
    current_user = check_auth(request)
    if not current_user.is_admin:
        raise HTTPException(status_code=401, detail="Unauthorized")
    db_jazz_standard = crud.get_jazz_standard(db, **utils.jazz_standardstr(jazz_standard))
    if db_jazz_standard is None:
        raise HTTPException(status_code=404, detail="Jazz Standard not found")
    crud.delete_jazz_standard(db, **utils.jazz_standardstr(jazz_standard))
    return db_jazz_standard

@app.post("/api/users/{user}/jazz_standards/{jazz_standard}", response_model=schemas.UserJazzStandard)
def add_standard_to_user(request: Request, user: str, jazz_standard: str, db: Session = Depends(get_db)):
    current_user = check_auth(request)
    # a boolean whether the user is id or username
    is_username_ = utils.is_username(user)
    if not current_user.is_admin:
        if is_username_:
            if current_user.username != user:
                raise HTTPException(status_code=401, detail="Unauthorized")
        else:
            if current_user.id != user:
                raise HTTPException(status_code=401, detail="Unauthorized")
    r = crud.add_standard_to_user(db, **utils.userstr(user), **utils.jazz_standardstr(jazz_standard))
    if r is None:
        raise HTTPException(status_code=400, detail="User Jazz Standard already exists")

@app.get("/api/users/{user}/jazz_standards/", response_model=List[schemas.UserJazzStandard])
def get_user_standards(request: Request, user: str, db: Session = Depends(get_db)):
    current_user = check_auth(request)
    is_username_ = utils.is_username(user)
    if not current_user.is_admin:
        if is_username_:
            if current_user.username != user:
                raise HTTPException(status_code=401, detail="Unauthorized")
        else:
            if current_user.id != user:
                raise HTTPException(status_code=401, detail="Unauthorized")
    return crud.get_user_standards(db, **utils.userstr(user))

@app.delete("/api/users/{user}/jazz_standards/{jazz_standard}", response_model=schemas.UserJazzStandard)
def delete_user_standard(request: Request, user: str, jazz_standard: str, db: Session = Depends(get_db)):
    current_user = check_auth(request)
    is_username_ = utils.is_username(user)
    if not current_user.is_admin:
        if is_username_:
            if current_user.username != user:
                raise HTTPException(status_code=401, detail="Unauthorized")
        else:
            if current_user.id != user:
                raise HTTPException(status_code=401, detail="Unauthorized")
    db_user_standard = crud.user_knows_standard(db, **utils.userstr(user), **utils.jazz_standardstr(jazz_standard))
    if db_user_standard is None:
        raise HTTPException(status_code=404, detail="User Jazz Standard not found")
    crud.delete_user_standard(db, **utils.userstr(user), **utils.jazz_standardstr(jazz_standard))
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
def login(username: Annotated[str, Form], password: Annotated[str, Form], response: Response):
    # check if there is a user with the given username
    user = crud.get_user_by_username(username)
    if user is None:
        raise HTTPException(status_code=404, detail="User not found")
    # get the salt of the user
    salt = crud.get_salt(username)
    # hash the password with the salt
    password_hash = bcrypt.hashpw(password.encode(), salt)
    # check if the password is correct
    if not crud.check_password(username, password_hash):
        raise HTTPException(status_code=401, detail="Incorrect password")
    # create a token for the user
    token = token_urlsafe(32)
    response.set_cookie("username", username)
    response.set_cookie("cookie_token", token)
    crud.set_token(username, token)
    return RedirectResponse(url="/")