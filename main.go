package main

import (
	"context"
	"fmt"
	"io"
	"os"
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

			if gameInfo.fetchCell(cell.X, cell.Y) != "?" {
				continue
			}
			*gameInfo.Game = move(client, gameInfo, cell)

			if gameInfo.IsFinished() {
				fmt.Println(gameInfo.PrettyBoardState)
				fmt.Println("returning due to", gameInfo.Status)
				break GameLoop
			}
			fmt.Printf("turn %d, opening (%d, %d) from queue\n", currentTurnNumber, cell.X, cell.Y)

			gameInfo.refreshBombs()
			printBoardState(os.Stdout, gameInfo)
			currentTurnNumber++
		}

		gameInfo.findSafeCells()

		if len(gameInfo.cellsToOpen) == 0 {
			loc, err := gameInfo.findLeastRiskyCell()
			if err != nil {
				panic("no obvious turn candidates!")
			}
			gameInfo.queueCellToOpen(loc)
		}
	}
	fmt.Println("finished")
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
