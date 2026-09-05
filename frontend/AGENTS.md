# Frontend guidelines (React + TypeScript)

This document contains coding standards for the Databasus frontend (React 19 + TypeScript + Vite + Ant Design + TailwindCSS).
For project-wide engineering philosophy, see the root `AGENTS.md`.

---

## Table of Contents

- [UI kit and icons](#ui-kit-and-icons)
- [React component structure](#react-component-structure)
- [Vertical spacing](#vertical-spacing)
- [Clipboard operations](#clipboard-operations)
- [Forms](#forms)
- [User-facing copy](#user-facing-copy)
- [FSD (Feature-Sliced Design)](#fsd-feature-sliced-design)
- [Refactoring](#refactoring)

---

## UI kit and icons

- **AntD 5 only** for components. Don't pull in Mantine, MUI, Chakra, shadcn, or Radix directly. Use AntD primitives (`Button`, `Input`, `Modal`, `Form`, `Table`, `Menu`, `Tabs`, etc.) plus Tailwind utility classes for layout and spacing.
- **`@ant-design/icons` only** for icons. Don't add `lucide-react`, `@heroicons/react`, `react-icons`, or FontAwesome.

---

## React component structure

Write React components with the following structure:

```typescript
interface Props {
   someValue: SomeValue;
}

const someHelperFunction = () => {
    ...
}

export const ReactComponent = ({ someValue }: Props): JSX.Element => {
    // First put states
    const [someState, setSomeState] = useState<...>(...)

    // Then place functions
    const loadSomeData = async () => {
        ...
    }

    // Then hooks
    useEffect(() => {
       loadSomeData();
    });

    // Then calculated values
    const calculatedValue = someValue.calculate();

    return <div> ... </div>
}
```

### Structure order

1. **Props interface** — Define component props
2. **Helper functions** (outside component) — Pure utility functions
3. **Component declaration**
   - **States** — `useState` declarations
   - **Plain functions** — Event handlers, async operations, in-component formatters. Anything that does _not_ call a React hook.
   - **Hooks** — `useRef` + ref mutation, `useCallback`, `useMemo`, `useEffect`. A function wrapped in `useCallback` is a hook — it lives here, not in the functions section.
   - **Calculated values** — Derived data computed inline (e.g. an AntD `columns` array).
   - **Return** — JSX markup

**All hooks (including every `useEffect`) come below every plain function definition.** If a `useEffect` reads a handler, the handler must be defined above it — don't reorder by putting handlers below the effects.

---

## Vertical spacing

- Structure function bodies with vertical rhythm. Put a blank line between logically distinct steps (setup / main work / return, independent branches, before and after a guard). Do not leave blank lines at the start or end of a body, never stack two in a row. Do not insert blanks inside a tight expression, a single statement split across lines, or a short (≤ 5-line) function. If blank lines alone aren't enough to navigate a body, extract — don't add comments.

---

## Clipboard operations

Always use `ClipboardHelper` (`shared/lib/ClipboardHelper.ts`) for clipboard operations — never call `navigator.clipboard` directly.

- **Copy:** `ClipboardHelper.copyToClipboard(text)` — uses `navigator.clipboard` with `execCommand('copy')` fallback for non-secure contexts (HTTP).
- **Paste:** Check `ClipboardHelper.isClipboardApiAvailable()` first. If available, use `ClipboardHelper.readFromClipboard()`. If not, show `ClipboardPasteModalComponent` (`shared/ui`) which lets the user paste manually via a text input modal.

---

## Forms

### Progressive disclosure

- Submit buttons (`Save`, `Update password`, etc.) render **only when the form is dirty**.
- Dependent fields (e.g. `Confirm password`) render **only after the field they depend on has a value**.

---

## User-facing copy

Use a plain hyphen `-` in any string the user will see — labels, descriptions, notifications, modal bodies, error messages. Reserve em dashes (`—`) and en dashes (`–`) for markdown docs and code comments only.

---

## FSD (Feature-Sliced Design)

The project follows FSD v2.1. Layers in use: `app/`, `pages/`, `widgets/`, `features/`, `entity/`, `shared/`.

### Import direction

`app → pages → widgets → features → entity → shared`

A module may only import from layers strictly **below** it. Cross-imports between slices on the **same** layer are forbidden.

Paths below are written from `src/` for readability. Real imports are relative; the project has no path alias.

```text
✅ Allowed
  a widget       imports  features/backups, entity/backups, shared/ui
  a feature      imports  entity/databases, shared/api
  an entity      imports  shared/lib

❌ Violations
  an entity      imports  features/databases      (upward: entity sits below features)
  a feature      imports  pages/AuthPageComponent (upward: pages sits above features)
  features/users imports  features/backups        (same layer: cross-slice import)
  shared/lib     imports  entity/users            (upward: shared is the bottom layer)
```

### Where new code goes

- Used in only one page → keep it in that `pages/` slice.
- Reusable infrastructure, **no business logic** → `shared/` (UI kit, utils, API client, route constants, auth tokens, CRUD helpers).
- User interaction reused in 2+ pages → `features/`.
- Domain model reused in 2+ pages/features → `entity/`.
- App-wide providers, router, theme → `app/`.

**When in doubt — keep it in `pages/`.** Extract only when a second real consumer appears.

### Quick placement table

| Scenario                 | Single use                            | Reused in 2+ places                   |
| ------------------------ | ------------------------------------- | ------------------------------------- |
| Profile form             | `pages/profile/ui/ProfileForm.tsx`    | `features/profile-form/`              |
| Database card            | `pages/databases/ui/DatabaseCard.tsx` | `entity/database/ui/DatabaseCard.tsx` |
| Data fetching for backup | `pages/backup/api/fetch-backup.ts`    | `entity/backup/api/`                  |
| Auth token / session     | `shared/auth/` (always)               | `shared/auth/` (always)               |
| Login form               | `pages/login/ui/LoginForm.tsx`        | `features/auth/`                      |
| CRUD helpers             | `shared/api/` (always)                | `shared/api/` (always)                |
| Date formatting util     | —                                     | `shared/lib/format-date.ts`           |
| Modal content            | `pages/[page]/ui/SomeModal.tsx`       | —                                     |

### MUST rules

1. **Downward-only imports.** No upward imports, no same-layer cross-imports.
2. **Public API via `index.ts`.** External consumers import only from a slice's `index.ts`, never its internal files.
   ```text
   ✅  features/users
   ❌  features/users/ui/AuthNavbarComponent
   ```
3. **Domain-based file names.** Name files by the domain they represent, not their technical role.
   ```text
   // ❌  model/types.ts, model/utils.ts, lib/helpers.ts
   // ✅  model/user.ts, model/backup.ts, api/fetch-database.ts
   ```
4. **No business logic in `shared/`.** Shared holds only infrastructure. Domain calculations live in `entity/` or higher.
5. **One type per file in `model/` / DTOs.** Each `interface`, `class`, or `enum` that represents a domain entity, DTO, request body, or response shape gets its own file in `model/` (or `models/`), named after the type. Don't co-locate sibling types like `Foo` + `FooStatus` + `FooResponse` in a single `Foo.ts` — split them. Re-export each from the slice `index.ts` so consumers still import from the slice's public API.
   ```text
   // ❌  model/RestoreVerification.ts — enums + table-stat + main interface all in one file
   // ✅  model/RestoreVerification.ts        — interface RestoreVerification
   // ✅  model/RestoreVerificationTableStat.ts — interface RestoreVerificationTableStat
   // ✅  model/VerificationStatus.ts         — enum VerificationStatus
   // ✅  model/VerificationTrigger.ts        — enum VerificationTrigger
   ```
   Tiny shape-only helper types tightly coupled to a parent type (a literal-union alias, a `Pick<>`) may stay in the parent's file. Splitting applies to anything with its own identity — anything you'd reasonably import on its own elsewhere.

### Segments inside a slice

- `ui/` — components, styles
- `model/` — state, types, domain logic, validation
- `api/` — backend calls, request functions, API-specific types
- `lib/` — internal helpers for this slice only
- `config/` — slice-level config / feature flags

`app/` and `shared/` have **segments only, no slices**. Segments within them may import from each other.

### AVOID

- Creating an entity prematurely (single consumer → keep it in the page).
- Putting CRUD inside `entity/` — CRUD is infrastructure, goes to `shared/api/`.
- Creating a `user` entity just for auth tokens/DTOs — those belong in `shared/auth/` or `shared/api/`.
- Extracting single-use code "for future reuse."
- God slices (`user-management/` covering auth + profile + password) — split by focused responsibility.
- Importing UI segments of one entity from another entity. Entity UI may be imported only from features/widgets/pages.
- Abusing the `@x` cross-import pattern — it is a last resort, not a tool.

### Cross-imports between same-layer slices

When two slices on the same layer need to share code, try **in order**:

1. **Merge** — if they always change together, they are one slice.
2. **Extract shared logic down to `entity/`** — keep UI in the features/widgets.
3. **Compose in a higher layer (IoC)** — the parent page/widget imports both and wires them together via props/slots.
4. **`@x` notation** — explicit, documented cross-import between entities only. Last resort.

---

## Refactoring

When applying changes, **do not forget to refactor old code**. You can shortify, make more readable, improve code quality, etc. Common logic can be extracted to functions, constants, files, etc.

**After each large change with more than ~50-100 lines of code:**

- Run `pnpm format` (from `frontend/` root folder)
- Run `pnpm lint` to verify the change
