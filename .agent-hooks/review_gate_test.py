import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import textwrap
import unittest


HOOK_PATH = Path(__file__).resolve().parent / "review_gate.py"
CHANGE_NAME = "add-widgets"
PROPOSAL_NAME = "add-gadgets"
STUB_NAMES = ("claude", "codex", "openspec", "golangci-lint", "pnpm", "npm", "just")

# The stub stands in for every external command: it takes its answer from <stubs>/<name>.json
# and records each call in <stubs>/calls, so a test sees the arguments, the working directory
# and the prompt. openspec answers by its first argument (openspec.json["status"]),
# npm by its script name (npm.json["build"]).
STUB_SOURCE = textwrap.dedent(
    """
    import json
    import os
    from pathlib import Path
    import sys
    import uuid

    stubs_dir = Path(os.environ["REVIEW_GATE_STUBS"])
    name = Path(sys.argv[0]).name
    arguments = sys.argv[1:]
    prompt = sys.stdin.read() if name in ("claude", "codex") else ""
    calls_dir = stubs_dir / "calls"
    calls_dir.mkdir(exist_ok=True)
    (calls_dir / f"{name}-{uuid.uuid4().hex}.json").write_text(
        json.dumps(
            {
                "name": name,
                "arguments": arguments,
                "cwd": os.getcwd(),
                "prompt": prompt,
                "effort_env": os.environ.get("CLAUDE_CODE_EFFORT_LEVEL"),
            }
        )
    )

    config_path = stubs_dir / f"{name}.json"
    config = json.loads(config_path.read_text()) if config_path.exists() else {}
    if name == "openspec":
        config = config.get(arguments[0], {})
    elif name == "npm":
        config = config.get(arguments[-1], {})

    if config.get("exit_code"):
        sys.stdout.write(config.get("output", ""))
        sys.exit(config["exit_code"])

    result = config.get("result", {"ok": True, "reason": "", "findings": []})
    if name == "claude":
        sys.stdout.write(json.dumps({"structured_output": result}))
    elif name == "codex":
        output_path = arguments[arguments.index("--output-last-message") + 1]
        Path(output_path).write_text(json.dumps(result))
    elif name == "openspec" and arguments[0] == "status":
        sys.stdout.write(config.get("output", json.dumps({"isPlanningComplete": True})))
    else:
        sys.stdout.write(config.get("output", ""))
    """
)
PASSING_HOOK_TEST = textwrap.dedent(
    """
    import unittest


    class SampleTest(unittest.TestCase):
        def test_passes(self):
            self.assertTrue(True)
    """
)
FAILING_HOOK_TEST = textwrap.dedent(
    """
    import unittest


    class SampleTest(unittest.TestCase):
        def test_reports_broken_gate(self):
            self.fail("gate returned block for an accepted turn")
    """
)


class HookRepository:
    def __init__(self, test_case):
        temporary_dir = tempfile.TemporaryDirectory(prefix="review-gate-test-")
        test_case.addCleanup(temporary_dir.cleanup)
        base = Path(temporary_dir.name)

        self.root = base / "repo"
        self.stubs_dir = base / "stubs"
        bin_dir = base / "bin"
        for directory in (self.root, self.stubs_dir, bin_dir):
            directory.mkdir()

        for name in STUB_NAMES:
            stub = bin_dir / name
            stub.write_text(f"#!{sys.executable}\n{STUB_SOURCE}")
            stub.chmod(0o755)

        self.environment = {
            name: value
            for name, value in os.environ.items()
            if not name.startswith("GIT_") and name != "AGENT_REVIEW_HOOK_ACTIVE"
        }
        self.environment["PATH"] = f"{bin_dir}{os.pathsep}{os.environ['PATH']}"
        self.environment["REVIEW_GATE_STUBS"] = str(self.stubs_dir)
        self.environment["PYTHONDONTWRITEBYTECODE"] = "1"

        self.git("init", "--quiet")
        self.git("config", "user.email", "test@example.com")
        self.git("config", "user.name", "test")
        self.write("README.md", "test repository\n")
        self.write_change(spec_names=("widgets",), tasks="- [x] 1.1 Build\n- [ ] 1.2 Verify\n")
        self.git("add", ".")
        self.git("commit", "--quiet", "-m", "init")

    def git(self, *arguments):
        subprocess.run(["git", *arguments], cwd=self.root, env=self.environment, check=True, capture_output=True)

    def write(self, relative_path, content):
        path = self.root / relative_path
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(content)

    def write_change(self, spec_names, tasks, name=CHANGE_NAME):
        change_dir = f"openspec/changes/{name}"
        self.write(f"{change_dir}/proposal.md", "# Proposal\n")
        self.write(f"{change_dir}/design.md", "# Design\n")
        self.write(f"{change_dir}/tasks.md", f"# Tasks\n\n{tasks}")
        for spec_name in spec_names:
            self.write(f"{change_dir}/specs/{spec_name}/spec.md", f"# {spec_name}\n")

    def write_proposal(self, spec_names=("gadgets",), tasks="- [ ] 1.1 Build\n- [ ] 1.2 Verify\n"):
        self.write_change(spec_names=spec_names, tasks=tasks, name=PROPOSAL_NAME)

    def complete_tasks(self, name=CHANGE_NAME):
        self.write(f"openspec/changes/{name}/tasks.md", "# Tasks\n\n- [x] 1.1 Build\n- [x] 1.2 Verify\n")

    def set_reviewer_result(self, result):
        for name in ("claude", "codex"):
            (self.stubs_dir / f"{name}.json").write_text(json.dumps({"result": result}))

    def set_reviewer_failure(self, output):
        for name in ("claude", "codex"):
            (self.stubs_dir / f"{name}.json").write_text(json.dumps({"exit_code": 1, "output": output}))

    def set_stub(self, name, config):
        (self.stubs_dir / f"{name}.json").write_text(json.dumps(config))

    def calls(self, name=None):
        calls_dir = self.stubs_dir / "calls"
        if not calls_dir.is_dir():
            return []
        calls = [json.loads(path.read_text()) for path in calls_dir.iterdir()]
        return [call for call in calls if name is None or call["name"] == name]

    def check_calls(self):
        return [call for call in self.calls() if call["name"] in ("golangci-lint", "pnpm", "npm", "just")]

    def reviewer_calls(self):
        return [call for call in self.calls() if call["name"] in ("claude", "codex")]

    def clear_calls(self):
        for path in (self.stubs_dir / "calls").glob("*.json"):
            path.unlink()

    def run_hook(self, action, payload, client="claude"):
        process = subprocess.run(
            [sys.executable, str(HOOK_PATH), action, "--client", client],
            cwd=self.root,
            env=self.environment,
            input=json.dumps({"cwd": str(self.root), **payload}),
            text=True,
            capture_output=True,
            check=True,
        )
        return json.loads(process.stdout) if process.stdout.strip() else None

    def submit_prompt(self, prompt, session_id="session-1", client="claude"):
        self.run_hook("snapshot", {"hook_event_name": "UserPromptSubmit", "session_id": session_id, "prompt": prompt}, client)

    def stop(self, action, session_id="session-1", client="claude", message="Done."):
        payload = {"hook_event_name": "Stop", "session_id": session_id, "last_assistant_message": message}
        return self.run_hook(action, payload, client)


