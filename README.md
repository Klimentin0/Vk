# тестовое задание
## Postgres

При билде контейнера с постгрес в компоузеуказал текущую папку для вольюма, чтобы прям тут хранилась DB
- postgres_data/ отправляется в .gitignore

```
volumes:
      - ./postgres_data:/var/lib/postgresql/data
```