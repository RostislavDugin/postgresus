# Dockerized Tests Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Все тесты проекта (go-тесты бэкенда и vitest фронтенда) запускаются одной командой `make test` в Docker, без единой зависимости на хосте кроме самого Docker.

**Architecture:** Тесты исполняются внутри контейнера, где клиенты PostgreSQL 12–18 ставятся из pgdg-репозитория тем же способом, что и в прод-образе. Инфраструктура (БД приложения, семь тестовых постгресов, MinIO, Azurite, Samba, FTP) — соседние сервисы в сети compose, порты наружу не публикуются, адресация по DNS-именам. Тесты на внешние SaaS (Telegram, Google Drive, Supabase) удаляются, вместе с ними уходят обязательные секреты.

**Tech Stack:** Docker Compose, Go 1.23.3, goose v3.24.3, PostgreSQL 12–18 (pgdg apt), MinIO, Azurite, Samba, vsftpd, Node 24 + vitest.

**Спека:** `docs/superpowers/specs/2026-08-11-dockerized-tests-design.md`

## Global Constraints

- Go — ровно `1.23.3` (как в `backend/go.mod` и прод-`Dockerfile`).
- goose — `v3.24.3` (версия из прод-`Dockerfile`).
- Base-образ прогонщика — `golang:1.23.3-bookworm`; клиенты PostgreSQL ставятся из
  `http://apt.postgresql.org/pub/repos/apt $(lsb_release -cs)-pgdg main`, версии 12, 13, 14, 15,
  16, 17, 18.
- Go-тесты запускаются только с `-p=1` — пакеты делят одну БД приложения.
- Порты сервисов наружу из `docker-compose.test.yml` не публикуются.
- Продакшн-код интеграций (Google Drive-сторэдж, Telegram-нотифаер) не трогаем — уезжают только
  тесты и тестовые переменные окружения.
- Все команды выполняются из корня репозитория `/Users/artem/Documents/Work/go/src/postgresus`,
  если не сказано иное.
- Базовая отметка: до правок `grep -rn '^func Test' backend/internal --include='*_test.go' | wc -l`
  даёт **237**. Task 1 удаляет четыре тест-функции, после него должно стать **233**.

---

## File Structure

**Создаются:**

- `backend/internal/util/testing/addr.go` — разбор `host:port` из ADDR-переменных.
- `backend/internal/util/testing/addr_test.go` — тесты разбора.
- `backend/Dockerfile.test` — образ прогонщика: Go + клиенты PostgreSQL 12–18 + goose.
- `backend/scripts/run-tests.sh` — entrypoint прогонщика: ждёт TCP-сервисы, гонит миграции, гонит тесты.
- `frontend/Dockerfile.test` — образ для vitest с вшитым `npm ci`.
- `docker-compose.test.yml` — тестовый стенд целиком.
- `docker-compose.dev.yml` — `dev-db` + `app` для повседневной разработки.
- `Makefile` (в корне) — единственная точка входа.

**Изменяются:**

- `backend/internal/config/config.go` — ADDR вместо портов, необязательный `.env`, работающий
  `POSTGRES_INSTALL_DIR`, удаление внешних тестовых полей.
- `backend/internal/features/tests/postgresql_backup_restore_test.go` — ADDR, удаление Supabase.
- `backend/internal/features/databases/databases/postgresql/readonly_user_test.go` — ADDR,
  удаление Supabase.
- `backend/internal/features/storages/model_test.go` — ADDR, удаление Google Drive.
- `backend/internal/features/notifiers/controller_test.go` — удаление двух Telegram-тестов.
- `backend/.env.development.example` — без `TEST_*`.
- `backend/README.md` — инструкция сводится к `make test`.
- `.github/workflows/ci-release.yml` — без тестовых job'ов.

**Удаляются:**

- `backend/tools/` целиком (4 файла + `.gitignore`).
- `backend/docker-compose.yml.example`.
- `backend/Makefile` (его таргеты переезжают в корневой `Makefile`).
- `backend/temp/` из индекса git.

---

### Task 1: Удалить тесты на внешние сервисы

Идёт первым: пока `config.go` требует `TEST_TELEGRAM_BOT_TOKEN` через `os.Exit(1)`, прогон в
контейнере невозможен в принципе.

**Files:**
- Modify: `backend/internal/features/notifiers/controller_test.go` (удалить строки 158-201 и функцию `createTelegramNotifier`, 1045-1056)
- Modify: `backend/internal/features/tests/postgresql_backup_restore_test.go` (удалить `Test_BackupAndRestoreSupabase_PublicSchemaOnly_RestoreIsSuccessful` 116-243, `createSupabaseDatabaseViaAPI` 927-978, `createSupabaseRestoreViaAPI` 979-1010)
- Modify: `backend/internal/features/databases/databases/postgresql/readonly_user_test.go` (удалить `Test_CreateReadOnlyUser_Supabase_UserCanReadButNotWrite` 324-429)
- Modify: `backend/internal/features/storages/model_test.go` (удалить блок Google Drive 150-168)
- Modify: `backend/internal/config/config.go` (удалить поля и проверки)

**Interfaces:**
- Consumes: ничего.
- Produces: конфиг без полей `TestGoogleDriveClientID`, `TestGoogleDriveClientSecret`,
  `TestGoogleDriveTokenJSON`, `TestTelegramBotToken`, `TestTelegramChatID`, `TestSupabaseHost`,
  `TestSupabasePort`, `TestSupabaseUsername`, `TestSupabasePassword`, `TestSupabaseDatabase`,
  `TestMinioConsolePort`. Task 2 переписывает то, что осталось.

- [ ] **Step 1: Зафиксировать базовую отметку количества тестов**

```bash
grep -rn '^func Test' backend/internal --include='*_test.go' | wc -l
```

Ожидается: `237`. Если число другое — остановиться и сверить с планом, дальше идти нельзя.

- [ ] **Step 2: Удалить два Telegram-теста и их фикстуру**

