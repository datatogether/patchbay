package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/julienschmidt/httprouter"
)

var (
	// cfg is the global configuration for the server. It's read in at startup from
	// the config.json file and enviornment variables, see config.go for more info.
	cfg *config
	// log output
	logger = log.New(os.Stderr, "", log.Ldate|log.Ltime|log.Lshortfile)
	// application database connection
	appDB *sql.DB

	room *Room
)

func main() {
	var err error
	cfg, err = initConfig(os.Getenv("GOLANG_ENV"))
	if err != nil {
		// panic if the server is missing a vital configuration detail
		panic(fmt.Errorf("server configuration error: %s", err.Error()))
	}

	connectToAppDb()

	room = newRoom()
	go room.run()

	// initialize a router to handle requests
	r := httprouter.New()

	r.GET("/", middleware(WebappHandler))
	r.GET("/url", middleware(WebappHandler))
	r.GET("/content/:hash", middleware(WebappHandler))
	r.GET("/metadata/:hash", middleware(WebappHandler))
	r.GET("/settings", middleware(WebappHandler))
	r.GET("/settings/keys", middleware(WebappHandler))
	r.GET("/users/:user", middleware(WebappHandler))
	r.GET("/signup", middleware(WebappHandler))
	r.GET("/login", middleware(WebappHandler))
	r.GET("/login/forgot", middleware(WebappHandler))
	r.GET("/primers", middleware(WebappHandler))
	r.GET("/primers/:id", middleware(WebappHandler))

	r.GET("/ws", middleware(HandleWebsocketUpgrade))

	// serve static content from public directories
	r.ServeFiles("/css/*filepath", http.Dir("public/css"))
	r.ServeFiles("/js/*filepath", http.Dir("public/js"))

	// print notable config settings
	printConfigInfo()

	// fire it up!
	fmt.Println("starting server on port", cfg.Port)
	// non-tls configured servers end here
	if !cfg.TLS {
		logger.Fatal(StartHttpServer(cfg.Port, r))
	}
	// start server wrapped in a log.Fatal b/c http.ListenAndServe will not
	// return unless there's an error
	logger.Fatal(StartHttpsServer(cfg.Port, r))
}
