package control

import (
	"encoding/json"
	"log"
	"net"
	"net/http"
	"os"
	"wag/firewall"
)

const controlSocket = "/tmp/wag.sock"

func block(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.NotFound(w, r)
		return
	}

	err := r.ParseForm()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	err = firewall.RemoveAuthorizedRoutes(r.FormValue("address"))
	if err != nil {
		http.Error(w, "not found: "+err.Error(), 404)
		return
	}

	w.Write([]byte("OK"))
}

func sessions(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	result, _ := json.Marshal(firewall.GetAllAllowed())

	w.Write(result)
}

func delete(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.NotFound(w, r)
		return
	}

	err := r.ParseForm()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	err = firewall.RemoveAllRoutes(r.FormValue("address"))
	if err != nil {
		http.Error(w, "not found: "+err.Error(), 404)
		return
	}

	w.Write([]byte("OK"))
}

func firewallRules(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.NotFound(w, r)
		return
	}

	rules, err := firewall.GetRules()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	result, _ := json.Marshal(rules)

	w.Write(result)
}

func StartControlSocket() error {
	l, err := net.Listen("unix", controlSocket)
	if err != nil {
		return err
	}

	if err := os.Chmod(controlSocket, 0700); err != nil {
		return err
	}

	log.Println("Started control socket: \n\t\t\t", controlSocket)

	http.HandleFunc("/device/block", block)
	http.HandleFunc("/device/sessions", sessions)
	http.HandleFunc("/device/delete", delete)

	http.HandleFunc("/firewall/list", firewallRules)

	go http.Serve(l, nil)

	return nil
}

func TearDown() {
	err := os.Remove(controlSocket)
	if err != nil {
		log.Println(err)
	}
}
