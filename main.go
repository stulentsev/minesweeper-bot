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

func main() {
	configuration := swagger.NewConfiguration()
	configuration.BasePath = "http://localhost:3000"
	client := swagger.NewAPIClient(configuration)

	initialGame, _, err := client.DefaultApi.NewgamePost(context.Background())
	if err != nil {
		panic(err)
	}

	gameInfo := newGameInfo(initialGame)
	fmt.Println(gameInfo.PrettyBoardState)

	// initial move, guaranteed safe
	initialCell := location{
		X: int(gameInfo.BoardWidth / 2),
		Y: int(gameInfo.BoardHeight / 2),
	}
	gameInfo.queueCellToOpen(initialCell)

	var currentTurnNumber int

GameLoop:
	for {
		gameInfo.addFullyRevealedLocations()

		for len(gameInfo.cellsToOpen) > 0 {
			cell := gameInfo.cellsToOpen[0]
			gameInfo.cellsToOpen = gameInfo.cellsToOpen[1:]

			if fetchCell(gameInfo, cell.X, cell.Y) != "?" {
				continue
			}
			*gameInfo.Game = move(client, gameInfo, cell)

			if gameIsFinished(gameInfo) {
				fmt.Println(gameInfo.PrettyBoardState)
				fmt.Println("returning due to", gameInfo.Status)
				break GameLoop
			}
			fmt.Printf("turn %d, opening (%d, %d) from queue\n", currentTurnNumber, cell.X, cell.Y)

			refreshBombs(gameInfo, gameInfo.bombLocations)
			printBoardState(os.Stdout, gameInfo)
			currentTurnNumber++
		}

		findSafeCells(&gameInfo)

		if len(gameInfo.cellsToOpen) == 0 {
			loc, err := findLeastRiskyCell(gameInfo)
			if err != nil {
				panic("no obvious turn candidates!")
			}
			gameInfo.queueCellToOpen(loc)
		}
	}
	fmt.Println("finished")
}

func refreshBombs(game gameInformation, bombLocations map[location]bool) {
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
}

func move(client *swagger.APIClient, game gameInformation, cell location) swagger.Game {
	newGameState, _, err := client.DefaultApi.MovePost(context.Background(), swagger.MoveInfo{
		GameId: game.GameId,
		X:      int32(cell.X),
		Y:      int32(cell.Y),
	})
	if err != nil {
		panic(err)
	}
	return newGameState
}

func markNewBombs(game gameInformation) []location {
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

		//fmt.Printf("looking for %d bombs around (%d,%d) offset %d\n", count, x, y, offset)

		locs := findUnknownCellsAround(game, x, y)

		//fmt.Printf("found %d unknown cells, but already see %d bombs\n", len(locs), len(bombLocs))

		if len(locs)+len(bombLocs) == count {
			result = locs
			break
		}
	}
	return result
}

func findUnknownCellsAround(game gameInformation, x int, y int) []location {
	return findCellsAround(game, x, y, "?")
}

func findBombsAround(game gameInformation, x int, y int) []location {
	return findCellsAround(game, x, y, "*")
}

func findCellsAround(game gameInformation, x int, y int, marker string) []location {
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

func fetchCell(game gameInformation, x, y int) string {
	offset := y*int(game.BoardWidth) + x
	return game.BoardState[offset]
}

func applyBombLocations(game gameInformation, bombLocations map[location]bool) {
	for loc := range bombLocations {
		offset := loc.Y*int(game.BoardWidth) + loc.X
		game.BoardState[offset] = "*"
	}
}

// if we got to this point, then multiple cells can contain a bomb. Some more likely than others.
// findLeastRiskyCells evaluates/intersects area of effect of numbered cells and tries to guess which
// cells are more likely to contain a bomb. And, as a consequence, we get the "least likely" cells.
func findLeastRiskyCell(game gameInformation) (location, error) {
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

		visibleBombs := findBombsAround(game, x, y)
		unknowns := findUnknownCellsAround(game, x, y)
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

func findSafeCells(game *gameInformation) {
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

		bombLocs := findBombsAround(*game, x, y)
		if len(bombLocs) == count { // cell at (x,y) already sees all its bombs. It's safe to open all unknowns
			unknownLocs := findUnknownCellsAround(*game, x, y)
			if len(unknownLocs) > 0 {
				for _, loc := range unknownLocs {
					game.queueCellToOpen(loc)
				}
			}
		}
	}
}

func gameIsFinished(game gameInformation) bool {
	return game.Status != ""
}

func printBoardState(w io.Writer, game gameInformation) {
	leftTopCorner := "\u250c"
	rightTopCorner := "\u2510"
	leftBottomCorner := "\u2514"
	rightBottomCorner := "\u2518"

	horizontalLine := "\u2500"
	verticalLine := "\u2502"

	_, _ = fmt.Fprint(w, "  ")
	for i := 0; i < int(game.BoardWidth); i++ {
		_, _ = fmt.Fprint(w, i)
		_, _ = fmt.Fprint(w, " ")
	}
	_, _ = fmt.Fprintln(w, "")

	_, _ = fmt.Fprintf(w, " %s%s%s\n", leftTopCorner, strings.Repeat(horizontalLine, int(game.BoardWidth)*2), rightTopCorner)
	for i := 0; i < int(game.BoardHeight); i++ {
		_, _ = fmt.Fprintf(w, "%d%s", i, verticalLine)
		for j := 0; j < int(game.BoardWidth); j++ {
			idx := j + i*int(game.BoardWidth)
			_, _ = fmt.Fprint(w, game.BoardState[idx])
			_, _ = fmt.Fprint(w, " ")
		}
		_, _ = fmt.Fprintf(w, "%s%d\n", verticalLine, i)
	}

	_, _ = fmt.Fprintf(w, " %s%s%s\n", leftBottomCorner, strings.Repeat(horizontalLine, int(game.BoardWidth)*2), rightBottomCorner)

	_, _ = fmt.Fprint(w, "  ")
	for i := 0; i < int(game.BoardWidth); i++ {
		_, _ = fmt.Fprint(w, i)
		_, _ = fmt.Fprint(w, " ")
	}
	_, _ = fmt.Fprintln(w, "")
}