def finding(category, summary, file="src/widgets.py:10"):
    return {"category": category, "file": file, "summary": summary}


class ShortReplyAssertions:
    def assert_asks_for_short_reply(self, reason):
        self.assertIn("one short line about your fixes that names every rejected finding with the reason", reason)
        self.assertIn("Do not write the summary for the user yet", reason)


class SummaryAssertions:
    def assert_asks_for_summary_without_review(self, reason):
        self.assertIn("as if there had been no review", reason)
        self.assertIn("Leave out review rounds, findings and the fixes made for them.", reason)
        self.assertNotIn("each round", reason)


class SessionStartTest(unittest.TestCase):
    def test_tells_agent_to_load_communication_skill(self):
        repository = HookRepository(self)

        response = repository.run_hook("session-start", {"hook_event_name": "SessionStart", "session_id": "session-1"})

        context = response["hookSpecificOutput"]["additionalContext"]
        self.assertIn(".agents/skills/humanizer/SKILL.md", context)


class PlanGateTest(unittest.TestCase):
    def exit_plan_mode(self, repository, gate, plan):
        payload = {
            "hook_event_name": "PreToolUse",
            "session_id": "session-1",
            "tool_name": "ExitPlanMode",
            "tool_input": {"plan": plan},
        }
        return repository.run_hook(gate, payload)

    def test_allows_plan_when_reviewer_approves(self):
        repository = HookRepository(self)
        repository.submit_prompt("Plan the widgets")

        response = self.exit_plan_mode(repository, "plan", "# Plan\n\nAdd widgets.\n")

        self.assertEqual(response, {})
        [call] = repository.reviewer_calls()
        self.assertIn("## Plan review", call["prompt"])
        self.assertIn("Add widgets.", call["prompt"])

    def test_denies_plan_with_reviewer_reason_when_reviewer_rejects(self):
        repository = HookRepository(self)
        repository.set_reviewer_result({"ok": False, "reason": "The plan keeps a compatibility alias."})
        repository.submit_prompt("Plan the widgets")

        response = self.exit_plan_mode(repository, "necessity", "# Plan\n")

        output = response["hookSpecificOutput"]
        self.assertEqual(output["permissionDecision"], "deny")
        self.assertIn("The plan keeps a compatibility alias.", output["permissionDecisionReason"])

    def test_denies_exit_plan_mode_without_plan_text(self):
        repository = HookRepository(self)
        repository.submit_prompt("Plan the widgets")

        response = self.exit_plan_mode(repository, "architecture", "")

        self.assertEqual(response["hookSpecificOutput"]["permissionDecision"], "deny")
        self.assertEqual(repository.reviewer_calls(), [])


class DiffGateTest(unittest.TestCase):
    def test_accepts_turn_when_reviewer_approves(self):
        repository = HookRepository(self)
        repository.submit_prompt("Add a widget")
        repository.write("src/widgets.py", "WIDGETS = []\n")

        response = repository.stop("diff")

        self.assertEqual(response, {})
        self.assertEqual(len(repository.reviewer_calls()), 1)

    def test_blocks_turn_with_reviewer_reason_when_reviewer_rejects(self):
        repository = HookRepository(self)
        repository.set_reviewer_result({"ok": False, "reason": "No test for an empty list."})
        repository.submit_prompt("Add a widget")
        repository.write("src/widgets.py", "WIDGETS = []\n")

        response = repository.stop("diff")

        self.assertEqual(response["decision"], "block")
        self.assertIn("No test for an empty list.", response["reason"])
        self.assertIn("a summary of the final state, without retelling this review", response["reason"])
        self.assertNotIn("Fix the result", response["reason"])

    def test_asks_to_revise_plan_when_necessity_blocks_codex_plan_in_final_message(self):
        repository = HookRepository(self)
        repository.set_reviewer_result({"ok": False, "reason": "The plan keeps a compatibility alias."})
        repository.submit_prompt("/plan the widgets", client="codex")

        response = repository.stop("necessity", client="codex", message="# Plan\n\nAdd widgets.\n")

        self.assertEqual(response["decision"], "block")
        self.assertIn("The plan keeps a compatibility alias.", response["reason"])
        self.assertIn("Revise the plan and show it again", response["reason"])
        self.assertNotIn("summary", response["reason"])

    def test_diff_reviewer_prompt_checks_agent_instructions(self):
        repository = HookRepository(self)
        repository.submit_prompt("Fix the docs")
        repository.write("AGENTS.md", "# AGENTS.md\n")

        repository.stop("diff")

        [call] = repository.reviewer_calls()
        self.assertIn("Check edits to agent instructions per the same section.", call["prompt"])

    def test_reviewer_prompt_maps_paths_to_module_docs(self):
        repository = HookRepository(self)
        repository.submit_prompt("Add a widget")
        repository.write("src/widgets.py", "WIDGETS = []\n")

        repository.stop("diff")

        [call] = repository.reviewer_calls()
        self.assertIn("| `agent/**` | `agent/verification/AGENTS.md` |", call["prompt"])

    def test_runs_claude_reviewer_on_opus_with_low_effort(self):
        repository = HookRepository(self)
        repository.submit_prompt("Add a widget")
        repository.write("src/widgets.py", "WIDGETS = []\n")

        repository.stop("diff")

        [call] = repository.calls("claude")
        self.assertIn("--model opus --effort low", " ".join(call["arguments"]))
        self.assertEqual(call["effort_env"], "low")

    def test_runs_codex_reviewer_on_gpt_with_low_effort(self):
        repository = HookRepository(self)
        repository.submit_prompt("Add a widget", client="codex")
        repository.write("src/widgets.py", "WIDGETS = []\n")

        repository.stop("diff", client="codex")

        [call] = repository.calls("codex")
        self.assertIn("--model gpt-6.1-sol --config model_reasoning_effort=\"low\"", " ".join(call["arguments"]))

    def test_stops_agent_after_fourth_rejection(self):
        repository = HookRepository(self)
        repository.set_reviewer_result({"ok": False, "reason": "No test for an empty list."})
        repository.submit_prompt("Add a widget")

        responses = []
        for attempt in range(4):
            repository.write("src/widgets.py", f"WIDGETS = [{attempt}]\n")
            responses.append(repository.stop("diff"))

        self.assertEqual([response.get("decision") for response in responses[:3]], ["block", "block", "block"])
        self.assertIs(responses[3]["continue"], False)


