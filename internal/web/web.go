package web

import (
	"net/http"
	"fmt"

	"github.com/Fishwaldo/go-logadapter"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/spf13/viper"
)

func init() {
	viper.SetDefault("web.port", 8080)
}

type WebServer struct {
	Echo *echo.Echo
	log  logadapter.Logger
}
var (
	Web WebServer
)

func (web *WebServer) Start(log logadapter.Logger) {
	web.log = log
	web.Echo = echo.New()
	// Middleware
	web.Echo.Use(middleware.Logger())
	web.Echo.Use(middleware.Recover())

	// Routes
	web.Echo.GET("/", homePage)

	// Start server
	go func() {
		port := viper.GetInt("web.port")
		bind := fmt.Sprintf(":%d", port)
		web.log.Fatal("%s", web.Echo.Start(bind))
	}()
}

func (web *WebServer) GetEchoServer() (*echo.Echo) {
	return Web.Echo
}

func homePage(c echo.Context) error {
	return c.String(http.StatusOK, "Hello, World!")
}
