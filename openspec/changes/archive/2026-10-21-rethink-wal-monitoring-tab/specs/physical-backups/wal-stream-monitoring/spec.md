## ADDED Requirements

### Requirement: The statistics state what the backups can restore

The statistics SHALL state the latest moment to which the database's backups can restore it at the time of the request. That moment SHALL be the one a restore to the latest available point reaches. It starts from the newest completed backup of the newest chain, full or incremental, and extends through the WAL that follows that backup without a gap. With WAL after that backup, it is the time Databasus received the last segment before the first gap after that backup, or the newest segment when there is no such gap. Without WAL after that backup, it is the backup's completion time. When no full backup has completed, the statistics SHALL state that no moment can be restored yet.

The statistics SHALL also state the stretches of time the newest chain can restore to, oldest first. Each completed backup of that chain, full or incremental, opens a stretch from its completion time to the moment the WAL after it reaches without a gap, by the same rule as above. Stretches that overlap or touch SHALL be merged. A WAL segment that is still being uploaded SHALL end a stretch like a gap. The last stretch SHALL end at the latest restorable moment. Without a completed full backup, the list SHALL be empty.

#### Scenario: Streaming without gaps

- **WHEN** the newest full backup completed yesterday and every WAL segment since then is archived without a gap
- **THEN** the latest restorable moment is the time Databasus received the newest archived segment

#### Scenario: Gap in the newest chain

- **WHEN** the slot was rebuilt at 07:51, which left a WAL gap after the segment received at 07:50, no backup completed after the gap, and segments after the gap keep arriving
- **THEN** the latest restorable moment is 07:50
- **AND** a restore to the latest available point reaches the same moment

#### Scenario: Incremental backup after the gap

- **WHEN** the newest chain has a WAL gap at 07:51, an incremental backup of that chain completed at 09:00, and every segment after 09:00 is archived without a gap
- **THEN** the latest restorable moment is the time Databasus received the newest archived segment
- **AND** the restorable stretches are two: from the full backup's completion to 07:50, and from 09:00 to the newest archived segment

#### Scenario: Incremental backup before the gap

- **WHEN** an incremental backup of the newest chain completed at 06:00, and the chain has a WAL gap at 07:51 with segments after it
- **THEN** the latest restorable moment is the time Databasus received the last segment before the gap
- **AND** the restorable stretches are one, from the full backup's completion to that moment

#### Scenario: Newer full backup

- **WHEN** an older chain has a gap, and a newer full backup completed after it
- **THEN** the restorable stretches start at the newer full backup's completion

#### Scenario: Full backup without WAL after it

- **WHEN** the only completed full backup finished at 14:02 and no WAL segment followed it
- **THEN** the latest restorable moment is 14:02

#### Scenario: No full backup yet

- **WHEN** no full backup of the database has completed
- **THEN** the statistics state that no moment can be restored yet
- **AND** the restorable stretches are empty

### Requirement: The Monitoring tab says to which moment the database can be restored

The verdict banner SHALL say, in one sentence, to which moment the database can be restored:

- when a moment is known and the WAL chain broken reason is listed: until a new full backup finishes, the database can be restored only up to that moment;
- when a moment is known, that reason is not listed and the newest restorable stretch covers the whole history: the database can be restored to any moment up to that moment;
- when a moment is known, that reason is not listed and an earlier stretch cannot be restored: the database can be restored to any moment from the start of the newest restorable stretch up to that moment, so the sentence never promises the moments the timeline marks as lost;
- when no moment can be restored yet and a full backup is requested or running: nothing can be restored until the first full backup finishes, which is on its way;
- when no moment can be restored yet and no full backup is requested or running: nothing can be restored yet, because no full backup has finished.

The moment SHALL be shown as a local time for today and as a local date and time otherwise, followed by how long ago it was, rounded for reading. When that age cannot be computed, the sentence SHALL leave it out instead of showing an invalid number. The sentence SHALL appear for every verdict, collecting data included, and SHALL keep the last known moment while the current state cannot be confirmed.

#### Scenario: Healthy database

- **WHEN** the verdict is healthy and the latest restorable moment was 30 seconds ago
- **THEN** the banner says that the database can be restored to any moment up to that time, 30 seconds ago

#### Scenario: Broken chain

- **WHEN** the WAL chain broken reason is listed, the latest restorable moment is 07:50 today and the newest segment was archived at 07:58
- **THEN** the banner says that until a new full backup finishes, the database can be restored only up to 07:50

#### Scenario: Broken chain repaired by an incremental backup

