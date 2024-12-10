# Jazz Standards Database

This project is a Jazz Standards Database application that allows users to manage jazz standards and save a list of standards that you play. The application provides a CLI and a REST API for interacting with the database.

## Features

- **User Management**: Create, read, update, and delete users.
- **Jazz Standards Management**: Add, read, and delete jazz standards.
- **save what you play**: Associate jazz standards with users.
- **CLI**: Command-line interface for managing users and jazz standards.
- **REST API**: RESTful API for interacting with the database.

## Installation

Clone the repository:
    ```sh
    git clone https://github.com/denizsincar29/jazz_standards_db.git
    cd jazz_standards_db
    ```


## Usage

### CLI

The CLI provides commands for managing users and jazz standards. Here are some examples:

- Add a user:
    ```sh
    python -m cli add_user <username> <name>
    ```

- Delete a user by ID:
    ```sh
    python -m cli delete_user_by_id <user_id>
    ```

- Add a jazz standard:
    ```sh
    python -m cli add_jazz_standard <title> <composer>
    ```

- Associate a jazz standard with a user:
    ```sh
    python -m cli add_standard_to_user <username> <jazz_standard_name>
    ```

### API

The API provides endpoints for managing users and jazz standards. Here are some examples:

- Create a user:
    ```sh
    curl -X POST "http://localhost:8000/api/users/" -H "Content-Type: application/json" -d '{"username": "test_user", "name": "Test User", "is_admin": true}'
    ```

- Get a user by ID:
    ```sh
    curl -X GET "http://localhost:8000/api/users/1"
    ```

- Add a jazz standard:
    ```sh
    curl -X POST "http://localhost:8000/api/jazz_standards/" -H "Content-Type: application/json" -d '{"title": "All the Things You Are", "composer": "Jerome Kern", "style": "swing"}'
    ```

- Associate a jazz standard with a user:
    ```sh
    curl -X POST "http://localhost:8000/api/users/1/jazz_standards/1"
    ```

## Testing

Run the tests using pytest:
```sh
pytest
```

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

## Contributing

Contributions are welcome! Please open an issue or submit a pull request.