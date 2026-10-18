# WAL Stream Monitoring Specification

## Purpose

Lets an operator of any experience see at a glance whether WAL archiving of a streaming PostgreSQL database is healthy, and if it is not, what is wrong, how urgent it is and where the unsaved changes are. The same statistics are available through the authenticated API for other consumers.

## Requirements

### Requirement: A streaming database is sampled every 30 seconds

While Databasus streams WAL for a database, it SHALL record one sample of that database's archiving health every 30 seconds. Each sample SHALL hold:

- when it was taken, and when the current streaming session started;
- the source's state: reachable, unreachable, or reachable but without Databasus's replication slot;
- since when the source has been unreachable, while it is, counted across streaming sessions;
- the source's current WAL position and the archive progress point;
- the WAL bytes the source wrote since the previous sample;
- the number of segments archived since the previous sample, their WAL size, and the bytes written to storage for them;
- the source's WAL segment size;
- the highest number of segment uploads running at the same time since the previous sample;
- the average and the longest time to write one segment to storage since the previous sample;
- the slot's status as the source reports it;
- since when the slot has kept extra WAL, meaning status `extended` or `unreserved`, while it does;
- the slot lag: how much WAL the source keeps for Databasus because Databasus has not received it yet;
- how much WAL the source can still write before the slot loses WAL, or that the source sets no limit;
- the unarchived WAL: everything the source wrote after the archive progress point, which is not in storage yet;
- since when the archive progress point has stood still while the source keeps writing, while it does;
- the size of the local queue in bytes, and the size at which the receiver is paused;
- the number of receiver restarts since the previous sample, not counting pauses for a full local queue;
- the number of receiver pauses for a full local queue since the previous sample;
- whether the receiver was running;
- whether the source refused the forced WAL switch in this session.

The two "since" times about the slot and the archive SHALL come from the tracking that decides the existing "WAL chain at risk" and "archiving fell behind" notifications. A sample and a notification then agree about when a condition began.

Source-side values SHALL be absent from a sample taken while the source was unreachable or without the slot. The unarchived WAL SHALL also be absent while nothing has been archived for the database yet. The other values SHALL still be recorded.

The WAL bytes the source wrote since the previous sample SHALL also be absent in three more cases:

- the first sample of a streaming session;
- the first sample after one without source-side values;
- a sample whose source position is behind the previous sample's position, as happens when the source switches to a new timeline.

Probing the source SHALL give up after 10 seconds, so a source that accepts connections but does not answer still yields a sample on time.

#### Scenario: Healthy streaming

- **WHEN** four consecutive sampling ticks run for a streaming database with the source reachable
- **THEN** four samples exist for it, all with the source reachable
- **AND** every sample after the first holds a source write value
- **AND** every sample holds either a remaining WAL allowance or the marker that the source sets no limit
- **AND** every sample holds the slot lag, the unarchived WAL, the local queue size and its pause size

#### Scenario: Source unreachable

- **WHEN** the source stops answering while the streamer keeps running
- **THEN** samples keep being recorded with the source unreachable
- **AND** their slot status, slot lag, remaining WAL allowance, unarchived WAL and source write values are absent
- **AND** they all carry the same time since which the source has been unreachable

#### Scenario: Unreachable across a session restart

- **WHEN** the newest sample found the source unreachable and a new streaming session takes its first sample with the source still unreachable
- **THEN** that sample carries the same unreachable-since time as the newest sample before it

#### Scenario: Silent source

- **WHEN** the source accepts connections but never answers the probe
- **THEN** a sample with the source unreachable is recorded no later than 10 seconds after its tick

#### Scenario: Slot removed on the source

- **WHEN** the source answers but Databasus's replication slot no longer exists there
- **THEN** the sample records the source as reachable without the slot, not as unreachable

#### Scenario: Slot lost on the source

- **WHEN** the source reports the slot's status as `lost`
- **THEN** the sample still holds the source's real current WAL position
- **AND** no unarchived WAL value in it is negative