- **WHEN** an incremental backup after the gap brought the latest restorable moment up to the newest archived segment
- **THEN** the WAL chain broken reason is not listed
- **AND** the banner says that the database can be restored to any moment up to that moment

#### Scenario: Old gap before the newest restorable stretch

- **WHEN** the database cannot be restored to moments between Oct 10 18:35 and Oct 20 07:51, and can be restored to every moment after that up to 08:32 today
- **THEN** the banner says that the database can be restored to any moment from Oct 20 07:51 up to 08:32

#### Scenario: Just enabled

- **WHEN** the verdict is collecting data and no full backup has completed or is requested
- **THEN** the banner says that nothing can be restored yet because no full backup has finished

#### Scenario: Saving stuck

- **WHEN** no new changes have reached storage for 40 minutes while the database keeps writing
- **THEN** the restore sentence names a moment about 40 minutes ago

#### Scenario: Before the first full backup

- **WHEN** no full backup has completed and none is requested or running
- **THEN** the banner says that nothing can be restored yet because no full backup has finished

### Requirement: The Monitoring tab offers to start a full backup when that is the fix

When a full backup is what repairs the situation, the verdict banner SHALL offer a button that starts one. That is the case while all of these hold:

- the verdict is healthy, needs attention or broken, and the current state is confirmed;
- the WAL chain broken reason or the slot missing reason is listed, or no full backup has completed;
- no full backup is requested or running.

The not streaming verdict SHALL never offer the button, because Databasus takes no backups while streaming is off. The collecting data verdict SHALL not offer it either: it lasts at most two minutes, and the restore sentence already says whether a first full backup exists.

The button SHALL request a full backup at once, the same way the Backups tab does. While the request runs, the button SHALL show that it is working and SHALL not accept another click. After the request succeeds, the tab SHALL refresh at once, so the banner says that a full backup is on its way without waiting for the next scheduled refresh. A failed request SHALL show the error and leave the button available.

#### Scenario: Broken chain fixed from the tab

- **WHEN** a user who may manage the database sees the WAL chain broken reason with no full backup pending, and presses the button
- **THEN** a full backup is requested
- **AND** the banner says that a full backup is on its way

#### Scenario: Full backup already pending

- **WHEN** the WAL chain broken reason is listed and a full backup is requested or running
- **THEN** the banner offers no button
- **AND** says that a new full backup is on its way

#### Scenario: Request refused

- **WHEN** the full backup request fails
- **THEN** the tab shows the error
- **AND** the button can be pressed again

#### Scenario: Missing slot

- **WHEN** a user who may manage the database sees the slot missing reason with no full backup pending
- **THEN** the banner offers the full backup button

#### Scenario: Healthy streaming before the first full backup

- **WHEN** the verdict is healthy, no full backup has completed and none is requested or running
- **THEN** the banner offers the full backup button

#### Scenario: Streaming off

- **WHEN** the verdict is not streaming because backups are disabled, and no full backup has completed
- **THEN** the banner offers no full backup button

### Requirement: Only users who may manage the database can start a full backup from the Monitoring tab

The full backup button SHALL be shown only to users who may manage the database's backups. A user who can only view the database SHALL see the problem and the sentence about what to do, without the button. A full backup request from such a user SHALL be refused by the API, whatever the tab shows.

#### Scenario: Read-only member

- **WHEN** a workspace member who can view but not manage the database opens the tab while the chain is broken
- **THEN** the banner shows the problem and what to do
- **AND** offers no full backup button

#### Scenario: Request without permission

- **WHEN** a user who cannot manage the database requests a full backup through the API
- **THEN** the request is refused
- **AND** no full backup is requested

### Requirement: The Monitoring tab shows the current flow as four figures

Under the verdict banner, the tab SHALL show four figures from the newest statistics, each as a label with a large value:

- when the last change was saved to storage, as a relative time, or that nothing is saved yet;
- how much is waiting to be saved to storage;
- how fast the database writes changes;
- how fast Databasus saves them.

The two rates SHALL use the same names and colors as the lines of the main chart, so a reader can match a figure to its line, and SHALL say that they are averages over the last 10 minutes, so a rate next to an empty queue does not read as a contradiction.

The figures SHALL be shown for the healthy, needs attention and broken verdicts, and while the current state cannot be confirmed, and not for the collecting data and not streaming verdicts. A value that is not known SHALL be shown as not measured, not as zero.

#### Scenario: Busy healthy database

- **WHEN** the database writes 10 MB/s, Databasus saves 10 MB/s, 20 MB is not saved yet and the newest segment was archived 15 seconds ago
- **THEN** the figures show 15 seconds ago, 20 MB, 10 MB/s of database writes and 10 MB/s saved to storage

