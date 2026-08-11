# Прогон всех тестов в Docker

Дата: 2026-08-11

## Задача

Сделать так, чтобы все тесты проекта — go-тесты бэкенда и vitest фронтенда — запускались одной
командой в Docker, без установки чего-либо на хост, и попутно вычистить из репозитория то, что
после этого становится лишним.

Локальный прогон — единственный. В CI тесты не гоняются.

## Почему это нужно

Сегодня `go test ./internal/...` требует от хоста:

- клиентов PostgreSQL 12–18 в `backend/tools/postgresql/postgresql-{версия}/bin`
  (`config.go:151` вызывает `VerifyPostgresesInstallation`, при отсутствии — `os.Exit(1)`);
- файла `backend/.env` (без него `loadEnvVariables` завершает процесс);
- установленного `goose` и прогнанных миграций;
- каталогов `postgresus-data/{backups,temp}` рядом с репозиторием;
- реального Telegram-бота: `TEST_TELEGRAM_BOT_TOKEN` и `TEST_TELEGRAM_CHAT_ID` проверяются в
  `config.go:209-216` через `os.Exit(1)`, а два теста шлют настоящие сообщения в API;
- двенадцати свободных портов, включая 5000 — на macOS он занят AirPlay Receiver.

На macOS ARM первый пункт невыполним разумной ценой: `postgresql@12` и `@13` в Homebrew disabled,
остаётся компиляция семи версий из исходников.

## Решение

Тесты исполняются внутри контейнера, где клиенты PostgreSQL ставятся из pgdg-репозитория тем же
способом, что и в прод-образе. Инфраструктура — соседние сервисы в сети compose.

### Состав `docker-compose.test.yml` (корень репозитория)

Инфраструктура, порты наружу не публикуются:

| Сервис | Образ | Роль |
|---|---|---|
| `app-db` | `postgres:17` | основная БД приложения |
| `test-postgres-12` … `test-postgres-18` | `postgres:{12..18}` | цели бэкапа и рестора |
| `test-minio` | `minio/minio` | S3-сторэдж |
| `test-azurite` | `mcr.microsoft.com/azure-storage/azurite` | Azure Blob |
| `test-nas` | `dperson/samba` | NAS-сторэдж |
| `test-ftp` | `delfer/alpine-ftp-server` | FTP-сторэдж |

Прогонщики:

- `backend-tests` — образ из `backend/Dockerfile.test`: `golang:1.23.3-bookworm`, клиенты
  PostgreSQL 12–18 из pgdg, `goose`. Симлинки на бинари складываются в `/opt/pg-tools` — вне
  смонтированного кода, иначе bind-mount их перекроет. Монтируется только `backend/` в
  `/app/backend`; `GOPATH/pkg/mod` и build-cache — именованные volume'ы. Entrypoint: `goose up`,
  затем `go test -p=1 -count=1 ./internal/...`.
- `frontend-tests` — образ из `frontend/Dockerfile.test` с `npm ci` на этапе сборки; `node_modules`
  — именованный volume, исходники — bind-mount; команда `vitest run`. От бэкенд-стека не зависит,
  идёт параллельно.

Готовность постгресов — через `healthcheck` (`pg_isready`) и `depends_on: condition:
service_healthy`. Для MinIO, Azurite, Samba и FTP готовность проверяется TCP-пробой в entrypoint
прогонщика через bash `/dev/tcp`: так мы не гадаем, какие утилиты лежат внутри сторонних образов.
Самописного цикла ожидания в духе текущего CI (`nc -z` по списку портов на хосте) не остаётся.

`stilliard/pure-ftpd` заменяется на `delfer/alpine-ftp-server`: текущий образ собран только под
amd64. Замена мультиарочная, arm64 нативно.

### Состав `docker-compose.dev.yml`

- `dev-db` — БД для повседневной разработки, порт 5437 опубликован.
- `app` — на том же образе, что `backend-tests` (клиенты и Go в нём уже есть), команда
  `go run cmd/main.go`, смонтированный код, опубликованный порт 4005.

Фронтенд в дев-режиме продолжает жить на хосте: `npm run dev` ходит на `localhost:4005`.

### Команды

`make test` в корне поднимает инфраструктуру, дожидается healthcheck'ов, прогоняет оба
прогонщика и возвращает ненулевой код, если упал любой. Дополнительно: `make test-backend`,
`make test-frontend`, `make test-clean` (сброс volume'ов), `make test-logs` (логи сервисов
последнего прогона), `make run`, `make migration-*`.

