# yt-thumbnails-go

gRPC прокси-сервис для загрузки thumbnail'ов с видеороликов Youtube.

## Зависимости:

- Go 1.21
- sqlite3

## Запуск:
```sh
# Сборка
# (пакет go-sqlite3 собирается больше 10 секунд)
make build

# Сервер
./build/server --addr=localhost:8080

# Клиент
# Указываем url как аргумент командной строки
./build/client --addr=localhost:8080 "https://www.youtube.com/watch?v=dQw4w9WgXcQ"

# Или можно указать файл с url на каждой строке
./build/client --async --max-parallel-requests=16 --input=testdata/test.txt --output=images
```

## a