class NamingGateTest(unittest.TestCase):
    def test_runs_reviewer_when_turn_changes_backend_go_file(self):
        repository = HookRepository(self)
        repository.submit_prompt("Add a widget")
        repository.write("backend/internal/features/widgets/service.go", "package widgets\n")

        response = repository.stop("naming")

        self.assertEqual(response, {})
        [call] = repository.reviewer_calls()
        self.assertIn("## Naming review", call["prompt"])

    def test_runs_reviewer_when_turn_changes_agent_go_file(self):
        repository = HookRepository(self)
        repository.submit_prompt("Add a heartbeat field")
        repository.write("agent/verification/internal/heartbeat/heartbeat.go", "package heartbeat\n")

        repository.stop("naming")

        self.assertEqual(len(repository.reviewer_calls()), 1)

    def test_runs_reviewer_when_turn_changes_frontend_tsx_file(self):
        repository = HookRepository(self)
        repository.submit_prompt("Add a widget")
        repository.write("frontend/src/pages/widgets/ui/WidgetsPage.tsx", "export {};\n")

        repository.stop("naming")

        self.assertEqual(len(repository.reviewer_calls()), 1)

    def test_runs_reviewer_when_turn_changes_website_tsx_file(self):
        repository = HookRepository(self)
        repository.submit_prompt("Add a page")
        repository.write("website/app/[lang]/pricing/page.tsx", "export {};\n")

        repository.stop("naming")

        self.assertEqual(len(repository.reviewer_calls()), 1)

    def test_blocks_turn_with_reviewer_reason_when_reviewer_rejects(self):
        repository = HookRepository(self)
        repository.set_reviewer_result({"ok": False, "reason": "Active() is a getter, name it GetRunningVerificationIDs()."})
        repository.submit_prompt("Add a widget")
        repository.write("backend/internal/features/widgets/service.go", "package widgets\n")

        response = repository.stop("naming")

        self.assertEqual(response["decision"], "block")
        self.assertIn("GetRunningVerificationIDs", response["reason"])

    def test_skips_turn_that_changes_only_markdown(self):
        repository = HookRepository(self)
        repository.submit_prompt("Fix the docs")
        repository.write("backend/AGENTS.md", "# AGENTS.md\n")

        response = repository.stop("naming")

        self.assertEqual(response, {})
        self.assertEqual(repository.reviewer_calls(), [])

    def test_skips_turn_that_changes_only_frontend_config(self):
        repository = HookRepository(self)
        repository.submit_prompt("Bump Vite")
        repository.write("frontend/vite.config.ts", "export default {};\n")

        repository.stop("naming")

        self.assertEqual(repository.reviewer_calls(), [])

    def test_skips_markdown_turn_when_go_file_was_changed_before_request(self):
        repository = HookRepository(self)
        repository.write("backend/internal/features/widgets/service.go", "package widgets\n")
        repository.submit_prompt("Fix the docs")
        repository.write("backend/AGENTS.md", "# AGENTS.md\n")

        repository.stop("naming")

        self.assertEqual(repository.reviewer_calls(), [])

    def test_skips_turn_that_changes_only_ready_proposal(self):
        repository = HookRepository(self)
        repository.submit_prompt("Write a spec")
        repository.write_proposal()

        response = repository.stop("naming")

        self.assertEqual(response, {})
        self.assertEqual(repository.reviewer_calls(), [])

    def test_runs_claude_reviewer_on_opus_with_low_effort(self):
        repository = HookRepository(self)
        repository.submit_prompt("Add a widget")
        repository.write("backend/internal/features/widgets/service.go", "package widgets\n")

        repository.stop("naming")

        [call] = repository.calls("claude")
        self.assertIn("--model opus --effort low", " ".join(call["arguments"]))

    def test_runs_codex_reviewer_on_gpt_with_low_effort(self):
        repository = HookRepository(self)
        repository.submit_prompt("Add a widget", client="codex")
        repository.write("backend/internal/features/widgets/service.go", "package widgets\n")

        repository.stop("naming", client="codex")

        [call] = repository.calls("codex")
        self.assertIn("--model gpt-6.1-sol --config model_reasoning_effort=\"low\"", " ".join(call["arguments"]))


