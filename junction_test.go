package greenwave

import "github.com/LdDl/greenwave/color"

func basicTestJuntions() []*Junction {
	junctions := []*Junction{
		NewJunction([]*Phase{
			NewPhase(0, []*Signal{
				NewSignal(30, color.GREEN),
				NewSignal(20, color.RED),
			}),
			NewPhase(1, []*Signal{
				NewSignal(20, color.GREEN),
				NewSignal(15, color.RED),
			}),
		}),
		NewJunction([]*Phase{
			NewPhase(10, []*Signal{
				NewSignal(20, color.RED),
				NewSignal(35, color.GREEN),
				NewSignal(5, color.YELLOW),
			}),
			NewPhase(11, []*Signal{
				NewSignal(10, color.RED),
				NewSignal(10, color.GREEN),
				NewSignal(5, color.YELLOW),
			}),
		}),
		NewJunction([]*Phase{
			NewPhase(20, []*Signal{
				NewSignal(45, color.RED),
				NewSignal(10, color.GREEN),
			}),
			NewPhase(21, []*Signal{
				NewSignal(7, color.RED),
				NewSignal(18, color.GREEN),
				NewSignal(5, color.YELLOW),
			}),
		}),
		NewJunction([]*Phase{
			NewPhase(20, []*Signal{
				NewSignal(40, color.RED),
				NewSignal(15, color.GREEN),
			}),
			NewPhase(21, []*Signal{
				NewSignal(10, color.RED),
				NewSignal(20, color.GREEN),
			}),
		}),
	}

	return junctions
}