#### Scenario: Source write volume after a restart

- **WHEN** the streamer starts and takes its first sample
- **THEN** that sample has no value for WAL written since the previous sample
- **AND** the second sample has one

#### Scenario: Source reachable again

- **WHEN** a sample is taken while the source is unreachable and the next one is taken with the source reachable again
- **THEN** the next sample has no value for WAL written since the previous sample
- **AND** the sample after it has one

#### Scenario: Source position moves backwards

- **WHEN** the source's current WAL position in a sample is lower than in the previous sample
- **THEN** that sample has no value for WAL written since the previous sample
- **AND** no sample holds a negative write value

#### Scenario: Pause for a full local queue

- **WHEN** the local queue reaches the pause size and the receiver is stopped and started again once it drains
- **THEN** the next sample counts one pause and no restart

#### Scenario: Database not streaming

- **WHEN** a physical database in full and incremental mode is reconciled alongside a WAL streaming database
- **AND** the WAL streaming database has its first sample
- **THEN** no sample exists for the full and incremental database

### Requirement: Samples are kept for 7 days

Databasus SHALL delete samples older than 7 days at least once an hour, whether or not the database still streams. Deleting a database SHALL delete its samples.

#### Scenario: Expired samples

- **WHEN** the hourly cleanup runs and a database has samples older than 7 days
- **THEN** those samples no longer exist and the newer ones remain

#### Scenario: Streaming switched off

- **WHEN** a database stopped streaming 8 days ago
- **THEN** none of its samples remain

#### Scenario: Database removed

- **WHEN** a database with samples is deleted
- **THEN** none of its samples remain

### Requirement: Only users who can view the database can read its WAL stream statistics

The WAL stream statistics of a database SHALL be returned only to an authenticated user who can view that database. Any other request SHALL be refused, the same way the other physical backup endpoints refuse it. The statistics SHALL contain no connection details, credentials, error messages from the source or WAL contents.

#### Scenario: Member of the database's workspace

- **WHEN** a member of the workspace that owns the database requests its statistics
- **THEN** the statistics are returned

#### Scenario: User from another workspace

- **WHEN** a user without access to the database's workspace requests its statistics
- **THEN** the request is refused

#### Scenario: Unauthenticated request

- **WHEN** a request without valid authentication asks for statistics
- **THEN** the request is refused with an authentication error

#### Scenario: Nonexistent database

- **WHEN** an authenticated user requests statistics for a database ID that does not exist
- **THEN** the request is refused
- **AND** no statistics are returned

### Requirement: The statistics API is a documented, unit-explicit contract

The statistics SHALL be served by an authenticated API endpoint documented with the rest of the API. Every amount SHALL carry its unit in its name, in bytes, bytes per second, seconds or milliseconds. Every time SHALL be an absolute UTC time. Every verdict, state, reason and notice SHALL be a stable code from a documented list, not display text, so other consumers can rely on it.

A response SHALL hold:

- the health verdict, its reasons and its notices;
- whether a full backup is requested or running;
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

### Requirement: The streamer's state accounts for its heartbeat and repeated failures

The streamer's state SHALL be running only while Databasus marks it running and its heartbeat is younger than the age after which another tick may take the streamer over. It SHALL be not running when it is marked failed or its heartbeat is older. It SHALL be not started when no streamer has ever run for the database.

Databasus SHALL remember since when the streamer has been failing: from its first failure after the last session that recorded a sample, until a session records a sample again. A failing streamer that is taken over and fails again SHALL keep the first failure time.

#### Scenario: Stale heartbeat

- **WHEN** the streamer is marked running but its last heartbeat is older than the takeover age
- **THEN** the state is not running

#### Scenario: Streamer never started

- **WHEN** no streamer has ever run for the database
- **THEN** the state is not started

#### Scenario: Crash loop before the first sample

- **WHEN** every new session fails before its first sample for five minutes
- **THEN** the failing-since time is the time of the first of those failures

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
| WAL chain broken | - | the newest completed full backup's chain has a WAL gap below the archive progress point; it clears when a newer full backup completes |
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

