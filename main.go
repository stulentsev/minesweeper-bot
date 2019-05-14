package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)
import "minesweeper-bot/swagger"

var cellsToOpen = make([]location, 0)

func main() {
	configuration := swagger.NewConfiguration()
	configuration.BasePath = "http://localhost:3000"
	client := swagger.NewAPIClient(configuration)
	game, _, err := client.DefaultApi.NewgamePost(context.Background())
	if err != nil {
		panic(err)
	}
	fmt.Println(game.PrettyBoardState)

	// initial move, guaranteed safe
	initialCell := location{
		X: int(game.BoardWidth / 2),
		Y: int(game.BoardHeight / 2),
	}
	queueCellToOpen(initialCell)

	bombLocations := make(map[location]bool)

	var currentTurnNumber int

GameLoop:
	for {
		for len(cellsToOpen) > 0 {
			cell := cellsToOpen[0]
			cellsToOpen = cellsToOpen[1:]

			if fetchCell(game, cell.X, cell.Y) != "?" {
				continue
			}
			game, _, err = client.DefaultApi.MovePost(context.Background(), swagger.MoveInfo{
				GameId: game.GameId,
				X:      int32(cell.X),
				Y:      int32(cell.Y),
			})
			if err != nil {
				panic(err)
			}
			if gameIsFinished(game) {
				fmt.Println(game.PrettyBoardState)
				fmt.Println("returning due to", game.Status)
				break GameLoop
			}
			fmt.Printf("turn %d, opening (%d, %d) from queue\n", currentTurnNumber, cell.X, cell.Y)

			// refresh bombs
			newBombLocs := markNewBombs(game)
			for len(newBombLocs) > 0 {
				applyBombLocations(game, bombLocations)
				for _, loc := range newBombLocs {
					if !bombLocations[loc] {
						fmt.Println("found new bomb", loc)
						bombLocations[loc] = true
					}
				}
				applyBombLocations(game, bombLocations)
				newBombLocs = markNewBombs(game)
			}
			printBoardState(os.Stdout, game)
			currentTurnNumber++

		}

		findSafeCells(game)

		if len(cellsToOpen) == 0 {
			panic("no obvious turn candidates!")
		}
	}
	fmt.Println("finished")
}

func queueCellToOpen(cell location) {
	for _, loc := range cellsToOpen {
		if cell == loc {
			return
		}
	}
	cellsToOpen = append(cellsToOpen, cell)
}

func markNewBombs(game swagger.Game) []location {
	result := make([]location, 0)
	for offset, cell := range game.BoardState {
		count, err := strconv.Atoi(cell)
		if err != nil {
			continue
		}
		y := offset / int(game.BoardWidth)
		x := offset - y*int(game.BoardWidth)
		bombLocs := findBombsAround(game, x, y)
		if len(bombLocs) == count {
			continue
		}

		fmt.Printf("looking for %d bombs around (%d,%d) offset %d\n", count, x, y, offset)

		locs := findUnknownCellsAround(game, x, y)

		fmt.Printf("found %d unknown cells, but already see %d bombs\n", len(locs), len(bombLocs))

		if len(locs)+len(bombLocs) == count {
			result = locs
			break
		}
	}
	return result
}

func findUnknownCellsAround(game swagger.Game, x int, y int) []location {
	return findCellsAround(game, x, y, "?")
}

func findBombsAround(game swagger.Game, x int, y int) []location {
	return findCellsAround(game, x, y, "*")
}

func findCellsAround(game swagger.Game, x int, y int, marker string) []location {
	result := make([]location, 0)
	for i := x - 1; i <= x+1; i++ {
		for j := y - 1; j <= y+1; j++ {
			if i == x && j == y || i < 0 || j < 0 || i >= int(game.BoardWidth) || j >= int(game.BoardHeight) {
				continue
			}
			if fetchCell(game, i, j) == marker {
				result = append(result, location{X: i, Y: j})
			}
		}
	}
	return result
}

func fetchCell(game swagger.Game, x, y int) string {
	offset := y*int(game.BoardWidth) + x
	return game.BoardState[offset]
}

func applyBombLocations(game swagger.Game, bombLocations map[location]bool) {
	for loc := range bombLocations {
		offset := loc.Y*int(game.BoardWidth) + loc.X
		game.BoardState[offset] = "*"
	}
}

func findSafeCells(game swagger.Game) {
	for offset, cell := range game.BoardState {
		count, err := strconv.Atoi(cell)
		if err != nil {
			continue
		}
		y := offset / int(game.BoardWidth)
		x := offset - y*int(game.BoardWidth)
		//fmt.Printf("safe: looking for %d bombs around (%d,%d) offset %d\n", count, x, y, offset)

		bombLocs := findBombsAround(game, x, y)
		if len(bombLocs) == count { // cell at (x,y) already sees all its bombs. It's safe to open all unknowns
			unknownLocs := findUnknownCellsAround(game, x, y)
			if len(unknownLocs) > 0 {
				for _, loc := range unknownLocs {
					fmt.Printf("queueing %v from cell (%d,%d)\n", loc, x, y)

					queueCellToOpen(loc)
				}
			}
		}
	}
}

func gameIsFinished(game swagger.Game) bool {
	return game.Status != ""
}

func printBoardState(w io.Writer, game swagger.Game) {
	leftTopCorner := "\u250c"
	rightTopCorner := "\u2510"
	leftBottomCorner := "\u2514"
	rightBottomCorner := "\u2518"

	horizontalLine := "\u2500"
	verticalLine := "\u2502"

	fmt.Fprint(w, "  ")
	for i := 0; i < int(game.BoardWidth); i++ {
		fmt.Fprint(w, i)
		fmt.Fprint(w, " ")
	}
	fmt.Fprintln(w, "")

	fmt.Fprintf(w, " %s%s%s\n", leftTopCorner, strings.Repeat(horizontalLine, int(game.BoardWidth)*2), rightTopCorner)
	for i := 0; i < int(game.BoardHeight); i++ {
		fmt.Fprintf(w, "%d%s", i, verticalLine)
		for j := 0; j < int(game.BoardWidth); j++ {
			idx := j + i*int(game.BoardHeight)
			fmt.Fprint(w, game.BoardState[idx])
			fmt.Fprint(w, " ")
		}
		fmt.Fprintf(w, "%s%d\n", verticalLine, i)
	}

	fmt.Fprintf(w, " %s%s%s\n", leftBottomCorner, strings.Repeat(horizontalLine, int(game.BoardWidth)*2), rightBottomCorner)

	fmt.Fprint(w, "  ")
	for i := 0; i < int(game.BoardWidth); i++ {
		fmt.Fprint(w, i)
		fmt.Fprint(w, " ")
	}
	fmt.Fprintln(w, "")
}
