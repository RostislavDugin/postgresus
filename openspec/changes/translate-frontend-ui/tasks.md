## 1. i18n infrastructure

- [ ] 1.1 Add `i18next`, `react-i18next` and `eslint-plugin-i18next` to `frontend/package.json` and verify `pnpm install` succeeds and `pnpm build` still passes
- [ ] 1.2 Create `shared/i18n/locales/en.ts` with the top-level namespaces (`common`, `status`, `errors`, plus one per domain) and verify `tsc -b` passes
- [ ] 1.3 Add the `i18next` module augmentation typing `resources` as `typeof en`, and verify that `t('does.not.exist')` is a type error while a real key is not
- [ ] 1.4 Create `shared/i18n/locales/ru.ts` typed `const ru: typeof en`, and verify that deleting one key from it makes `tsc -b` fail with that key named
- [ ] 1.5 Configure the i18next instance with `returnNull: false`, `nsSeparator: false`, `interpolation.escapeValue: false` and `fallbackLng: 'en'`, and verify `t()` is typed `string` at call sites
- [ ] 1.6 Add `LocaleProvider` and `useLocale` in `shared/i18n`, mirroring `shared/theme/ThemeProvider.tsx`, reading `databasus-locale` from `localStorage` during initialization and falling back to `navigator.language` matched against the six supported languages, then English, and verify a stored locale is applied on first paint with no flash of English
- [ ] 1.7 Wrap the tree in `LocaleProvider` in `App.tsx` and pass the matching AntD locale to `ConfigProvider`, and verify table pagination and an empty table render in Russian when Russian is selected
- [ ] 1.8 Wire `dayjs.locale()` to follow the selected locale and verify month and weekday names in a `DatePicker` change with the language
- [ ] 1.8a Set `document.documentElement.lang` and the document title from the selected locale, replacing the hardcoded `lang="en"` and English `<title>` in `frontend/index.html`, and verify the tab title and the reported language both change with the selection
- [ ] 1.8b Add a locale-aware number formatter in `shared/i18n` and verify with a unit test that an English browser with French selected groups digits the French way, that Chinese groups Western-style rather than in the 万 form, and that unit words are not its concern
- [ ] 1.8c Confirm that `shared/time/getUserTimeFormat.ts:3,43` and `shared/time/utils.ts:5` keep deriving clock format and date order from `navigator.language`, and record in `frontend/AGENTS.md` why those three reads are correct where a bare `toLocaleString()` is not, so a later reader does not "fix" them
- [ ] 1.9 Build the paired language and theme control in `shared/ui`, with the language half showing the current language and opening its native-name list on hover, using contiguous hover padding rather than a margin so crossing from button to list does not close it, and verify the list stays open while the pointer moves onto it
- [ ] 1.10 Replace `ThemeToggleComponent` with the paired control at all three call sites (`widgets/main/MainScreenComponent.tsx:254`, `widgets/main/SidebarComponent.tsx:132`, `features/users/ui/AuthNavbarComponent.tsx:49`) and verify the control renders in the navbar, in the narrow-viewport sidebar and on the authentication screen
- [ ] 1.11 Verify end to end that a language chosen before signing in survives the sign-in, that a first visit on a Spanish system starts in Spanish, that an unsupported system language starts in English, and that an explicit choice outranks the system language on later visits

## 2. Lint enforcement

- [ ] 2.1 Add an `i18next/no-literal-string` config object to `frontend/eslint.config.js` scoped to `src/features/settings/**` with exclusions for technical strings and non-prose AntD props, and verify `pnpm lint` passes on the migrated slice and fails when a literal label is reintroduced
- [x] 2.2 Amend the "Language in code" section of the root `AGENTS.md` to permit target-language text in `frontend/src/shared/i18n/locales/<locale>.ts` and nowhere else under `frontend/src/`, and verify the amended wording still forbids translated fallback copy and error messages outside that directory. Do not start section 5 until this task is done and approved
- [ ] 2.3 Document the i18n convention in `frontend/AGENTS.md`: dictionary keys are domain-scoped, English is the key source of truth, technical strings stay literal, and each migrated directory gets a lint config entry

## 3. Error handling contract

- [ ] 3.1 Add an `errors` namespace keyed by backend error code and a `translateApiError` function that resolves an `ApiError` by `code`, falling back to `message` when the code is absent or unknown, and verify all three branches with a unit test
- [ ] 3.2 Route the existing `physicalConnectionErrorContent` lookup in `EditPostgreSqlPhysicalSpecificDataComponent.tsx` through that helper and verify the physical connection error paths still show their specific copy

