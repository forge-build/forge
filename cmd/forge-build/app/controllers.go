/*
Copyright 2024 The Forge contributors.

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

package app

import (
	"fmt"

	"github.com/forge-build/forge/cmd/forge-build/app/options"
	buildcontroller "github.com/forge-build/forge/pkg/controllers/build"
	shellcontroller "github.com/forge-build/forge/provisioner/shell/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller"
)

type controllerCreator func(*options.ControllerContext) error

// AllControllers stores the list of all controllers that we want to run,
// each entry holds the name of the controller and the corresponding
// start function that will essentially run the controller.
var AllControllers = map[string]controllerCreator{
	buildcontroller.ControllerName: createBuildController,
	shellcontroller.ControllerName: createShellController,
}

func createAllControllers(ctrlCtx *options.ControllerContext) error {
	for name, create := range AllControllers {
		if err := create(ctrlCtx); err != nil {
			return fmt.Errorf("failed to create %q controller: %w", name, err)
		}
	}

	return nil
}

func createBuildController(ctrlCtx *options.ControllerContext) error {
	return buildcontroller.Add(ctrlCtx.Ctx, ctrlCtx.Mgr, 1, *ctrlCtx.Log, controller.Options{MaxConcurrentReconciles: ctrlCtx.RunOptions.WorkerNumber})
}

func createShellController(ctrlCtx *options.ControllerContext) error {
	return shellcontroller.Add(ctrlCtx.Ctx, ctrlCtx.Mgr, *ctrlCtx.Log, "forge-core")
}
