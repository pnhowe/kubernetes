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
	"errors"
)

// validateHost validates Structure resource for creation.
func (s *Structure) validateStructure() []error {
	var errs []error

	if s.Spec.BluePrint != "" {
		if err := validateBluePrint(s.Spec.BluePrint); err != nil {
			errs = append(errs, err)
		}
	}

	if err := validateConfigurationValues(s.Spec.ConfigurationValues); err != nil {
		errs = append(errs, err)
	}

	return errs
}

// validateChanges validates Structure resource on changes
// but also covers the validations of creation.
func (s *Structure) validateChanges(old *Structure) []error {
	var errs []error

	if err := s.validateStructure(); err != nil {
		errs = append(errs, err...)
	}

	if s.Spec.BluePrint != old.Spec.BluePrint &&
		(old.Status.State != "planned" || s.Status.State != "planned" ||
			old.Spec.State != "planned" || s.Spec.State != "planned") {
		errs = append(errs, errors.New("can not change the BluePrint while not in 'Planned' State"))
	}

	if s.Spec.BluePrint != old.Spec.BluePrint &&
		(old.Status.Job != nil || s.Status.Job != nil) {
		errs = append(errs, errors.New("can not change the BluePrint while there is a Job"))
	}

	if s.Spec.State != old.Spec.State &&
		(old.Status.Job != nil || s.Status.Job != nil) {
		errs = append(errs, errors.New("can not change the State while there is a Job"))
	}

	return errs
}

func validateBluePrint(blueprint string) error {
	// if err != nil {
	// 	return fmt.Errorf("blueprint %s is invalid: %w", imageURL, err)
	// }

	return nil
}

func validateConfigurationValues(configurationValues map[string]ConfigValue) error {
	// if err != nil {
	// 	return fmt.Errorf("blueprint %s is invalid: %w", imageURL, err)
	// }

	return nil
}
