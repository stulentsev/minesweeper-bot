package main

import (
	"fmt"
	"minesweeper-bot/swagger"
	"strconv"
)

type location struct {
	X, Y int
}

type gameInformation struct {
	*swagger.Game
	cellsToOpen   []location
	bombLocations map[location]bool

	fullyRevealedLocations map[location]bool
}

func newGameInfo(game swagger.Game) gameInformation {
	return gameInformation{
		Game:                   &game,
		cellsToOpen:            make([]location, 0),
		bombLocations:          make(map[location]bool),
		fullyRevealedLocations: make(map[location]bool),
	}
}

func (game *gameInformation) queueCellToOpen(cell location) {
	for _, loc := range game.cellsToOpen {
		if cell == loc {
			return
		}
	}

	game.cellsToOpen = append(game.cellsToOpen, cell)
	fmt.Println(game.cellsToOpen)
}

func (game *gameInformation) addFullyRevealedLocations() {
	for offset := range game.BoardState {
		y := offset / int(game.BoardWidth)
		x := offset - y*int(game.BoardWidth)

		loc := location{x, y}
		if game.fullyRevealedLocations[loc] {
			continue
		}

		unknownLocs := game.findUnknownCellsAround(x, y)
		if len(unknownLocs) == 0 {
			game.fullyRevealedLocations[loc] = true
		}
	}
}

func (game *gameInformation) refreshBombs() {
	newBombLocs := game.markNewBombs()
	for len(newBombLocs) > 0 {
		game.applyBombLocations(game.bombLocations)
		for _, loc := range newBombLocs {
			if !game.bombLocations[loc] {
				fmt.Println("found new bomb", loc)
				game.bombLocations[loc] = true
			}
		}
		game.applyBombLocations(game.bombLocations)
		newBombLocs = game.markNewBombs()
	}
}

func (game *gameInformation) markNewBombs() []location {
	result := make([]location, 0)
	for offset, cell := range game.BoardState {
		count, err := strconv.Atoi(cell)
		if err != nil {
			continue
		}
		y := offset / int(game.BoardWidth)
		x := offset - y*int(game.BoardWidth)
		bombLocs := game.findBombsAround(x, y)
		if len(bombLocs) == count {
			continue
		}

		//fmt.Printf("looking for %d bombs around (%d,%d) offset %d\n", count, x, y, offset)

		locs := game.findUnknownCellsAround(x, y)

		//fmt.Printf("found %d unknown cells, but already see %d bombs\n", len(locs), len(bombLocs))

		if len(locs)+len(bombLocs) == count {
			result = locs
			break
		}
	}
	return result
}

func (game *gameInformation) findUnknownCellsAround(x int, y int) []location {
	return game.findCellsAround(x, y, "?")
}

func (game *gameInformation) findBombsAround(x int, y int) []location {
	return game.findCellsAround(x, y, "*")
}

func (game *gameInformation) findCellsAround(x int, y int, marker string) []location {
	result := make([]location, 0)
	for i := x - 1; i <= x+1; i++ {
		for j := y - 1; j <= y+1; j++ {
			if i == x && j == y || i < 0 || j < 0 || i >= int(game.BoardWidth) || j >= int(game.BoardHeight) {
				continue
			}
			if game.fetchCell(i, j) == marker {
				result = append(result, location{X: i, Y: j})
			}
		}
	}
	return result
}

func (game *gameInformation) fetchCell(x, y int) string {
	offset := y*int(game.BoardWidth) + x
	return game.BoardState[offset]
}

func (game *gameInformation) applyBombLocations(bombLocations map[location]bool) {
	for loc := range bombLocations {
		offset := loc.Y*int(game.BoardWidth) + loc.X
		game.BoardState[offset] = "*"
	}
}

// if we got to this point, then multiple cells can contain a bomb. Some more likely than others.
// findLeastRiskyCells evaluates/intersects area of effect of numbered cells and tries to guess which
// cells are more likely to contain a bomb. And, as a consequence, we get the "least likely" cells.
func (game *gameInformation) findLeastRiskyCell() (location, error) {
	probabilitiesOfBomb := make(map[location]float64)

	for offset, cellState := range game.BoardState {
		y := offset / int(game.BoardWidth)
		x := offset - y*int(game.BoardWidth)

		if game.fullyRevealedLocations[location{x, y}] {
			continue
		}

		count, err := strconv.Atoi(cellState)
		if err != nil { // not a numbered cell
			continue
		}

		visibleBombs := game.findBombsAround(x, y)
		unknowns := game.findUnknownCellsAround(x, y)
		for _, loc := range unknowns {
			additionalRisk := float64(count-len(visibleBombs)) / float64(len(unknowns))
			fmt.Printf("new risk from cell (%d,%d) for cell %v: %f\n", x, y, loc, additionalRisk)
			probabilitiesOfBomb[loc] += additionalRisk
		}
	}

	fmt.Println("bomb probabilities", probabilitiesOfBomb)
	// find loc with lowest probability
	var leastRiskyLoc location
	var leastRisk float64
	found := false

	for loc, risk := range probabilitiesOfBomb {
		if !found {
			leastRiskyLoc = loc
			leastRisk = risk
			found = true
		} else {
			if risk < leastRisk {
				fmt.Printf("new least risky cell %f %v\n", risk, loc)
				leastRisk = risk
				leastRiskyLoc = loc
			}
		}
	}
	if found {
		return leastRiskyLoc, nil
	}
	return location{}, fmt.Errorf("can't find least risky cell")
}

func (game *gameInformation) findSafeCells() {
	for offset, cellState := range game.BoardState {
		y := offset / int(game.BoardWidth)
		x := offset - y*int(game.BoardWidth)

		if game.fullyRevealedLocations[location{x, y}] {
			continue
		}
		count, err := strconv.Atoi(cellState)
		if err != nil {
			continue
		}
		//fmt.Printf("safe: looking for %d bombs around (%d,%d) offset %d\n", count, x, y, offset)

		bombLocs := game.findBombsAround(x, y)
		if len(bombLocs) == count { // cell at (x,y) already sees all its bombs. It's safe to open all unknowns
			unknownLocs := game.findUnknownCellsAround(x, y)
			if len(unknownLocs) > 0 {
				for _, loc := range unknownLocs {
					game.queueCellToOpen(loc)
				}
			}
		}
	}
}

func (game *gameInformation) IsFinished() bool {
	return game.Status != ""
}
