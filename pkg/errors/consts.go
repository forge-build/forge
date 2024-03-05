package errors

// BuildStatusError defines errors states for Build objects.
type BuildStatusError string

const (
	// InvalidConfigurationBuildError indicates that the Build
	// configuration is invalid.
	InvalidConfigurationBuildError BuildStatusError = "InvalidConfiguration"

	// UnsupportedChangeBuildError indicates that the Build
	// spec has been updated in an unsupported way. That cannot be
	// reconciled.
	UnsupportedChangeBuildError BuildStatusError = "UnsupportedChange"

	// CreateBuildError indicates that an error was encountered
	// when trying to create the Build.
	CreateBuildError BuildStatusError = "CreateError"

	// UpdateBuildError indicates that an error was encountered
	// when trying to update the Build.
	UpdateBuildError BuildStatusError = "UpdateError"

	// DeleteBuildError indicates that an error was encountered
	// when trying to delete the Build.
	DeleteBuildError BuildStatusError = "DeleteError"
)
