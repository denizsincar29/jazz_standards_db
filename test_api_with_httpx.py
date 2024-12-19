import httpx

url = "http://localhost:8000/api"

with httpx.Client() as client:
    print("Creating user")
    response = client.post(f"{url}/users/", json={"username": "auth_user", "name": "Auth User", "password": "securepass", "is_admin": True})
    response.raise_for_status()
    user = response.json()
    print("Created user", user)
    print("Logging in")
    response = client.post(f"{url}/login", json={"username": "auth_user", "password": "securepass"})
    response.raise_for_status()
    cookies = response.cookies
    print("Logged in", cookies)
    cookie_token = cookies.get("cookie_token")
    assert cookie_token is not None
    headers = {"Authorization": f"Bearer {cookie_token}"}
    print("Reading root")
    response = client.get(f"{url}/", headers=headers)
    response.raise_for_status()
    print("Root response", response.json())