Прогонщики запускаются через `docker compose run --rm`, поэтому код возврата `go test` и `vitest`
становится кодом возврата `make test`.

## Изменения в бэкенд-коде

### Адресация сервисов

Внутри сети compose у каждого сервиса своё DNS-имя, поэтому переменные-порты заменяются на
переменные-адреса. Количество переменных не растёт.

| Было | Стало |
|---|---|
| `TEST_POSTGRES_12_PORT` … `TEST_POSTGRES_18_PORT` | `TEST_POSTGRES_12_ADDR` … `TEST_POSTGRES_18_ADDR` (`test-postgres-12:5432`) |
| `TEST_MINIO_PORT` | `TEST_MINIO_ADDR` (`test-minio:9000`) |
| `TEST_AZURITE_BLOB_PORT` | `TEST_AZURITE_ADDR` (`test-azurite:10000`) |
| `TEST_NAS_PORT` | `TEST_NAS_ADDR` (`test-nas:445`) |
| `TEST_FTP_PORT` | `TEST_FTP_ADDR` (`test-ftp:21`) |
| `TEST_MINIO_CONSOLE_PORT` | удаляется |

Из тестов уходят захардкоженные `localhost` и `127.0.0.1`: `storages/model_test.go:264,310`,
NAS- и FTP-кейсы там же, `tests/postgresql_backup_restore_test.go`,
`databases/postgresql/readonly_user_test.go`. Разбор `host:port` — один хелпер в
`internal/util/testing`.

Записи с `localhost:5432` в фикстурах (`databases/testing.go:24`, `databases/controller_test.go`,
`restores/controller_test.go` и другие) остаются как есть: это данные сохраняемых сущностей, а не
адреса, куда тесты реально ходят. Единственный тест, который по ним пытается соединиться,
`Test_TestConnection_PermissionsEnforced`, штатно допускает 400 с ошибкой соединения.

### `.env` перестаёт быть обязательным

`loadEnvVariables` при отсутствии файла пишет предупреждение и продолжает работу с переменными
окружения вместо `os.Exit(1)`. Проверки `DATABASE_DSN` и `ENV_MODE` сохраняются.

### `POSTGRES_INSTALL_DIR` начинает работать

Сейчас `config.go:151` безусловно перетирает поле путём `<backendRoot>/tools/postgresql`, из-за
чего объявленная в структуре переменная мертва. Становится: значение задано — используем его, не
задано — прежний дефолт. Прогонщик выставляет `/opt/pg-tools`.

### Изоляция путей данных

В прогонщик монтируется только `backend/`, поэтому `DataFolder`, `TempFolder` и `SecretKeyPath`
(`config.go:155-157`) резолвятся в `/app/postgresus-data/...` внутри контейнера. Samba-сервис
переезжает с bind-mount `./temp/nas` на именованный volume — именно этот bind-mount оставил в
репозитории пять файлов в `backend/temp/nas`.

### Удаляемые тесты

Продакшн-код интеграций не трогается — только тесты и тестовые переменные.

- `notifiers/controller_test.go`: `Test_SendTestNotificationDirect_NotificationSent`,
  `Test_SendTestNotificationExisting_NotificationSent` и фикстура `createTelegramNotifier` —
  она вызывается только из этих двух тестов (строки 163 и 178), больше нигде. Фикстура
  `notifiers.CreateTestNotifier`, которой пользуются другие пакеты, работает на webhook-нотифаере
  и Telegram не касается.
- `storages/model_test.go`: тест-кейс Google Drive и его ветка в `validateEnvVariables`.
- `tests/postgresql_backup_restore_test.go`:
  `Test_BackupAndRestoreSupabase_PublicSchemaOnly_RestoreIsSuccessful`.
- `databases/postgresql/readonly_user_test.go`: Supabase-ветка со скипом на строке 328.
- `config.go`: поля и проверки `TestTelegram*`, `TestGoogleDrive*`, `TestSupabase*`,
  `TestMinioConsolePort`.

## Чистка репозитория

- `git rm -r --cached backend/temp` — пять файлов от прошлого прогона; правило `temp/` в
  `.gitignore` на уже отслеживаемые файлы не действует.
- `backend/docker-compose.yml.example` удаляется, его содержимое расходится по
  `docker-compose.test.yml` и `docker-compose.dev.yml`. Оба файла трекаются, копировать руками
  ничего не нужно. Корневой `docker-compose.yml.example` не трогаем — он про локальную сборку
  прод-образа.