В `backend/internal/features/notifiers/controller_test.go` удалить целиком функции
`Test_SendTestNotificationDirect_NotificationSent` и `Test_SendTestNotificationExisting_NotificationSent`
(строки 158-204, следующая функция начинается на 206) и функцию `createTelegramNotifier`
(строки 1045-1056).

`createTelegramNotifier` вызывается только из этих двух тестов — проверить, что после удаления
не осталось ссылок:

```bash
grep -rn 'createTelegramNotifier' backend/internal
```

Ожидается: пусто.

Если после удаления импорт `telegram_notifier` или `config` в этом файле стал неиспользуемым —
удалить и его. Проверяется компиляцией на шаге 6.

- [ ] **Step 3: Удалить Supabase-тесты**

В `backend/internal/features/tests/postgresql_backup_restore_test.go` удалить:
- `Test_BackupAndRestoreSupabase_PublicSchemaOnly_RestoreIsSuccessful` (строки 116-243),
- `createSupabaseDatabaseViaAPI` (строки 927-978),
- `createSupabaseRestoreViaAPI` (строки 979-1010).

В `backend/internal/features/databases/databases/postgresql/readonly_user_test.go` удалить
`Test_CreateReadOnlyUser_Supabase_UserCanReadButNotWrite` (строки 324-429).

Проверить, что вспомогательные функции больше нигде не нужны:

```bash
grep -rn 'createSupabaseDatabaseViaAPI\|createSupabaseRestoreViaAPI\|TestSupabase' backend/internal
```

Ожидается: пусто.

- [ ] **Step 4: Удалить кейс Google Drive из тестов сторэджей**

В `backend/internal/features/storages/model_test.go` удалить блок строк 150-168 (от комментария
`// Add Google Drive storage test only if environment variables are available` до закрывающей
скобки `else`-ветки включительно) и импорт
`google_drive_storage "postgresus-backend/internal/features/storages/models/google_drive"`.

- [ ] **Step 5: Вычистить внешние поля из конфига**

В `backend/internal/config/config.go` удалить из структуры `EnvVariables` поля
`TestGoogleDriveClientID`, `TestGoogleDriveClientSecret`, `TestGoogleDriveTokenJSON`,
`TestMinioConsolePort`, `TestTelegramBotToken`, `TestTelegramChatID`, `TestSupabaseHost`,
`TestSupabasePort`, `TestSupabaseUsername`, `TestSupabasePassword`, `TestSupabaseDatabase`
(строки 33-35, 46, 59-68) вместе с комментариями `// testing Telegram` и `// testing Supabase`.

В блоке `if env.IsTesting` (строки 160-218) удалить проверки `TestMinioConsolePort`,
`TestTelegramBotToken`, `TestTelegramChatID`. Остальные проверки пока не трогать — их
переписывает Task 2.

- [ ] **Step 6: Проверить, что всё компилируется**

```bash
cd backend && go vet ./internal/... && go build ./...
```

Ожидается: без ошибок. Типичная ошибка на этом шаге — неиспользуемый импорт в затронутом файле;
удалить импорт и повторить.

- [ ] **Step 7: Проверить количество тестов**

```bash
grep -rn '^func Test' backend/internal --include='*_test.go' | wc -l
```

Ожидается: `233`. Удалены ровно четыре тест-функции: `Test_SendTestNotificationDirect_NotificationSent`,
`Test_SendTestNotificationExisting_NotificationSent`,
`Test_BackupAndRestoreSupabase_PublicSchemaOnly_RestoreIsSuccessful`,
`Test_CreateReadOnlyUser_Supabase_UserCanReadButNotWrite`. Кейс Google Drive в счёт не идёт — это
элемент таблицы внутри `Test_Storage_BasicOperations`, а не отдельная функция.

Если число другое — найти лишнее удаление или недоудалённую функцию и исправить, прежде чем идти
дальше.

- [ ] **Step 8: Commit**

```bash
git add backend/internal
git commit -m "REFACTOR (tests): remove tests depending on external SaaS"
```

---

### Task 2: Перевести тесты с портов на адреса сервисов

**Files:**
- Create: `backend/internal/util/testing/addr.go`
- Create: `backend/internal/util/testing/addr_test.go`
- Modify: `backend/internal/config/config.go`
- Modify: `backend/internal/features/tests/postgresql_backup_restore_test.go`
- Modify: `backend/internal/features/databases/databases/postgresql/readonly_user_test.go`
- Modify: `backend/internal/features/storages/model_test.go`

**Interfaces:**
- Consumes: конфиг после Task 1.
- Produces: `test_utils.SplitAddr(addr string) (string, int, error)` из пакета
  `postgresus-backend/internal/util/testing`; поля конфига `TestPostgres12Addr` …
  `TestPostgres18Addr`, `TestMinioAddr`, `TestAzuriteAddr`, `TestNasAddr`, `TestFtpAddr` (все
  `string`, формат `host:port`). Task 4 задаёт их значения в compose.

- [ ] **Step 1: Написать падающий тест разбора адреса**

Создать `backend/internal/util/testing/addr_test.go`:

```go
package testing

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_SplitAddr_ValidAddress_ReturnsHostAndPort(t *testing.T) {
	host, port, err := SplitAddr("test-postgres-16:5432")

	require.NoError(t, err)
	assert.Equal(t, "test-postgres-16", host)
	assert.Equal(t, 5432, port)
}

func Test_SplitAddr_NoPort_ReturnsError(t *testing.T) {
	_, _, err := SplitAddr("test-postgres-16")

	assert.Error(t, err)
}

func Test_SplitAddr_EmptyHost_ReturnsError(t *testing.T) {
	_, _, err := SplitAddr(":5432")

	assert.Error(t, err)
}

func Test_SplitAddr_NonNumericPort_ReturnsError(t *testing.T) {
	_, _, err := SplitAddr("test-minio:not-a-port")

	assert.Error(t, err)
}
```

- [ ] **Step 2: Убедиться, что тест падает**

```bash
cd backend && go test ./internal/util/testing/... -run Test_SplitAddr -v
```

