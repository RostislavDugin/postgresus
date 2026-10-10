## Purpose

Lets a Telegram notifier deliver messages through a self-hosted Telegram Bot API server or a reverse proxy instead of the official Telegram API, and defines which addresses it accepts and how the address is shown.

## ADDED Requirements

### Requirement: A Telegram notifier sends to its Bot API URL or to the official API

A Telegram notifier SHALL have an optional Bot API URL. When the URL is empty, the notifier SHALL send its messages to the official Telegram Bot API at `https://api.telegram.org`. When the URL is set, the notifier SHALL send its messages to that server instead, with the standard Bot API path for sending a message appended to the URL, including any path prefix the URL has. A trailing slash in the URL SHALL NOT change the address the request is sent to. This SHALL hold for scheduled notifications and for test notifications alike, and a notifier with a proxy enabled SHALL reach the Bot API URL through that proxy.

#### Scenario: Notifier without a Bot API URL

- **WHEN** a Telegram notifier has no Bot API URL and sends a notification
- **THEN** the message is sent to the official Telegram Bot API, as before this change

#### Scenario: Notifier with a self-hosted server

- **WHEN** a Telegram notifier has the Bot API URL `http://telegram-bot-api:8081` and sends a notification
- **THEN** the message is sent to `http://telegram-bot-api:8081/bot<token>/sendMessage` with the notifier's chat, topic and text

#### Scenario: Bot API URL with a path prefix and a trailing slash

- **WHEN** a Telegram notifier has the Bot API URL `https://gateway.example.com/telegram/` and sends a notification
- **THEN** the message is sent to `https://gateway.example.com/telegram/bot<token>/sendMessage`

#### Scenario: Test notification uses the Bot API URL from the form

- **WHEN** the user enters a Bot API URL in the Telegram form and sends a test notification before saving
- **THEN** the test message is sent to the entered server

#### Scenario: Bot API URL together with a proxy

- **WHEN** a Telegram notifier has both a Bot API URL and a proxy enabled
- **THEN** the message is sent to the Bot API URL through the configured proxy

#### Scenario: Existing notifiers after the upgrade

- **WHEN** Databasus is upgraded and a Telegram notifier created before the upgrade sends a notification
- **THEN** the notifier has no Bot API URL and the message is sent to the official Telegram Bot API

### Requirement: Only a plain HTTP or HTTPS address is accepted as a Bot API URL

Saving a Telegram notifier and sending a test notification SHALL be rejected with an error when the Bot API URL is set and does not satisfy all of these: the scheme is `http` or `https`, the URL has a host, and it has no query string and no fragment. A path prefix SHALL be accepted. The form SHALL refuse to save or test the notifier under the same conditions. An empty Bot API URL SHALL always be accepted.

#### Scenario: Unsupported scheme

- **WHEN** the user saves a Telegram notifier with the Bot API URL `ftp://telegram.example.com`
- **THEN** the save is rejected and the notifier keeps its previous settings

#### Scenario: Address without a host

- **WHEN** the user saves a Telegram notifier with the Bot API URL `telegram-bot-api:8081` or `https://`
- **THEN** the save is rejected

#### Scenario: Query string or fragment

- **WHEN** the user saves a Telegram notifier with the Bot API URL `https://gateway.example.com/telegram?key=1` or `https://gateway.example.com/#bot`
- **THEN** the save is rejected

#### Scenario: Test send of an unsaved notifier with an invalid address

- **WHEN** the user sends a test notification from the form for a Telegram notifier that has not been saved yet and whose Bot API URL is `ftp://telegram.example.com` or `https://user:password@gateway.example.com`
- **THEN** the test send returns an error and no request leaves the backend

#### Scenario: Valid addresses

- **WHEN** the user saves a Telegram notifier with the Bot API URL `http://localhost:8081`, `https://gateway.example.com` or `https://gateway.example.com/telegram/`
- **THEN** the notifier is saved with that URL

### Requirement: The Bot API URL holds no credentials and stays visible

A Bot API URL that contains a username or a password SHALL be rejected, on save and on test send, in the API and in the form. Because the URL then holds no secret, it SHALL be stored as entered, returned when the notifier is read and shown on the notifier's page, unlike the bot token and the proxy URL, which stay hidden. Errors from a failed send, whether stored as the notifier's last error, written to logs or returned by a test send, SHALL NOT contain the bot token when the notifier uses a Bot API URL, including one with a path prefix.

#### Scenario: Credentials in the URL

- **WHEN** the user saves a Telegram notifier with the Bot API URL `https://user:password@gateway.example.com`
- **THEN** the save is rejected

#### Scenario: Reading a notifier with a Bot API URL

- **WHEN** the user opens a Telegram notifier whose Bot API URL is `https://gateway.example.com/telegram`
- **THEN** the page shows that URL, while the bot token and the proxy URL stay hidden

#### Scenario: Send fails against a custom server

- **WHEN** a Telegram notifier with the Bot API URL `https://gateway.example.com/telegram` fails to connect to that server
- **THEN** the reported error names the host but does not contain the bot token

#### Scenario: Custom server echoes the request path in an error response

- **WHEN** the server at the Bot API URL answers with a non-OK status and a body such as `Cannot POST /telegram/bot<token>/sendMessage`
- **THEN** the reported error, whether returned by a test send or stored as the notifier's last error, keeps the body with `redacted` in place of the bot token

### Requirement: Bot API URL and proxy live in the advanced settings of the Telegram form

The Telegram notifier form SHALL show the Bot API URL field, the proxy switch and the proxy URL in a collapsible advanced settings section. The section SHALL start expanded when the notifier being edited has a Bot API URL or a proxy enabled, and collapsed otherwise. The bot token, the chat ID and the topic settings SHALL stay outside the section.

#### Scenario: New notifier

- **WHEN** the user creates a Telegram notifier
- **THEN** the advanced settings section is collapsed and the bot token, chat ID and topic settings are visible

#### Scenario: Editing a notifier with a proxy

- **WHEN** the user edits a Telegram notifier that has a proxy enabled or a Bot API URL set
- **THEN** the advanced settings section is expanded and shows those values
