package main

import (
	"fmt"

	"github.com/ewilliams-labs/overture/backend/internal/core/domain"
)

func main() {
	fmt.Println("Overture API starting...")
	pl := domain.Playlist{Name: "My First Orchestration"}
	fmt.Println(pl.Name)
}
