from typing import List, Annotated
from fastapi import FastAPI, HTTPException, Depends, Form, Response, status
from fastapi.responses import HTMLResponse, RedirectResponse
from fastapi.requests import Request
from fastapi.templating import Jinja2Templates
import logging
import utils
import base64
import bcrypt  # for hashing passwords
from secrets import token_urlsafe
import os
import dotenv
dotenv.load_dotenv()
from sqlalchemy.orm import Session #noqa  # import not on top because of the above
import schemas # noqa
from db import get_db, crud, models, init_db, teardown_db, JazzStyle # noqa

testenv = os.getenv("TEST_ENV")
if testenv is not None:
    logging.info(f"Using test database #{testenv}")



logging.basicConfig(level=logging.INFO)

#models.Base.metadata.create_all(bind=engine)
init_db()

app = FastAPI()
templates = Jinja2Templates(directory="templates")


# a helper function that checks if header contains either username+password or cookie_token
def check_auth(db: Session, request: Request, must_be_admin: bool = False) -> models.User:
    # check if there is an Authorization header
    if "Authorization" in request.headers:
        auth = request.headers["Authorization"]
        # check if it is Basic or Bearer
        if auth.startswith("Basic "):
            auth = auth[6:]
            username, password = base64.b64decode(auth).decode().split(":")
            user = crud.get_user(db, username = username)
            if user is None:
                raise HTTPException(status_code=404, detail="User not found")
            # lets get salt and hash the password
            salt = crud.get_salt(db, username)
            password_hash = bcrypt.hashpw(password.encode(), salt)
            if not crud.check_password(db, username, password_hash):
                raise HTTPException(status_code=401, detail="Incorrect password")
            if must_be_admin and not user.is_admin:
                raise HTTPException(status_code=401, detail="Unauthorized")
            return user
        if auth.startswith("Bearer "):
            token = auth[7:]
            
            user = crud.get_user_by_token(db, token)
            if user is None:
                raise HTTPException(status_code=404, detail="User not found")
            if must_be_admin and not user.is_admin:
                raise HTTPException(status_code=401, detail="Unauthorized")
            return user
    if "cookie_token" in request.cookies:
        token = request.cookies["cookie_token"]
        user = crud.get_user_by_token(db, token)
        if user is None:
            raise HTTPException(status_code=404, detail="User not found")
        if must_be_admin and not user.is_admin:
            raise HTTPException(status_code=401, detail="Unauthorized")
        return user
    raise HTTPException(status_code=401, detail="Unauthorized")

# web browsers must show beautiful error pages
# use try checkauth except return error page
def check_auth_web(db: Session, request: Request, must_be_admin: bool = False, go_back: str = None, go_login: bool = False) -> models.User:
    try:
        return check_auth(db, request, must_be_admin), None
    except HTTPException as e:
        if go_login:
            with open("html/login.html", encoding="UTF-8") as f:
                return None, HTMLResponse(content=f.read())
        return None, error_page(request, e.status_code, e.detail, go_back)
        # should I instead make a dict "russian"? Later, maybe

# in web routes, use this function instead of raise HTTPException. Just return error_page(404, "User not found") for example
def error_page(request: Request, code: int, message: str, goback: str = None):
    context = {"code": code, "message": message}
    if goback is not None:
        context["goback"] = goback
    return templates.TemplateResponse(request, "error.html", context)

@app.get("/api/", response_model=schemas.Root)
def api_root(request: Request, db: Session = Depends(get_db)):
    logging.info("api_root started")  # debug
    # check if the user is authorized
    user = check_auth(db, request)
    
    if user.is_admin:
        logging.info("admin")
        users = crud.get_users(db)
        jazz_standards = crud.get_jazz_standards(db)
        response = schemas.Root(users=users, jazz_standards=jazz_standards)
        return response
    else:
        logging.info("user")
        # lets keep json same to not break the frontend
        user_jazz_standards = crud.get_user_standards(db, user_id=user.id)
        response = schemas.Root(users=[], jazz_standards=user_jazz_standards)
        return response

