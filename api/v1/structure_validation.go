/*
Copyright 2024.

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

package v1

import (
	"context"
	"errors"
	"fmt"
	"regexp"

	client "github.com/t3kton/contractor_goclient"
)

var config_name_regex = regexp.MustCompile(`^[<>\-~]?[a-zA-Z0-9][a-zA-Z0-9_\-]*(:[a-zA-Z0-9]+)?$`)

// ValidateStructure Validates that the structure is valid
func (s *Structure) ValidateStructure(ctx context.Context, client *client.Contractor) []error {
	var errs []error

	if s.Spec.ID == 0 {
		errs = append(errs, fmt.Errorf("ID not specified"))
	} else {
		_, err := client.BuildingStructureGet(ctx, s.Spec.ID)
		if err != nil {
			errs = append(errs, fmt.Errorf("structure not found"))
		}
	}

	if s.Spec.BluePrint == "" { // TODO: We need to make sure the blueprint is valid for the foundation/structure combination also
		errs = append(errs, fmt.Errorf("blueprint not specified"))
	} else {
		_, err := client.BlueprintStructureBluePrintGet(ctx, s.Spec.BluePrint)
		if err != nil {
			errs = append(errs, fmt.Errorf("blueprint not found"))
		}
	}

	if err := validateConfigValues(s.Spec.ConfigValues); err != nil {
		errs = append(errs, err)
	}

	return errs
}

// ValidateChanges validates that changes happening to the structure are valid
func (s *Structure) ValidateChanges(ctx context.Context, client *client.Contractor, old *Structure) []error {
	var errs []error

	if err := s.ValidateStructure(ctx, client); err != nil {
		errs = append(errs, err...)
	}

	if s.Spec.ID != old.Spec.ID {
		errs = append(errs, errors.New("can not change the ID"))
	}

	if s.Spec.BluePrint != old.Spec.BluePrint {
		if old.Status.State != "planned" || s.Status.State != "planned" ||
			old.Spec.State != "planned" || s.Spec.State != "planned" {
			errs = append(errs, errors.New("can not change the BluePrint while not in 'Planned' State"))
		}

		if old.Status.Job != nil || s.Status.Job != nil {
			errs = append(errs, errors.New("can not change the BluePrint while there is a Job"))
		}
	}

	if s.Spec.State != old.Spec.State &&
		(old.Status.Job != nil || s.Status.Job != nil) {
		errs = append(errs, errors.New("can not change the State while there is a Job"))
	}

	// TODO: do we allow changing confivalues at any time?  I think so?
	// TODO: make sure the blueprint value is valid

	return errs
}

func validateConfigValues(configurationValues map[string]ConfigValue) error {
	for name := range configurationValues {
		if !config_name_regex.MatchString(name) {
			return fmt.Errorf("invalid configuration value name '%s'", name)
		}
	}

	return nil
}
