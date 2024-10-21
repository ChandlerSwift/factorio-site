package main

import (
	"crypto/tls"
	_ "embed"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"

	"github.com/james4k/rcon"
	"golang.org/x/crypto/acme/autocert"
)

//go:embed templates/index.html
var indexhtml string

type config struct {
	Title       string   `json:"title"`
	Content     string   `json:"content"`
	Servers     []server `json:"servers"`
	UseTLS      bool     `json:"useTLS"`
	TLSHostname string   `json:"tlshostname"`
	ServerPort  int      `json:"serverport"`
	BackupDir   string   `json:"backupDir"`
}

type server struct {
	Host           string `json:"host"` // for display only
	Port           int    `json:"port"`
	RCONHost       string `json:"rconhost"` // not displayed, but used to connect; leave blank for no RCON connection
	RCONPort       int    `json:"rconport"`
	RCONPassword   string `json:"rconpassword"`
	Title          string `json:"title"`       // TODO: get this from RCON?
	Description    string `json:"description"` // TODO: get this from RCON?
	Version        string // Populated by RCON
	Players        string // Populated by RCON
	rconConnection *rcon.RemoteConsole
}

type pageData struct {
	Title        string
	Content      template.HTML
	Servers      []server
	ServeBackups bool
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
	configData, err := os.ReadFile("./config.json")
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

	data := pageData{
		Title:        config.Title,
		Content:      template.HTML(config.Content),
		Servers:      config.Servers,
		ServeBackups: serveBackups,
	}

	// Set up templates
	fmt.Print("Parsing templates...\n")
	t, err := template.New("index").Parse(indexhtml)
	if err != nil {
		log.Fatalf("Error parsing HTML template: %v\n", err)
	}

	// Connect to RCON servers
	for i := range config.Servers {
		s := config.Servers[i]
		if s.RCONHost != "" {
			config.Servers[i].rconConnection, err = rcon.Dial(fmt.Sprintf("%v:%v", s.RCONHost, s.RCONPort), s.RCONPassword)
			if err != nil {
				log.Fatalf("Error making RCON connection to %v: %v", s.Title, err)
			}
			defer config.Servers[i].rconConnection.Close()
		}
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {

		// Update servers with current data
		for i, s := range config.Servers {
			if s.rconConnection != nil {
				config.Servers[i].Players, err = s.rconCommand("/players o")
				if err != nil {
					log.Printf("Error executing players online command: %v\n", err)
				}

				config.Servers[i].Version, err = s.rconCommand("/version")
				if err != nil {
					log.Printf("Error executing version command: %v\n", err)
				}
			}
		}

		err = t.Execute(w, data)
		if err != nil {
			log.Printf("Error executing template: %v\n", err)
		}
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

		if config.ServerPort == 0 { // Value not set in JSON
			config.ServerPort = 443
		}
		fmt.Printf("Serving HTTPS on port %v...\n", config.ServerPort)
		srv.ListenAndServeTLS("", "") // Key/cert come from srv.TLSConfig

	} else {
		if config.ServerPort == 0 { // Value not set in JSON
			config.ServerPort = 80
		}
		fmt.Printf("Serving HTTP on port %v...\n", config.ServerPort)
		http.ListenAndServe(fmt.Sprintf(":%v", config.ServerPort), nil)
	}

}
