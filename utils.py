# import to url decode the jazz_standard string
from urllib.parse import unquote

# Description: Utility functions for the API

# Function to convert user to dictionary
# it will be unpacked to a kwarg in crud functions
def userstr(user):
    # if integer, return {"user_id": user}
    if isinstance(user, int):
        return {"user_id": user}
    # if string but integer, return {"user_id": int(user)}
    elif user.isdigit():
        return {"user_id": int(user)}
    else:
        return {"username": user}

# Function to check if the user is an username
def is_username(user):
    # if integer, return False
    if isinstance(user, int):
        return False
    # if string but integer, return False
    elif user.isdigit():
        return False
    else:
        return True


# Function to convert jazz_standard to dictionary
# it will be unpacked to a kwarg in crud functions
def jazz_standardstr(jazz_standard):
    # if integer, return {"jazz_standard_id": jazz_standard}
    if isinstance(jazz_standard, int):
        return {"jazz_standard_id": jazz_standard}
    # if string but integer, return {"jazz_standard_id": int(jazz_standard)}
    elif jazz_standard.isdigit():
        return {"jazz_standard_id": int(jazz_standard)}
    else:
        return {"title": unquote(jazz_standard)}

