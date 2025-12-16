package main

import (
	"strconv"

	"github.com/bravo68web/githut/internal/server"
	"github.com/bravo68web/githut/internal/transport/http/router"
)

func main() {
	s := server.New()

	s.DB.AutoMigrate()

	router.NewRouter(s).RegisterRoutes()

	if err := s.Run(":" + strconv.Itoa(s.Config.Server.Port)); err != nil {
		panic(err)
	}
}
