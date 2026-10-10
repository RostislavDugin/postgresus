## 1. Invitation link

- [x] 1.1 Add `SetEmailSender` to `MembershipService` in `backend/internal/features/workspaces/services/membership_service.go`, mirroring `UserService.SetEmailSender`, and verify with `go build ./...` in `backend/`
- [x] 1.2 Add controller tests to `backend/internal/features/workspaces/controllers/membership_controller_test.go` that replace the sender with `workspaces_testing.MockEmailSender`, set `config.GetEnv().DatabasusURL` to `https://databasus.example.com` and to `https://databasus.example.com/`, add a member with an unknown address, and assert that the captured body contains `href="<configured value>"` exactly. Restore the sender and the URL in `t.Cleanup`. Verify that both tests fail with `just test-package ./internal/features/workspaces/...` in `backend/` while the link still appends `/sign-up`
- [x] 1.3 Make `buildInvitationEmailHTML` link to `env.DatabasusURL` with no path appended, and verify that the tests from 1.2 pass with `just test-package ./internal/features/workspaces/... ./internal/features/users/...` in `backend/`. The same run includes the existing `Test_InviteUserToWorkspace_MembershipReceivedAfterSignUp` in `backend/internal/features/users/controllers/management_controller_test.go`, which pins the membership half of the "Invited person opens the link" scenario

## 2. Unknown addresses in the web app

- [x] 2.1 Move `frontend/src/App.tsx` to `frontend/src/app/ui/App.tsx` with `git mv`, update its relative imports and the import in `frontend/src/main.tsx`, and verify with `pnpm typecheck`, `pnpm lint` and `pnpm build` in `frontend/`
- [x] 2.2 Add a catch-all `*` route rendering `<Navigate to="/" replace />` to the route table in `frontend/src/app/ui/App.tsx`, after the existing routes, and verify with `pnpm typecheck` and `pnpm lint` in `frontend/`. The three scenarios of "An unknown address in the web app leads to the main screen" and the "sign-up form is shown" half of "Invited person opens the link" are deliberately left without an automated test, per the design decision "No automated frontend test for the catch-all"

## 3. Verification

- [x] 3.1 Run `just lint` at the root, and `pnpm test` and `pnpm build` in `frontend/`
- [x] 3.2 Run `openspec validate fix-invitation-sign-up-link --strict`
