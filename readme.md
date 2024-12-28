# Jazz Standards Database

This project is a Jazz Standards Database application that allows users to manage there list of jazz standards and save a list of standards that you play. The application provides a CLI and a REST API for interacting with the database.

## Features

- **User Management**: Create, read, update, and delete users.
- **Jazz Standards Management**: Add, read, and delete jazz standards.
- **save what you play**: Associate jazz standards with users.
- **CLI**: Command-line interface for managing users and jazz standards.
- **REST API**: RESTful API for interacting with the database.
- **Docker**: Dockerfile for building and running the application in a container.

## Installation
    From now on, there is an option to use docker+docker-compose to run the application. You can skip the next steps and go to the [Docker](#docker) section if you want to use docker.

- Clone the repository:
    ```sh
    git clone https://github.com/denizsincar29/jazz_standards_db.git
    cd jazz_standards_db
    ```
- Install the dependencies:
    ```sh
    pip install -r requirements.txt
    ```

    - PostgreSQL is used as the database. You can install it using the following command on unix:
    ```sh
    sudo apt update
    sudo apt install postgresql
    ```
    on windows you can download the installer from [here](https://www.postgresql.org/download/).
- Create an environment
    create a .env file in the root directory and add the following lines (you can change the values according to your needs):
    ```env
    DB_USER = jazz
    DB_PASSWORD = jazz
    DB_NAME = jazz
    DB_HOST = localhost
    ```
- create the database and tables:
    Edit your sqls/init.sql file according to your .env file. Then, run the following command:
    ```sh
    psql -U postgres -f sqls/init.sql
    ```

    All the tables will be created via sqlalchemy, so the init.sql just creates the database and the user.
- Run the application:
    ```sh
    uvicorn main:app --reload
    ```

I think you can easily switch to sqlite. In your .env file, you can set USE_SQLITE to True. Then, you can run the application without installing PostgreSQL.
SQLite method is not meant to be used with docker. The container is already configured to use PostgreSQL.

## Docker
From now on, you can use docker to run the application. For this, you need to have docker and docker-compose installed on your machine. You can install them from [here](https://docs.docker.com/get-docker/).
To build and run the application, you can use the following command:
```sh
docker-compose up --build
```
After the build is completed, you can access the application from your browser by going to http://localhost:8000.
You can also run the application in the background:
command:
```sh
docker-compose up -d
# stop:
docker-compose down
```



## Usage

### CLI

The CLI provides commands for managing users and jazz standards. Here are some examples:

- Add a user:
    ```sh
    python -m cli add-user <username> <name>
    ```

- Delete a user by ID:
    ```sh
    python -m cli delete-user-by-id <user_id>
    ```

- Add a jazz standard:
    ```sh
    python -m cli add-jazz-standard <title> <composer>
    ```

- Associate a jazz standard with a user:
    ```sh
    python -m cli add-standard-to-user <username> <jazz_standard_name>
    ```

### API

The API provides endpoints for managing users and jazz standards. Here are some examples:

#### Authorization

Set the `AUTH` environment variable to the desired Authorization header value, e.g.:

```sh
export AUTH="Bearer YOUR_TOKEN_HERE"
# or
export AUTH="Basic BASE64_ENCODED_USERNAME_PASSWORD"
```

Use this variable in your API requests to include the Authorization header.

#### Examples

- Create a user:
    ```sh
    curl -X POST "http://localhost:8000/api/users/" -H "Content-Type: application/json" -d '{"username": "test_user", "name": "Test User", "is_admin": false}'
    ```

- Get a user by ID:
    ```sh
    curl -X GET "http://localhost:8000/api/users/1" -H "Authorization: $AUTH"
    ```

- Add a jazz standard:
    ```sh
    curl -X POST "http://localhost:8000/api/jazz_standards/" -H "Content-Type: application/json" -H "Authorization: $AUTH" -d '{"title": "All the Things You Are", "composer": "Jerome Kern", "style": "swing"}'
    ```

- Associate a jazz standard with a user:
    ```sh
    curl -X POST "http://localhost:8000/api/users/1/jazz_standards/1" -H "Authorization: $AUTH"
    


## Testing

Run the tests using pytest:
```sh
pytest
# or run specific tests:
pytest tests/test_crud.py
pytest tests/test_api.py
```

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

## Contributing

Contributions are welcome! Please open an issue or submit a pull request.
