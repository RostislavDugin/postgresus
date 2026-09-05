## Why

The marketing website ships in six languages, but the product behind it is English only. A user who arrives through the Russian or Chinese landing page logs in and hits an entirely English interface. The website translation set the audience expectation; the application has to meet it.

There is also a window that closes. Today's 32,700 lines of frontend code carry roughly 1,500 hardcoded strings, and that number grows with every feature. Extraction gets more expensive the longer it waits, and nothing currently stops a new hardcoded string from landing.

## Governing docs

This change answers to [`AGENTS.md`](../../../AGENTS.md) at the repo root and to [`frontend/AGENTS.md`](../../../frontend/AGENTS.md). It also amends the root doc's language rule, which [`assets/readme/AGENTS.md`](../../../assets/readme/AGENTS.md) restates, so that doc is in scope for consistency even though no README translation changes. It touches no backend, agent or website code, so those module docs do not apply. `website/AGENTS.md` is relevant only as prior art: its translation-quality rules are the reference for how each language is written.

### **BREAKING** rule change, approved and applied

The root doc's "Language in code" section permitted translated end-user content in three places and nowhere else: the website page copies, the README translations, and the six language-switcher labels in the root `README.md`. It forbade any language other than English inside `frontend/src/`, fallback copy and error messages included.

This change puts Russian, Spanish, Portuguese, Chinese and French text into `frontend/src/shared/i18n/locales/*.ts`, which that wording forbade. The rule now names a fourth permitted location, and nothing about the change is implementable without it.

The amendment adds exactly one location, `frontend/src/shared/i18n/locales/<locale>.ts`, and changes nothing else: identifiers, file names, dictionary keys, comments, log messages, API strings, test assertions and commit messages stay English, and every other file under `frontend/src/` stays English including fallback copy and error messages. It is recorded in the root `AGENTS.md` under "Language in code" (task 2.2).

## What Changes

- Add an i18n layer under `frontend/src/shared/i18n`: `i18next` + `react-i18next`, a `LocaleProvider` mirroring the existing `ThemeProvider`, and a language control in the navbar.
- Ship the same six languages as the website: `en` (source of truth) plus `ru`, `es`, `pt`, `zh`, `fr`.
- Author locale dictionaries as TypeScript modules typed against the English one, so a missing or stray key fails `tsc -b` instead of silently falling back at runtime.
- Replace every user-facing hardcoded string in `frontend/src` with a dictionary lookup, slice by slice.
- Add `eslint-plugin-i18next` and enable `i18next/no-literal-string` per directory as each slice is migrated, so migrated code cannot regress.
- Route AntD's own strings through `ConfigProvider locale` and date formatting through `dayjs.locale`, both following the selected language.
- Group numbers by the selected language instead of the browser's, so digits and dates on the same screen stop disagreeing.
- Set the document's language attribute and title from the selected language, replacing the hardcoded `lang="en"` and English `<title>` in `frontend/index.html`.
- Translate status labels via lookup tables keyed by the existing status enums, replacing the copy-pasted `renderStatusTag` / `renderStatus` variants. Enum values stay English: they are wire protocol.
- **BREAKING** for contributors, not users: after a slice is migrated, a literal string in its JSX fails the lint job.

### The language control

The website's `LanguageSelectorComponent` is the reference: a compact button in the navbar showing the current language, a dropdown that opens on hover and lists the six languages by their native names.

Here it is merged with the existing theme control into one paired button, language on one side and Dark/Light/System on the other, in the style the website already uses to glue the language selector to the GitHub button (`isSplitEnd`).

That merge also settles placement. `ThemeToggleComponent` already renders in three places, and the paired control replaces it at all three:

- `widgets/main/MainScreenComponent.tsx:254` - the main navbar
- `widgets/main/SidebarComponent.tsx:132` - the sidebar, for narrow viewports
- `features/users/ui/AuthNavbarComponent.tsx:49` - the authentication screen

So the language is selectable before login, not only after it. The choice is stored in `localStorage`, and on a first visit with nothing stored the interface starts in the highest-ranked language the browser prefers that the product actually ships.

### Out of scope

- The backend. Its ~416 free-text error responses keep reaching the user in English through `alert(e.message)`. This change only fixes the frontend's side of the contract: translate by `ApiError.code` when one is present, fall back to `message` when it is not, so the backend work later is additive.
- Replacing `alert()` with AntD notifications (20+ sites). A real improvement, a separate change.
- Unit words stay English: `h`/`m`/`s` in `formatDuration`, `MB` and `GB` in sizes. `formatDuration` needs no change at all, its digits being too small to group. The digits in sizes and counts are in scope and follow the selected language.
- Clock format and date order stay derived from the browser, not from the selected language. See design.md - Two axes.
- Technical strings: engine names, `pg_dump` and friends, connection strings, command blocks, cron expressions.
- Persisting the language choice server side. It lives in `localStorage`, like the theme.

## Capabilities

### New Capabilities

- `frontend-localization`: the frontend renders its interface in a user-selected language, keeps dictionaries provably complete, and prevents untranslated strings from being added.

### Modified Capabilities

None. This is the repository's first change; `openspec/specs/` is empty.

## Impact

- `frontend/package.json`: adds `i18next`, `react-i18next`, `eslint-plugin-i18next`. Runtime cost is roughly 40 KB plus one dictionary per session.
- `frontend/eslint.config.js`: gains one config object per migrated directory.
- `frontend/src/App.tsx`: `LocaleProvider` wraps the tree, `ConfigProvider` receives an AntD locale.
- `frontend/index.html`: the hardcoded `lang="en"` and English `<title>` are set from the selected locale instead.
- Eight `toLocaleString()` call sites: number grouping follows the selected language rather than the browser's.
- `frontend/src/shared/ui/ThemeToggleComponent.tsx`: becomes half of a paired control; its three call sites change with it.
- `frontend/src/features/**` (105 files, 23,800 lines), `frontend/src/entity/**` (~136 strings), `frontend/src/widgets/**`, `frontend/src/pages/**`: every user-facing literal moves to a dictionary key.
- `AGENTS.md` (root): the "Language in code" rule gains a fourth permitted location. See the rule change above.
- `frontend/AGENTS.md`: the copy rules gain the i18n convention.
- Existing frontend tests that assert on English text.
- No backend, agent, or API changes.