#### Scenario: No write rate yet

- **WHEN** the window of the last 10 minutes holds no sample with a source write value
- **THEN** both rate figures say not measured

### Requirement: The Monitoring tab shows tips and technical detail below the chart

Notices other than the no full backup yet notice SHALL appear as tips: quiet notes, each with its title and two sentences, inside the technical details section under the path of changes. The no full backup yet notice SHALL be shown only through the banner's restore sentence, which appears for every verdict. A tip SHALL never use a warning or error color. The no slot WAL limit tip SHALL explain in plain words what the limit protects and what it costs.

Everything else SHALL sit in one technical details section below the chart, with its own heading and shown without a click: the path of changes, the tips, the chart of what is not saved and not received yet, the buffer chart, and the raw values.

#### Scenario: Default source settings

- **WHEN** the source sets no limit on WAL kept for the slot and nothing else is wrong
- **THEN** the technical details section shows one tip with its text

#### Scenario: Engineer looks for raw values

- **WHEN** a reader scrolls below the main chart
- **THEN** the technical details section shows the path of changes, the two additional charts and the raw values

### Requirement: The Monitoring tab shows which moments can be restored

Under the figures, the tab SHALL show a timeline of the newest chain, from the start of its first restorable stretch to now. The timeline SHALL mark the restorable stretches, the stretches between them that cannot be restored, and the time after the latest restorable moment that is not saved yet, in different colors and patterns. A stretch that cannot be restored SHALL stay visible however short it is compared with the whole timeline. Under the timeline, the tab SHALL name each stretch that cannot be restored, up to three, in a sentence with its start and end, so color is not the only signal. The timeline SHALL not appear when the statistics list no restorable stretch.

#### Scenario: Old gap bridged by an incremental backup

- **WHEN** the newest chain cannot restore to moments between Oct 10 18:35 and Oct 20 07:51, and can restore to every moment after that up to a minute ago
- **THEN** the banner is green
- **AND** the timeline marks the stretch from Oct 10 18:35 to Oct 20 07:51 as not restorable
- **AND** a sentence under it says that the database cannot be restored to moments between those times

#### Scenario: No full backup yet

- **WHEN** no full backup has completed
- **THEN** the tab shows no timeline

### Requirement: The Monitoring tab reads well on wide screens

The tab SHALL stop growing at a maximum width of about 1150 px, so on wide screens the text, the figures and the charts keep the sizes they have on a laptop instead of stretching. The banner's text SHALL keep a readable line length, the figures SHALL sit in cards next to each other, and the charts SHALL keep a fixed height. The path of changes SHALL put its stages in a row only when its own section is wide enough for them, and stack them otherwise.

#### Scenario: 2560 px wide screen

- **WHEN** the tab is shown in a 2560 px wide window
- **THEN** the tab is no wider than in a 1600 px wide window
- **AND** the four figures sit in one row of cards

#### Scenario: Laptop screen

- **WHEN** the tab is shown in a 1240 px wide window next to the sidebar
- **THEN** the stages of the path of changes sit in one row without horizontal scrolling

### Requirement: The API serves the current values every second

A second authenticated endpoint SHALL return the current values of a database's running WAL stream, read from the running streamer at the time of the request and not stored: the time of the reading, the local queue and its pause size, whether receiving runs, the archive progress point, the time the newest WAL segment was archived, and how much is not saved to storage yet. The values from the source database, its current position, the slot's status, the slot lag and the remaining allowance, SHALL come from the newest probe of the source that the streamer already makes, at least every 10 seconds, together with the time of that probe; the endpoint SHALL not open a connection to the source itself. Only users who can view the database SHALL read it. While no streamer runs for the database, the response SHALL say so and carry no values.

#### Scenario: Buffer drains between two samples

- **WHEN** the local queue drains from 300 MB to 0 B within five seconds
- **THEN** a request one second later reports 0 B in the local queue

#### Scenario: Source values follow the probe

- **WHEN** the newest source probe ran 7 seconds ago
- **THEN** the source-side values are those of that probe, with its time

#### Scenario: Saving moves past the probed position

- **WHEN** segments reached storage after the newest source probe, past the source position it found
- **THEN** the response says that nothing is waiting to be saved, not that the amount is unknown

#### Scenario: Streamer not running

- **WHEN** WAL streaming is off for the database
- **THEN** the response says that no streamer runs and carries no values

#### Scenario: Member without access

- **WHEN** a user who cannot view the database requests its current values
- **THEN** the request is refused

## MODIFIED Requirements

### Requirement: The statistics state one health verdict with reasons

The statistics SHALL state one verdict:

