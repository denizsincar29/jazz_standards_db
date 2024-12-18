import os
try:
    os.remove("test.db")  # remove the test db file if it exists
except FileNotFoundError:
    pass
os.environ["JAZZ_DB_FILE"] = "test.db"  # because :memory: is doing some weird stuff
from fastapi.testclient import TestClient  # noqa
from app import app  # noqa

client = TestClient(app)

COOKIE_TOKEN = ""

# Create the first user for authentication
def create_or_get_auth_user(is_admin=True):
    auth_username = "auth_admin" if is_admin else "auth_user"
    response = client.get(f"/api/users/auth_{'admin' if is_admin else 'user'}")
    if response.status_code == 200:
        print("got the user", response.cookies)
        return response.json(), response.cookies
    # else create the user
    response = client.post(
        "/api/users/",
        json={"username": auth_username, "name": "Auth User", "password": "qwerty", "is_admin": is_admin},
    )
    assert response.status_code == 200
    # login
    response = client.post(
        "/api/login",
        json={"username": auth_username, "password": "qwerty"},
    )
    print("logged in", response.text)
    assert response.status_code == 200
    print("created the user", dict(response.cookies))
    return (response.json(), response.cookies)

def get_auth_headers(is_admin=True):
    auth_user, delicious_cookies = create_or_get_auth_user(is_admin)  # cookies are delicious, aren't they?
    return {"Authorization": f"Bearer {delicious_cookies['cookie_token']}"}

def test_read_root():
    headers = get_auth_headers()
    response = client.get("/api/", headers=headers)
    assert response.status_code == 200
    # well, check that the response is a dict but not the content, and it has keys users and jazz standards
    assert isinstance(response.json(), dict)
    assert "users" in response.json()
    assert "jazz_standards" in response.json()

def test_create_user():
    #headers = get_auth_headers()  # or it could be a chane: to register you need to be registered, lol
    response = client.post(
        "/api/users/",
        json={"username": "test_user", "name": "Test User", "password": "securepass", "is_admin": True},
    )
    assert response.status_code == 200
    assert response.json() == {"id": 2, "username": "test_user", "name": "Test User", "is_admin": True}

def test_read_user():
    headers = get_auth_headers()
    response = client.post(
        "/api/users/",
        json={"username": "other_user", "name": "Other User", "password": "securepass2", "is_admin": False},
        headers=headers
    )
    user_id = response.json()["id"]
    response = client.get(f"/api/users/{user_id}", headers=headers)
    assert response.status_code == 200
    assert response.json() == {"id": user_id, "username": "other_user", "name": "Other User", "is_admin": False}

def test_read_users():
    headers = get_auth_headers()
    client.post(
        "/api/users/",
        json={"username": "test_user1", "name": "Test User 1", "password": "securepass3", "is_admin": True},
        headers=headers
    )
    client.post(
        "/api/users/",
        json={"username": "test_user2", "name": "Test User 2", "password": "securepass4", "is_admin": False},
        headers=headers
    )

    response = client.get("/api/users/", headers=headers)
    assert response.status_code == 200
    # i think we have the previous users from the previous tests, so lets just check with "in"
    j = response.json()
    assert {"id": 3, "username": "test_user1", "name": "Test User 1", "is_admin": True} in j
    assert {"id": 4, "username": "test_user2", "name": "Test User 2", "is_admin": False} in j
    headers = get_auth_headers(False)  # nope, you're not admin
    response = client.get("/api/users/", headers=headers)
    assert response.status_code == 401  # yikes! you're not admin, debil :D

def test_delete_user():
    headers = get_auth_headers()
    response = client.post(
        "/api/users/",
        json={"username": "delete_user", "name": "Delete User", "password": "securepass5", "is_admin": False},
        headers=headers
    )
    j = response.json()
    print(j)
    user_id = j["id"]
    response = client.delete(f"/api/users/{user_id}", headers=headers)
    assert response.status_code == 200
    assert response.json() == {"id": user_id, "username": "delete_user", "name": "Delete User", "is_admin": False}
    # now lets delete non existing user
    response = client.delete(f"/api/users/{user_id}", headers=headers)
    assert response.status_code == 404  # yikes!

