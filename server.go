package main

import (
	"flag"
	"fmt"
	"html/template"
	"net/http"
)

type serverData struct {
	IPAddr  string
	Title   string
	Players string
}

func main() {

	port := flag.Int("port", 65536, "Port on which the HTTP server should serve")
	server := flag.String("serverAddr", "factorio.blackolivepineapple.pizza", "Server to check status of")
	flag.Parse()

	if *port < 1 || *port > 65535 {
		fmt.Printf("Invalid port %v\n", *port)
		return
	}

	fmt.Print("Parsing templates...\n")
	t, err := template.ParseFiles("templates/index.html")
	if err != nil {
		fmt.Printf("Error parsing HTML template: %v\n", err)
	}

	data := serverData{
		*server,
		"bopp server",
		"none",
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Execute(w, data)
	})

	fmt.Printf("Serving on :%v...\n", *port)
	http.ListenAndServe(fmt.Sprintf(":%v", *port), nil)

}