class ImplementationGateStartTest(unittest.TestCase):
    def test_runs_full_review_when_agent_completes_last_task(self):
        repository = HookRepository(self)
        repository.submit_prompt("Continue")
        repository.complete_tasks()

        response = repository.stop("implementation")

        self.assertEqual(response, {})
        self.assertEqual(len(repository.reviewer_calls()), 2)

    def test_skips_review_when_tasks_remain_unchecked(self):
        repository = HookRepository(self)
        repository.submit_prompt("Continue")
        repository.write("src/widgets.py", "WIDGETS = []\n")

        response = repository.stop("implementation")

        self.assertEqual(response, {})
        self.assertEqual(repository.calls(), [])

    def test_skips_review_when_workspace_is_unchanged_during_turn(self):
        repository = HookRepository(self)
        repository.complete_tasks()
        repository.submit_prompt("What is left?")

        response = repository.stop("implementation")

        self.assertEqual(response, {})
        self.assertEqual(repository.calls(), [])

    def test_skips_review_in_new_session_when_workspace_matches_accepted_state(self):
        repository = HookRepository(self)
        repository.submit_prompt("Continue", session_id="first")
        repository.submit_prompt("Look at the tasks", session_id="second")
        repository.complete_tasks()
        repository.stop("implementation", session_id="first")
        repository.clear_calls()

        response = repository.stop("implementation", session_id="second")

        self.assertEqual(response, {})
        self.assertEqual(repository.calls(), [])

    def test_runs_review_again_when_files_change_after_accepted_review(self):
        repository = HookRepository(self)
        repository.submit_prompt("Continue")
        repository.complete_tasks()
        repository.stop("implementation")
        repository.clear_calls()
        repository.submit_prompt("Rename it")
        repository.write("src/widgets.py", "WIDGETS = ()\n")

        repository.stop("implementation")

        self.assertEqual(len(repository.reviewer_calls()), 2)

    def test_skips_review_when_prompt_has_openspec_planning_command(self):
        repository = HookRepository(self)
        repository.submit_prompt("/opsx:propose add widgets")
        repository.complete_tasks()

        response = repository.stop("implementation")

        self.assertEqual(response, {})
        self.assertEqual(repository.calls(), [])

    def test_skips_archived_changes(self):
        repository = HookRepository(self)
        repository.git("rm", "-r", "--quiet", f"openspec/changes/{CHANGE_NAME}")
        repository.git("commit", "--quiet", "-m", "archive")
        repository.submit_prompt("Archive it")
        repository.write_change(spec_names=("widgets",), tasks="- [x] 1.1 Build\n", name=f"archive/2026-10-01-{CHANGE_NAME}")

        response = repository.stop("implementation")

        self.assertEqual(response, {})
        self.assertEqual(repository.calls(), [])

    def test_returns_empty_response_without_payload(self):
        repository = HookRepository(self)

        process = subprocess.run(
            [sys.executable, str(HOOK_PATH), "implementation", "--client", "claude"],
            cwd=repository.root,
            env=repository.environment,
            input="",
            text=True,
            capture_output=True,
        )

        self.assertEqual(process.returncode, 0)
        self.assertEqual(json.loads(process.stdout), {})


class ImplementationGateChecksTest(ShortReplyAssertions, unittest.TestCase):
    def test_runs_only_backend_linter_when_change_touches_backend(self):
        repository = HookRepository(self)
        repository.submit_prompt("Continue")
        repository.write("backend/internal/features/widgets/service.go", "package widgets\n")
        repository.complete_tasks()

        repository.stop("implementation")

        [call] = repository.check_calls()
        self.assertEqual(call["name"], "golangci-lint")
        self.assertEqual(call["arguments"], ["run", "./cmd/...", "./internal/..."])
        self.assertEqual(Path(call["cwd"]), repository.root / "backend")

    def test_runs_agent_linter_in_verification_agent_directory(self):
        repository = HookRepository(self)
        repository.submit_prompt("Continue")
        repository.write("agent/verification/internal/heartbeat/heartbeat.go", "package heartbeat\n")
        repository.complete_tasks()

        repository.stop("implementation")

        [call] = repository.check_calls()
        self.assertEqual(call["arguments"], ["run", "./cmd/...", "./internal/...", "./e2e/..."])
        self.assertEqual(Path(call["cwd"]), repository.root / "agent" / "verification")

    def test_runs_only_frontend_linter_when_change_touches_frontend(self):
        repository = HookRepository(self)
        repository.submit_prompt("Continue")
        repository.write("frontend/src/pages/widgets/ui/WidgetsPage.tsx", "export {};\n")
        repository.complete_tasks()

        repository.stop("implementation")

        self.assertEqual([call["arguments"] for call in repository.check_calls()], [["--dir", "frontend", "lint"]])

    def test_runs_website_lint_and_build_when_change_touches_website(self):
        repository = HookRepository(self)
        repository.submit_prompt("Continue")
        repository.write("website/app/[lang]/pricing/page.tsx", "export {};\n")
        repository.complete_tasks()

        repository.stop("implementation")

        self.assertEqual(
            sorted(call["arguments"] for call in repository.calls("npm")),
            [["--prefix", "website", "run", "build"], ["--prefix", "website", "run", "lint"]],
        )

    def test_runs_no_checks_when_change_touches_no_module(self):
        repository = HookRepository(self)
        repository.submit_prompt("Continue")
        repository.write("docs/widgets.md", "# Widgets\n")
        repository.complete_tasks()

        repository.stop("implementation")

        self.assertEqual(repository.check_calls(), [])
        self.assertEqual(len(repository.reviewer_calls()), 2)

    def test_blocks_turn_with_output_tail_when_linter_fails(self):
        repository = HookRepository(self)
        repository.set_stub("golangci-lint", {"exit_code": 1, "output": "service.go:12:2: ineffectual assignment to err\n"})
        repository.submit_prompt("Continue")
        repository.write("backend/internal/features/widgets/service.go", "package widgets\n")
        repository.complete_tasks()

        response = repository.stop("implementation")

        self.assertEqual(response["decision"], "block")
        self.assertIn("`golangci-lint run ./cmd/... ./internal/...` in `backend`", response["reason"])
        self.assertIn("service.go:12:2: ineffectual assignment to err", response["reason"])
        self.assert_asks_for_short_reply(response["reason"])
        self.assertEqual(repository.reviewer_calls(), [])

    def test_never_runs_formatters_or_just(self):
        repository = HookRepository(self)
        repository.submit_prompt("Continue")
        repository.write("backend/internal/features/widgets/service.go", "package widgets\n")
        repository.write("frontend/src/pages/widgets/ui/WidgetsPage.tsx", "export {};\n")
        repository.complete_tasks()

        response = repository.stop("implementation")

        self.assertEqual(response, {})
        self.assertEqual(repository.calls("just"), [])
        arguments = [" ".join(call["arguments"]) for call in repository.check_calls()]
        self.assertEqual(len(arguments), 2)
        self.assertFalse(any("fmt" in argument or "format" in argument for argument in arguments))
        self.assertEqual(len(repository.reviewer_calls()), 2)

    def test_runs_hook_tests_when_change_touches_hooks(self):
        repository = HookRepository(self)
        repository.submit_prompt("Continue")
        repository.write(".agent-hooks/sample_test.py", PASSING_HOOK_TEST)
        repository.complete_tasks()

        response = repository.stop("implementation")

        self.assertEqual(response, {})
        self.assertEqual(len(repository.reviewer_calls()), 2)

    def test_blocks_turn_when_hook_tests_fail(self):
        repository = HookRepository(self)
        repository.submit_prompt("Continue")
        repository.write(".agent-hooks/sample_test.py", FAILING_HOOK_TEST)
        repository.complete_tasks()

        response = repository.stop("implementation")

        self.assertEqual(response["decision"], "block")
        self.assertIn("gate returned block for an accepted turn", response["reason"])
        self.assertEqual(repository.reviewer_calls(), [])


