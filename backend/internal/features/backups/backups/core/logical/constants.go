package backups_core_logical

import "time"

// DumpToolOutputWaitDelay bounds how long a backup waits for the dump tool's output to end after
// the tool itself exited. A process the tool left behind can hold that output open.
const DumpToolOutputWaitDelay = 10 * time.Second