Ожидается: ошибка компиляции `undefined: SplitAddr`.

- [ ] **Step 3: Реализовать разбор адреса**

Создать `backend/internal/util/testing/addr.go`:

```go
package testing

import (
	"fmt"
	"strconv"
	"strings"
)

// SplitAddr splits a "host:port" service address into its parts.
// Test services are addressed by their compose DNS name, so the host part
// is not necessarily localhost.
func SplitAddr(addr string) (string, int, error) {
	host, portStr, found := strings.Cut(addr, ":")
	if !found || host == "" || portStr == "" {
		return "", 0, fmt.Errorf("invalid address %q, expected host:port", addr)
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		return "", 0, fmt.Errorf("invalid port in address %q: %w", addr, err)
	}

	return host, port, nil
}
```

- [ ] **Step 4: Убедиться, что тест проходит**

```bash
cd backend && go test ./internal/util/testing/... -run Test_SplitAddr -v
```

Ожидается: PASS, четыре теста.

Этот пакет не ходит в БД, поэтому запускается без инфраструктуры.

- [ ] **Step 5: Заменить порты на адреса в конфиге**

В `backend/internal/config/config.go` заменить блок полей (строки 37-51 после Task 1) на:

```go
	TestPostgres12Addr string `env:"TEST_POSTGRES_12_ADDR"`
	TestPostgres13Addr string `env:"TEST_POSTGRES_13_ADDR"`
	TestPostgres14Addr string `env:"TEST_POSTGRES_14_ADDR"`
	TestPostgres15Addr string `env:"TEST_POSTGRES_15_ADDR"`
	TestPostgres16Addr string `env:"TEST_POSTGRES_16_ADDR"`
	TestPostgres17Addr string `env:"TEST_POSTGRES_17_ADDR"`
	TestPostgres18Addr string `env:"TEST_POSTGRES_18_ADDR"`

	TestMinioAddr   string `env:"TEST_MINIO_ADDR"`
	TestAzuriteAddr string `env:"TEST_AZURITE_ADDR"`
	TestNasAddr     string `env:"TEST_NAS_ADDR"`
	TestFtpAddr     string `env:"TEST_FTP_ADDR"`
```

Блок проверок `if env.IsTesting { ... }` (строки 160-218 до Task 1) заменить на компактный цикл:

```go
	if env.IsTesting {
		required := map[string]string{
			"TEST_POSTGRES_12_ADDR": env.TestPostgres12Addr,
			"TEST_POSTGRES_13_ADDR": env.TestPostgres13Addr,
			"TEST_POSTGRES_14_ADDR": env.TestPostgres14Addr,
			"TEST_POSTGRES_15_ADDR": env.TestPostgres15Addr,
			"TEST_POSTGRES_16_ADDR": env.TestPostgres16Addr,
			"TEST_POSTGRES_17_ADDR": env.TestPostgres17Addr,
			"TEST_POSTGRES_18_ADDR": env.TestPostgres18Addr,
			"TEST_MINIO_ADDR":       env.TestMinioAddr,
			"TEST_AZURITE_ADDR":     env.TestAzuriteAddr,
			"TEST_NAS_ADDR":         env.TestNasAddr,
			"TEST_FTP_ADDR":         env.TestFtpAddr,
		}

		for name, value := range required {
			if value == "" {
				log.Error("required test environment variable is empty", "variable", name)
				os.Exit(1)
			}
		}
	}
```

- [ ] **Step 6: Перевести тесты бэкапа/рестора на адреса**

В `backend/internal/features/tests/postgresql_backup_restore_test.go`:

Добавить импорт `test_utils "postgresus-backend/internal/util/testing"` — он уже есть в этом
файле, дополнительно ничего добавлять не нужно.

В `Test_BackupAndRestorePostgresql_RestoreIsSuccesful` и
`Test_BackupAndRestorePostgresqlWithEncryption_RestoreIsSuccessful` переименовать поле `port` в
`addr` и подставить новые поля конфига:

```go
	cases := []struct {
		name    string
		version string
		addr    string
	}{
		{"PostgreSQL 12", "12", env.TestPostgres12Addr},
		{"PostgreSQL 13", "13", env.TestPostgres13Addr},
		{"PostgreSQL 14", "14", env.TestPostgres14Addr},
		{"PostgreSQL 15", "15", env.TestPostgres15Addr},
		{"PostgreSQL 16", "16", env.TestPostgres16Addr},
		{"PostgreSQL 17", "17", env.TestPostgres17Addr},
		{"PostgreSQL 18", "18", env.TestPostgres18Addr},
	}
```

и вызовы `testBackupRestoreForVersion(t, tc.version, tc.addr)` /
`testBackupRestoreWithEncryptionForVersion(t, tc.version, tc.addr)`.

Сигнатуры хелперов (строки 492 и 576) поменять на `(t *testing.T, pgVersion string, addr string)`,
внутри них вызовы `connectToPostgresContainer(pgVersion, addr)`.

В `Test_BackupPostgresql_SchemaSelection_AllSchemasWhenNoneSpecified` (строка 247) и
`Test_BackupPostgresql_SchemaSelection_OnlySpecifiedSchemas` (строка 371) заменить
`connectToPostgresContainer("16", env.TestPostgres16Port)` на
`connectToPostgresContainer("16", env.TestPostgres16Addr)`.

Сам хелпер (строка 1032) переписать:

```go
func connectToPostgresContainer(version string, addr string) (*PostgresContainer, error) {
	dbName := "testdb"
	password := "testpassword"
	username := "testuser"

	host, port, err := test_utils.SplitAddr(addr)
	if err != nil {
		return nil, err
	}

	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		host, port, username, password, dbName)

	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	return &PostgresContainer{
		Host:     host,
		Port:     port,
		Username: username,
		Password: password,
		Database: dbName,
		DB:       db,
	}, nil
}
```

Если после правок импорт `strconv` в файле стал неиспользуемым — удалить.

- [ ] **Step 7: Перевести тесты readonly-пользователя на адреса**