class ImplementationGateReviewTest(ShortReplyAssertions, unittest.TestCase):
    def test_runs_reviewer_per_spec_and_one_for_docs_on_claude_with_medium_effort(self):
        repository = HookRepository(self)
        repository.write_change(spec_names=("widgets", "gadgets"), tasks="- [x] 1.1 Build\n")
        repository.submit_prompt("Continue")
        repository.complete_tasks()

        repository.stop("implementation")

        calls = repository.calls("claude")
        self.assertEqual(len(calls), 3)
        for call in calls:
            self.assertIn("--model opus --effort medium", " ".join(call["arguments"]))
            self.assertEqual(call["effort_env"], "medium")
        roles = sorted(line for call in calls for line in call["prompt"].splitlines() if line.startswith("Role:"))
        self.assertEqual(
            roles,
            [
                f"Role: delta spec `openspec/changes/{CHANGE_NAME}/specs/gadgets/spec.md`",
                f"Role: delta spec `openspec/changes/{CHANGE_NAME}/specs/widgets/spec.md`",
                "Role: documentation and tasks",
            ],
        )

    def test_runs_codex_reviewers_on_gpt_with_high_effort(self):
        repository = HookRepository(self)
        repository.write_change(spec_names=("widgets", "gadgets"), tasks="- [x] 1.1 Build\n")
        repository.submit_prompt("Continue", client="codex")
        repository.complete_tasks()

        repository.stop("implementation", client="codex")

        calls = repository.calls("codex")
        self.assertEqual(len(calls), 3)
        for call in calls:
            self.assertIn("--model gpt-6.1-sol --config model_reasoning_effort=\"high\"", " ".join(call["arguments"]))

    def test_implementation_docs_reviewer_prompt_checks_agent_instructions(self):
        repository = HookRepository(self)
        repository.submit_prompt("Continue")
        repository.complete_tasks()

        repository.stop("implementation")

        [call] = [call for call in repository.reviewer_calls() if "\nRole: documentation and tasks\n" in call["prompt"]]
        prompt = call["prompt"]
        docs_role = prompt[prompt.index("### Role: documentation and tasks") : prompt.index("### Role: repeat review")]
        self.assertIn("Check the edits to agent instructions the change made", docs_role)

    def test_blocks_turn_and_requires_agent_to_resolve_findings_when_reviewer_finds_bug(self):
        repository = HookRepository(self)
        repository.set_reviewer_result({"findings": [finding("bug", "Race between the check and the write.")]})
        repository.submit_prompt("Continue")
        repository.complete_tasks()

        response = repository.stop("implementation")

        self.assertEqual(response["decision"], "block")
        self.assertIn("[bug] src/widgets.py:10: Race between the check and the write.", response["reason"])
        self.assertIn("without asking the user", response["reason"])
        self.assert_asks_for_short_reply(response["reason"])

    def test_marks_minor_findings_optional_when_turn_is_blocked(self):
        repository = HookRepository(self)
        repository.set_reviewer_result(
            {"findings": [finding("docs", "The README does not mention the command."), finding("minor", "Long function.")]}
        )
        repository.submit_prompt("Continue")
        repository.complete_tasks()

        response = repository.stop("implementation")

        self.assertIn("[minor] (optional) src/widgets.py:10: Long function.", response["reason"])

    def test_accepts_change_and_shows_minor_findings_to_user(self):
        repository = HookRepository(self)
        repository.set_reviewer_result({"findings": [finding("minor", "Long function.")]})
        repository.submit_prompt("Continue")
        repository.complete_tasks()

        response = repository.stop("implementation")
        repository.clear_calls()
        repeated_response = repository.stop("implementation")

        self.assertNotIn("decision", response)
        self.assertIn("Long function.", response["systemMessage"])
        self.assertEqual(repeated_response, {})
        self.assertEqual(repository.calls(), [])

    def test_blocks_turn_when_reviewer_answer_does_not_match_schema(self):
        repository = HookRepository(self)
        repository.set_reviewer_result({"findings": [{"category": "typo", "file": "a", "summary": "b"}]})
        repository.submit_prompt("Continue")
        repository.complete_tasks()

        response = repository.stop("implementation")

        self.assertEqual(response["decision"], "block")
        self.assertIn("does not match the schema", response["reason"])

    def test_blocks_turn_with_failure_reason_when_reviewer_crashes(self):
        repository = HookRepository(self)
        repository.set_reviewer_failure("rate limit exceeded")
        repository.submit_prompt("Continue")
        repository.complete_tasks()

        response = repository.stop("implementation")

        self.assertEqual(response["decision"], "block")
        self.assertIn("rate limit exceeded", response["reason"])


