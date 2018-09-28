package main

import (
	"log"
	"net/http"

	"github.com/labstack/echo"
	"github.com/labstack/echo/engine/standard"
	"github.com/labstack/echo/middleware"
)

func main() {

	// Initialize echo webserver
	echoServer := GetWebSvr()

	std := standard.New(":8082")
	std.SetHandler(echoServer)

	// gracehttp.Serve(echoServer.Server)
	log.Fatal(std.Start())
}

func GetWebSvr() *echo.Echo {
	// Initialize echo webserver
	echoServer := echo.New()
	echoServer.Use(middleware.Logger())
	echoServer.Use(middleware.Recover())

	echoServer.GET("/healthcheck", getHealthcheckEp)

	return echoServer
}

func getHealthcheckEp(ctx echo.Context) error {
	return ctx.JSON(http.StatusOK, map[string]interface{}{
		"code":    "success",
		"message": "success",
	})
}
