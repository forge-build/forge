package v1alpha1

const (
	// BuildNameLabel is the label set on InfraBuild linked to a Build and
	//provisioners.
	BuildNameLabel = "forge.build/build-name"

	// ProviderNameLabel is the label set on components in the provider manifest.
	// This label allows to easily identify all the components belonging to a provider; the forgectl
	// tool uses this label for implementing provider's lifecycle operations.
	ProviderNameLabel = "forge.build/provider"

	// ManagedByAnnotation is an annotation that can be applied to InfraBuild resources to signify that
	// some external system is managing the build infrastructure.
	//
	// Provider InfraBuild controllers will ignore resources with this annotation.
	// An external controller must fulfill the contract of the InfraBuild resource.
	// External infrastructure providers should ensure that the annotation, once set, cannot be removed.
	ManagedByAnnotation = "forge.build/managed-by"

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

const (
	// TemplateSuffix is the object kind suffix used by template types.
	TemplateSuffix = "Template"
)
