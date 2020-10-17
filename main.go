package main

import (
	"github.com/NERON/tran/database"
	"html/template"
	"log"

	"net/http"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

var TemplateManager *template.Template

func InitRouting() *mux.Router {

	r := mux.NewRouter()

	r.HandleFunc("/info/{symbol}/{interval}", IndexHandler)
	r.HandleFunc("/chart/{symbol}/{interval}", ChartUpdateHandler)
	r.HandleFunc("/rsiJSON", RSIJSONHandler)
	r.HandleFunc("/test/{symbol}/{interval}", TestHandler)
	r.HandleFunc("/load", SaveCandlesHandler)

	return r
}

func main() {

	var err error
	TemplateManager, err = template.ParseFiles("./templates/chartPage.html", "./templates/RSIReverseStat.html")

	err = database.OpenDatabaseConnection()

	if err != nil {

		log.Fatal("Database connection error: ", err.Error())
	}

	router := InitRouting()

	log.Fatal(http.ListenAndServeTLS(":8085", "server.crt", "server.key", router))

}
