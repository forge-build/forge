/*
Copyright 2024 The Forge Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

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

	// ProvisionerFailedError indicates that the provisioner failed.
	ProvisionerFailedError BuildStatusError = "ProvisionerFailed"
)