В `backend/internal/features/databases/databases/postgresql/readonly_user_test.go` в обеих
табличных структурах (строки 28-33 и 61-66) заменить `env.TestPostgresNNPort` на
`env.TestPostgresNNAddr`, поле структуры `port` переименовать в `addr`, вызовы —
`connectToPostgresContainer(t, tc.addr)`. Одиночные вызовы на строках 159, 201, 251 —
`connectToPostgresContainer(t, env.TestPostgres16Addr)`.

Хелпер (строка 440) переписать:

```go
func connectToPostgresContainer(t *testing.T, addr string) *PostgresContainer {
	dbName := "testdb"
	password := "testpassword"
	username := "testuser"

	host, port, err := test_utils.SplitAddr(addr)
	assert.NoError(t, err)

	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		host, port, username, password, dbName)

	db, err := sqlx.Connect("postgres", dsn)
	assert.NoError(t, err)

	var versionStr string
	err = db.Get(&versionStr, "SELECT version()")
	assert.NoError(t, err)

	return &PostgresContainer{
		Host:     host,
		Port:     port,
		Username: username,
		Password: password,
		Database: dbName,
		DB:       db,
	}
}
```

Добавить импорт `test_utils "postgresus-backend/internal/util/testing"`. Если `strconv` стал
неиспользуемым — удалить.

- [ ] **Step 8: Перевести тесты сторэджей на адреса**

В `backend/internal/features/storages/model_test.go`:

`setupS3Container` (строка 264):

```go
	endpoint := env.TestMinioAddr
```

`setupAzuriteContainer` (строки 310 и 315-321):

```go
	serviceURL := fmt.Sprintf("http://%s/%s", env.TestAzuriteAddr, accountName)
	...
	connectionString := fmt.Sprintf(
		"DefaultEndpointsProtocol=http;AccountName=%s;AccountKey=%s;BlobEndpoint=http://%s/%s",
		accountName,
		accountKey,
		env.TestAzuriteAddr,
		accountName,
	)
```

Блок вычисления `nasPort`/`ftpPort` (строки 66-79) заменить на разбор адресов:

```go
	nasHost, nasPort, err := test_utils.SplitAddr(config.GetEnv().TestNasAddr)
	require.NoError(t, err, "TEST_NAS_ADDR must be host:port")

	ftpHost, ftpPort, err := test_utils.SplitAddr(config.GetEnv().TestFtpAddr)
	require.NoError(t, err, "TEST_FTP_ADDR must be host:port")
```

В тест-кейсе `NASStorage` заменить `Host: "localhost"` на `Host: nasHost`, в `FTPStorage` —
`Host: "localhost"` на `Host: ftpHost`.

`validateEnvVariables` (строка 354):

```go
func validateEnvVariables(t *testing.T) {
	env := config.GetEnv()
	assert.NotEmpty(t, env.TestMinioAddr, "TEST_MINIO_ADDR is empty")
	assert.NotEmpty(t, env.TestAzuriteAddr, "TEST_AZURITE_ADDR is empty")
	assert.NotEmpty(t, env.TestNasAddr, "TEST_NAS_ADDR is empty")
	assert.NotEmpty(t, env.TestFtpAddr, "TEST_FTP_ADDR is empty")
}
```

Добавить импорт `test_utils "postgresus-backend/internal/util/testing"`, удалить ставший
неиспользуемым `strconv`.

- [ ] **Step 9: Проверить сборку и юнит-тесты**

```bash
cd backend && go vet ./internal/... && go test ./internal/util/... -count=1
```

Ожидается: `go vet` без ошибок, тесты пакетов `util` проходят. Тесты, которым нужна БД, здесь не
запускаются — они поедут в Task 4.

- [ ] **Step 10: Commit**

```bash
git add backend/internal
git commit -m "REFACTOR (tests): address test services by host:port instead of port only"
```

---

### Task 3: Сделать `.env` необязательным и оживить `POSTGRES_INSTALL_DIR`

**Files:**
- Modify: `backend/internal/config/config.go:103-121` (загрузка `.env`), `:151` (путь к клиентам)

**Interfaces:**
- Consumes: конфиг после Task 2.
- Produces: переменная окружения `POSTGRES_INSTALL_DIR` — если задана, используется как каталог с
  клиентами PostgreSQL в формате `<dir>/postgresql-<версия>/bin`. Task 4 выставляет её в
  `/opt/pg-tools`.

- [ ] **Step 1: Сделать отсутствие `.env` не фатальным**

В `backend/internal/config/config.go` заменить блок строк 118-121:

```go
	if !loaded {
		log.Error("Error loading .env file: could not find .env in any location")
		os.Exit(1)
	}
```

на:

```go
	if !loaded {
		log.Info("No .env file found, reading configuration from environment variables")
	}
```

Проверки `DATABASE_DSN` и `ENV_MODE` ниже по коду остаются нетронутыми — они и ловят случай, когда
переменных нет ни в файле, ни в окружении.

- [ ] **Step 2: Заставить `POSTGRES_INSTALL_DIR` работать**

Заменить строку 151:

```go
	env.PostgresesInstallDir = filepath.Join(backendRoot, "tools", "postgresql")
```

на:

```go
	if env.PostgresesInstallDir == "" {
		env.PostgresesInstallDir = filepath.Join(backendRoot, "tools", "postgresql")
	}
```

- [ ] **Step 3: Проверить сборку**

```bash
cd backend && go vet ./internal/... && go build ./...
```

Ожидается: без ошибок.

- [ ] **Step 4: Commit**

```bash
git add backend/internal/config/config.go
git commit -m "FIX (config): make .env optional and honor POSTGRES_INSTALL_DIR"
```

---

### Task 4: Образ прогонщика и тестовый стенд

Самая крупная задача: её результат — первый зелёный прогон go-тестов в докере.

**Files:**
- Create: `backend/Dockerfile.test`
- Create: `backend/scripts/run-tests.sh`
- Create: `docker-compose.test.yml`
- Create: `Makefile`

