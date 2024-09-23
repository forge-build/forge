package v1alpha1

const (
	// BuildNameLabel is the label set on InfraBuild linked to a Build and
	//provisioners.
	BuildNameLabel = "forge.build/build-name"

	// ProviderNameLabel is the label set on components in the provider manifest.
	// This label allows to easily identify all the components belonging to a provider; the forgectl
	// tool uses this label for implementing provider's lifecycle operations.
	ProviderNameLabel = "cluster.x-k8s.io/provider"

	// PausedAnnotation is an annotation that can be applied to any Cluster API
	// object to prevent a controller from processing a resource.
	//
	// Controllers working with Cluster API objects must check the existence of this annotation
	// on the reconciled object.
	PausedAnnotation = "forge.build/paused"

	// WatchLabel is a label othat can be applied to any Build API object.
	//
	// Controllers which allow for selective reconciliation may check this label and proceed
	// with reconciliation of the object only if this label and a configured value is present.
	WatchLabel = "cluster.x-k8s.io/watch-filter"
)