class ImplementationGateRepeatTest(SummaryAssertions, unittest.TestCase):
    def block_with_bug(self, repository):
        repository.set_reviewer_result({"findings": [finding("bug", "Race between the check and the write.")]})
        repository.submit_prompt("Continue")
        repository.write("src/widgets.py", "WIDGETS = []\n")
        repository.write("src/gadgets.py", "GADGETS = []\n")
        repository.complete_tasks()
        repository.stop("implementation")
        repository.clear_calls()

    def test_repeat_runs_one_reviewer_with_previous_findings_and_paths_changed_after_review(self):
        repository = HookRepository(self)
        self.block_with_bug(repository)
        repository.write("src/widgets.py", "WIDGETS = ()\n")

        repository.stop("implementation")

        [call] = repository.reviewer_calls()
        self.assertIn("Role: repeat review", call["prompt"])
        self.assertIn("Race between the check and the write.", call["prompt"])
        changed_paths = call["prompt"].split("Paths changed after the previous review:")[1]
        self.assertIn("- src/widgets.py", changed_paths)
        self.assertNotIn("src/gadgets.py", changed_paths)

    def test_accepted_repeat_clears_findings(self):
        repository = HookRepository(self)
        self.block_with_bug(repository)
        repository.set_reviewer_result({"findings": []})
        repository.write("src/widgets.py", "WIDGETS = ()\n")

        repository.stop("implementation")
        repository.write("src/widgets.py", "WIDGETS = (1,)\n")
        repository.clear_calls()
        repository.stop("implementation")

        self.assertEqual(len(repository.reviewer_calls()), 2)

    def test_asks_for_summary_without_review_once_after_repeat_accepts_change(self):
        repository = HookRepository(self)
        self.block_with_bug(repository)
        repository.set_reviewer_result({"findings": [finding("minor", "Long function.")]})
        repository.write("src/widgets.py", "WIDGETS = ()\n")

        response = repository.stop("implementation")
        repository.clear_calls()
        summary_response = repository.stop("implementation")

        self.assertEqual(response["decision"], "block")
        self.assertIn("accepted the change", response["reason"])
        self.assertIn("summary of the implementation", response["reason"])
        self.assert_asks_for_summary_without_review(response["reason"])
        self.assertIn("Long function.", response["reason"])
        self.assertEqual(summary_response, {})
        self.assertEqual(repository.calls(), [])

    def test_stops_agent_with_remaining_findings_after_third_block(self):
        repository = HookRepository(self)
        self.block_with_bug(repository)
        repository.write("src/widgets.py", "WIDGETS = ()\n")
        second_response = repository.stop("implementation")
        repository.write("src/widgets.py", "WIDGETS = (1,)\n")

        third_response = repository.stop("implementation")

        self.assertEqual(second_response["decision"], "block")
        self.assertIs(third_response["continue"], False)
        self.assertIn("Race between the check and the write.", third_response["stopReason"])


class NecessityDuringImplementationReviewTest(unittest.TestCase):
    def block_with_spec_mismatch(self, repository):
        repository.set_reviewer_result({"findings": [finding("spec_mismatch", "The spec requires 409, the code answers 400.")]})
        repository.submit_prompt("Continue")
        repository.complete_tasks()
        repository.stop("implementation")
        repository.clear_calls()

    def test_skips_necessity_when_agent_fixes_delta_spec_after_finding(self):
        repository = HookRepository(self)
        self.block_with_spec_mismatch(repository)
        repository.write(f"openspec/changes/{CHANGE_NAME}/specs/widgets/spec.md", "# widgets\n\nAnswers 400.\n")

        response = repository.stop("necessity")

        self.assertEqual(response, {})
        self.assertEqual(repository.calls(), [])

    def test_runs_necessity_when_prompt_has_openspec_planning_command(self):
        repository = HookRepository(self)
        self.block_with_spec_mismatch(repository)
        repository.submit_prompt("/opsx:propose a new change")
        repository.write(f"openspec/changes/{CHANGE_NAME}/specs/widgets/spec.md", "# widgets\n\nAnswers 400.\n")

        repository.stop("necessity")

        self.assertEqual(len(repository.reviewer_calls()), 1)


class ProposalGateStartTest(unittest.TestCase):
    def test_runs_full_review_when_agent_writes_all_artifacts_without_command(self):
        repository = HookRepository(self)
        repository.submit_prompt("Write a spec for gadgets")
        repository.write_proposal()

        response = repository.stop("proposal")

        self.assertEqual(response, {})
        self.assertEqual(len(repository.reviewer_calls()), 3)

    def test_skips_review_without_calling_openspec_when_tasks_are_missing(self):
        repository = HookRepository(self)
        repository.submit_prompt("Write a spec for gadgets")
        repository.write(f"openspec/changes/{PROPOSAL_NAME}/proposal.md", "# Proposal\n")

        response = repository.stop("proposal")

        self.assertEqual(response, {})
        self.assertEqual(repository.calls(), [])

    def test_skips_review_without_calling_openspec_when_tasks_file_has_no_tasks(self):
        repository = HookRepository(self)
        repository.submit_prompt("Write a spec for gadgets")
        repository.write_proposal(tasks="")

        response = repository.stop("proposal")

        self.assertEqual(response, {})
        self.assertEqual(repository.calls(), [])

    def test_skips_review_when_openspec_reports_incomplete_artifacts(self):
        repository = HookRepository(self)
        repository.set_stub("openspec", {"status": {"output": json.dumps({"isPlanningComplete": False})}})
        repository.submit_prompt("Write a spec for gadgets")
        repository.write_proposal()

        response = repository.stop("proposal")

        self.assertEqual(response, {})
        self.assertEqual(repository.reviewer_calls(), [])

    def test_skips_review_when_openspec_status_fails(self):
        repository = HookRepository(self)
        repository.set_stub("openspec", {"status": {"exit_code": 1, "output": "openspec: unknown change"}})
        repository.submit_prompt("Write a spec for gadgets")
        repository.write_proposal()

        response = repository.stop("proposal")

        self.assertEqual(response, {})
        self.assertEqual(repository.reviewer_calls(), [])

    def test_skips_review_without_calling_openspec_when_task_is_checked(self):
        repository = HookRepository(self)
        repository.submit_prompt("Fix the widgets spec")
        repository.write(f"openspec/changes/{CHANGE_NAME}/specs/widgets/spec.md", "# widgets\n\nAnswers 409.\n")

        response = repository.stop("proposal")

        self.assertEqual(response, {})
        self.assertEqual(repository.calls(), [])

    def test_skips_review_when_change_files_are_unchanged_during_turn(self):
        repository = HookRepository(self)
        repository.write_proposal()
        repository.submit_prompt("What is in the proposal?")

        response = repository.stop("proposal")

        self.assertEqual(response, {})
        self.assertEqual(repository.calls(), [])

    def test_skips_review_in_new_session_when_files_match_accepted_state(self):
        repository = HookRepository(self)
        repository.submit_prompt("Write a spec", session_id="first")
        repository.submit_prompt("Look at the spec", session_id="second")
        repository.write_proposal()
        repository.stop("proposal", session_id="first")
        repository.clear_calls()

        response = repository.stop("proposal", session_id="second")

        self.assertEqual(response, {})
        self.assertEqual(repository.calls(), [])

    def test_runs_review_again_when_design_changes_after_accepted_review(self):
        repository = HookRepository(self)
        repository.submit_prompt("Write a spec")
        repository.write_proposal()
        repository.stop("proposal")
        repository.clear_calls()
        repository.submit_prompt("Fix the design")
        repository.write(f"openspec/changes/{PROPOSAL_NAME}/design.md", "# Design\n\nIn-memory cache.\n")

        repository.stop("proposal")

        self.assertEqual(len(repository.reviewer_calls()), 3)

    def test_skips_archived_changes(self):
        repository = HookRepository(self)
        repository.submit_prompt("Archive it")
        repository.write_change(spec_names=("gadgets",), tasks="- [ ] 1.1 Build\n", name=f"archive/2026-10-01-{PROPOSAL_NAME}")

        response = repository.stop("proposal")

        self.assertEqual(response, {})
        self.assertEqual(repository.calls(), [])


