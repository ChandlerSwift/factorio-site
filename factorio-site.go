package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"

	"github.com/james4k/rcon"
	"golang.org/x/crypto/acme/autocert"
)

type serverData struct {
	IPAddr  string
	Port    int
	Title   string
	Players string
}

func main() {

	serverAddr := flag.String("serverAddr", "localhost", "Server to check status of (optional; defaults to localhost")
	serverPort := flag.Int("serverPort", 34196, "RCON port on the Factorio server (optional; defaults to 34196)")
	password := flag.String("password", "", "RCON password of the server (required)")
	flag.Parse()

	if *serverPort < 1 || *serverPort > 65535 {
		fmt.Printf("Invalid server port %v\n", *serverPort)
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

		data := []serverData{}
		data = append(data, serverData{
			*serverAddr,
			34197,
			"Server with Bob's Mod, est. Feb 2020",
			playersOnline,
		})

		t.Execute(w, data)
	})

	certManager := autocert.Manager{
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist("factorio.blackolivepineapple.pizza"), // TODO: add config
		Cache:      autocert.DirCache("certs"),
	}

	server := &http.Server{
		Addr: ":https",
		TLSConfig: &tls.Config{
			GetCertificate: certManager.GetCertificate,
		},
	}

	go http.ListenAndServe(":http", certManager.HTTPHandler(nil)) // Handler for LetsEncrypt

	fmt.Println("Serving...")
	server.ListenAndServeTLS("", "") // Key/cert come from server.TLSConfig

}
