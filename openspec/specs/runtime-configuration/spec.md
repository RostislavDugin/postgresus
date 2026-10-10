# Runtime Configuration Specification

## Purpose

Defines where Databasus takes its settings from in the published Docker image and in a development checkout, and how startup behaves when no environment file is present.

## Requirements

### Requirement: The published image carries no preset settings

The published Docker image SHALL NOT contain an environment file or any other source of setting values beyond the few the image itself defines to run: the application mode, the version, the processor architecture and the connection to the embedded database, which the container prepares at every start. Every other setting SHALL come from the operator or, when the operator leaves it unset, from the default the application defines for it. No environment file from the build machine SHALL reach the image, neither the backend's nor the frontend's.

The application defaults for an unset setting are: log level `info`, log file enabled, database installation verification logs shown, and the instance mail server, OAuth sign-in, bot protection and log export all off.

#### Scenario: Operator sets nothing optional

- **WHEN** an operator starts the published image without any optional setting
- **THEN** the settings screen reports the mail server as not configured, the sign-in screen offers neither GitHub nor Google sign-in nor a bot check, informational messages are logged, a log file is written, and the database installation verification output appears in the startup log

#### Scenario: Image built on a developer machine

- **WHEN** a developer whose checkout contains environment files for the backend and the frontend, including local and production variants, builds the image locally
- **THEN** the resulting image contains none of those files, and the built frontend contains none of their values

### Requirement: The environment file is optional

When the application starts without an environment file, it SHALL read its settings from the process environment alone and SHALL report in its log that no environment file was loaded. When an environment file exists, the application SHALL read it and SHALL NOT let it override a variable that is already set in the process environment, even one set to an empty value. A required setting that is missing from both SHALL still stop startup with a log message saying that a required setting is missing.

#### Scenario: Container without an environment file

- **WHEN** the published image starts with only the variables the operator passed in
- **THEN** the application starts and logs that no environment file was loaded

#### Scenario: Development checkout without an environment file

- **WHEN** a developer starts the backend from a checkout that has no environment file, with neither the application mode nor the database connection set in the shell
- **THEN** the log says that no environment file was loaded, startup stops with a message saying a required setting is missing, and the process exits with a non-zero status

#### Scenario: Operator value wins over the file

- **WHEN** a variable is set in the process environment and the environment file sets it to a different value
- **THEN** the application uses the value from the process environment
