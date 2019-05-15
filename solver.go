package main

import (
	"fmt"
	"minesweeper-bot/swagger"
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

func (gi *gameInformation) queueCellToOpen(cell location) {
	for _, loc := range gi.cellsToOpen {
		if cell == loc {
			return
		}
	}

	gi.cellsToOpen = append(gi.cellsToOpen, cell)
	fmt.Println(gi.cellsToOpen)
}

func (gi *gameInformation) addFullyRevealedLocations() {
	for offset := range gi.BoardState {
		y := offset / int(gi.BoardWidth)
		x := offset - y*int(gi.BoardWidth)

		loc := location{x, y}
		if gi.fullyRevealedLocations[loc] {
			continue
		}

		unknownLocs := findUnknownCellsAround(*gi, x, y)
		if len(unknownLocs) == 0 {
			gi.fullyRevealedLocations[loc] = true
		}
	}
}
