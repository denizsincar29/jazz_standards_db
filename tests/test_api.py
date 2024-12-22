import pytest
import base64
import os
os.environ["TEST_ENV"] = "2"  # use other test database
from db.models import JazzStyle # noqa
from fastapi.testclient import TestClient  # noqa
from app import app  # noqa

client = TestClient(app)

BASIC_AUTH = base64.b64encode(b"auth_admin:qwerty").decode("utf-8")
BASIC_AUTH_USER = base64.b64encode(b"auth_user:qwerty").decode("utf-8")

# Create the first user for authentication
def create_or_get_auth_user(is_admin=True):
    auth_username = "auth_admin" if is_admin else "auth_user"
    bauth = BASIC_AUTH if is_admin else BASIC_AUTH_USER
    response = client.get("/api/users/me", headers={"Authorization": f"Basic {bauth}"})
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

# assert 2 jsons but ignore id. It can be everything, not just users, just ignore id.
def assert_jsons(j1, j2):
    assert isinstance(j1, dict)
    assert isinstance(j2, dict)
    # copy the dicts and delete the id key
    j1 = j1.copy()
    j2 = j2.copy()
    del j1["id"]  # j1 is guaranteed to have id key because it's from the db
    if "id" in j2:
        del j2["id"]  # j2 may not have id key because it's template that we want to compare
    assert j1 == j2  # we do this to view beautiful diff of pytest


@pytest.fixture(scope="module")
def headers():
    print("Setting up the test db")
    # Create the first user for authentication
    admin = get_auth_headers(True)
    user = get_auth_headers(False)
    yield admin, user
    print("Tearing down the test db")
    # send /api/teardown
    response = client.get("/api/teardown")  # it doesn't need any headers because it's used for testing
    assert response.status_code == 200
    # remove the test db file
    try:
        os.remove("test.db")
    except FileNotFoundError:
        pass

def test_read_root(headers):
    admin, user = headers
    response = client.get("/api/", headers=admin)
    assert response.status_code == 200
    # well, check that the response is a dict but not the content, and it has keys users and jazz standards
    assert isinstance(response.json(), dict)
    assert "users" in response.json()
    assert "jazz_standards" in response.json()

def test_create_user():
    response = client.post(
        "/api/users/",
        json={"username": "test_user", "name": "Test User", "password": "securepass", "is_admin": True},
    )
    assert response.status_code == 200, response.text
    assert_jsons(response.json(), {"username": "test_user", "name": "Test User", "is_admin": True})  # in test_crud.py i didn't think about ignoring id, so there i matched ids perfectly per each test xD

def test_read_user(headers):
    # create a user
    response = client.post(
        "/api/users/",
        json={"username": "other_user", "name": "Other User", "password": "securepass2", "is_admin": False},
        headers=headers[0]
    )
    user_id = response.json()["id"]
    # read the user
    response = client.get(f"/api/users/{user_id}", headers=headers[0])
    assert response.status_code == 200, response.text
    assert_jsons(response.json(), {"username": "other_user", "name": "Other User", "is_admin": False})

def test_read_users(headers):
    client.post(
        "/api/users/",
        json={"username": "test_user1", "name": "Test User 1", "password": "securepass3", "is_admin": True},
        headers=headers[0]
    )
    client.post(
        "/api/users/",
        json={"username": "test_user2", "name": "Test User 2", "password": "securepass4", "is_admin": False},
        headers=headers[0]
    )
    response = client.get("/api/users/", headers=headers[0])
    assert response.status_code == 200, response.text
    # delete all dicts that's name is not "Test User 1" or "Test User 2" and assert that they are the same
    j = response.json()
    assert isinstance(j, list)
    new_j = [i for i in j if i["name"] in ["Test User 1", "Test User 2"]]
    assert len(new_j) == 2
    assert_jsons(new_j[0], {"username": "test_user1", "name": "Test User 1", "is_admin": True})
    response = client.get("/api/users/", headers=headers[1])  # non admin user
    assert response.status_code == 401  # yikes! you're not admin, debil :D

