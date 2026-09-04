## Purpose

Lets a user read the Databasus interface in their own language, and gives the project a way to keep every language complete as the product changes.

## ADDED Requirements

### Requirement: The interface renders in the selected language

The frontend SHALL render all user-facing copy in one of six languages: English, Russian, Spanish, Portuguese, Chinese and French. Copy includes labels, placeholders, button text, headings, table columns, empty states, confirmation dialogs, tooltips and validation messages.

Technical strings are excluded and stay identical in every language: command lines and code blocks, connection strings, cron expressions, database engine and tool names, file paths, and duration readouts such as `2h 15m 3s` in full, and the unit words in size readouts such as the `GB` in `1.4 GB`. The digits in a size or a count are not excluded; see the number formatting requirement below.

#### Scenario: A user switches language

- **WHEN** a user selects Russian
- **THEN** every visible label, button and message re-renders in Russian without a page reload
- **AND** command blocks, connection strings and engine names remain byte-identical to what English showed

### Requirement: The language control sits beside the theme control

The language control SHALL be presented together with the light/dark theme control as a single paired element, and SHALL appear everywhere that element appears: the main navigation bar, the narrow-viewport sidebar, and the authentication screen. A user SHALL be able to choose a language before signing in.

The control SHALL show which language is currently active, and SHALL list all six by their native names rather than by language code or flag.

#### Scenario: Choosing a language before signing in

- **WHEN** an unauthenticated user opens the authentication screen
- **THEN** the language control is visible there
- **AND** selecting a language translates the authentication screen immediately and keeps that language after signing in

#### Scenario: Opening the language list

- **WHEN** a user moves the pointer onto the language control
- **THEN** the list of six languages opens without requiring a click
- **AND** moving the pointer from the control to the list does not close it

#### Scenario: Narrow viewport

- **WHEN** the viewport is too narrow for the main navigation bar
- **THEN** the paired language and theme control remains reachable in the sidebar

### Requirement: The language choice defaults to the browser's preference and persists

On a first visit with no stored choice, the frontend SHALL take the browser's ordered list of preferred languages and use the first entry that matches a supported language, ignoring any region subtag, and English when none matches. Once a user makes a choice, that choice SHALL be stored in the browser and applied on subsequent visits.

#### Scenario: First visit with Spanish preferred

- **WHEN** a user whose browser reports Spanish as its preferred language opens the application for the first time
- **THEN** the interface renders in Spanish

#### Scenario: The first preference is unsupported

- **WHEN** a user whose browser reports German first and Russian second opens the application for the first time
- **THEN** the interface renders in Russian rather than English, because Russian is the highest-ranked supported preference

#### Scenario: No preference is supported

- **WHEN** a user whose browser reports no supported language opens the application for the first time
- **THEN** the interface renders in English

#### Scenario: A regional variant with no dictionary of its own

- **WHEN** a user whose browser reports Taiwanese Chinese or European Portuguese opens the application for the first time
- **THEN** the interface renders in the single Chinese or Portuguese variant the product ships, rather than falling through to English

#### Scenario: An explicit choice outranks the browser preference

- **WHEN** a user whose browser prefers Spanish selects English
- **THEN** the interface renders in English on this and every later visit from the same browser

#### Scenario: The choice survives a reload

- **WHEN** a user has selected a language and then reloads the page or returns in a new session on the same browser
- **THEN** the interface renders in the previously selected language
- **AND** the first paint is already in that language, with no visible flash of English

### Requirement: Library and framework text follows the selected language

Text produced by the component library and by date formatting SHALL follow the selected language, not the browser default and not English.

#### Scenario: Component library strings

- **WHEN** the interface renders in a non-English language
- **THEN** table pagination, empty-table placeholders, date pickers, select dropdowns and confirmation popovers display their built-in text in that language

#### Scenario: Dates and times

- **WHEN** the interface renders in a non-English language
- **THEN** month names, weekday names and relative times render in that language

### Requirement: Numbers are grouped by the selected language

