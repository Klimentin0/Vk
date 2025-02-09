# Профильное задание от VK

---
- [Установка и запуск](#установка-и-запуск)
- [Краткое описание](#краткое-описание)

## Установка и запуск

### Предварительные установки
На системе должны быть установлены 
> **git** 

> **docker engine** 

> **docker compose plugin**

Далее создаём рабочую директорию программы при помощи:
```
git clone https://github.com/Klimentin0/Vk
```
Как директория будет создана далее:
```
cd Vk

docker compose up

```

## Краткое описание


## Postgres

При билде контейнера с постгрес в компоузеуказал текущую папку для вольюма, чтобы прям тут хранилась DB
- postgres_data/ отправляется в .gitignore

```
volumes:
      - ./postgres_data:/var/lib/postgresql/data
```