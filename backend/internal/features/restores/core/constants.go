package restores_core

import "time"

// ToolOutputWaitDelay bounds how long a restore waits for the restore tool's output to end
// after the tool itself exited. A process the tool left behind can hold that output open.
const ToolOutputWaitDelay = 10 * time.Second
