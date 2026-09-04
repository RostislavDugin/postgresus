## Context

See proposal.md - Why.

The state that shapes the approach:

- `frontend/src` is 32,700 lines. Strings are inlined at their point of use, with no copy layer. A crude grep finds 299 text attributes, 411 JSX text nodes and 136 literals in `.ts` files; the real count is closer to 1,500 once multi-line JSX, template literals, option arrays and `alert()` calls are included.
- Copy is not confined to `ui/` segments. `entity/databases/model/postgresql/ConnectionStringParser.ts` returns error prose, `entity/notifiers/models/getNotifierNameFromType.ts` returns display names. The dictionary has to be reachable from `entity/` and `shared/`, which puts it in `shared/`.
- Two rules in `frontend/AGENTS.md` bear on where the dictionaries go. FSD MUST rule 4 says shared holds only infrastructure, and the dictionaries are domain-scoped (`backups`, `databases`, `storages`). The reading this design adopts: the provider, `t()` and the locale wiring are infrastructure, and copy strings carry no domain calculation, so `shared/i18n` is the right layer. The alternative reading, that copy is domain content belonging in `entity/`, would scatter one dictionary across seven slices and break the single-source-of-truth typing that the whole approach rests on. Recorded here so it is a decision rather than an oversight.
- `ThemeProvider` (`shared/theme/`) already establishes the pattern for a user preference: React context, `localStorage`, a toggle in the header. Locale follows it rather than inventing a second shape.
- `ThemeToggleComponent` already renders at three call sites: `widgets/main/MainScreenComponent.tsx:254`, `widgets/main/SidebarComponent.tsx:132` and `features/users/ui/AuthNavbarComponent.tsx:49`. Merging the language control into it covers the navbar, the narrow-viewport sidebar and the pre-login screen in one move.
- The website's `website/app/components/LanguageSelectorComponent.tsx` is working prior art, including an `isSplitEnd` variant that glues the selector to a neighbouring button as one split control.
- ESLint uses flat config (`tseslint.config(...)`). There is no `overrides` key; per-directory rules are additional config objects.
- `shared/api/apiHelper.ts` already parses `json.code` into `ApiError.code`, and `EditPostgreSqlPhysicalSpecificDataComponent.tsx:204` already branches on it. The error-code contract exists in one place and needs generalizing, not inventing.

The website solved the same problem differently, with five full page copies per page and no dictionaries. That works there because each URL needs its own static HTML for search engines and the pages are prose. Applying it here would mean 525 component copies held structurally in sync. The application diverges deliberately.

## Goals / Non-Goals

**Goals:**

- A forgotten translation is a compile error, not a runtime fallback.
- The migration is a burn-down with a machine-checkable finish line, not an open-ended sweep.
- Each language reads as if written in it, so the key is the identity and the English text is just the English value.

**Non-Goals:**

- Right-to-left languages. None of the six need it, and no layout work assumes it.
- Lazy-loading dictionaries per feature slice. One dictionary per language, loaded whole. Revisit if a dictionary passes a few hundred KB.
- Machine translation in the build. Translations are written and reviewed, like the website's.

## Decisions

### Dictionaries, not per-language component copies

See Context. Component reuse is the deciding factor: the same `EditStorageComponent` serves six languages, so language must be a runtime lookup rather than a file variant.

### `i18next` + `react-i18next`

Alternatives considered:

- **LinguiJS.** Rejected on the key strategy. Lingui uses the English source text as the key, so editing an English phrase silently invalidates all five translations. Since each language is written independently rather than rendered literally, keys must be stable and text-independent.
- **A hand-rolled `t()`, roughly 60 lines.** Attractive given the project's tight dependency budget (AntD only, one icon set). Rejected because Russian needs three plural forms, and interpolation, plural selection, and namespacing grow into a quarter of a library within a year. If usage stays limited to `{name}` substitution and plurals, this decision is cheap to revisit.
- **`react-intl` / FormatJS.** Comparable, but a heavier compile step for ICU messages than this codebase needs.

Required configuration, because the defaults are wrong here:

- `returnNull: false`, otherwise `t()` is typed `string | null` and every call site needs a guard.
- `nsSeparator: false`, otherwise a key containing a colon is misparsed as a namespace reference.
- `interpolation.escapeValue: false`, since React escapes already.
- `fallbackLng: 'en'` as a last-resort runtime guard only. Completeness is enforced by types (below); the fallback exists for a dictionary that failed to load, not for a missing key.