**Interfaces:**
- Consumes: `SplitAddr` и ADDR-поля конфига из Task 2, `POSTGRES_INSTALL_DIR` из Task 3.
- Produces: сервис `backend-tests` в `docker-compose.test.yml`; цели `make test-backend`,
  `make test-logs`, `make test-clean`. Task 5 добавляет к ним `frontend-tests` и `make test`.

- [ ] **Step 1: Написать Dockerfile прогонщика**

Создать `backend/Dockerfile.test`:

```dockerfile
FROM golang:1.23.3-bookworm

# PostgreSQL client tools 12-18 from the official pgdg repository.
# Same source as the production image, so tests run against the same binaries.
RUN apt-get update && apt-get install -y --no-install-recommends \
  wget ca-certificates gnupg lsb-release && \
  wget -qO- https://www.postgresql.org/media/keys/ACCC4CF8.asc | apt-key add - && \
  echo "deb http://apt.postgresql.org/pub/repos/apt $(lsb_release -cs)-pgdg main" \
  > /etc/apt/sources.list.d/pgdg.list && \
  apt-get update && \
  apt-get install -y --no-install-recommends \
  postgresql-client-12 postgresql-client-13 postgresql-client-14 postgresql-client-15 \
  postgresql-client-16 postgresql-client-17 postgresql-client-18 && \
  rm -rf /var/lib/apt/lists/*

# The app expects <dir>/postgresql-<version>/bin/{pg_dump,psql}.
# /opt is outside the bind-mounted source tree, so the mount cannot shadow it.
RUN for v in 12 13 14 15 16 17 18; do \
  mkdir -p /opt/pg-tools/postgresql-$v/bin && \
  for b in pg_dump pg_dumpall psql pg_restore createdb dropdb; do \
  ln -sf /usr/lib/postgresql/$v/bin/$b /opt/pg-tools/postgresql-$v/bin/$b; \
  done; \
  done

RUN go install github.com/pressly/goose/v3/cmd/goose@v3.24.3

WORKDIR /app/backend

COPY scripts/run-tests.sh /usr/local/bin/run-tests.sh
RUN chmod +x /usr/local/bin/run-tests.sh

ENTRYPOINT ["/usr/local/bin/run-tests.sh"]
```

- [ ] **Step 2: Написать entrypoint прогонщика**

Создать `backend/scripts/run-tests.sh`:

```bash
#!/usr/bin/env bash
set -euo pipefail

# Wait for a TCP endpoint given as host:port. Uses bash /dev/tcp so we make
# no assumptions about what tools third-party images ship with.
wait_for_tcp() {
  local addr="$1"
  local host="${addr%%:*}"
  local port="${addr##*:}"
  local attempts=60

  echo "waiting for ${host}:${port}"
  for _ in $(seq 1 "${attempts}"); do
    if (echo > "/dev/tcp/${host}/${port}") 2>/dev/null; then
      echo "${host}:${port} is up"
      return 0
    fi
    sleep 1
  done

  echo "timed out waiting for ${host}:${port}" >&2
  return 1
}

for addr in \
  "${TEST_MINIO_ADDR}" \
  "${TEST_AZURITE_ADDR}" \
  "${TEST_NAS_ADDR}" \
  "${TEST_FTP_ADDR}"; do
  wait_for_tcp "${addr}"
done

echo "running migrations"
cd /app/backend/migrations
goose up
cd /app/backend

echo "running tests"
exec go test -p=1 -count=1 "$@" ./internal/...
```

Постгресы в этом списке отсутствуют намеренно: у них есть healthcheck в compose, и
`depends_on: service_healthy` не даст контейнеру стартовать раньше времени.

- [ ] **Step 3: Написать тестовый compose**

Создать `docker-compose.test.yml` в корне репозитория:

```yaml
name: postgresus-test

x-test-postgres: &test-postgres
  environment:
    POSTGRES_DB: testdb
    POSTGRES_USER: testuser
    POSTGRES_PASSWORD: testpassword
  healthcheck:
    test: ["CMD-SHELL", "pg_isready -U testuser -d testdb"]
    interval: 2s
    timeout: 5s
    retries: 30
  shm_size: 1gb

services:
  app-db:
    image: postgres:17
    environment:
      POSTGRES_DB: postgresus
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: Q1234567
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U postgres -d postgresus"]
      interval: 2s
      timeout: 5s
      retries: 30
    shm_size: 1gb

  test-postgres-12:
    image: postgres:12
    <<: *test-postgres
  test-postgres-13:
    image: postgres:13
    <<: *test-postgres
  test-postgres-14:
    image: postgres:14
    <<: *test-postgres
  test-postgres-15:
    image: postgres:15
    <<: *test-postgres
  test-postgres-16:
    image: postgres:16
    <<: *test-postgres
  test-postgres-17:
    image: postgres:17
    <<: *test-postgres
  test-postgres-18:
    image: postgres:18
    <<: *test-postgres

  test-minio:
    image: minio/minio:latest
    environment:
      MINIO_ROOT_USER: testuser
      MINIO_ROOT_PASSWORD: testpassword
    command: server /data --console-address ":9001"

  test-azurite:
    image: mcr.microsoft.com/azure-storage/azurite
    command: azurite-blob --blobHost 0.0.0.0

  test-nas:
    image: dperson/samba:latest
    environment:
      USERID: 1000
      GROUPID: 1000
    volumes:
      - test-nas-data:/shared
    command: >
      -u "testuser;testpassword"
      -s "backups;/shared;yes;no;no;testuser"
      -p

  test-ftp:
    image: delfer/alpine-ftp-server:latest
    environment:
      USERS: "testuser|testpassword|/ftp/testuser"
      ADDRESS: test-ftp
      MIN_PORT: 30000
      MAX_PORT: 30009

  backend-tests:
    build:
      context: ./backend
      dockerfile: Dockerfile.test
    depends_on:
      app-db:
        condition: service_healthy
      test-postgres-12:
        condition: service_healthy
      test-postgres-13:
        condition: service_healthy
      test-postgres-14:
        condition: service_healthy
      test-postgres-15:
        condition: service_healthy
      test-postgres-16:
        condition: service_healthy
      test-postgres-17:
        condition: service_healthy
      test-postgres-18:
        condition: service_healthy
      test-minio:
        condition: service_started
      test-azurite:
        condition: service_started
      test-nas:
        condition: service_started
      test-ftp:
        condition: service_started
    environment:
      ENV_MODE: development
      POSTGRES_INSTALL_DIR: /opt/pg-tools
      DATABASE_DSN: "host=app-db user=postgres password=Q1234567 dbname=postgresus port=5432 sslmode=disable"
      GOOSE_DRIVER: postgres
      GOOSE_DBSTRING: "postgres://postgres:Q1234567@app-db:5432/postgresus?sslmode=disable"
      GOOSE_MIGRATION_DIR: "."
      TEST_POSTGRES_12_ADDR: test-postgres-12:5432
      TEST_POSTGRES_13_ADDR: test-postgres-13:5432
      TEST_POSTGRES_14_ADDR: test-postgres-14:5432
      TEST_POSTGRES_15_ADDR: test-postgres-15:5432
      TEST_POSTGRES_16_ADDR: test-postgres-16:5432
      TEST_POSTGRES_17_ADDR: test-postgres-17:5432
      TEST_POSTGRES_18_ADDR: test-postgres-18:5432
      TEST_MINIO_ADDR: test-minio:9000
      TEST_AZURITE_ADDR: test-azurite:10000
      TEST_NAS_ADDR: test-nas:445
      TEST_FTP_ADDR: test-ftp:21
    volumes:
      - ./backend:/app/backend
      - go-mod-cache:/go/pkg/mod
      - go-build-cache:/root/.cache/go-build

volumes:
  test-nas-data:
  go-mod-cache:
  go-build-cache:
```

