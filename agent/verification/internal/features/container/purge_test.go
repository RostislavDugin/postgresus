package container

import (
	"errors"
	"testing"

	cerrdefs "github.com/containerd/errdefs"
	"github.com/stretchr/testify/assert"

	"databasus-verification-agent/internal/testutil"
)

func Test_PurgeAgentContainers_RemovesEveryContainerAndNetwork(t *testing.T) {
	host := &fakeDockerHost{}
	host.addContainer("c1", attemptLabels("agent-1", "v1"))
	host.addContainer("c2", attemptLabels("agent-1", "v2"))
	host.addNetwork("n1", attemptLabels("agent-1", "v1"))
	host.addNetwork("n2", attemptLabels("agent-1", "v2"))

	purgeAgentContainers(t.Context(), host, "agent-1", testutil.DiscardLogger())

	assert.Empty(t, host.containerIDs())
	assert.Empty(t, host.networkIDs())
}

func Test_PurgeAgentContainers_WhenNothingLabeled_RemovesNothing(t *testing.T) {
	host := &fakeDockerHost{}

	purgeAgentContainers(t.Context(), host, "agent-1", testutil.DiscardLogger())

	assert.Equal(t, []string{
		"list containers " + LabelAgentID + "=agent-1",
		"list networks " + LabelAgentID + "=agent-1",
	}, host.log)
}

func Test_PurgeAgentContainers_WhenContainerRemovalFails_ContinuesWithRemainingResources(t *testing.T) {
	host := &fakeDockerHost{}
	host.addContainer("c1", attemptLabels("agent-1", "v1"))
	host.addContainer("c2", attemptLabels("agent-1", "v2"))
	host.addNetwork("n2", attemptLabels("agent-1", "v2"))
	host.failRemoval("c1", errors.New("container in use"))

	assert.NotPanics(t, func() {
		purgeAgentContainers(t.Context(), host, "agent-1", testutil.DiscardLogger())
	})

	assert.Equal(t, []string{"c1"}, host.containerIDs())
	assert.Empty(t, host.networkIDs())
}

func Test_PurgeAgentContainers_WhenNetworkHasNoContainer_RemovesNetwork(t *testing.T) {
	host := &fakeDockerHost{}
	host.addNetwork("n1", attemptLabels("agent-1", "v1"))

	purgeAgentContainers(t.Context(), host, "agent-1", testutil.DiscardLogger())

	assert.Empty(t, host.networkIDs())
}

func Test_PurgeAgentContainers_WhenContainerAndNetworkExist_RemovesContainerBeforeNetwork(t *testing.T) {
	host := &fakeDockerHost{}
	host.addContainer("c1", attemptLabels("agent-1", "v1"))
	host.addNetwork("n1", attemptLabels("agent-1", "v1"))

	purgeAgentContainers(t.Context(), host, "agent-1", testutil.DiscardLogger())

	assert.Equal(t, []string{
		"list containers " + LabelAgentID + "=agent-1",
		"remove container c1",
		"list networks " + LabelAgentID + "=agent-1",
		"remove network n1",
	}, host.log)
}

func Test_PurgeAgentContainers_WhenResourcesBelongToOtherAgentOrNoAgent_LeavesThemInPlace(t *testing.T) {
	host := &fakeDockerHost{}
	host.addContainer("foreign-container", attemptLabels("agent-2", "v1"))
	host.addNetwork("foreign-network", attemptLabels("agent-2", "v1"))
	host.addNetwork("unrelated-network", map[string]string{"com.example.app": "web"})

	purgeAgentContainers(t.Context(), host, "agent-1", testutil.DiscardLogger())

	assert.Equal(t, []string{"foreign-container"}, host.containerIDs())
	assert.Equal(t, []string{"foreign-network", "unrelated-network"}, host.networkIDs())
}

func Test_PurgeAgentContainers_FiltersByAgentLabelOnly(t *testing.T) {
	host := &fakeDockerHost{}

	purgeAgentContainers(t.Context(), host, "agent-1", testutil.DiscardLogger())

	assert.Equal(t, []string{
		"list containers " + formatLabels(agentLabels("agent-1")),
		"list networks " + formatLabels(agentLabels("agent-1")),
	}, host.log)
}

func Test_PurgeAgentContainers_WhenNetworkListFails_LogsAndRemovesContainers(t *testing.T) {
	host := &fakeDockerHost{listNetworksErr: errors.New("docker unreachable")}
	host.addContainer("c1", attemptLabels("agent-1", "v1"))
	host.addNetwork("n1", attemptLabels("agent-1", "v1"))
	log, logs := testutil.CaptureLogger()

	assert.NotPanics(t, func() {
		purgeAgentContainers(t.Context(), host, "agent-1", log)
	})

	assert.Empty(t, host.containerIDs())
	assert.Equal(t, []string{"n1"}, host.networkIDs())
	assert.Contains(t, logs.String(), "level=WARN msg=\"startup purge left some resources in place\"")
	assert.Contains(t, logs.String(), "list networks: docker unreachable")
}

func Test_PurgeAgentContainers_WhenRemovalReturnsNotFound_ContinuesWithRemainingResources(t *testing.T) {
	host := &fakeDockerHost{}
	host.addContainer("c1", attemptLabels("agent-1", "v1"))
	host.addContainer("c2", attemptLabels("agent-1", "v2"))
	host.addNetwork("n1", attemptLabels("agent-1", "v1"))
	host.failRemoval("c1", cerrdefs.ErrNotFound)
	log, logs := testutil.CaptureLogger()

	purgeAgentContainers(t.Context(), host, "agent-1", log)

	assert.Equal(t, []string{
		"list containers " + LabelAgentID + "=agent-1",
		"remove container c1",
		"remove container c2",
		"list networks " + LabelAgentID + "=agent-1",
		"remove network n1",
	}, host.log)
	assert.Empty(t, host.networkIDs())
	assert.NotContains(t, logs.String(), "level=WARN")
	assert.Contains(t, logs.String(), "startup purge removed 3 stale container(s) and network(s)")
}

func Test_PurgeAgentContainers_WhenContainerListFails_LogsAndRemovesNetworks(t *testing.T) {
	host := &fakeDockerHost{listContainersErr: errors.New("docker unreachable")}
	host.addContainer("c1", attemptLabels("agent-1", "v1"))
	host.addNetwork("n1", attemptLabels("agent-1", "v1"))
	host.addNetwork("n2", attemptLabels("agent-1", "v2"))
	host.failRemoval("n1", errors.New("network has active endpoints"))
	log, logs := testutil.CaptureLogger()

	assert.NotPanics(t, func() {
		purgeAgentContainers(t.Context(), host, "agent-1", log)
	})

	assert.Equal(t, []string{"c1"}, host.containerIDs())
	assert.Equal(t, []string{"n1"}, host.networkIDs())
	assert.Contains(t, logs.String(), "level=WARN msg=\"startup purge left some resources in place\"")
	assert.Contains(t, logs.String(), "list containers: docker unreachable")
	assert.Contains(t, logs.String(), "remove network n1 of verification v1: network has active endpoints")
}
