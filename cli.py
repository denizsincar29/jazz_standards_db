# A cli program for the user to interact with the database of jazz standards

import click  # argparse is bad!
from db import crud, Session, init_db
import os

@click.group()
def cli():
    pass

@cli.command()
@click.argument('username')
@click.argument('name')
def add_user(username, name):
    db = Session()
    user = crud.create_user(db, username, name)
    print(user)
    db.close()

# get a user either by id or by username
@cli.command()
@click.option('--id', type=int, help='User ID', default=None)
@click.option('--username', help='Username', default=None)
def get_user(id=None, username=None):
    db = Session()
    user = crud.get_user(db, user_id=id, username=username)
    print(user)
    db.close()

@cli.command()
@click.option('--skip', default=0, help='Skip this many records')
@click.option('--limit', default=100, help='Limit the number of records to this')
def users(skip, limit):
    db = Session()
    users = crud.get_users(db, skip, limit)
    for user in users:
        print(user)
    db.close()

@cli.command()
@click.argument('user_id', type=int)
def delete_user_by_id(user_id):
    db = Session()
    crud.delete_user(db, user_id)
    db.close()

@cli.command()
@click.argument('username')
def delete_user(username):
    db = Session()
    user = crud.get_user(db, username=username)
    if user:
        crud.delete_user(db, user.id)
    else:
        click.echo(f"User {username} not found")
    db.close()

@cli.command()
@click.argument('title')
@click.argument('composer')
def add_jazz_standard(title, composer):
    db = Session()
    jazz_standard = crud.add_jazz_standard(db, title, composer)
    print(jazz_standard)
    db.close()

@cli.command()
@click.argument('jazz_standard_id', type=int)
def get_jazz_standard_by_id(jazz_standard_id):
    db = Session()
    jazz_standard = crud.get_jazz_standard(db, jazz_standard_id)
    print(jazz_standard)
    db.close()

@cli.command()
@click.argument('title')
def get_jazz_standard(title):
    db = Session()
    jazz_standard = crud.get_jazz_standard(db, title=title)
    print(jazz_standard)
    db.close()

@cli.command()
@click.argument('search')
def search_jazz_standard(search):
    db = Session()
    jazz_standards = crud.search_jazz_standard(db, search)
    for jazz_standard in jazz_standards:
        print(jazz_standard)
    db.close()

@cli.command()
@click.option('--skip', default=0, help='Skip this many records')
@click.option('--limit', default=100, help='Limit the number of records to this')
def jazz_standards(skip, limit):
    db = Session()
    jazz_standards = crud.get_jazz_standards(db, skip, limit)
    for jazz_standard in jazz_standards:
        print(jazz_standard)
    db.close()

@cli.command()
@click.argument('jazz_standard_id', type=int)
def delete_jazz_standard(jazz_standard_id):
    db = Session()
    crud.delete_jazz_standard(db, jazz_standard_id)
    db.close()

@cli.command()
@click.argument('user_id', type=int)
@click.argument('jazz_standard_id', type=int)
def add_standard_to_user_by_id(user_id, jazz_standard_id):
    db = Session()
    user_standard = crud.add_standard_to_user(db, user_id, jazz_standard_id)
    print(user_standard)
    db.close()

@cli.command()
@click.argument('username')
@click.argument('jazz_standard_name')
def add_standard_to_user(username, jazz_standard_name):
    db = Session()
    user = crud.get_user(db, username=username)
    jazz_standard = crud.get_jazz_standard(db, title=jazz_standard_name)
    if user and jazz_standard:
        user_standard = crud.add_standard_to_user(db, user.id, jazz_standard.id)
        print(user_standard)
    else:
        click.echo(f"User {username} or Jazz Standard {jazz_standard_name} not found")
    db.close()

@cli.command()
@click.argument('username')
def get_standards_for_user(username):
    db = Session()
    user = crud.get_user(db, username=username)
    if user:
        standards = crud.get_user_standards(db, user.id)
        for standard in standards:
            print(standard)
    else:
        click.echo(f"User {username} not found")
    db.close()

@cli.command()
@click.argument('id', type=int)
def get_standards_for_user_by_id(id):
    db = Session()
    standards = crud.get_standards_for_user_by_username(db, id)
    for standard in standards:
        print(standard)
    db.close()

# now make a command that prints every user and their standards
@cli.command()
def users_play_standards():
    db = Session()
    users = crud.get_users(db)
    for user in users:
        print(user)
        standards = crud.get_user_standards(db, user.id)
        for standard in standards:
            print(f"    {standard}")
    db.close()


if __name__ == '__main__':
    init_db()
    cli()