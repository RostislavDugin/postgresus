package v1alpha1

// SecretKeyRef references a key within a Kubernetes Secret.
type SecretKeyRef struct {
	// Name of the Secret in the same namespace.
	Name string `json:"name"`
	// Key within the Secret data.
	Key string `json:"key"`
}
