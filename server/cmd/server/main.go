// Command server is the My Bad, Coach backend entry point.
//
// This tracer-bullet stub proves the binary imports and runs the shared point
// reducer before any HTTP or persistence wiring (ticket 07).
package main

import (
	"fmt"

	"github.com/alecbakholdin/mybadcoach/server/internal/reduce"
)

// initialViewLine reduces an empty point log and renders the initial view.
func initialViewLine() string {
	state := reduce.ReducePoint(nil, reduce.Offense)
	return "Initial view: " + string(state.View)
}

func main() {
	fmt.Println(initialViewLine())
}