- **not streaming**: WAL streaming is not expected to run, because the database is not in WAL streaming mode, its backups are disabled or it has no storage;
- **collecting data**: the streamer is not failing, no sample is newer than the current session's start, and that session started less than 2 minutes ago, or no streamer has started yet;
- **broken**: at least one reason applies at the broken level;
- **needs attention**: at least one reason applies at the needs-attention level and none at the broken level;
- **healthy**: no reason applies.

The first matching verdict in this order wins.

Each reason SHALL carry its code and its level. It SHALL also carry the time the condition began when that time is known, and the measured amount and the threshold it was compared with when the rule compares an amount. A needs-attention reason that is read from samples SHALL appear only when its condition holds in the two newest samples, so a single unusual sample does not change the verdict. Reasons whose rule already waits for a duration are exempt.

| Reason | Needs attention | Broken |
|---|---|---|
| Streaming stopped | - | the streamer has been failing for 1 minute or longer, or the newest sample is older than 2 minutes, and the newest sample did not find the source unreachable |
| Source unreachable | the newest sample found the source unreachable, and the source has been unreachable for 1 to 5 minutes since | the same for 5 minutes or longer, also when streaming stopped after that sample |
| WAL chain broken | - | the newest completed full backup's chain has a WAL gap below the archive progress point that the latest restore cannot cross, because no completed backup of that chain starts after the gap; it clears when a completed incremental backup of that chain or a newer full backup starts after the gap |
| Slot missing | - | the newest sample found the source without the slot |
| Slot lost | - | the newest sample reports the slot's status as `lost` |
| Archive not advancing | the archive progress point has stood still for 10 minutes or longer while the source keeps writing | the same for the archive staleness threshold of 30 minutes or longer, which is when the "archiving fell behind" notification fires |
| Archive slower than source | the unarchived WAL grew by at least the backlog threshold over the last 10 minutes | - |
| Slot keeps extra WAL | the slot's status has been `unreserved` for longer than 5 minutes, or `extended` for longer than 5 minutes while the slot lag grows | - |
| Slot limit approaching | the source sets a limit, and the remaining WAL allowance would run out in under 1 hour if the slot lag keeps growing at its rate of the last 10 minutes | - |
| Near slot rebuild | the slot lag is at least 80% of the configured lag at which Databasus rebuilds the slot, and the slot's status is neither `extended` nor `unreserved` | - |
| Local queue filling | the local queue is at least half the size at which the receiver is paused | - |
| Frequent restarts | receiver restarts and streaming session starts in the last hour add up to 3 or more | - |
| Forced WAL switch refused | the newest sample says the source refused the forced WAL switch | - |

The backlog threshold is 64 MiB or four WAL segments, whichever is larger. Growth of the unarchived WAL SHALL be measured between the lowest value in the first 2.5 minutes of the window and the lowest value in its last 2.5 minutes, so normal upload batches do not count as growth.

When the newest sample is more than 2 minutes old, reasons read from that sample SHALL be dropped, except the source unreachable reason described in the table. The WAL chain broken reason does not depend on samples and stays.

Reasons read from the slot or the archive progress point SHALL be dropped while the newest sample has no source-side values.

The statistics SHALL also list notices, which never change the verdict:

- **no slot WAL limit**: the newest sample was taken with the source reachable, the slot's status is not `lost`, and the source sets no limit on WAL kept for the slot;
- **slot keeps extra WAL steadily**: the slot's status has been `extended` for longer than 5 minutes while the slot lag does not grow;
- **no full backup yet**: no full backup of the database has completed, so archived WAL cannot be restored yet.

#### Scenario: Everything in order

- **WHEN** the source is reachable, archiving keeps pace and no threshold above is crossed
- **THEN** the verdict is healthy and no reason is listed

#### Scenario: Single unusual sample

- **WHEN** only the newest sample shows the local queue above half its pause size
- **THEN** the local queue filling reason is not listed

#### Scenario: Short network blip

- **WHEN** the source stopped answering 30 seconds ago
- **THEN** the source unreachable reason is not listed

#### Scenario: Source unreachable for a few minutes

- **WHEN** the source has been unreachable for 3 minutes and the streamer keeps sampling
- **THEN** the verdict is needs attention with the source unreachable reason and the time it began

#### Scenario: Source down for a while

- **WHEN** the newest sample found the source unreachable, the streamer then stopped, and the source has been unreachable for 6 minutes
- **THEN** the verdict is broken with the source unreachable reason, not the streaming stopped reason

#### Scenario: Archive stuck

- **WHEN** the source keeps writing and the archive progress point has not moved for 12 minutes
- **THEN** the verdict is needs attention with the archive not advancing reason
- **AND** after 30 minutes without movement the same reason is listed at the broken level

