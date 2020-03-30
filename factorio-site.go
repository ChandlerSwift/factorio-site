package main

import (
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"

	"github.com/james4k/rcon"
)

type serverData struct {
	IPAddr  string
	Port    int
	Title   string
	Players string
}

func main() {

	serverAddr := flag.String("serverAddr", "factorio.blackolivepineapple.pizza", "Server to check status of (optional, defaults to factorio.bopp")
	serverPort := flag.Int("serverport", 34196, "RCON port on the Factorio server")
	password := flag.String("password", "", "RCON password of the server (required)")
	flag.Parse()

	if *port < 1 || *port > 65535 {
		fmt.Printf("Invalid port %v\n", *port)
		return
	}

	if *password == "" {
		fmt.Printf("Password flag is required")
	}

	fmt.Print("Parsing templates...\n")
	t, err := template.ParseFiles("templates/index.html")
	if err != nil {
		fmt.Printf("Error parsing HTML template: %v\n", err)
	}

	rconConnection, err := rcon.Dial(fmt.Sprintf("%v:%v", *serverAddr, *serverPort), *password)
	if err != nil {
		log.Fatalf("Error making RCON connection: %v", err)
	}
	defer rconConnection.Close()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {

		_, err := rconConnection.Write("/players o")
		if err != nil {
			fmt.Print(w, "Error connecting to server")
			return
		}

		playersOnline, _, err := rconConnection.Read()
		if err != nil {
			fmt.Print(w, "Error receiving data from server")
			return
		}

		data := serverData{
			*serverAddr,
			34197,
			"Server with Bob's Mod, est. Feb 2020",
			playersOnline,
		}

		t.Execute(w, data)
	})

	fmt.Printf("Serving on :%v...\n", *port)
	http.ListenAndServe(fmt.Sprintf(":%v", *port), nil)

}