@app.post("/api/users/", response_model=schemas.User)
def create_user(request: Request, user: schemas.UserCreate, db: Session = Depends(get_db)):
    # this can be done by whoever, but only admins can create admins
    if user.is_admin:
        users_count = crud.get_users_count(db)
        # if there are no users, you can create an admin while not being an admin and even not being authorized
        if users_count > 0:  # if there are users, disallow creating an admin
            check_auth(db, request, must_be_admin=True)
    salt = bcrypt.gensalt()
    password_hash = bcrypt.hashpw(user.password.encode(), salt)
    r = crud.create_user(db, username=user.username, name=user.name, is_admin=user.is_admin, password_hash=password_hash, salt=salt)
    if r is None:
        raise HTTPException(status_code=400, detail="Username already exists")
    return r

# create an admin user
# if there are no users, this will create the first user as an admin
# if there are already users, this will fail
# thus after deployment, you should run this endpoint to create the first admin user if you don't want other random people to create an owner and take over the system haha!
@app.post("/api/admin/", response_model=schemas.User)
def create_admin(user: schemas.AdminCreate, db: Session = Depends(get_db)):
    # check if there are any users
    if crud.get_users(db):
        raise HTTPException(status_code=400, detail="Admin already exists")
    salt = bcrypt.gensalt()
    password_hash = bcrypt.hashpw(user.password.encode(), salt)
    r = crud.create_user(db, username=user.username, name=user.name, is_admin=True, password_hash=password_hash, salt=salt)
    if r is None:
        raise HTTPException(status_code=400, detail="Username already exists")
    return r

@app.get("/api/users/me", response_model=schemas.User)
def read_user_me_api(request: Request, db: Session = Depends(get_db)):
    # check if the user is authorized
    user = check_auth(db, request)
    return user

# login from the api, not the web
@app.post("/api/login", response_model=schemas.User)
def login_user(request: Request, response: Response, user: schemas.UserLogin, db: Session = Depends(get_db)):
    c_user = crud.get_user(db, username = user.username)
    if c_user is None:
        raise HTTPException(status_code=404, detail="User not found")
    salt = crud.get_salt(db, user.username)
    password_hash = bcrypt.hashpw(user.password.encode(), salt)
    if not crud.check_password(db, user.username, password_hash):
        raise HTTPException(status_code=401, detail="Incorrect password")
    token = token_urlsafe(32)
    crud.update_user_token(db, username = user.username, token = token)
    response.set_cookie("cookie_token", token)
    response.set_cookie("username", user.username)
    return c_user

@app.get("/api/users/{user}", response_model=schemas.User)
def read_user(request: Request, user: str, db: Session = Depends(get_db)):
    # check if the user is authorized, but adminity is not necessary
    check_auth(db, request)
    db_user = crud.get_user(db, **utils.userstr(user))
    if db_user is None:
        raise HTTPException(status_code=404, detail="User not found")
    return db_user

# getme
@app.get("/api/users/me", response_model=schemas.User)
def read_user_me(request: Request, response: Response, db: Session = Depends(get_db)):
    user: models.User = check_auth(db, request)
    response.set_cookie("username", user.username)
    response.set_cookie("cookie_token", user.private.token)  # todo!
    return user

@app.get("/api/users/", response_model=List[schemas.User])
def read_users(request: Request, skip: int = 0, limit: int = 100, db: Session = Depends(get_db)):
    # only admin can see all users
    check_auth(db, request, must_be_admin=True)
    users = crud.get_users(db, skip=skip, limit=limit)
    return users