#### Scenario: Streamer stopped

- **WHEN** the streamer failed and the newest sample, taken with the source reachable, is ten minutes old
- **THEN** the verdict is broken with the streaming stopped reason
- **AND** the response carries the time of that newest sample
- **AND** no other reason read from that sample is listed

#### Scenario: Restart after a deploy

- **WHEN** Databasus restarted 20 seconds ago after 10 minutes of downtime, and the new session has not taken a sample yet
- **THEN** the verdict is collecting data, not broken

#### Scenario: Crash loop on a new database

- **WHEN** WAL streaming was enabled five minutes ago and every session has failed before its first sample
- **THEN** the verdict is broken with the streaming stopped reason

#### Scenario: Slot rebuilt

- **WHEN** Databasus rebuilt the slot after it was lost and the requested full backup has not completed yet
- **THEN** the verdict is broken with the WAL chain broken reason, even though the new slot is healthy
- **AND** once a newer full backup completes, the reason is no longer listed

#### Scenario: Gap bridged by an incremental backup

- **WHEN** the newest chain has a WAL gap below the archive progress point, an incremental backup of that chain completed after the gap, and every segment after that backup is archived without a gap
- **THEN** the WAL chain broken reason is not listed
- **AND** the latest restorable moment is the time Databasus received the newest archived segment

#### Scenario: Gap after the newest incremental backup

- **WHEN** an incremental backup of the newest chain completed, and a WAL gap below the archive progress point follows it
- **THEN** the verdict is broken with the WAL chain broken reason

#### Scenario: Failed incremental backup after the gap

- **WHEN** the only incremental backup started after the gap failed or is still running
- **THEN** the verdict is broken with the WAL chain broken reason

#### Scenario: Slot removed on the source

- **WHEN** the newest sample found the source without the slot
- **THEN** the verdict is broken with the slot missing reason, not the source unreachable reason

#### Scenario: Just enabled

- **WHEN** WAL streaming was just enabled and no sample exists yet
- **THEN** the verdict is collecting data

#### Scenario: Backups disabled

- **WHEN** a database in WAL streaming mode has its backups disabled
- **THEN** the verdict is not streaming
- **AND** the series still holds the samples recorded before

#### Scenario: Several problems at once

- **WHEN** the local queue has been half full for two samples and the slot is missing
- **THEN** the verdict is broken
- **AND** both reasons are listed, each with its own level

#### Scenario: Busy cluster with steady extra WAL

- **WHEN** the slot's status has been `extended` for an hour while the slot lag stays flat
- **THEN** the slot keeps extra WAL reason is not listed
- **AND** the slot keeps extra WAL steadily notice is listed

#### Scenario: Paused receiver above the rebuild lag

- **WHEN** the receiver is paused for a full local queue and the slot lag is at 90% of the rebuild lag with status `reserved`
- **THEN** the near slot rebuild reason is listed

#### Scenario: No limit on the source

- **WHEN** the source's `max_slot_wal_keep_size` is `-1` and nothing else is wrong
- **THEN** the verdict is healthy
- **AND** the no slot WAL limit notice is listed

#### Scenario: Slot already lost

- **WHEN** the newest sample reports the slot's status as `lost`
- **THEN** the no slot WAL limit notice is not listed

### Requirement: The statistics API is a documented, unit-explicit contract

The statistics SHALL be served by an authenticated API endpoint documented with the rest of the API. Every amount SHALL carry its unit in its name, in bytes, bytes per second, seconds or milliseconds. Every time SHALL be an absolute UTC time. Every verdict, state, reason and notice SHALL be a stable code from a documented list, not display text, so other consumers can rely on it.

A response SHALL hold:

- the health verdict, its reasons and its notices;
- whether a full backup is requested or running;
- the latest restorable moment, or that none exists yet;
- the restorable stretches of the newest chain, each with its start and end;
- the streamer's state;
- the current values from the newest sample, the time of that sample and the time the newest WAL segment was archived;
- the keeping-up summary;
- a chart series for the requested period.

A request for a database that is not a PostgreSQL physical database SHALL be refused as invalid. A PostgreSQL physical database outside WAL streaming mode SHALL get the not streaming verdict and whatever samples it still has.

#### Scenario: Codes, not text

- **WHEN** a client reads a response for a database that needs attention
- **THEN** the verdict, each reason and each notice are codes from the documented lists
- **AND** no field holds a sentence meant for display

#### Scenario: Logical database

- **WHEN** a user requests statistics for a logical database they can view
- **THEN** the request is refused as invalid

