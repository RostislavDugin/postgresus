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

# Swagger

To generate swagger docs (run from the repo root):

> make swagger

Swagger URL is:

> http://localhost:4005/api/v1/docs/swagger/index.html#/

# Project structure

Default endpoint structure is:

/feature
/feature/controller.go
/feature/service.go
/feature/repository.go
/feature/model.go
/feature/dto.go

If there are couple of models:
/feature/models/model1.go
/feature/models/model2.go
...

# Project rules

Read .cursor/rules folder, it contains all the rules for the project.