# return "ok" string
@app.delete("/api/users/{user}", response_model=str)
def delete_user(request: Request, user: str, db: Session = Depends(get_db)):
    current_user = check_auth(db, request)
    # if it's me, I can delete myself
    # if it's not me, I can delete anyone if I'm an admin
    if current_user.id != user and current_user.username != user and not current_user.is_admin:
        raise HTTPException(status_code=401, detail="Unauthorized")
    deleted = crud.delete_user(db, **utils.userstr(user))
    if not deleted:  # how natural english this is!!!
        raise HTTPException(status_code=404, detail="User not found")
    return "ok"

@app.post("/api/jazz_standards/", response_model=schemas.JazzStandard)
def create_jazz_standard(request: Request, jazz_standard: schemas.JazzStandardCreate, db: Session = Depends(get_db)):
    check_auth(db, request)
    r=crud.add_jazz_standard(db, title=jazz_standard.title, composer=jazz_standard.composer, style=jazz_standard.style)
    if r is None:
        raise HTTPException(status_code=400, detail="Jazz Standard already exists")
    return r

@app.get("/api/jazz_standards/{jazz_standard}", response_model=schemas.JazzStandard)
def read_jazz_standard(request: Request, jazz_standard: str, db: Session = Depends(get_db)):
    check_auth(db, request)
    db_jazz_standard = crud.get_jazz_standard(db, **utils.jazz_standardstr(jazz_standard))
    if db_jazz_standard is None:
        raise HTTPException(status_code=404, detail="Jazz Standard not found")
    return db_jazz_standard

@app.get("/api/jazz_standards/", response_model=List[schemas.JazzStandard])
def read_jazz_standards(request: Request, skip: int = 0, limit: int = 100, db: Session = Depends(get_db)):
    check_auth(db, request, must_be_admin=True)
    jazz_standards = crud.get_jazz_standards(db, skip=skip, limit=limit)
    return jazz_standards

@app.delete("/api/jazz_standards/{jazz_standard}", response_model=schemas.JazzStandard)
def delete_jazz_standard(request: Request, jazz_standard: str, db: Session = Depends(get_db)):
    check_auth(db, request, must_be_admin=True)
    db_jazz_standard = crud.get_jazz_standard(db, **utils.jazz_standardstr(jazz_standard))
    if db_jazz_standard is None:
        raise HTTPException(status_code=404, detail="Jazz Standard not found")
    crud.delete_jazz_standard(db, **utils.jazz_standardstr(jazz_standard))
    return db_jazz_standard

@app.post("/api/users/{user}/jazz_standards/{jazz_standard}", response_model=schemas.UserJazzStandard)
def add_standard_to_user(request: Request, user: str, jazz_standard: str, db: Session = Depends(get_db)):
    current_user = check_auth(db, request)
    # a boolean whether the user is id or username
    is_username_ = utils.is_username(user)
    if not current_user.is_admin:
        if is_username_:
            if current_user.username != user:
                raise HTTPException(status_code=401, detail="Unauthorized")
        else:
            if current_user.id != user:
                raise HTTPException(status_code=401, detail="Unauthorized")
    try:
        r = crud.add_standard_to_user(db, **utils.userstr(user), **utils.jazz_standardstr(jazz_standard))
    except ValueError:
        raise HTTPException(status_code=404, detail="User or Jazz Standard not found")
    if r is None:
        raise HTTPException(status_code=400, detail="User Jazz Standard already exists")
    return r

@app.get("/api/users/{user}/jazz_standards/", response_model=List[schemas.UserJazzStandard])
def get_user_standards(request: Request, user: str, db: Session = Depends(get_db)):
    current_user = check_auth(db, request)
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
    current_user = check_auth(db, request)
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

# tear down endpoint that works only in test mode
@app.get("/api/teardown")
def teardown(db: Session = Depends(get_db)):
    if testenv is not None:
        teardown_db()
        return {"message": "Database dropped"}
    else:
        raise HTTPException(status_code=401, detail="Unauthorized")


# web routes

