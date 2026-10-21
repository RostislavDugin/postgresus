---
name: testing
description: Write, change, run and review tests in Databasus by the rules shared by every module. Use when writing, changing, running or reviewing tests, when fixing a bug, and when choosing which test commands a change runs.
---

# Testing

These rules apply in every module. The module docs add the rules of their stack: test names, the level of test they prefer, where tests live and how they clean up. Read the one for the area you work in:

- [`backend/AGENTS.md`, "Testing"](../../../backend/AGENTS.md#testing)
- [`agent/verification/AGENTS.md`, "Testing"](../../../agent/verification/AGENTS.md#testing)

## Run what you wrote

Run every test you wrote or changed and see it pass before you hand the work over. A package that compiles has not passed its tests: when you only built it or ran `go vet`, say so in the report instead of calling it tested.

## Start a bug fix with a failing test

A bug fix starts with a test that reproduces the bug. Run it on the code before the fix and see it fail for the reason the bug describes, then fix the code and see it pass. A test that passes on the broken code proves nothing about the fix.

## Test what a caller observes

Test the behavior a caller sees, at the level the module doc prefers: through the HTTP API in the backend, through exported functions and CLI commands in the verification agent. A test that pins private helpers breaks on every refactoring and still misses the contract.

Mock only external systems the test cannot run, such as a third-party API, and failures that are unsafe to reproduce, such as a full disk. A database, a container or a file system the test fixtures provide stays real.

## Wait for a condition, not for time

A test that needs something to happen waits for that condition up to a limit and fails with a message when the limit passes. A fixed pause is either too short on a loaded machine or wastes time on a fast one. A negative check, such as "no notification was sent", first waits for a sign that the action was processed, so it does not pass merely because it looked too early.

## Run the tests the change touched

Which tests to run, and which command never to run, is set in [Several agents, one set of containers](../../../AGENTS.md#several-agents-one-set-of-containers), and the commands in the table of its parent section, [Development environment](../../../AGENTS.md#development-environment).

When package tests fail because the shared test fixtures are down, such as a refused database connection or a missing container, tell the user. Do not start the fixtures through `just test`: the developer starts the first full run.

## Clean up after an interrupted run

After a test run you stopped or that crashed, remove only what that run left behind: its containers, processes and temporary directories. Never remove all containers, prune Docker or stop processes you did not start; another agent's run may be using them.

## Before handing the work over

- Every test you wrote or changed ran and passed, and the report says which commands ran.
- A bug fix has a test that failed before the fix.
- No test sleeps for a fixed time to wait for a result.
- Every test removes the data, files and processes it created, as the module doc describes.
- An OpenSpec change whose tasks add or change tests lists their commands in the "Change verification" section of its `tasks.md`, as the tasks rule in [`openspec/config.yaml`](../../../openspec/config.yaml) describes.