### Requirement: The statistics say whether archiving keeps up with the source

The keeping-up summary SHALL cover the samples of the last 10 minutes and hold:

- the source's write rate and the archive rate in WAL bytes per second, both over the seconds covered by samples that hold a source write value;
- the unarchived WAL now and how fast it changes, in bytes per second, negative while it shrinks;
- the slot lag now and how fast it changes;
- the configured slot lag at which Databasus rebuilds the slot;
- an estimate of the time until the unarchived WAL is archived, while it shrinks;
- an estimate of the time until the slot lag reaches the rebuild lag, while the slot lag grows and the slot's status is neither `extended` nor `unreserved`;
- an estimate of the time until the remaining WAL allowance runs out, while the slot lag grows and the source sets a limit;
- one keeping-up state.

The keeping-up state SHALL be the first of these that matches:

1. **unknown**: the window holds no sample with a source write value;
2. **falling behind**: the archive slower than source reason applies, ignoring its two-sample rule;
3. **catching up**: the unarchived WAL is at least the backlog threshold and its lowest value in the last 2.5 minutes is below its lowest value in the first 2.5 minutes;
4. **idle**: the source wrote less than 1 MiB over the window and the unarchived WAL is at most one segment;
5. **keeping up**: anything else.

#### Scenario: Archiving keeps up

- **WHEN** the source writes 10 MB/s and Databasus archives 10 MB/s for ten minutes
- **THEN** the keeping-up state is keeping up
- **AND** both rates are about 10 MB/s

#### Scenario: Archiving falls behind

- **WHEN** the source writes 21 MB/s and Databasus archives 12 MB/s for ten minutes
- **THEN** the keeping-up state is falling behind
- **AND** the unarchived WAL grows by about 9 MB/s

#### Scenario: Catching up after a burst

- **WHEN** 3 GiB of WAL is unarchived and it shrank by 600 MiB over the last 10 minutes
- **THEN** the keeping-up state is catching up
- **AND** the estimate until it is archived is about 50 minutes

#### Scenario: Running out of allowance

- **WHEN** the source sets a limit, 5 GiB of allowance remains and the slot lag grew by 1 GiB over the last 10 minutes
- **THEN** the estimate until the allowance runs out is about 50 minutes

#### Scenario: Quiet database

- **WHEN** the source wrote 200 KiB over the last 10 minutes and 100 KiB is unarchived
- **THEN** the keeping-up state is idle

#### Scenario: Quiet database with a stuck backlog

- **WHEN** the source wrote 200 KiB over the last 10 minutes and 2 GiB is unarchived and not shrinking
- **THEN** the keeping-up state is not idle

### Requirement: The statistics cover the last hour, day or week

A request SHALL name a period of the last hour, the last day or the last week. The series SHALL hold at most 360 points of equal length and state that length: 1 minute for an hour, 4 minutes for a day and 28 minutes for a week. The last point covers the interval that contains the current moment. In each point:

- write and archive volumes become rates in bytes per second over the seconds its samples cover;
- slot lag, unarchived WAL and local queue size take the highest value in the interval;
- receiver restarts, receiver pauses and streaming session starts are summed;
- upload concurrency and the longest segment write take the highest value, and the average segment write is weighted by the segments archived.

An interval with no samples SHALL appear as a point without values, not as zero.

#### Scenario: Week view

- **WHEN** a user requests the last week for a database that streamed the whole week
- **THEN** the series has at most 360 points
- **AND** each point's archive rate equals the WAL bytes archived in its interval divided by the seconds its samples cover

#### Scenario: Streamer was down for an hour

- **WHEN** the streamer recorded no samples for one hour of the requested day
- **THEN** the points covering that hour carry no values

#### Scenario: Unknown period

- **WHEN** a request names a period other than hour, day or week
- **THEN** the request is refused as invalid

### Requirement: The Monitoring tab appears only for WAL streaming databases

