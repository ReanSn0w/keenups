package driver

import (
	"time"

	"github.com/reansnow/keenups/internal/state"
)

func stateForTest() *state.Store { return state.New(0 * time.Second) }