Портов наружу нет ни у одного сервиса — это осознанно.

- [ ] **Step 4: Написать корневой Makefile**

Создать `Makefile` в корне репозитория:

```makefile
COMPOSE_TEST = docker compose -f docker-compose.test.yml

.PHONY: test test-backend test-frontend test-logs test-clean

test: test-backend

test-backend:
	$(COMPOSE_TEST) run --rm backend-tests

test-logs:
	$(COMPOSE_TEST) logs

test-clean:
	$(COMPOSE_TEST) down -v --remove-orphans
```

Цель `test-frontend` и параллельный запуск добавляет Task 5.

- [ ] **Step 5: Первый прогон**

```bash
make test-backend
```

Ожидается: образ собирается, поднимаются 12 сервисов, проходят миграции, гоняются go-тесты.
Первая сборка занимает несколько минут.

Что делать, если упало:

- `postgresql-client-12` не найден под arm64 — добавить в `docker-compose.test.yml` в сервис
  `backend-tests` строку `platform: linux/amd64` рядом с `build:` и повторить.
- Тесты FTP не проходят — заменить сервис `test-ftp` на прежний образ:
  `image: stilliard/pure-ftpd:latest` с `platform: linux/amd64`, переменными
  `PUBLICHOST: test-ftp`, `FTP_USER_NAME: testuser`, `FTP_USER_PASS: testpassword`,
  `FTP_USER_HOME: /home/ftpusers/testuser`, `FTP_PASSIVE_PORTS: "30000:30009"`.
- Тесты NAS не проходят — добавить `platform: linux/amd64` в сервис `test-nas`.

- [ ] **Step 6: Проверить чистоту рабочего каталога**

```bash
git status --porcelain
```

Ожидается: в выводе нет `backend/temp/` и `postgresus-data/`. Если они появились — значит
что-то смонтировано лишнее, проверить блок `volumes` у `backend-tests`.

- [ ] **Step 7: Проверить повторный прогон**

```bash
make test-backend
```

Ожидается: образ не пересобирается, зависимости не скачиваются заново (работают кэш-volume'ы),
прогон заметно быстрее первого.

- [ ] **Step 8: Commit**

```bash
git add Makefile docker-compose.test.yml backend/Dockerfile.test backend/scripts/run-tests.sh
git commit -m "FEATURE (tests): run backend tests in docker"
```

---

### Task 5: Тесты фронтенда в докере

**Files:**
- Create: `frontend/Dockerfile.test`
- Create: `frontend/.dockerignore`
- Modify: `docker-compose.test.yml` (сервис `frontend-tests`)
- Modify: `Makefile` (цели `test`, `test-frontend`)

**Interfaces:**
- Consumes: `docker-compose.test.yml` и `Makefile` из Task 4.
- Produces: `make test` — прогон обоих наборов тестов, ненулевой код возврата при падении любого.

- [ ] **Step 1: Написать Dockerfile для vitest**

Изначальный вариант монтировал `node_modules` через именованный volume, прогреваемый из
образа при первом запуске. Fix round 1 (review finding, human ruling) отказался от этой
схемы: именованный volume переживает пересборки, поэтому после смены
`package.json`/`package-lock.json` `npm ci` в образе перезаписывается старым содержимым
volume и тесты молча гоняются против устаревших зависимостей. Вместо volume и bind-mount
исходники запекаются в образ, а сам образ пересобирается при каждом запуске (см. Step 3).
`npm ci` остаётся в отдельном слое над копированием исходников, чтобы установка
зависимостей кэшировалась при изменении только исходников.

Создать `frontend/Dockerfile.test`:

```dockerfile
FROM node:24-alpine

WORKDIR /app/frontend

# npm ci sits in its own layer above the source copy so dependency
# installation stays cached across rebuilds that only touch sources.
COPY package.json package-lock.json ./
RUN npm ci

# Sources are baked into the image (no bind mount / volume at runtime),
# so the image is self-contained and never runs against stale
# dependencies. frontend/.dockerignore keeps node_modules, dist and
# .env* out of the build context.
COPY . .

CMD ["npx", "vitest", "run"]
```

Так как теперь весь контекст сборки (включая `COPY . .`) уходит в образ, без
`.dockerignore` в него попал бы локальный `node_modules` (~309 МБ на хосте разработки).
Создать `frontend/.dockerignore`:

```
node_modules
dist
dist-ssr
.env*
```

- [ ] **Step 2: Добавить сервис в тестовый compose**

В `docker-compose.test.yml` добавить в `services`. Секции `volumes` у сервиса нет
намеренно — исходники и зависимости запечены в образ (Step 1), поэтому монтировать
нечего:

```yaml
  frontend-tests:
    build:
      context: ./frontend
      dockerfile: Dockerfile.test
```

Именованный volume `frontend-node-modules` из top-level `volumes:` не создаётся — его не
существует в этой схеме. Остальные volume'ы (`test-nas-data`, `go-mod-cache`,
`go-build-cache`) не трогаются.