Numeric values displayed to the user SHALL be grouped and separated according to the selected language rather than the browser's own locale, using each language's standard digit grouping. Unit words attached to a number stay English. Abbreviated forms that read as prose, such as the Chinese 万 form the website uses for millions in running text, are not used in the interface: counts and sizes appear in full.

#### Scenario: A browser locale that differs from the selected language

- **WHEN** a user whose browser is English selects French, and a screen shows a row count of one million eight hundred thousand
- **THEN** the number is grouped in the French convention rather than the English one
- **AND** it does not disagree with the dates on the same screen, which already follow the selected language

#### Scenario: Clock and calendar conventions are regional, not linguistic

- **WHEN** a user selects a language whose speakers commonly use a 24-hour clock, while their browser reports a region that uses a 12-hour clock
- **THEN** times continue to display in the 12-hour form their browser reports
- **AND** the words around those times, including month and weekday names, are still in the selected language

#### Scenario: A size readout

- **WHEN** a backup size is displayed in a language that groups digits differently from English
- **THEN** the digits are grouped for that language
- **AND** the unit remains `MB` or `GB`

### Requirement: The document reports the selected language

The page SHALL declare the selected language to the browser, and its title SHALL be translated.

#### Scenario: Assistive technology and browser translation

- **WHEN** the interface renders in a given language
- **THEN** the document's language attribute reports that language
- **AND** the browser does not offer to translate a page that is already in the reader's language

#### Scenario: The browser tab

- **WHEN** the interface renders in a non-English language
- **THEN** the browser tab and any bookmark made from it show a title in that language

### Requirement: Status labels are translated while wire values stay stable

Status values exchanged with the backend SHALL remain unchanged English identifiers. Their displayed labels SHALL be translated.

#### Scenario: A backup status is displayed

- **WHEN** the backend reports a backup status of `COMPLETED` and the interface renders in Spanish
- **THEN** the user sees the Spanish label for a completed backup
- **AND** the value sent back to the backend in any subsequent request is still `COMPLETED`

#### Scenario: A new status value is introduced

- **WHEN** a new value is added to a status enum and no label is supplied for it
- **THEN** the type check fails

### Requirement: Dictionary completeness is enforced by the type checker

Each non-English dictionary SHALL be checked against the English dictionary at build time. English is the source of truth for the set of keys; the wording of each language is written independently and is not a literal rendering of the English text.

#### Scenario: A key is missing from a translation

- **WHEN** a key exists in the English dictionary and is absent from any other language
- **THEN** the type check fails and names the missing key

#### Scenario: A key is left over in a translation

- **WHEN** a key is removed from the English dictionary but remains in another language
- **THEN** the type check fails and names the stray key

#### Scenario: A lookup uses an unknown key

- **WHEN** code requests a key that does not exist in the English dictionary
- **THEN** the type check fails

### Requirement: Migrated code cannot regress to hardcoded strings

Once a directory has been migrated, the lint job SHALL reject a user-facing string literal added to it. Directories not yet migrated are exempt until they are.

#### Scenario: A hardcoded string is added to migrated code

- **WHEN** a contributor adds a literal label, placeholder or message to a migrated directory
- **THEN** the lint job fails and points at the literal

#### Scenario: A technical string is added to migrated code

- **WHEN** a contributor adds a command block, connection string or engine name to a migrated directory
- **THEN** the lint job accepts it, either through the configured exclusions or through an inline suppression carrying a reason

### Requirement: Backend errors are translated by code, with the message as fallback

The frontend SHALL translate an error response by its stable error code when the backend supplies one, and SHALL display the backend's message text only when no code is present. This change does not add codes to the backend.

#### Scenario: The backend supplies a known error code

- **WHEN** a request fails with an error code the frontend has a translation for
- **THEN** the user sees the translated message in the selected language

#### Scenario: The backend supplies no code

- **WHEN** a request fails with only a free-text message
- **THEN** the user sees that message as-is, in a container whose surrounding copy is translated

#### Scenario: The backend supplies an unrecognized code

- **WHEN** a request fails with an error code the frontend has no translation for
- **THEN** the user sees the backend's message text rather than a raw code or a blank message
