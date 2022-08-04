package common

import (
	"fmt"
	v1 "k8s.io/api/core/v1"
)

type Pod struct {
	ObjectMeta ObjectMeta `json:"objectMeta"`
	TypeMeta   TypeMeta   `json:"typeMeta"`

	// Status determined based on the same logic as kubectl.
	Status string `json:"status"`

	// RestartCount of containers restarts.
	RestartCount int32 `json:"restartCount"`

	// Pod metrics.
	Metrics *PodMetrics `json:"metrics"`

	// NodeName of the Node this Pod runs on.
	NodeName string `json:"nodeName"`

	// ContainerImages holds a list of the Pod images.
	ContainerImages []string `json:"containerImages"`
}
type PodList struct {
	ListMeta ListMeta `json:"listMeta"`

	// Basic information about resources status on the list.
	Status ResourceStatus `json:"status"`

	// Unordered list of Pods.
	Pods []Pod `json:"pods"`

	// List of non-critical errors, that occurred during resource retrieval.
	Errors []error `json:"errors"`
}

type NamespaceQuery struct {
	namespaces []string
}

// ToRequestParam returns K8s API namespace query for list of objects from this namespaces.
// This is an optimization to query for single namespace if one was selected and for all
// namespaces otherwise.
func (n *NamespaceQuery) ToRequestParam() string {
	if len(n.namespaces) == 1 {
		return n.namespaces[0]
	}
	return NamespaceAll
}

// Matches returns true when the given namespace matches this query.
func (n *NamespaceQuery) Matches(namespace string) bool {
	if len(n.namespaces) == 0 {
		return true
	}

	for _, queryNamespace := range n.namespaces {
		if namespace == queryNamespace {
			return true
		}
	}
	return false
}
func GetPodStatus(pod v1.Pod) string {
	restarts := 0
	readyContainers := 0

	reason := string(pod.Status.Phase)
	if pod.Status.Reason != "" {
		reason = pod.Status.Reason
	}

	initializing := false
	for i := range pod.Status.InitContainerStatuses {
		container := pod.Status.InitContainerStatuses[i]
		restarts += int(container.RestartCount)
		switch {
		case container.State.Terminated != nil && container.State.Terminated.ExitCode == 0:
			continue
		case container.State.Terminated != nil:
			// initialization is failed
			if len(container.State.Terminated.Reason) == 0 {
				if container.State.Terminated.Signal != 0 {
					reason = fmt.Sprintf("Init: Signal %d", container.State.Terminated.Signal)
				} else {
					reason = fmt.Sprintf("Init: ExitCode %d", container.State.Terminated.ExitCode)
				}
			} else {
				reason = "Init:" + container.State.Terminated.Reason
			}
			initializing = true
		case container.State.Waiting != nil && len(container.State.Waiting.Reason) > 0 && container.State.Waiting.Reason != "PodInitializing":
			reason = fmt.Sprintf("Init: %s", container.State.Waiting.Reason)
			initializing = true
		default:
			reason = fmt.Sprintf("Init: %d/%d", i, len(pod.Spec.InitContainers))
			initializing = true
		}
		break
	}
	if !initializing {
		restarts = 0
		hasRunning := false
		for i := len(pod.Status.ContainerStatuses) - 1; i >= 0; i-- {
			container := pod.Status.ContainerStatuses[i]

			restarts += int(container.RestartCount)
			if container.State.Waiting != nil && container.State.Waiting.Reason != "" {
				reason = container.State.Waiting.Reason
			} else if container.State.Terminated != nil && container.State.Terminated.Reason != "" {
				reason = container.State.Terminated.Reason
			} else if container.State.Terminated != nil && container.State.Terminated.Reason == "" {
				if container.State.Terminated.Signal != 0 {
					reason = fmt.Sprintf("Signal: %d", container.State.Terminated.Signal)
				} else {
					reason = fmt.Sprintf("ExitCode: %d", container.State.Terminated.ExitCode)
				}
			} else if container.Ready && container.State.Running != nil {
				hasRunning = true
				readyContainers++
			}
		}

		// change pod status back to "Running" if there is at least one container still reporting as "Running" status
		if reason == "Completed" && hasRunning {
			if hasPodReadyCondition(pod.Status.Conditions) {
				reason = string(v1.PodRunning)
			} else {
				reason = "NotReady"
			}
		}
	}

	if pod.DeletionTimestamp != nil && pod.Status.Reason == "NodeLost" {
		reason = string(v1.PodUnknown)
	} else if pod.DeletionTimestamp != nil {
		reason = "Terminating"
	}

	if len(reason) == 0 {
		reason = string(v1.PodUnknown)
	}

	return reason
}
func hasPodReadyCondition(conditions []v1.PodCondition) bool {
	for _, condition := range conditions {
		if condition.Type == v1.PodReady && condition.Status == v1.ConditionTrue {
			return true
		}
	}
	return false
}

// getRestartCount return the restart count of given pod (total number of its containers restarts).
func GetRestartCount(pod v1.Pod) int32 {
	var restartCount int32 = 0
	for _, containerStatus := range pod.Status.ContainerStatuses {
		restartCount += containerStatus.RestartCount
	}
	return restartCount
}

// GetContainerImages returns container image strings from the given pod spec.
func GetContainerImages(podTemplate *v1.PodSpec) []string {
	var containerImages []string
	for _, container := range podTemplate.Containers {
		containerImages = append(containerImages, container.Image)
	}
	return containerImages
}
func GetStatus(list *v1.PodList) ResourceStatus {
	info := ResourceStatus{}
	if list == nil {
		return info
	}

	for _, pod := range list.Items {
		switch GetPodStatusPhase(pod) {
		case v1.PodFailed:
			info.Failed++
		case v1.PodSucceeded:
			info.Succeeded++
		case v1.PodRunning:
			info.Running++
		case v1.PodPending:
			info.Pending++
		case v1.PodUnknown:
			info.Unknown++
		case "Terminating":
			info.Terminating++
		}
	}

	return info
}

// getPodStatusPhase returns one of four pod status phases (Pending, Running, Succeeded, Failed, Unknown, Terminating)
func GetPodStatusPhase(pod v1.Pod) v1.PodPhase {
	// For terminated pods that failed
	if pod.Status.Phase == v1.PodFailed {
		return v1.PodFailed
	}

	// For successfully terminated pods
	if pod.Status.Phase == v1.PodSucceeded {
		return v1.PodSucceeded
	}

	ready := false
	initialized := false
	for _, c := range pod.Status.Conditions {
		if c.Type == v1.PodReady {
			ready = c.Status == v1.ConditionTrue
		}
		if c.Type == v1.PodInitialized {
			initialized = c.Status == v1.ConditionTrue
		}
	}

	if initialized && ready && pod.Status.Phase == v1.PodRunning {
		return v1.PodRunning
	}

	if pod.DeletionTimestamp != nil && pod.Status.Reason == "NodeLost" {
		return v1.PodUnknown
	} else if pod.DeletionTimestamp != nil {
		return "Terminating"
	}

	// pending
	return v1.PodPending
}