### Type safety comes from the locale files, not the library

This is the load-bearing decision, and no library provides it out of the box: i18next falls back at runtime, Lingui reports gaps in a CLI run. Both are checks a person can ignore.

Two independent layers:

```
shared/i18n/locales/en.ts        <- source of truth for the key set
    export const en = { backups: { logical: { title: 'Logical backups' } } }
             |                                    |
   layer 1: valid keys                  layer 2: complete dictionaries
   declare module 'i18next' {           locales/ru.ts:
     interface CustomTypeOptions {        import type { en } from './en'
       resources: { translation: typeof en }
     }                                    export const ru: typeof en = { ... }
   }
   t('backups.logical.titel') fails     missing key OR stray key fails tsc -b
```

`const ru: typeof en` is an exact structural check including nesting: it rejects a missing key and a leftover one alike. `en` is declared without `as const`, so values widen to `string` and translations are not forced to match the English text.

The `import type` link from each translation to `en` is erased at build time, so the English dictionary does not end up in the Russian bundle.

Consequence worth naming: a language cannot be added partially. Either its dictionary is complete or the build is red. That is why the four remaining languages come last, after the key set has settled.

### Keys are domain-scoped, not path-scoped

`t('backups.logical.emptyState.title')`, not `t('features.backups.logical.ui.EditLogicalBackupConfigComponent.emptyState.title')`.

Top-level namespaces follow the domain, not the file tree: `backups`, `databases`, `storages`, `notifiers`, `verification`, `workspaces`, `users`, `settings`, `status`, `errors`, `common`. A path-shaped key would look tidy today and rot at the first component move, and FSD already permits a component to migrate between `pages/`, `features/` and `entity/` as consumers appear.

### Status labels become typed lookup tables

Today the same five words are copy-pasted with different markup:

```
features/verification/runs/ui/VerificationDetailDrawer.tsx:51   renderStatusTag()  -> <Tag color="green">Successful</Tag>
features/verification/runs/ui/VerificationsComponent.tsx:58     renderStatus()     -> <span>Successful</span>
```

Seven enums are affected: `VerificationStatus`, `RestoreStatus`, `LogicalBackupStatus`, `PhysicalBackupStatus`, `RestoreVerificationStatus`, `HealthStatus`, `AddMemberStatusEnum`.

The label lookup lives in `entity/*/model/` in its own file next to the enum (`VerificationStatusLabelKeys.ts` beside `VerificationStatus.ts`), re-exported from the slice's `index.ts` so consumers reach it through the slice's public API. FSD MUST rule 5 governs types rather than constants, so co-locating the `Record` inside the enum's own file would also be legal; naming the choice here keeps the seven slices from diverging.

It is an explicit `Record`, not a template key:

```ts
// no: a dynamic key defeats the type check entirely
t(`status.verification.${status}`)

// yes: adding an enum member leaves the Record incomplete and fails the build
const verificationStatusLabelKeys: Record<VerificationStatus, TranslationKey> = { ... }
```

Components keep their own markup (a tag here, an icon and span there) and take the words from one place. This deduplicates as a side effect; that is a bonus, not the goal.

### The lint rule is what makes the migration finite

`eslint-plugin-i18next`, rule `i18next/no-literal-string`. The already-installed `react/jsx-no-literals` is not enough: it sees JSX text nodes and misses `placeholder="..."`, `okText="..."` and `alert('...')`, which together are more than half the strings.

Enabled per directory, appended to the flat config as each slice lands:

```js
{
  files: ['src/features/settings/**/*.tsx'],
  plugins: { i18next },
  rules: { 'i18next/no-literal-string': 'error' },
},
```

Exclusions are needed for 51 sites containing `pg_dump`, `pg_basebackup`, `mongodump`, `localhost` or `https://`, 27 sites with code and command blocks, and AntD props that look like prose but are not (`color`, `type`, `size`, `variant`). Configured exclusions cover the recurring ones; one-off cases take an inline suppression with a stated reason, which keeps the config from turning into a dumping ground. The exact option names are version-specific and get pinned when the plugin is installed.

Slices are migrated smallest first, so the infrastructure is debugged on `features/settings` (4 strings) rather than `features/databases` (159).

### The language control is merged with the theme control

The website's selector is the reference: a compact button showing the current language, a dropdown listing all six by native name, opening on `onMouseEnter` and closing on `onMouseLeave`, with the panel using top padding rather than a margin so the hover area between button and list stays contiguous. That detail is not cosmetic; a gap makes the dropdown close while the pointer crosses it.

