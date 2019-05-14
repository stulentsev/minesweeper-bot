package main

import (
	"context"
	"fmt"
)
import "minesweeper-bot/swagger"

func main() {
	configuration := swagger.NewConfiguration()
	configuration.BasePath = "http://localhost:3000"
	client := swagger.NewAPIClient(configuration)
	game, _, err := client.DefaultApi.NewgamePost(context.Background())
	if err != nil {
		panic(err)
	}
	fmt.Println(game.PrettyBoardState)
}
