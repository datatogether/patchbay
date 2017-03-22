package main

import (
	"database/sql"
	"fmt"
	"github.com/gchaincl/dotsql"
	"log"
	"net/http"
	"os"
)

var (
	// cfg is the global configuration for the server. It's read in at startup from
	// the config.json file and enviornment variables, see config.go for more info.
	cfg *config
	// log output
	logger = log.New(os.Stderr, "", log.Ldate|log.Ltime|log.Lshortfile)
	// application database connection
	appDB *sql.DB

	sqlCmds *dotsql.DotSql

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

	s := &http.Server{}
	m := http.NewServeMux()
	m.HandleFunc("/.well-known/acme-challenge/", CertbotHandler)

	m.Handle("/", middleware(WebappHandler))
	m.Handle("/url", middleware(WebappHandler))
	m.Handle("/content/", middleware(WebappHandler))
	m.Handle("/metadata/", middleware(WebappHandler))
	m.Handle("/settings", middleware(WebappHandler))
	m.Handle("/settings/keys", middleware(WebappHandler))
	m.Handle("/users/", middleware(WebappHandler))
	m.Handle("/signup", middleware(WebappHandler))
	m.Handle("/login", middleware(WebappHandler))
	m.Handle("/login/forgot", middleware(WebappHandler))
	m.Handle("/primers", middleware(WebappHandler))
	m.Handle("/primers/", middleware(WebappHandler))
	m.Handle("/subprimers", middleware(WebappHandler))
	m.Handle("/subprimers/", middleware(WebappHandler))

	m.Handle("/ws", middleware(HandleWebsocketUpgrade))

	// connect mux to server
	s.Handler = m

	// print notable config settings
	printConfigInfo()

	// fire it up!
	fmt.Println("starting server on port", cfg.Port)

	// start server wrapped in a log.Fatal b/c http.ListenAndServe will not
	// return unless there's an error
	logger.Fatal(StartServer(cfg, s))
}
