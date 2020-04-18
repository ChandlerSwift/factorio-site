package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/james4k/rcon"
	"golang.org/x/crypto/acme/autocert"
)

type config struct {
	UseTLS          bool     `json:"useTLS"`
	Servers         []server `json:"servers"`
	TLSHostname     string   `json:"tlshostname"`
	DebugServerPort int      `json:"debugserverport"`
	BackupDir       string   `json:"backupDir"`
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

	serveBackups := true
	if stat, err := os.Stat(config.BackupDir); os.IsNotExist(err) || !stat.IsDir() {
		log.Printf("Backup directory %v does not exist; not serving backups.", config.BackupDir)
		serveBackups = false
	}

	// Set up templates
	fmt.Print("Parsing templates...\n")
	t, err := template.ParseFiles("templates/index.html")
	if err != nil {
		fmt.Printf("Error parsing HTML template: %v\n", err)
	}

	// Connect to RCON servers
	for i := range config.Servers {
		s := config.Servers[i]
		config.Servers[i].rconConnection, err = rcon.Dial(fmt.Sprintf("%v:%v", s.Host, s.RCONPort), s.RCONPassword)
		if err != nil {
			log.Fatalf("Error making RCON connection to %v: %v", s.Title, err)
		}
		defer s.rconConnection.Close()
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {

		data := []serverData{}

		for _, s := range config.Servers {

			playersOnline, err := s.rconCommand("/players o")
			if err != nil {
				log.Printf("Error executing players online command: %v\n", err)
			}

			version, err := s.rconCommand("/version")
			if err != nil {
				log.Printf("Error executing version command: %v\n", err)
			}

			data = append(data, serverData{
				s.Host,
				s.Port,
				s.Title,
				playersOnline,
				version,
				s.Description,
			})

		}

		t.Execute(w, data)
	})

	// Serve backup directory
	if serveBackups {
		http.Handle("/backups/", http.StripPrefix("/backups/", http.FileServer(http.Dir(config.BackupDir))))
	}

	if config.UseTLS {

		certManager := autocert.Manager{
			Prompt:     autocert.AcceptTOS,
			HostPolicy: autocert.HostWhitelist(config.TLSHostname),
			Cache:      autocert.DirCache("certs"),
		}

		srv := &http.Server{
			Addr: ":https",
			TLSConfig: &tls.Config{
				GetCertificate: certManager.GetCertificate,
			},
		}

		go http.ListenAndServe(":http", certManager.HTTPHandler(nil)) // Handler for LetsEncrypt

		fmt.Println("Serving...")
		srv.ListenAndServeTLS("", "") // Key/cert come from srv.TLSConfig

	} else { // Debug
		fmt.Println("Serving...")
		if config.DebugServerPort == 0 { // Value not set in JSON
			config.DebugServerPort = 8080
		}
		http.ListenAndServe(fmt.Sprintf(":%v", config.DebugServerPort), nil) // TODO: pass as config value
	}

}