Rather than adding a second standalone button, the language selector and the theme selector become one paired control, split down the middle, the way the website glues its selector to the GitHub button. Two reasons: the navbar has no room for a third control at narrow widths, and the two settings are the same kind of thing (a per-browser display preference), so grouping them is honest rather than decorative.

The paired control replaces `ThemeToggleComponent` at its three existing call sites, which is what satisfies the spec's placement requirement without a separate placement task.

Alternatives considered:

- **A separate language button beside the theme button.** Simpler to build, but adds a third control to a navbar that already competes for width, and the website has already established the paired shape.
- **Language inside the theme dropdown as a submenu.** Fewer pixels, but buries a setting a first-time visitor needs to find immediately, and nested dropdowns on hover are fragile.

Trigger consistency is an open question below.

### Three surfaces that are not React state

Three things follow the language but sit outside the component tree, and each needs wiring the dictionary alone does not give:

- `index.html` hardcodes `<html lang="en">` and an English `<title>`. Both are set from the provider once the locale resolves. The `lang` attribute is what screen readers switch voices on and what stops the browser offering to translate a page already in the reader's language; the title is what a bookmark keeps.
- Eight `toLocaleString()` calls carry no locale argument, so they follow the browser rather than the selection. On an English browser with Russian selected, dates would read Russian and numbers English on the same screen. They are in `LogicalBackupsComponent.tsx:513,515`, `PhysicalBackupsComponent.tsx:49,52`, `VerificationDetailDrawer.tsx:45,48,157` and `StarButtonComponent.tsx:56`. The formatter moves to `shared/i18n` and takes the selected locale.
- The unit words stay English (`MB`, `GB`), only the digits regroup.

Two decisions this raises:

`index.html` is a static Vite entry, so its `lang` and `<title>` can only follow the language at runtime. They are set from `LocaleProvider` rather than from a separate `useDocumentLanguage` hook: the provider already owns the resolved locale and runs before first paint, and a second hook would need a consumer in the tree to fire at all, which is exactly the coupling the provider exists to avoid. The markup keeps English values as the pre-hydration fallback.

Chinese does not follow the website's number rule, and the spec deliberately says so. `website/AGENTS.md` fixes `zh 180 万` for millions, qualified as the form for prose. `(1800000).toLocaleString('zh')` returns `1,800,000`; the 万 form needs `Intl.NumberFormat` with `notation: 'compact'`. A row count in a table cell is not prose, and a compact form loses precision a size or a count needs, so the interface shows full grouped numerals in every language and reserves 万 for the website's marketing copy. Verified: `en 1,800,000`, `ru 1 800 000`, `fr 1 800 000`, `es/pt 1.800.000`, `zh 1,800,000`.

### Two axes: words follow the language, clocks follow the region

The product ships one control, and it selects a language. Formatting conventions are a second, independent axis, and conflating them silently is how a plan like this goes wrong. The split adopted here:

- Words follow the selected language. Labels, month and weekday names, component-library text, status labels.
- Digit grouping follows the selected language too. A user who picks French and reads `1,234.5` will file it as a bug, and the product gives them no other control to fix it. This is a deliberate simplification, not a claim that grouping is linguistic.
- Clock format and date order keep following the browser. `shared/time/getUserTimeFormat.ts:3,43` and `shared/time/utils.ts:5` derive 12-hour versus 24-hour and the day/month order from `navigator.language`, and they stay that way. Language does not determine these: English speakers use 12-hour in the United States and 24-hour in the United Kingdom, so deriving a clock from a language selection would be guessing. The browser reports the user's actual region, which is the better signal.

The residual oddity is that a French selection can show French month names above 12-hour AM/PM times. That is correct rather than inconsistent: the words are in the reader's language and the clock is in their region's convention.

Durations (`formatDuration`, `2h 15m 3s`) need no work under this split. They are technical readouts whose digits never reach a grouping threshold, and their unit letters stay English along with the other unit words.

### Locale persistence mirrors the theme

`localStorage` under `databasus-locale`. No user profile field: the backend has no notion of a user language, and adding one only becomes necessary when notifier emails need translating, which is outside this change.

Reading the stored value during provider initialization (as `ThemeProvider` does with `getStoredTheme`) is what satisfies the "no flash of English" scenario in the spec.

Initial detection reads the browser, not the operating system: a page cannot observe the system setting. What it has is `navigator.languages`, the browser's ordered preference list, which usually but not always follows the system.