class ProposalGateValidationTest(unittest.TestCase):
    def test_blocks_turn_with_validation_output_when_requirement_has_no_scenario(self):
        repository = HookRepository(self)
        repository.set_stub(
            "openspec",
            {"validate": {"exit_code": 1, "output": '✗ [ERROR] gadgets/spec.md: ADDED "Gadgets" must include at least one scenario\n'}},
        )
        repository.submit_prompt("Write a spec")
        repository.write_proposal()

        response = repository.stop("proposal")

        self.assertEqual(response["decision"], "block")
        self.assertIn(f"`openspec validate {PROPOSAL_NAME} --strict`", response["reason"])
        self.assertIn("must include at least one scenario", response["reason"])
        self.assertEqual(repository.reviewer_calls(), [])


class ProposalGateReviewTest(ShortReplyAssertions, SummaryAssertions, unittest.TestCase):
    def test_runs_reviewer_per_spec_and_for_proposal_and_tasks_on_claude_with_medium_effort(self):
        repository = HookRepository(self)
        repository.submit_prompt("Write a spec")
        repository.write_proposal(spec_names=("gadgets", "gizmos"))

        repository.stop("proposal")

        calls = repository.calls("claude")
        self.assertEqual(len(calls), 4)
        for call in calls:
            self.assertIn("--model opus --effort medium", " ".join(call["arguments"]))
            self.assertEqual(call["effort_env"], "medium")
        roles = sorted(line for call in calls for line in call["prompt"].splitlines() if line.startswith("Role:"))
        self.assertEqual(
            roles,
            [
                f"Role: delta spec `openspec/changes/{PROPOSAL_NAME}/specs/gadgets/spec.md`",
                f"Role: delta spec `openspec/changes/{PROPOSAL_NAME}/specs/gizmos/spec.md`",
                "Role: proposal and design",
                "Role: tasks",
            ],
        )

    def test_runs_codex_reviewers_on_gpt_with_high_effort(self):
        repository = HookRepository(self)
        repository.submit_prompt("Write a spec", client="codex")
        repository.write_proposal(spec_names=("gadgets", "gizmos"))

        repository.stop("proposal", client="codex")

        calls = repository.calls("codex")
        self.assertEqual(len(calls), 4)
        for call in calls:
            self.assertIn("--model gpt-6.1-sol --config model_reasoning_effort=\"high\"", " ".join(call["arguments"]))

    def test_tasks_reviewer_prompt_requires_every_scenario_covered_by_task_with_test(self):
        repository = HookRepository(self)
        repository.submit_prompt("Write a spec")
        repository.write_proposal()

        repository.stop("proposal")

        [call] = [call for call in repository.reviewer_calls() if "\nRole: tasks\n" in call["prompt"]]
        self.assertIn("every scenario of the delta specs is covered by a task with a test", call["prompt"])
        self.assertIn("the task order can be followed", call["prompt"])

    def test_blocks_turn_and_requires_agent_to_resolve_simplify_finding_without_asking_user(self):
        repository = HookRepository(self)
        repository.set_reviewer_result(
            {"findings": [finding("simplify", "A field in an existing table replaces the new table.", file="design.md:12")]}
        )
        repository.submit_prompt("Write a spec")
        repository.write_proposal()

        response = repository.stop("proposal")

        self.assertEqual(response["decision"], "block")
        self.assertIn("[simplify] design.md:12: A field in an existing table replaces the new table.", response["reason"])
        self.assertIn("without asking the user", response["reason"])
        self.assert_asks_for_short_reply(response["reason"])

    def test_asks_for_summary_with_scope_finding_without_requiring_fix(self):
        repository = HookRepository(self)
        repository.set_reviewer_result({"findings": [finding("scope", "The settings screen can be dropped.", file="proposal.md:5")]})
        repository.submit_prompt("Write a spec")
        repository.write_proposal()

        response = repository.stop("proposal")
        repository.clear_calls()
        summary_response = repository.stop("proposal")

        self.assertEqual(response["decision"], "block")
        self.assertIn("Full proposal review accepted the proposal.", response["reason"])
        self.assertIn("[scope] (the user decides) proposal.md:5: The settings screen can be dropped.", response["reason"])
        self.assertIn("summary of the proposal", response["reason"])
        self.assert_asks_for_summary_without_review(response["reason"])
        self.assertIn("what the user loses and what they save", response["reason"])
        self.assertEqual(summary_response, {})
        self.assertEqual(repository.calls(), [])

    def test_accepts_proposal_and_shows_minor_findings_to_user(self):
        repository = HookRepository(self)
        repository.set_reviewer_result({"findings": [finding("minor", "Long paragraph.", file="design.md:3")]})
        repository.submit_prompt("Write a spec")
        repository.write_proposal()

        response = repository.stop("proposal")
        repository.clear_calls()
        repeated_response = repository.stop("proposal")

        self.assertNotIn("decision", response)
        self.assertIn("Long paragraph.", response["systemMessage"])
        self.assertEqual(repeated_response, {})
        self.assertEqual(repository.calls(), [])

    def test_blocks_turn_when_reviewer_uses_implementation_category(self):
        repository = HookRepository(self)
        repository.set_reviewer_result({"findings": [finding("bug", "Race.")]})
        repository.submit_prompt("Write a spec")
        repository.write_proposal()

        response = repository.stop("proposal")

        self.assertEqual(response["decision"], "block")
        self.assertIn("does not match the schema", response["reason"])


