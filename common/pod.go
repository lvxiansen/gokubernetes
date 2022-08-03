package common

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
