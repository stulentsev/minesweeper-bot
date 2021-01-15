package main

import (
	"context"
	"fmt"
	"io"
	"minesweeper-bot/swagger"
	"os"
	"sort"
	"strings"
)

func main() {
	configuration := swagger.NewConfiguration()
	configuration.BasePath = "http://localhost:3000"
	client := swagger.NewAPIClient(configuration)

	gamesToPlay := 1000
	results := make(map[string]int)
	progress := make(map[int]int)
	for i := 0; i < gamesToPlay; i++ {
		thisGameResult := playNewGame(client)
		results[thisGameResult.Status]++

		progress[thisGameResult.MinesFound]++

		fmt.Println(results)
	}
	printProgressStats(progress)
}

func printProgressStats(gamesByMinesFound map[int]int) {
	fmt.Println("progress in lost games")
	minesFoundKeys := make([]int, 0, len(gamesByMinesFound))
	for key := range gamesByMinesFound {
		minesFoundKeys = append(minesFoundKeys, key)
	}
	sort.Ints(minesFoundKeys)

	for _, key := range minesFoundKeys {
		fmt.Printf("Mines found: %d, games: %d\n", key, gamesByMinesFound[key])
	}
}

type gameResult struct {
	Status     string
	MinesFound int
	MinesTotal int
}

func (gr gameResult) MinesFoundPercentage() float64 {
	return float64(gr.MinesFound) / float64(gr.MinesTotal)
}

func playNewGame(client *swagger.APIClient) gameResult {
	initialGame, _, err := client.DefaultApi.NewgamePost(context.Background())
	if err != nil {
		panic(err)
	}
	gameInfo := newGameInfo(initialGame)
	//fmt.Println(gameInfo.PrettyBoardState)
	// initial move, guaranteed safe
	initialCell := location{
		X: int(gameInfo.BoardWidth / 2),
		Y: int(gameInfo.BoardHeight / 2),
	}
	gameInfo.queueCellToOpen(initialCell)
	var currentTurnNumber int

	for {
		gameInfo.addFullyRevealedLocations()

		for len(gameInfo.cellsToOpen) > 0 {
			cell := gameInfo.cellsToOpen[0]
			gameInfo.cellsToOpen = gameInfo.cellsToOpen[1:]

			if gameInfo.fetchCell(cell.X, cell.Y) != "?" {
				continue
			}
			*gameInfo.Game = move(client, gameInfo, cell)

			if gameInfo.IsFinished() {
				//fmt.Println(gameInfo.PrettyBoardState)
				//fmt.Println("returning due to", gameInfo.Status)
				return gameInfo.Result()
			}
			//fmt.Printf("turn %d, opening (%d, %d) from queue\n", currentTurnNumber, cell.X, cell.Y)

			gameInfo.refreshBombs()
			printBoardState(os.Stdout, gameInfo)
			currentTurnNumber++
		}

		gameInfo.findSafeCells()

		if len(gameInfo.cellsToOpen) == 0 {
			loc, err := gameInfo.findLeastRiskyCell()
			if err != nil {
				return gameResult{
					Status:     "unsure",
					MinesFound: gameInfo.NumberOfCorrectlyGuessedBombs(),
					MinesTotal: int(gameInfo.MinesCount),
				}
			}
			gameInfo.queueCellToOpen(loc)
		}
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

func printBoardState(w io.Writer, game gameInformation) {
	leftTopCorner := "\u250c"
	rightTopCorner := "\u2510"
	leftBottomCorner := "\u2514"
	rightBottomCorner := "\u2518"

	horizontalLine := "\u2500"
	verticalLine := "\u2502"

	_, _ = fmt.Fprint(w, "  ")
	for i := 0; i < int(game.BoardWidth); i++ {
		if i%10 == 0 {
			_, _ = fmt.Fprint(w, i)
		} else {
			_, _ = fmt.Fprint(w, ".")
		}
		_, _ = fmt.Fprint(w, " ")
	}
	_, _ = fmt.Fprintln(w, "")

	_, _ = fmt.Fprintf(w, "  %s%s%s\n", leftTopCorner, strings.Repeat(horizontalLine, int(game.BoardWidth)*2), rightTopCorner)
	for i := 0; i < int(game.BoardHeight); i++ {
		_, _ = fmt.Fprintf(w, "%2d%s", i, verticalLine)
		for j := 0; j < int(game.BoardWidth); j++ {
			idx := j + i*int(game.BoardWidth)
			_, _ = fmt.Fprint(w, game.BoardState[idx])
			_, _ = fmt.Fprint(w, " ")
		}
		_, _ = fmt.Fprintf(w, "%s%d\n", verticalLine, i)
	}

	_, _ = fmt.Fprintf(w, "  %s%s%s\n", leftBottomCorner, strings.Repeat(horizontalLine, int(game.BoardWidth)*2), rightBottomCorner)

	_, _ = fmt.Fprint(w, "  ")
	for i := 0; i < int(game.BoardWidth); i++ {
		if i%10 == 0 {
			_, _ = fmt.Fprint(w, i)
		} else {
			_, _ = fmt.Fprint(w, ".")
		}
		_, _ = fmt.Fprint(w, " ")
	}
	_, _ = fmt.Fprintln(w, "")
}
