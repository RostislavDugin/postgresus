package container

import (
	"context"
	"fmt"
	"log/slog"
)

// purgeAgentContainers runs once at startup, with no age grace and no
// active-set check, so each agent process begins from a clean slate
// ("always fresh"). It finds networks by label, not through containers, so a
// network whose container is already gone is removed too.
//
// Best-effort by design: a failed listing or removal is logged and never
// blocks startup. A leftover the purge misses cannot block a retry, because
// each attempt clears its own verification's leftovers first (see
// prepareAttemptNetwork); degraded beats refusing to start on a transient
// Docker hiccup.
func purgeAgentContainers(ctx context.Context, engine leftoverEngine, agentID string, log *slog.Logger) {
	removed, err := removeLeftovers(ctx, engine, agentLabels(agentID))
	if err != nil {
		log.Warn("startup purge left some resources in place", "error", err)
	}

	if removed > 0 {
		log.Info(fmt.Sprintf("startup purge removed %d stale container(s) and network(s)", removed))
	}
}