class ProposalGateRepeatTest(ShortReplyAssertions, SummaryAssertions, unittest.TestCase):
    def block_with_gap(self, repository):
        repository.set_reviewer_result({"findings": [finding("gap", "The answer for an empty list is missing.", file="specs/gadgets/spec.md:8")]})
        repository.submit_prompt("Write a spec")
        repository.write_proposal()
        repository.stop("proposal")
        repository.clear_calls()

    def test_blocks_turn_with_gap_and_asks_for_short_reply_without_summary(self):
        repository = HookRepository(self)
        repository.set_reviewer_result({"findings": [finding("gap", "The answer for an empty list is missing.", file="specs/gadgets/spec.md:8")]})
        repository.submit_prompt("Write a spec")
        repository.write_proposal()

        response = repository.stop("proposal")

        self.assertEqual(response["decision"], "block")
        self.assertIn("[gap] specs/gadgets/spec.md:8: The answer for an empty list is missing.", response["reason"])
        self.assertIn("without asking the user", response["reason"])
        self.assert_asks_for_short_reply(response["reason"])

    def test_repeat_runs_one_reviewer_with_previous_findings_and_files_changed_after_review(self):
        repository = HookRepository(self)
        self.block_with_gap(repository)
        repository.write(f"openspec/changes/{PROPOSAL_NAME}/specs/gadgets/spec.md", "# gadgets\n\nEmpty list.\n")

        repository.stop("proposal")

        [call] = repository.reviewer_calls()
        self.assertIn("Role: repeat review", call["prompt"])
        self.assertIn("The answer for an empty list is missing.", call["prompt"])
        changed_paths = call["prompt"].split("Paths changed after the previous review:")[1]
        self.assertIn(f"- openspec/changes/{PROPOSAL_NAME}/specs/gadgets/spec.md", changed_paths)
        self.assertNotIn("design.md", changed_paths)

    def test_asks_for_summary_without_review_once_after_repeat_accepts_proposal(self):
        repository = HookRepository(self)
        self.block_with_gap(repository)
        repository.set_reviewer_result({"findings": []})
        repository.write(f"openspec/changes/{PROPOSAL_NAME}/specs/gadgets/spec.md", "# gadgets\n\nEmpty list.\n")

        response = repository.stop("proposal")
        repository.clear_calls()
        summary_response = repository.stop("proposal")

        self.assertEqual(response["decision"], "block")
        self.assertIn("accepted the proposal", response["reason"])
        self.assertIn("summary of the proposal", response["reason"])
        self.assert_asks_for_summary_without_review(response["reason"])
        self.assertEqual(summary_response, {})
        self.assertEqual(repository.calls(), [])

    def test_lists_scope_option_in_summary_without_review_when_repeat_accepts_proposal(self):
        repository = HookRepository(self)
        self.block_with_gap(repository)
        repository.set_reviewer_result({"findings": [finding("scope", "The settings screen can be dropped.", file="proposal.md:5")]})
        repository.write(f"openspec/changes/{PROPOSAL_NAME}/specs/gadgets/spec.md", "# gadgets\n\nEmpty list.\n")

        response = repository.stop("proposal")

        self.assertEqual(response["decision"], "block")
        self.assertIn("[scope] (the user decides) proposal.md:5: The settings screen can be dropped.", response["reason"])
        self.assertIn("List every scope finding separately: what the user loses and what they save", response["reason"])
        self.assert_asks_for_summary_without_review(response["reason"])

    def test_stops_agent_with_remaining_findings_after_third_block(self):
        repository = HookRepository(self)
        self.block_with_gap(repository)
        repository.write(f"openspec/changes/{PROPOSAL_NAME}/design.md", "# Design\n\nSecond attempt.\n")
        second_response = repository.stop("proposal")
        repository.write(f"openspec/changes/{PROPOSAL_NAME}/design.md", "# Design\n\nThird attempt.\n")

        third_response = repository.stop("proposal")

        self.assertEqual(second_response["decision"], "block")
        self.assertIs(third_response["continue"], False)
        self.assertIn("Full proposal review", third_response["stopReason"])
        self.assertIn("The answer for an empty list is missing.", third_response["stopReason"])


class QuickGatesDuringProposalReviewTest(unittest.TestCase):
    def test_skips_quick_checks_when_turn_changes_only_ready_proposal(self):
        repository = HookRepository(self)
        repository.submit_prompt("Write a spec")
        repository.write_proposal()

        responses = [repository.stop(gate) for gate in ("necessity", "architecture", "diff")]

        self.assertEqual(responses, [{}, {}, {}])
        self.assertEqual(repository.reviewer_calls(), [])

    def test_skips_quick_checks_in_summary_turn_after_accepted_review(self):
        repository = HookRepository(self)
        repository.set_reviewer_result({"findings": [finding("scope", "The settings screen can be dropped.")]})
        repository.submit_prompt("Write a spec")
        repository.write_proposal()
        repository.stop("proposal")
        repository.clear_calls()

        responses = [repository.stop(gate, message="Proposal summary.") for gate in ("necessity", "architecture", "diff")]

        self.assertEqual(responses, [{}, {}, {}])
        self.assertEqual(repository.reviewer_calls(), [])

    def test_runs_necessity_when_agent_wrote_only_proposal(self):
        repository = HookRepository(self)
        repository.submit_prompt("Write a spec")
        repository.write(f"openspec/changes/{PROPOSAL_NAME}/proposal.md", "# Proposal\n")

        repository.stop("necessity", message="Is the settings screen needed?")

        [call] = repository.reviewer_calls()
        self.assertIn("Under review: OpenSpec proposal", call["prompt"])

    def test_runs_quick_checks_without_calling_openspec_when_code_changes_too(self):
        repository = HookRepository(self)
        repository.submit_prompt("Write a spec and start")
        repository.write_proposal()
        repository.write("src/gadgets.py", "GADGETS = []\n")

        response = repository.stop("diff")

        self.assertEqual(response, {})
        self.assertEqual(len(repository.reviewer_calls()), 1)
        self.assertEqual(repository.calls("openspec"), [])


if __name__ == "__main__":
    unittest.main()