The rule: walk `navigator.languages` in order, take the first entry whose primary subtag matches a supported language, fall back to English. Walking the list matters. A user whose preferences are `["de", "ru"]` genuinely prefers Russian over English, and checking only `navigator.language` would hand them English.

Region subtags are stripped, which forces two mappings that should be stated rather than discovered:

- `zh-TW` and `zh-HK` resolve to the Simplified Chinese dictionary, though their users read Traditional. Simplified is what the website ships and what this change ships.
- `pt-PT` resolves to the Brazilian Portuguese dictionary, matching `website/AGENTS.md`, which fixes Portuguese as pt-BR.

Both are the least-bad option while only one variant exists per language, and both are a reason to add a variant later rather than a defect to fix now. `website/app/i18n.ts` already carries the same kind of mapping in `HTML_LANG_CODES` (`zh: "zh-CN"`).

### Backend errors: fix the frontend half of the contract now

The backend returns roughly 416 free-text English messages, surfaced by 20+ `alert(e.message)` calls. Translating them is out of scope, but the frontend's handling is not: it standardizes on translating by `ApiError.code` and falling back to `message`. The backend can then acquire codes incrementally without any further frontend rework. `ConnectionErrorCode` in `backend/internal/features/databases/databases/postgresql/shared/connection_error.go` is the existing precedent on the backend side.

## Risks / Trade-offs

- **The migration stalls half-finished.** → The per-directory lint rule makes progress visible and irreversible, and each slice is an independently shippable commit. The interface is bilingual in the meantime, which is worse than English but better than a long-lived branch.
- **Text expansion breaks layouts.** Russian and French run 20-30% longer than English, and AntD forms with fixed `labelCol` will wrap or clip. → Visual checks are done in Russian, not English. Russian is migrated alongside English from the first slice precisely so this surfaces immediately rather than at step 4.
- **Chinese is shorter and denser**, which hides different problems (line height, mixed CJK and Latin spacing). → Spot-check Chinese on the table-heavy screens once its dictionary lands.
- **Existing tests assert English text.** Twelve test files exist; some assert on user-facing copy. → Assert on keys or on stable identifiers rather than rendered prose, changed per slice as it is migrated.
- **The exclusion list for technical strings grows into a loophole.** → Prefer inline suppression with a reason for one-offs; only recurring patterns go in the config.
- **Six dictionaries in one bundle.** At ~1,500 strings each the total is meaningful but not alarming. → Ship all six initially and measure; dynamic import per language is a contained follow-up if the number justifies it.
- **The type-level completeness check makes adding a language all-or-nothing.** → Accepted deliberately. A partially translated language is worse than an absent one, and this is the mechanism that guarantees it cannot ship.

## Migration Plan

No data migration and no deployment coupling: this is a frontend build. Rollback for any slice is reverting its commit, which also reverts its lint config entry.

Sequencing is by slice, smallest first, English and Russian together:

```
settings(4)  -> databases(159) -> restores(13) -> healthcheck(19) -> workspaces(34)
  -> users(48) -> verification(60) -> notifiers(102) -> backups(119) -> storages(136)
  -> entity/, widgets/, pages/, shared/ui/ (~136 strings)
```

Counts are the crude-grep numbers and understate the real ones by roughly half; the ordering holds regardless.

`databases` is second rather than last on purpose. Strict smallest-first validates the key namespace only on trivial slices, so a design flaw surfacing in the largest one would mean reworking every slice already migrated. Doing it right after the infrastructure is proven costs fluency and buys the ability to revise the key layout while only one other slice depends on it.

Spanish, Portuguese, Chinese and French are written only after the key set has stopped moving. Translating earlier means redoing the work each time a key is renamed.

## Open Questions

- Which AntD locale bundles to import statically versus dynamically. Affects bundle size only, decidable once real numbers exist.
- Whether the paired control should open both halves on hover, or keep the theme half on click as it is today. The website opens the language list on hover; `ThemeToggleComponent` uses an AntD `Dropdown` with `trigger={['click']}`. Two different triggers inside one visually joined button is odd, but changing the theme control's behaviour is scope the user has not asked for. Default: the language half opens on hover, the theme half keeps click, revisit after seeing it built.
- Whether `common.*` should hold shared verbs (`Save`, `Cancel`, `Delete`) or whether each domain repeats them. Sharing risks a verb that reads correctly in one context and wrong in another, in languages with grammatical gender or aspect. Resolvable during the first two slices, with the default being: share only where every language confirms the wording carries over.