The database page SHALL show a Monitoring tab for a PostgreSQL physical database whose backup mode is full + incremental + WAL streaming, and for no other database.

#### Scenario: WAL streaming database

- **WHEN** a user opens a PostgreSQL physical database in full + incremental + WAL streaming mode
- **THEN** the page offers Config, Backups and Monitoring tabs

#### Scenario: Other databases

- **WHEN** a user opens a logical database or a physical database in a mode without WAL streaming
- **THEN** the page offers no Monitoring tab

### Requirement: The Monitoring tab speaks plain language

Every label, headline, sentence, legend and chart title the tab shows by default SHALL avoid the terms "WAL", "slot", "LSN" and "lag". The tab SHALL call the unarchived WAL "not saved to storage yet" and the slot lag "not received by Databasus yet", and SHALL say that the second is part of the first. Technical names SHALL appear only in explanations shown on hover or keyboard focus, in the details section, and in a "what to do" sentence that names a setting or permission to change, such as `max_slot_wal_keep_size`.

Each reason and each notice SHALL have one sentence about what happened, with its amounts and, where known, how long it has lasted, and one sentence about what to do. When Databasus resolves a condition by itself, the second sentence SHALL say so, and only then. For a broken chain or a missing slot, the second sentence SHALL ask the user to start a full backup unless one is requested or running. Durations SHALL be rounded for reading: minutes under an hour, hours and minutes under a day, days and hours beyond.

#### Scenario: Plain names

- **WHEN** a user opens the Monitoring tab of a healthy database
- **THEN** no default label, legend or chart title contains "WAL", "slot", "LSN" or "lag"

#### Scenario: Broken chain without a pending full backup

- **WHEN** the WAL chain broken reason is shown and no full backup is requested or running
- **THEN** the tab asks the user to start a full backup
- **AND** does not say that Databasus handles it by itself

#### Scenario: Every reason has an action

- **WHEN** any reason or notice is shown
- **THEN** it has both a sentence about what happened and a sentence about what to do

### Requirement: The Monitoring tab leads with the verdict

The tab SHALL open with the verdict as a headline: all is well, something needs attention, something is broken, data is being collected, or streaming is off. The headline SHALL use the verdict's color, green, orange, red or neutral gray, together with an icon and words, so color is never the only signal. Screen readers SHALL announce a change of the headline.

The not streaming headline SHALL say how to turn streaming on and link to the database's configuration. Below the headline, the tab SHALL list the reasons, broken ones first. Notices SHALL appear after the reasons in a neutral informational style, and the no slot WAL limit notice SHALL explain in plain words what the limit protects and what it costs.

#### Scenario: Healthy database

- **WHEN** the verdict is healthy
- **THEN** the headline is green and says that everything is in order
- **AND** no reason list is shown

#### Scenario: Problems listed

- **WHEN** the verdict is broken because the slot is missing and the local queue is filling
- **THEN** the headline is red
- **AND** the slot missing reason is listed before the local queue reason

### Requirement: The Monitoring tab explains whether archiving keeps up

Right below the headline, the tab SHALL describe the keeping-up state in one or two plain sentences:

- keeping up: Databasus saves changes as fast as the database writes them, with both rates;
- falling behind: the database writes changes faster than Databasus saves them, by how much per second, how much is not saved yet and how fast that amount grows;
- catching up: Databasus is saving a backlog, how much is left and about how long it takes;
- idle: the database writes almost nothing, and new changes reach storage within minutes;
- unknown: no current numbers are available.

When an estimate until the rebuild lag or until the WAL allowance runs out exists and is under one day, the tab SHALL add what happens then: Databasus will start a new backup chain from a full backup, or the source will drop changes that Databasus has not received yet.

#### Scenario: Falling behind explained

- **WHEN** the source writes 21 MB/s, Databasus archives 12 MB/s and 3.4 GB is unarchived
- **THEN** the tab says that the database writes changes faster than Databasus saves them, by about 9 MB/s
- **AND** says that 3.4 GB is not saved yet and grows by about 32 GB an hour

