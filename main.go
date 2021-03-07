package main

import (
	"github.com/NERON/tran/database"
	"github.com/NERON/tran/manager"
	"html/template"
	"log"

	"net/http"

	"github.com/gorilla/mux"
	_ "gopkg.in/lib/pq.v1"
)

var TemplateManager *template.Template

func InitRouting() *mux.Router {

	r := mux.NewRouter()

	r.HandleFunc("/info/{symbol}/{interval}/{centralRSI}", IndexHandler)
	r.HandleFunc("/chart/{symbol}/{interval}/{centralRSI}", ChartUpdateHandler)
	r.HandleFunc("/test/{symbol}/{interval}/{centralRSI}", TestHandler)
	r.HandleFunc("/getTriples/{symbol}/{centralRSI}/{mode}/{groupCount}/{timestamp}", GetTriplesHandler)
	r.HandleFunc("/getDD/{symbol}/{centralRSI}/{mode}/{groupCount}/{timestamp}", SaveCandlesHandler)
	r.HandleFunc("/getInter/{symbol}/{centralRSI}", GetIntervalHandler)

	return r
}

func main() {

	var err error
	TemplateManager, err = template.ParseFiles("./templates/chartPage.html", "./templates/RSIReverseStat.html", "./templates/triples.html")

	err = database.OpenDatabaseConnection()

	database.InitializeDatabase()

	if err != nil {

		log.Fatal("Database connection error: ", err.Error())
	}

	manager.KLineCacher, err = manager.NewLastKlinesCacher([]string{"ETHUSDT", "ADAETH"})

	if err != nil {
		log.Fatal(err.Error())
	}

	router := InitRouting()

	log.Fatal(http.ListenAndServeTLS(":8085", "server.crt", "server.key", router))

}