- `backend/tools/` удаляется целиком: `download_linux.sh`, `download_macos.sh`,
  `download_windows.bat`, `readme.md`, `.gitignore`. Логика linux-скрипта переезжает в
  `Dockerfile.test`.
- `backend/Makefile` переписывается под докер; попутно чинится таргет `test` с виндовыми слэшами
  (`.\internal\...`), не работавший на macOS.
- `backend/.env.development.example` и `.env.production.example`: убираются все `TEST_*`.
- `backend/README.md`: раздел про копирование файлов и установку семи версий PostgreSQL меняется
  на `make test`.

## CI

Из `.github/workflows/ci-release.yml` удаляются job'ы `test-backend` и `test-frontend` вместе с
генерацией `.env` и секретами `TEST_TELEGRAM_*`, `TEST_SUPABASE_*`, `TEST_GOOGLE_DRIVE_*`.
Зависимости `determine-version.needs` и `build-only.needs` перевешиваются с тестовых job'ов на
линтовые, чтобы релиз не остался без гейта. Сборка и публикация образа, определение версии,
changelog и публикация helm-чарта не меняются.

## Критерии готовности

- На машине, где есть только докер и git, `git clone && make test` даёт зелёный прогон: без
  `.env`, brew, goose, секретов и свободного порта 5000.
- После прогона `git status` чистый: ни `backend/temp/nas`, ни `postgresus-data` в рабочем
  каталоге не появляется.
- Список тестов до и после правок различается ровно на перечисленные выше удаления: два
  телеграм-теста, один Supabase backup/restore, одна Supabase-ветка в `readonly_user_test.go`,
  один кейс Google Drive внутри табличного теста сторэджей. Сверка — по исходникам
  (`grep -rn '^func Test' backend/internal`): запустить `go test -list` на исходном коде нельзя,
  пока конфиг требует Telegram-переменные. Базовая отметка, снятая до правок: **237 тест-функций**,
  после чистки должно остаться **233** (удаляются четыре функции; кейс Google Drive — элемент
  таблицы внутри `Test_Storage_BasicOperations`, отдельной функцией не является). Все оставшиеся
  тесты проходят.
- Падение любого теста даёт ненулевой код возврата `make test`.
- Не поднявшийся сервис даёт внятную ошибку зависимости, а не таймаут посреди теста.

## Порядок работ

1. Удаление тестов на внешние сервисы и связанных полей конфига (`TestTelegram*`,
   `TestGoogleDrive*`, `TestSupabase*`, `TestMinioConsolePort`). Идёт первым: пока обязательная
   проверка Telegram-переменных жива, прогон в контейнере невозможен в принципе.
2. Правки конфига: адреса вместо портов, необязательный `.env`, работающий
   `POSTGRES_INSTALL_DIR`. Проверка на этом шаге — сборка и `go vet`; полноценно прогнать тесты
   можно будет только после шага 3, так как на macOS нет клиентов PostgreSQL 12–18.
3. `backend/Dockerfile.test`, `frontend/Dockerfile.test`, `docker-compose.test.yml`, таргеты
   Makefile. Первый зелёный прогон в докере.
4. `docker-compose.dev.yml` и перевод `make run` в контейнер.
5. Чистка репозитория, `.env`-примеры, README, CI.

## Риски

| Риск | Запасной ход |
|---|---|
| Нет пакета `postgresql-client-12` под arm64 в pgdg | Собирать образ прогонщика под `linux/amd64` через эмуляцию |
| Passive-режим FTP на `delfer/alpine-ftp-server` | Вернуть `stilliard/pure-ftpd` с `platform: linux/amd64` |
| Samba под arm64 | То же — принудительный `linux/amd64` |

Косвенное свидетельство в пользу arm64-клиентов: прод-образ собирается под `linux/arm64` и ставит
`postgresql-client-12` … `-18` тем же способом.

## Вне скоупа

- Перевод тестов на testcontainers.
- Изменения прод-кода интеграций (Google Drive-сторэдж, Telegram-нотифаер).
- Ветка `EnvMode == production` в `internal/util/tools/postgresql.go`.
- Devcontainer: поверх образа прогонщика добавляется позже почти бесплатно.
- Линтеры в докер-прогоне: `golangci-lint` и `eslint` остаются на хосте и в CI.