#### Scenario: Switched out of WAL streaming

- **WHEN** a physical database switched from WAL streaming to full and incremental mode yesterday
- **THEN** the verdict is not streaming
- **AND** the series holds the samples recorded before the switch

#### Scenario: Restorable moment as UTC time

- **WHEN** a client reads a response for a database with a completed full backup
- **THEN** the latest restorable moment is an absolute UTC time
- **AND** each restorable stretch starts and ends at an absolute UTC time

### Requirement: The Monitoring tab speaks plain language

Every label, headline, sentence, legend and chart title the tab shows by default SHALL avoid the terms "WAL", "slot", "LSN" and "lag". The tab SHALL call the unarchived WAL "not saved to storage yet" and the slot lag "not received by Databasus yet", and SHALL say that the second is part of the first. Technical names SHALL appear only in explanations shown on hover or keyboard focus, in the technical details section, and in a "what to do" sentence that names a setting or permission to change, such as `max_slot_wal_keep_size`.

Each reason and each notice SHALL have one sentence about what happened, with its amounts and, where known, how long it has lasted, and one sentence about what to do. When Databasus resolves a condition by itself, the second sentence SHALL say so, and only then. For a broken chain or a missing slot, the second sentence SHALL ask the user to start a full backup unless one is requested or running. Where the banner offers the full backup button, the button takes the place of that sentence. Durations SHALL be rounded for reading: minutes under an hour, hours and minutes under a day, days and hours beyond.

#### Scenario: Plain names

- **WHEN** a user opens the Monitoring tab of a healthy database
- **THEN** no default label, legend or chart title contains "WAL", "slot", "LSN" or "lag"

#### Scenario: Broken chain without a pending full backup

- **WHEN** the WAL chain broken reason is shown and no full backup is requested or running
- **THEN** the tab asks the user to start a full backup, through the button or, for a user who cannot manage the database, through the sentence about what to do
- **AND** does not say that Databasus handles it by itself

#### Scenario: Every reason has an action

- **WHEN** any reason or notice is shown
- **THEN** it has both a sentence about what happened and a sentence about what to do

### Requirement: The Monitoring tab leads with the verdict

The tab SHALL open with a verdict banner. Its headline SHALL be:

- for the healthy verdict: all changes are being saved;
- for the needs attention and broken verdicts: the title of the most severe reason, broken reasons before needs-attention ones and, within one level, in the order the statistics list them;
- for the collecting data and not streaming verdicts, and while the current state cannot be confirmed: what they are today.

The banner SHALL use the verdict's color, green, orange, red or neutral gray, together with an icon and words, so color is never the only signal. Screen readers SHALL announce a change of the headline.

Under the headline, the banner SHALL show the headline reason's sentence about what happened and its sentence about what to do, then the restore sentence. While the banner offers the full backup button, it SHALL leave out the headline reason's sentence about what to do, because the button is that action. The headline reason's sentence about what happened SHALL not repeat the headline. The time of the newest measurement SHALL sit in the headline row. Every other reason SHALL follow inside the same banner as a compact line with its level icon, its title and both its sentences, broken ones first. The banner SHALL not show the keeping-up sentence. The not streaming banner SHALL say how to turn streaming on and link to the database's configuration.

#### Scenario: Healthy database

- **WHEN** the verdict is healthy
- **THEN** the headline is green and says that all changes are being saved
- **AND** no reason is shown

#### Scenario: Problem named in the headline

- **WHEN** the verdict is broken because the WAL chain is broken
- **THEN** the headline is red and says that changes were lost between backups
- **AND** the banner does not say that the database writes almost nothing

#### Scenario: Problems listed

- **WHEN** the verdict is broken because the slot is missing and the local queue is filling
- **THEN** the headline is red and names the slot missing reason
- **AND** the local queue reason follows inside the banner as a compact line

### Requirement: The Monitoring tab explains whether archiving keeps up

Under the figures, the tab SHALL describe the keeping-up state in one or two plain sentences:

- keeping up: Databasus saves changes as fast as the database writes them, with both rates;
- falling behind: the database writes changes faster than Databasus saves them, by how much per second, how much is not saved yet and how fast that amount grows;
- catching up: Databasus is saving a backlog, how much is left and about how long it takes;
- idle: the database writes almost nothing, and new changes reach storage within minutes;
- unknown: no sentence, because the figures already say not measured.

When an estimate until the rebuild lag or until the WAL allowance runs out exists and is under one day, the tab SHALL add what happens then: Databasus will start a new backup chain from a full backup, or the source will drop changes that Databasus has not received yet.

#### Scenario: Falling behind explained