def test_delete_user(headers):
    response = client.post(
        "/api/users/",
        json={"username": "delete_user", "name": "Delete User", "password": "securepass5", "is_admin": False},
        headers=headers[0]
    )
    j = response.json()
    print(j)
    user_id = j["id"]
    response = client.delete(f"/api/users/{user_id}", headers=headers[0])
    assert response.status_code == 200, response.text
    assert response.json() == {"id": user_id, "username": "delete_user", "name": "Delete User", "is_admin": False}
    # now lets delete non existing user
    response = client.delete(f"/api/users/{user_id}", headers=headers[0])
    assert response.status_code == 404, f"User should have been died already! Expected 404, got {response.status_code}"

def test_login_api(headers):
    response = client.post(
        "/api/login",
        json={"username": "test_user", "password": "securepass"},
    )
    assert response.status_code == 200
    assert "cookie_token" in response.cookies
    assert "username" in response.cookies

def test_read_user_me(headers):
    response = client.get("/api/users/me", headers=headers[0])
    assert response.status_code == 200
    assert response.json()["username"] == "auth_admin"

def test_create_jazz_standard(headers):
    response = client.post(
        "/api/jazz_standards/",
        json={"title": "Giant Steps", "composer": "John Coltrane", "style": JazzStyle.bebop.value},
        headers=headers[0]
    )
    assert response.status_code == 200
    assert response.json()["title"] == "Giant Steps"

def test_read_jazz_standard(headers):
    response = client.post(
        "/api/jazz_standards/",
        json={"title": "Blue Bossa", "composer": "Kenny Dorham", "style": JazzStyle.bossa_nova.value},
        headers=headers[0]
    )
    j = response.json()
    print(j)
    standard_id = j["id"]
    response = client.get(f"/api/jazz_standards/{standard_id}", headers=headers[0])
    assert response.status_code == 200
    assert response.json()["title"] == "Blue Bossa"

def test_read_jazz_standards(headers):
    response = client.get("/api/jazz_standards/", headers=headers[0])
    assert response.status_code == 200
    assert isinstance(response.json(), list)
    response = client.get("/api/jazz_standards/", headers=headers[1])
    assert response.status_code == 401  # non-admin user

def test_delete_jazz_standard(headers):
    response = client.post(
        "/api/jazz_standards/",
        json={"title": "Autumn Leaves", "composer": "Joseph Kosma", "style": JazzStyle.swing.value},
        headers=headers[0]
    )
    standard_id = response.json()["id"]
    response = client.delete(f"/api/jazz_standards/{standard_id}", headers=headers[0])
    assert response.status_code == 200
    response = client.get(f"/api/jazz_standards/{standard_id}", headers=headers[0])
    assert response.status_code == 404

def test_add_standard_to_user(headers):
    # Create standard
    response = client.post(
        "/api/jazz_standards/",
        json={"title": "So What", "composer": "Miles Davis", "style": JazzStyle.modal.value},
        headers=headers[0]
    )
    assert response.status_code == 200, response.text
    standard_id = response.json()["id"]
    
    # Add standard to user
    response = client.post(
        f"/api/users/auth_admin/jazz_standards/{standard_id}",
        headers=headers[0]
    )
    assert response.status_code == 200

def test_get_user_standards(headers):
    response = client.get("/api/users/auth_admin/jazz_standards/", headers=headers[0])
    assert response.status_code == 200
    assert isinstance(response.json(), list)

def test_delete_user_standard(headers):
    # Create and add standard
    response = client.post(
        "/api/jazz_standards/",
        json={"title": "Take Five", "composer": "Paul Desmond", "style": JazzStyle.swing.value},
        headers=headers[0]
    )
    standard_id = response.json()["id"]
    client.post(
        f"/api/users/auth_admin/jazz_standards/{standard_id}",
        headers=headers[0]
    )
    
    # Delete user standard
    response = client.delete(
        f"/api/users/auth_admin/jazz_standards/{standard_id}",
        headers=headers[0]
    )
    assert response.status_code == 200

def test_teardown():
    response = client.get("/api/teardown")
    assert response.status_code == 200
    assert response.json() == {"message": "Database dropped"}