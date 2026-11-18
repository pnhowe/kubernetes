/*
Copyright 2025.

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

package e2e

import (
	"context"
	"fmt"
	"time"

	"t3kton.com/pkg/contractor"
)

// fixture is a Foundation and Structure in the demo contractor that the tests work on
type fixture struct {
	Foundation  string // the locator of the foundation
	StructureID int
}

// find looks up the Structure of the Foundation
func (f *fixture) find(ctx context.Context) error {
	client, err := contractor.GetClient(ctx)
	if err != nil {
		return err
	}

	foundationURI := "/api/v1/Building/Foundation:" + f.Foundation + ":"

	structures, err := client.BuildingStructureList(ctx, "", nil)
	if err != nil {
		return err
	}

	// read it all, so the listing's goroutine finishes
	for structure := range structures {
		if structure.ID != nil && structure.Foundation != nil && *structure.Foundation == foundationURI {
			f.StructureID = *structure.ID
		}
	}

	if f.StructureID == 0 {
		return fmt.Errorf("no structure found for foundation '%s', is the demo data loaded?", f.Foundation)
	}

	return nil
}

// locate gets the Foundation to the located state, the same way a machine that was booted into the bootstrap image does:
// by registering with the Cartographer (the survey), being assigned the Foundation, and saying it is done.
// The demo foundations have nothing else that would locate them, and they can't be built until they are.
func (f *fixture) locate(ctx context.Context) error {
	client, err := contractor.GetClient(ctx)
	if err != nil {
		return err
	}

	identifier := "e2e-" + f.Foundation
	if err = client.SurveyCartographerCallRegister(ctx, identifier); err != nil {
		return fmt.Errorf("register: %w", err)
	}

	cartographer := client.SurveyCartographerNewWithID(identifier)
	if err = cartographer.CallAssign(ctx, "/api/v1/Building/Foundation:"+f.Foundation+":"); err != nil {
		return fmt.Errorf("assign: %w", err)
	}

	if err = cartographer.CallDone(ctx); err != nil {
		return fmt.Errorf("done: %w", err)
	}

	return nil
}

// ensureFoundationBuilt makes sure the Foundation is built, the Structure can't be created without it
func (f *fixture) ensureFoundationBuilt(ctx context.Context) error {
	client, err := contractor.GetClient(ctx)
	if err != nil {
		return err
	}

	foundation, err := client.BuildingFoundationGet(ctx, f.Foundation)
	if err != nil {
		return err
	}

	if foundation.State != nil && *foundation.State == "planned" {
		if err = f.locate(ctx); err != nil {
			return fmt.Errorf("unable to locate foundation '%s': %w", f.Foundation, err)
		}
		if foundation, err = client.BuildingFoundationGet(ctx, f.Foundation); err != nil {
			return err
		}
	}

	if foundation.State != nil && *foundation.State == "located" {
		jobURI, err := foundation.CallGetJob(ctx)
		if err != nil {
			return err
		}
		if jobURI == "" { // not already building
			if _, err = foundation.CallDoCreate(ctx); err != nil {
				return fmt.Errorf("unable to build foundation '%s': %w", f.Foundation, err)
			}
		}
	}

	deadline := time.Now().Add(5 * time.Minute)
	for {
		foundation, err = client.BuildingFoundationGet(ctx, f.Foundation)
		if err != nil {
			return err
		}
		if foundation.State != nil && *foundation.State == "built" {
			return nil
		}
		if time.Now().After(deadline) {
			state := "unknown"
			if foundation.State != nil {
				state = *foundation.State
			}
			return fmt.Errorf("foundation '%s' is not built after 5 minutes, state '%s'", f.Foundation, state)
		}
		time.Sleep(2 * time.Second)
	}
}

// configValues returns the Structure's config values as they are in contractor right now
func (f *fixture) configValues(ctx context.Context) (map[string]any, error) {
	client, err := contractor.GetClient(ctx)
	if err != nil {
		return nil, err
	}

	structure, err := client.BuildingStructureGet(ctx, f.StructureID)
	if err != nil {
		return nil, err
	}

	if structure.ConfigValues == nil {
		return map[string]any{}, nil
	}
	return *structure.ConfigValues, nil
}

// setConfigValues changes the Structure's config values directly in contractor, like someone using the UI would
func (f *fixture) setConfigValues(ctx context.Context, values map[string]any) error {
	client, err := contractor.GetClient(ctx)
	if err != nil {
		return err
	}

	structure := client.BuildingStructureNewWithID(f.StructureID)
	structure.ConfigValues = &values
	return structure.Update(ctx)
}

// state returns the Structure's state, and if it has a job, as they are in contractor right now
func (f *fixture) state(ctx context.Context) (string, bool, error) {
	client, err := contractor.GetClient(ctx)
	if err != nil {
		return "", false, err
	}

	structure, err := client.BuildingStructureGet(ctx, f.StructureID)
	if err != nil {
		return "", false, err
	}

	jobURI, err := structure.CallGetJob(ctx)
	if err != nil {
		return "", false, err
	}

	state := ""
	if structure.State != nil {
		state = *structure.State
	}
	return state, jobURI != "", nil
}
