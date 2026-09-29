package maxpressure

import (
	"github.com/LdDl/go-gmns/gmns"
	"github.com/LdDl/greenwave/junction"
)

// StagesFromJunction is a compatibility wrapper. CompileProgram also returns
// timing windows and validation errors and is preferred for new callers.
func StagesFromJunction(jun *junction.Junction, groupConnectors map[junction.GroupID][]gmns.LinkID) []Stage {
	program, err := CompileProgram(jun, groupConnectors)
	if err != nil {
		return nil
	}
	return program.Stages
}