- **WHEN** the source writes 21 MB/s, Databasus archives 12 MB/s and 3.4 GB is unarchived
- **THEN** the tab says that the database writes changes faster than Databasus saves them, by about 9 MB/s
- **AND** says that 3.4 GB is not saved yet and grows by about 32 GB an hour

#### Scenario: Consequence only when near

- **WHEN** the estimate until the rebuild lag is three days
- **THEN** the tab adds no sentence about a new backup chain

#### Scenario: Unknown state

- **WHEN** the keeping-up state is unknown
- **THEN** the tab shows no keeping-up sentence

### Requirement: The Monitoring tab shows where unsaved changes are

The technical details section SHALL show the path of changes as three stages, source, Databasus and storage, with:

- at the source: what is not received by Databasus yet, against the rebuild lag and, when the source sets a limit, the remaining allowance;
- in Databasus: the local queue, against the size at which the receiver is paused, and whether receiving is paused;
- in storage: how long ago the newest WAL segment was archived;
- between the stages: the source's write rate and the archive rate, on a connector that points to the next stage.

The stages SHALL sit in one frame as nodes of a single line, each with an icon and its name, and each rate SHALL sit on the line between two nodes, like the throughput of a pipe, with its name and the color of its chart line under it. Under the nodes the section SHALL say that the rates are averages over the last 10 minutes and the amounts are current.

Each stage SHALL show its amount under a short label. An empty amount SHALL read as nothing waiting, with a success icon, instead of 0 B over an empty meter. An amount above zero SHALL show as a number and, when it has a limit, say how much of the limit it is, such as "300 MB of 512 MB", next to a thin meter; the limit's sentence, such as the size at which receiving pauses, SHALL appear only then and otherwise stay in the explanation on hover or keyboard focus. An amount above its limit SHALL show as over the limit, not as a full bar. A stage without a problem SHALL stay neutral; a stage with a problem SHALL take the color of the most severe reason about that stage on its icon, together with an icon and the word for that level, so color is never the only signal. The WAL chain broken reason concerns the backup history rather than a stage, so it SHALL color no stage. Each amount on a stage SHALL offer a short explanation on hover or keyboard focus. When the section is wide enough the stages SHALL sit in a row joined by arrows pointing right; otherwise they SHALL stack vertically joined by arrows pointing down. Only one arrow direction SHALL be visible at a time.

#### Scenario: Backlog in the local queue

- **WHEN** the slot lag is small and the local queue holds 300 MB of its 512 MB pause size for two samples
- **THEN** the Databasus stage shows 300 MB of 512 MB in the needs-attention color with its icon and word
- **AND** the source stage stays neutral

#### Scenario: Database writing in bursts

- **WHEN** the database writes a burst every three minutes, each burst reaches storage within seconds, and the average write rate over 10 minutes is 54.6 KB/s
- **THEN** the source and Databasus stages say that nothing is waiting
- **AND** the links show 54.6 KB/s
- **AND** the section says that the rates are averages over the last 10 minutes and the amounts are current

#### Scenario: Broken chain with healthy stages

- **WHEN** the only reason is the WAL chain broken reason and the newest segment was archived seconds ago
- **THEN** every stage stays neutral, without a problem icon

#### Scenario: Wide screen

- **WHEN** the path of changes has room for its stages in a row
- **THEN** each link between stages shows one arrow pointing right and no arrow pointing down

#### Scenario: Narrow screen

- **WHEN** the path of changes has no room for its stages in a row
- **THEN** the stages stack vertically
- **AND** each link between stages shows one arrow pointing down and no arrow pointing right

### Requirement: The Monitoring tab shows charts for the selected period

The tab SHALL offer the last hour, day and week, and show one chart for the selected period below the figures: the source write rate and the archive rate. The technical details section SHALL show, for the same period:

- what is not saved to storage yet and what is not received by Databasus yet on one chart;
- the local queue size with its pause size, and receiver restarts and pauses.

Axes SHALL show formatted amounts and local times. The amount axis SHALL mark round amounts in the unit it shows, such as 256 KB, 512 KB and 768 KB, not fractions of another unit. Each chart SHALL show its title, its legend and, for the main chart, the period switch in one header above the plot. The plot SHALL keep a fixed height, because the tab itself has a maximum width. Hovering a time on one chart SHALL mark the same time on the others and show the formatted values. When the source write rate and the archive rate are equal, both lines SHALL stay distinguishable.

Intervals without samples SHALL show as gaps. A run of two or more intervals without samples after the data starts SHALL be shaded on the chart, and the tab SHALL say from when to when there are no measurements.

