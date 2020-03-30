package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/james4k/rcon"
	"golang.org/x/crypto/acme/autocert"
)

type config struct {
	Servers []server `json:"servers"`
}

type server struct {
	Host           string `json:"host"`
	Port           int    `json:"port"`
	RCONPort       int    `json:"rconport"`
	RCONPassword   string `json:"rconpassword"`
	Title          string `json:"title"`
	Description    string `json:"description"`
	rconConnection *rcon.RemoteConsole
}

type serverData struct {
	IPAddr      string
	Port        int
	Title       string
	Players     string
	Version     string
	Description string
}

// rconCommand executes a command on the server, and returns the server's
// response as a string.
func (s server) rconCommand(command string) (response string, err error) {
	_, err = s.rconConnection.Write(command)
	if err != nil {
		return "", err
	}

	response, _, err = s.rconConnection.Read()
	if err != nil {
		return "", err
	}
	return
}

func main() {

	// Parse config file
	configData, err := ioutil.ReadFile("./config.json")
	if err != nil {
		log.Fatalf("Error reading config file: %v\n", err)
	}

	var config config

	err = json.Unmarshal(configData, &config)
	if err != nil {
		log.Fatalf("Error parsing config file: %v\n", err)
	}

	fmt.Print("Parsing templates...\n")
	t, err := template.ParseFiles("templates/index.html")
	if err != nil {
		fmt.Printf("Error parsing HTML template: %v\n", err)
	}

	for _, server := range config.Servers {
		server.rconConnection, err = rcon.Dial(fmt.Sprintf("%v:%v", server.Host, server.RCONPort), server.RCONPassword)
		if err != nil {
			log.Fatalf("Error making RCON connection to %v: %v", server.Title, err)
		}
		defer server.rconConnection.Close()
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {

		data := []serverData{}

		for _, server := range config.Servers {

			playersOnline, err := server.rconCommand("/players o")
			if err != nil {
				log.Printf("Error executing players online command: %v\n", err)
			}

			version, err := server.rconCommand("/version")
			if err != nil {
				log.Printf("Error executing version command: %v\n", err)
			}

			data = append(data, serverData{
				server.Host,
				server.Port,
				server.Title,
				playersOnline,
				version,
				server.Description,
			})

		}

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
