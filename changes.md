# 08.12.2024

- Начата работа над проектом
- Созданы модели данных на sqlalchemy/sqlite
- Написаны тесты для моделей на unittest

# 09.12.2024

- исправлено очень много ошибок с crud операциями и тестами.
- закончена реализация базы данных
    На данном этапе я надоел всем в чате sqlalchemy!
- переписаны тесты на pytest
- Сделана небольшая консольная утилита для работы с базой данных
- Начата работа над API

# 10.12.2024

- Написаны pydantic модели для валидации данных
- Добавлен почему-то забытый столбец jazz style в базу данных
- Сделана первая версия API
- Написаны тесты для API, они работают поотдельности из-за проблем с переменными окружения
- Начата работа над веб-интерфейсом
- сделана процедура авторизации

# 11.12.2024

- Переписан crud и тесты для него
- Убран легаси код из моделей
- Добавлены новые поля в модели
- Добавлен файл sapg.py для экспериментов с sqlalchemy в консоли питона

# 12.12.2024 - 16.12.2024

- Не работал над проектом, так как был занят другими задачами

# 17.12.2024

- Завершена работа над авторизацией с паролем и токенами
- переписаны тесты для API (почти все)

# 18.12.2024 - 19.12.2024

- По немногу исправлял ошибки в тестах и в самом API

# 20.12.2024

- доисправил ошибки в тестах
- сделал так, чтобы тесты работали вместе
- исправил ошибки авторизации и куков
- обновил CLI утилиту, чтобы она работала с новыми полями
- Добавил и починил новые тесты для API
- добавил форму регистрации
- Добавил форму добавления джазового стандарта
- Если в базе данных нет пользователей, то один раз в жизни можно зарегистрироваться как админ через API. Если админ уже есть, то он может создать нового админа сам через API
- Начал читать про postgresql

# 21.12.2024

- Начал переносить проект на postgresql

# 22.12.2024

- Закончил перенос проекта на postgresql
    Комментарий: После переноса проекта на новую СУБД мне постгрес так навалял за детские ошибки с sqlalchemy/orm, что пришлось все модели переписать уже похорошему, читая про каскады и связи между таблицами.
    Но теперь я доволен. SQLite был слишком добрый, и все мои ошибки он прощал, а вот постгрес - нет.
- Начал работу над запиской к проекту

# 23.12.2024

- Поработал над фронтендом:
    Теперь можно добавлять джазовые стандарты через веб-интерфейс. Стандарты сортируются по стилю джаза.
- починил регистрацию и авторизацию через веб-интерфейс
- Исправил очередные ошибки в CRUD операциях
- начал работу над телеграм ботом

# 25.12.2024

- Отошёл от телеграм бота.
- Добавил отвязку джазовых стандартов от пользователей, по простому, удаление из своего списка.

# 27.12.2024

- Доделал docker файлы! Теперь можно запустить проект в докере

# 28.12.2024

- Обновил readme.md
    От времени последнего обновления я поменял многое в проекте, и теперь это отражено в readme.md. добавил в readme инструкцию по запуску проекта в докере.
- добавил инициализацию базы данных в .sql файл для недокеристов

# 29.12.2024

- исправил ошибку после переименования app.py в main.py, в докере оставалась команда uvicorn app:app
- Страница входа теперь не направляет на 404/401 при неправильном вводе данных, а показывает под заголовком <p>Неправильный логин или пароль</p>