@app.get("/", response_class=HTMLResponse)
def read_root(request: Request, db: Session = Depends(get_db)):
    # if there is no username cookie, redirect to html/login.html
    if "username" not in request.cookies:
        with open("html/login.html", encoding="UTF-8") as f:
            return HTMLResponse(content=f.read())
    # if there is a username cookie, redirect to html/index.html but check if the user is authorized
    user, error = check_auth_web(db, request, go_login=True)
    if error is not None:
        return error  # if the user is not authorized, return the login page
    # get list of jazz_style names
    styles = [style.value for style in JazzStyle]
    # get the user's jazz standards
    user_standards = crud.get_user_standards(db, user_id=user.id)
    # sort by style:
    # yeah but this must be a list of jazz_standard objects, not user_jazz_standard objects
    standards = [standard.jazz_standard for standard in user_standards]
    standard_dict = {}
    for standard in standards:
        if standard.style.value not in standard_dict:
            standard_dict[standard.style.value] = []
        standard_dict[standard.style.value].append(standard)
    # return the index page
    return templates.TemplateResponse(request, "index.html", {"styles": styles, "user": user, "standards": standard_dict})

@app.post("/login/", response_class=HTMLResponse)
def login(username: Annotated[str, Form()], password: Annotated[str, Form()], request: Request, db: Session = Depends(get_db)):
    # check if there is a user with the given username
    user = crud.get_user(db, username = username)
    if user is None:
        return error_page(Request, 404, "Пользователь не найден", goback="/")
    # get the salt of the user
    salt = crud.get_salt(db, username)
    # hash the password with the salt
    password_hash = bcrypt.hashpw(password.encode(), salt)
    # check if the password is correct
    if not crud.check_password(db, username, password_hash):
        return error_page(request, 401, "Неправильный пароль", goback="/")
    # create a token for the user
    token = token_urlsafe(32)
    response = RedirectResponse(url="/", status_code=status.HTTP_303_SEE_OTHER)
    response.set_cookie("username", username)
    response.set_cookie("cookie_token", token)
    crud.update_user_token(db, username=username, token=token)  # it needs kwargs, not args
    return response

# logout
@app.get("/logout/", response_class=HTMLResponse)
def logout(response: Response):
    response.delete_cookie("username")
    response.delete_cookie("cookie_token")
    return RedirectResponse(url="/")

# register
@app.post("/register/", response_class=HTMLResponse)
def register(name: Annotated[str, Form()], username: Annotated[str, Form()], password: Annotated[str, Form()], request: Request, db: Session = Depends(get_db)):
    logging.info(f"registering {username}")
    # check if the username is already taken
    if crud.get_user(db, username = username) is not None:
        return error_page(request, 400, "Пользователь уже существует", goback="/")
    # create a salt
    salt = bcrypt.gensalt()
    # hash the password
    password_hash = bcrypt.hashpw(password.encode(), salt)
    # create the user
    crud.create_user(db, username=username, name=name, is_admin=False, password_hash=password_hash, salt=salt)  # admins can be created only from the direct db access, you fool
    return RedirectResponse(url="/", status_code=status.HTTP_303_SEE_OTHER)

# add jazz standard
@app.post("/add_standard/", response_class=HTMLResponse)
def add_standard(title: Annotated[str, Form()], composer: Annotated[str, Form()], style: Annotated[str, Form()], request: Request, db: Session = Depends(get_db)):
    # check if the user is authorized
    user, error = check_auth_web(db, request)
    if error is not None:
        return error
    # add a jazz standard if it doesn't exist
    if crud.get_jazz_standard(db, title=title) is None:
        crud.add_jazz_standard(db, title, composer, style)
    # assign the jazz standard to the user
    try:
        crud.add_standard_to_user(db, user.id, title=title)
    except ValueError as e:
        logging.exception(e)
        return error_page(request, 404, "Вы уже отметили этот стандарт, как изученный", goback="/")
    return RedirectResponse(url="/", status_code=status.HTTP_303_SEE_OTHER)