A chart SHALL appear only when the series holds at least four points with values. Otherwise the tab SHALL say that there is not enough data for this period yet and that a shorter period may already have a chart. When the data starts inside the period, the time axis SHALL start at the first point with values and the tab SHALL say since when it has data. The amount axis SHALL reach at least 64 KiB/s on the rate chart and at least one WAL segment on the amount charts, so small values stay near the bottom instead of filling the chart.

The technical details section SHALL also show the raw values with their technical names: the slot's status, the remaining allowance, the average and longest segment write, the upload concurrency, the ratio of WAL to stored bytes, the WAL segment size and the streaming session's start.

#### Scenario: Archiving falls behind on the chart

- **WHEN** every point of the hour holds a source write rate of 21 MB/s and an archive rate of 12 MB/s, and a rising unarchived amount
- **THEN** the archive series lies below the source series at every point
- **AND** the not-saved-yet series in the technical details rises across the hour

#### Scenario: No history yet

- **WHEN** the verdict is collecting data and the series holds no point with values
- **THEN** the tab shows the explanation instead of the chart

#### Scenario: Two minutes of data

- **WHEN** streaming started two minutes ago and the hour series holds two points with values
- **THEN** the tab shows the explanation instead of the chart

#### Scenario: Data started inside the period

- **WHEN** the hour series holds values only for its last 20 minutes
- **THEN** the chart's time axis starts 20 minutes ago
- **AND** the tab says since when it has data

#### Scenario: Measurements missing for five minutes

- **WHEN** the hour series has values except for five consecutive one-minute points in its middle
- **THEN** the chart shades those five minutes
- **AND** the tab says that there are no measurements from the first to the last of them

#### Scenario: Round rate ticks

- **WHEN** the rate axis reaches 1 MiB/s
- **THEN** its ticks read 0 B/s, 256 KB/s, 512 KB/s, 768 KB/s and 1 MB/s

#### Scenario: Equal rates

- **WHEN** the database writes and Databasus saves at the same rate at every point
- **THEN** both lines are visible on the chart

#### Scenario: Quiet database

- **WHEN** every point of the hour holds a source write rate of 2 KB/s
- **THEN** the rate axis reaches at least 64 KB/s and the line stays near its bottom

### Requirement: The Monitoring tab stays current without disturbing the reader

The tab SHALL read the current values every second, and the full statistics every 30 seconds, in every period while it is open and the browser tab is visible. It SHALL refresh both at once when the browser tab becomes visible again, and the statistics when the period changes. The figures, the restore timeline's end, the path of changes and the time of the newest measurement SHALL use the current values while they are newer than the newest sample. While the latest restorable moment of the statistics equals their newest save, the banner's restorable moment and the end of the newest restorable stretch SHALL follow the newest save of the current values, so the banner never names an older moment than the figure next to it; the verdict, the keeping-up summary, the rates and the charts SHALL use the statistics.

The tab SHALL show the age of the newest measurement, updated every second, and mark it as live only while that measurement is younger than 2 minutes. When the last successful refresh of the statistics is older than 2 minutes, the headline SHALL turn neutral gray and say that the current state cannot be confirmed, with the last known verdict and its age. A failed reading of the current values SHALL leave the values of the statistics on screen without an error of its own.

A refresh SHALL keep the previous content on screen until the new content arrives, keep the scroll position and the selected period, and not replay the charts' entry animation. Changed numbers and the headline color SHALL change with a short transition. When the reader's system asks for reduced motion, there SHALL be no transitions, no pulsing and no chart animation. A failed refresh of the statistics SHALL keep the last content, say that refreshing failed, and keep retrying on the same schedule. A failed first load SHALL show the error and a retry button.

#### Scenario: Status changes while open

- **WHEN** the tab is open on the day view and the verdict changes from healthy to needs attention
- **THEN** the headline changes within 30 seconds without reloading the page

#### Scenario: Buffer drains while open

- **WHEN** the local queue drains while the tab is open
- **THEN** the path of changes shows the new amount within about a second

#### Scenario: Segment saved between two statistics refreshes

- **WHEN** the latest restorable moment equals the newest save and a new segment reaches storage 5 seconds after the statistics refresh
- **THEN** within about a second the banner names the new save's time as the moment the database can be restored to

#### Scenario: Browser tab hidden

- **WHEN** the browser tab is hidden for ten minutes and then shown again
- **THEN** no refresh happened while it was hidden
- **AND** a refresh happens as soon as it is shown

#### Scenario: Refresh keeps failing

- **WHEN** refreshes have failed for 3 minutes after a healthy response
- **THEN** the headline is gray and says the state cannot be confirmed, last known healthy 3 minutes ago