## 4. Status labels

- [ ] 4.1 Add a `status` namespace and a typed `Record<Enum, TranslationKey>` beside each of the seven status enums in `entity/*/model/`, and verify that adding an enum member without a label fails `tsc -b`
- [ ] 4.2 Replace the duplicated `renderStatusTag` in `VerificationDetailDrawer.tsx` and `renderStatus` in `VerificationsComponent.tsx` with lookups against the shared table, keeping each component's own markup, and verify both screens render the same labels

## 5. Slice migration (English and Russian together)

Ordering is deliberate and not by size alone. `features/settings` comes first to prove the infrastructure on four strings. `features/databases` comes second, out of size order, because it is the largest and most varied slice: if the key namespace design breaks anywhere it breaks there, and finding that out after nine migrated slices would mean reworking all of them. The cost is doing the hardest slice before the pattern is fluent. The rest then run ascending.

- [ ] 5.1 Migrate `features/settings` and enable the lint rule for it, and verify `pnpm lint`, `pnpm test` and `tsc -b` pass
- [ ] 5.2 Migrate `features/databases`, the largest slice, and enable the lint rule for it, then review whether the namespace design survived it and revise the key layout before continuing if it did not, and verify the same three commands pass
- [ ] 5.3 Migrate `features/restores` and enable the lint rule for it, and verify the same three commands pass
- [ ] 5.4 Migrate `features/healthcheck` and enable the lint rule for it, and verify the same three commands pass
- [ ] 5.5 Migrate `features/workspaces` and enable the lint rule for it, and verify the same three commands pass
- [ ] 5.6 Migrate `features/users` and enable the lint rule for it, and verify the same three commands pass
- [ ] 5.7 Migrate `features/verification` and enable the lint rule for it, and verify the same three commands pass
- [ ] 5.8 Migrate `features/notifiers` and enable the lint rule for it, and verify the same three commands pass
- [ ] 5.9 Migrate `features/backups` and enable the lint rule for it, and verify the same three commands pass
- [ ] 5.10 Migrate `features/storages` and enable the lint rule for it, and verify the same three commands pass
- [ ] 5.10a Replace the bare `toLocaleString()` calls with the formatter from 1.8b as each owning slice is migrated (`LogicalBackupsComponent.tsx:513,515` and `PhysicalBackupsComponent.tsx:49,52` with `features/backups`, `VerificationDetailDrawer.tsx:45,48,157` with `features/verification`, `StarButtonComponent.tsx:56` with `shared/ui/`), rather than in a separate pass over the same files, and verify no bare `toLocaleString()` survives
- [ ] 5.11 Migrate the remaining copy in `entity/`, `widgets/`, `pages/` and `shared/ui/`, including `ConnectionStringParser` error text and `getNotifierNameFromType`, and enable the lint rule for those directories, and verify the same three commands pass
- [ ] 5.12 Update frontend tests that assert on rendered English prose to assert on keys or stable identifiers instead, and verify `pnpm test` passes

## 6. Remaining languages

- [ ] 6.1 Confirm the key set has stopped changing after section 5, then write `es.ts`, and verify `tsc -b` passes with no missing or stray keys
- [ ] 6.2 Write `pt.ts` (Brazilian Portuguese, matching the website's pt-BR conventions) and verify `tsc -b` passes
- [ ] 6.3 Write `fr.ts` and verify `tsc -b` passes
- [ ] 6.4 Write `zh.ts` (Simplified, matching the website's conventions) and verify `tsc -b` passes
- [ ] 6.5 Add each new locale to the switcher, the AntD locale map and the dayjs locale map, and verify each language renders end to end

## 7. Verification

- [ ] 7.1 Walk every screen in Russian and French looking for clipped or wrapped labels in AntD forms with fixed `labelCol`, and fix the layouts that break
- [ ] 7.2 Spot-check Chinese on the table-heavy screens (backups, databases, verification runs) for line height and CJK-Latin spacing
- [ ] 7.3 Confirm in every language that command blocks, connection strings, cron expressions, engine names and the unit words in duration and size readouts are byte-identical to English, while the digits inside those readouts follow the selected language
- [ ] 7.4 Confirm in every language that the browser tab title and the document language attribute match the selection, and that no bare `toLocaleString()` remains
- [ ] 7.5 Run `pnpm lint`, `pnpm format`, `pnpm test` and `pnpm build` from `frontend/` and confirm all pass
