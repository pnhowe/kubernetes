package v1

import (
	"testing"
)

func TestStructureSpecValidation(t *testing.T) {
	// ID can not change
	// blueprint can not change if not curently planned, and there is no Job
	// Configuration Values can Change at any time
	// state can't change if there is a job
	// test Valid configuration values/names
	// config_name_regex = re.compile( r'^[<>\-~]?[a-zA-Z0-9][a-zA-Z0-9_\-]*(:[a-zA-Z0-9]+)?$' )
}

func TestStructureNeedsJob(t *testing.T) {
	// change in state and no existing job
	// not when state does not change
	// not shen already has job
}