- [ ] **Step 3: Добавить цели в Makefile**

Заменить в корневом `Makefile` цель `test` и добавить `test-frontend`. `--build`
обязателен: без него `docker compose run` переиспользовал бы уже собранный образ и не
подхватил бы изменения в исходниках или в `package.json`/`package-lock.json`:

```makefile
test:
	$(MAKE) -j2 test-backend test-frontend

test-frontend:
	$(COMPOSE_TEST) run --rm --build frontend-tests
```

- [ ] **Step 4: Прогнать фронтовые тесты**

```bash
make test-frontend
```

Ожидается: `Test Files 1 passed (1)`, `Tests 49 passed (49)`.

- [ ] **Step 5: Прогнать всё вместе**

```bash
make test
```

Ожидается: оба набора проходят, код возврата 0. Вывод двух прогонов перемешан — это ожидаемо для
параллельного запуска.

- [ ] **Step 6: Проверить, что падение теста роняет команду**

Временно сломать один фронтовый тест (например, поменять ожидаемое значение в
`frontend/src/entity/databases/model/postgresql/ConnectionStringParser.test.ts`), затем:

```bash
make test-frontend; echo "exit code: $?"
```

Ожидается: ненулевой код возврата. После проверки вернуть тест в исходное состояние
(`git checkout frontend/src/entity/databases/model/postgresql/ConnectionStringParser.test.ts`).

- [ ] **Step 7: Commit**

```bash
git add frontend/Dockerfile.test docker-compose.test.yml Makefile
git commit -m "FEATURE (tests): run frontend tests in docker"
```

---

### Task 6: Дев-окружение в докере и удаление хостовых инструментов

**Files:**
- Create: `docker-compose.dev.yml`
- Modify: `Makefile` (цели `run`, `migration-*`, `lint-backend`, `swagger`)
- Delete: `backend/Makefile`, `backend/tools/`, `backend/docker-compose.yml.example`

**Interfaces:**
- Consumes: образ из `backend/Dockerfile.test` (в нём уже есть Go, goose и клиенты PostgreSQL).
- Produces: `make run` — приложение на `http://localhost:4005`; `make migration-up`,
  `make migration-down`, `make migration-create name=<имя>`.

- [ ] **Step 1: Написать дев-compose**

Создать `docker-compose.dev.yml` в корне репозитория:

```yaml
name: postgresus-dev

services:
  dev-db:
    image: postgres:17
    environment:
      POSTGRES_DB: postgresus
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: Q1234567
    ports:
      - "5437:5432"
    volumes:
      - dev-db-data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U postgres -d postgresus"]
      interval: 2s
      timeout: 5s
      retries: 30
    shm_size: 1gb

  app:
    build:
      context: ./backend
      dockerfile: Dockerfile.test
    depends_on:
      dev-db:
        condition: service_healthy
    entrypoint: []
    command: ["go", "run", "cmd/main.go"]
    environment:
      ENV_MODE: development
      POSTGRES_INSTALL_DIR: /opt/pg-tools
      DATABASE_DSN: "host=dev-db user=postgres password=Q1234567 dbname=postgresus port=5432 sslmode=disable"
      GOOSE_DRIVER: postgres
      GOOSE_DBSTRING: "postgres://postgres:Q1234567@dev-db:5432/postgresus?sslmode=disable"
      GOOSE_MIGRATION_DIR: "."
    ports:
      - "4005:4005"
    volumes:
      - ./backend:/app/backend
      - go-mod-cache:/go/pkg/mod
      - go-build-cache:/root/.cache/go-build

volumes:
  dev-db-data:
  go-mod-cache:
  go-build-cache:
```

`entrypoint: []` нужен, потому что образ прогонщика по умолчанию запускает `run-tests.sh`.

- [ ] **Step 2: Перенести оставшиеся цели в корневой Makefile**

Дописать в корневой `Makefile`:

```makefile
COMPOSE_DEV = docker compose -f docker-compose.dev.yml

.PHONY: run migration-up migration-down migration-create lint-backend swagger

run:
	$(COMPOSE_DEV) up

migration-up:
	$(COMPOSE_DEV) run --rm --entrypoint goose -w /app/backend/migrations app up

migration-down:
	$(COMPOSE_DEV) run --rm --entrypoint goose -w /app/backend/migrations app down

migration-create:
	$(COMPOSE_DEV) run --rm --entrypoint goose -w /app/backend/migrations app create $(name) sql

lint-backend:
	cd backend && golangci-lint fmt && golangci-lint run

swagger:
	cd backend && swag init -g ./cmd/main.go -o swagger
```

`lint-backend` и `swagger` остаются хостовыми — это осознанно, линтеры в докер не переезжают.

- [ ] **Step 3: Проверить, что приложение поднимается**

```bash
make run
```

Ожидается: в логах `Database migrations completed successfully` и старт HTTP-сервера. Проверить
из другого терминала:

```bash
curl -s -o /dev/null -w '%{http_code}\n' http://localhost:4005/api/v1/docs/swagger/index.html
```

Ожидается: `200`. Затем остановить: `Ctrl+C`, после чего `docker compose -f docker-compose.dev.yml down`.