#### Scenario: Consequence only when near

- **WHEN** the estimate until the rebuild lag is three days
- **THEN** the tab adds no sentence about a new backup chain

### Requirement: The Monitoring tab shows where unsaved changes are

The tab SHALL show the path of changes as three stages, source, Databasus and storage, with:

- at the source: what is not received by Databasus yet, against the rebuild lag and, when the source sets a limit, the remaining allowance;
- in Databasus: the local queue, against the size at which the receiver is paused, and whether receiving is paused;
- in storage: how long ago the newest WAL segment was archived;
- between the stages: the source's write rate and the archive rate.

An amount above its limit SHALL show as over the limit, not as a full bar. Each stage SHALL take the color of the most severe reason about it. Each amount on a stage SHALL offer a short explanation on hover or keyboard focus. On a narrow screen the stages SHALL stack vertically.

#### Scenario: Backlog in the local queue

- **WHEN** the slot lag is small and the local queue holds 300 MB of its 512 MB pause size for two samples
- **THEN** the Databasus stage shows 300 MB of 512 MB in the needs-attention color
- **AND** the source stage stays in the healthy color

### Requirement: The Monitoring tab shows charts for the selected period

The tab SHALL offer the last hour, day and week, and show for the selected period:

- the source write rate and the archive rate on one chart;
- what is not saved to storage yet and what is not received by Databasus yet on one chart;
- the local queue size with its pause size, and receiver restarts and pauses.

Axes SHALL show formatted amounts and local times. Hovering a time on one chart SHALL mark the same time on the others and show the formatted values. Intervals without samples SHALL show as gaps. When the series holds no point with values, the tab SHALL say that charts appear after a few minutes of streaming instead of showing empty charts. When the data starts inside the period, the tab SHALL say since when it has data.

A details section, collapsed by default, SHALL show the raw values with their technical names: the slot's status, the remaining allowance, the average and longest segment write, the upload concurrency, the ratio of WAL to stored bytes, the WAL segment size and the streaming session's start.

#### Scenario: Archiving falls behind on the chart

- **WHEN** every point of the hour holds a source write rate of 21 MB/s and an archive rate of 12 MB/s, and a rising unarchived amount
- **THEN** the archive series lies below the source series at every point
- **AND** the not-saved-yet series rises across the hour

#### Scenario: No history yet

- **WHEN** the verdict is collecting data and the series holds no point with values
- **THEN** the tab shows the explanation instead of the charts

### Requirement: The Monitoring tab stays current without disturbing the reader

The tab SHALL refresh every 30 seconds in every period while it is open and the browser tab is visible. It SHALL refresh at once when the browser tab becomes visible again and when the period changes.

The tab SHALL show the age of the newest sample, updated every second, and mark it as live only while that sample is younger than 2 minutes. When the last successful refresh is older than 2 minutes, the headline SHALL turn neutral gray and say that the current state cannot be confirmed, with the last known verdict and its age.

A refresh SHALL keep the previous content on screen until the new content arrives, keep the scroll position and the selected period, and not replay the charts' entry animation. Changed numbers and the headline color SHALL change with a short transition. When the reader's system asks for reduced motion, there SHALL be no transitions, no pulsing and no chart animation. A failed refresh SHALL keep the last content, say that refreshing failed, and keep retrying on the same schedule. A failed first load SHALL show the error and a retry button.

#### Scenario: Status changes while open

- **WHEN** the tab is open on the day view and the verdict changes from healthy to needs attention
- **THEN** the headline changes within 30 seconds without reloading the page

#### Scenario: Browser tab hidden

- **WHEN** the browser tab is hidden for ten minutes and then shown again
- **THEN** no refresh happened while it was hidden
- **AND** a refresh happens as soon as it is shown

#### Scenario: Refresh keeps failing

- **WHEN** refreshes have failed for 3 minutes after a healthy response
- **THEN** the headline is gray and says the state cannot be confirmed, last known healthy 3 minutes ago