- [ ] **Step 4: Проверить миграции**

```bash
make migration-up
```

Ожидается: `goose: no migrations to run` (миграции уже применены на шаге 3) либо список
применённых миграций.

- [ ] **Step 5: Удалить хостовые инструменты и старые файлы**

```bash
git rm -r backend/tools
git rm backend/Makefile backend/docker-compose.yml.example
```

- [ ] **Step 6: Убедиться, что ничего на них не ссылается**

```bash
grep -rn 'tools/postgresql\|download_linux\|download_macos\|download_windows' \
  --include='*.go' --include='*.yml' --include='*.yaml' --include='*.md' \
  --include='Dockerfile*' --include='Makefile' . | grep -v docs/superpowers
```

Ожидается: единственное совпадение — дефолт `filepath.Join(backendRoot, "tools", "postgresql")` в
`backend/internal/config/config.go`. Он остаётся сознательно: это фолбэк на случай запуска без
`POSTGRES_INSTALL_DIR`.

- [ ] **Step 7: Проверить, что тесты по-прежнему зелёные**

```bash
make test
```

Ожидается: оба набора проходят.

- [ ] **Step 8: Commit**

```bash
git add -A Makefile docker-compose.dev.yml backend
git commit -m "FEATURE (dev): run app and migrations in docker, drop host postgres tooling"
```

---

### Task 7: Чистка репозитория, документации и CI

**Files:**
- Modify: `backend/.env.development.example`
- Modify: `backend/README.md`
- Modify: `.github/workflows/ci-release.yml`
- Delete from index: `backend/temp/`

**Interfaces:**
- Consumes: рабочий `make test` из Task 5 и `make run` из Task 6.
- Produces: ничего для последующих задач — это финальная задача плана.

- [ ] **Step 1: Убрать из git мусор от прошлых прогонов**

```bash
git rm -r --cached backend/temp
```

Правило `temp/` в `backend/.gitignore` уже есть, но на отслеживаемые файлы оно не действовало.
Проверить:

```bash
git ls-files backend/temp | wc -l
```

Ожидается: `0`.

- [ ] **Step 2: Почистить пример `.env`**

Заменить содержимое `backend/.env.development.example` на:

```
# app
ENV_MODE=development

# db
DATABASE_DSN=host=localhost user=postgres password=Q1234567 dbname=postgresus port=5437 sslmode=disable

# migrations
GOOSE_DRIVER=postgres
GOOSE_DBSTRING=postgres://postgres:Q1234567@localhost:5437/postgresus?sslmode=disable
GOOSE_MIGRATION_DIR=./migrations

# oauth (optional, see docs/how-extrnal-oauth-works.md)
GITHUB_CLIENT_ID=
GITHUB_CLIENT_SECRET=
GOOGLE_CLIENT_ID=
GOOGLE_CLIENT_SECRET=
```

`backend/.env.production.example` не трогать — `TEST_*` переменных в нём нет.

- [ ] **Step 3: Переписать README бэкенда**

Заменить в `backend/README.md` разделы `# Before run`, `# Run` и `# Migrations` на:

```markdown
# Run

Всё запускается из корня репозитория, на хосте нужен только Docker.

Приложение (http://localhost:4005):

> make run

Все тесты (бэкенд + фронтенд):

> make test

Только бэкенд или только фронтенд:

> make test-backend
> make test-frontend

Логи сервисов последнего прогона и полный сброс окружения:

> make test-logs
> make test-clean

Линтеры остаются хостовыми (нужен установленный `golangci-lint`):

> make lint-backend

# Migrations

> make migration-create name=MIGRATION_NAME
> make migration-up
> make migration-down
```

Раздел `# Swagger` и всё, что ниже, оставить как есть, заменив в нём команду `make swagger`
(она теперь в корневом Makefile).

- [ ] **Step 4: Выкинуть тестовые job'ы из CI**

В `.github/workflows/ci-release.yml`:

- удалить job `test-backend` целиком (вместе с генерацией `.env`, стартом контейнеров,
  установкой клиентов PostgreSQL и всеми `secrets.TEST_*`),
- удалить job `test-frontend` целиком,
- в job `determine-version` заменить `needs: [test-backend, test-frontend]` на
  `needs: [lint-backend, lint-frontend]`,
- в job `build-only` заменить `needs: [test-backend, test-frontend]` на
  `needs: [lint-backend, lint-frontend]`.

Остальные job'ы (`lint-backend`, `lint-frontend`, `build-and-push`, `release`,
`publish-helm-chart`) не трогать.

- [ ] **Step 5: Проверить синтаксис workflow и отсутствие ссылок на удалённое**

```bash
python3 -c "import yaml,sys; yaml.safe_load(open('.github/workflows/ci-release.yml'))" && echo "yaml ok"
grep -n 'test-backend\|test-frontend\|TEST_TELEGRAM\|TEST_SUPABASE\|TEST_GOOGLE_DRIVE' .github/workflows/ci-release.yml
```

Ожидается: `yaml ok` и пустой вывод grep.

- [ ] **Step 6: Финальная проверка критериев готовности**

```bash
make test-clean
make test
git status --porcelain
```

Ожидается: полный прогон с нуля зелёный, `git status --porcelain` пустой.

- [ ] **Step 7: Commit**

```bash
git add -A
git commit -m "REFACTOR (ci): drop test jobs and stale test env, document docker workflow"
```

---

## Проверка результата целиком

После всех задач должно выполняться:

- `make test` на машине, где есть только Docker и git, даёт зелёный прогон — без `.env`, brew,
  goose, секретов и свободного порта 5000.
- `git status --porcelain` после прогона пустой.
- `grep -rn '^func Test' backend/internal --include='*_test.go' | wc -l` даёт **233**.
- Падение любого теста даёт ненулевой код возврата `make test`.
- `backend/tools/`, `backend/Makefile`, `backend/docker-compose.yml.example` отсутствуют.
- В `.github/workflows/ci-release.yml` нет job'ов `test-backend` и `test-frontend`.
