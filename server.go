package main

import (
	"bytes"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"text/template"
	"encoding/json"
	"io/ioutil"

	"github.com/a-h/templ"
	"github.com/acaloiaro/hugo-htmx-go-template/partials"
)

type Ability struct {
	Ability struct {
		Name string `json:"name"`
		URL  string `json:"url"`
	} `json:"ability"`
	IsHidden bool `json:"is_hidden"`
	Slot     int  `json:"slot"`
}

type Pokemon struct {
	Abilities    []Ability  `json:"abilities"`
	Height       int        `json:"height"`
}

//go:embed all:public
var content embed.FS

func main() {
	mux := http.NewServeMux()
	serverRoot, _ := fs.Sub(content, "public")

	// Serve all hugo content (the 'public' directory) at the root url
	mux.Handle("/", http.FileServer(http.FS(serverRoot)))

	cors := func(h http.Handler) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			// in development, the Origin is the the Hugo server, i.e. http://localhost:1313
			// but in production, it is the domain name where one's site is deployed
			//
			// CHANGE THIS: You likely do not want to allow any origin (*) in production. The value should be the base URL of
			// where your static content is served
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, hx-target, hx-current-url, hx-request")
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			h.ServeHTTP(w, r)
		}
	}

	// Add any number of handlers for custom endpoints here
	mux.HandleFunc("/goodbyeworld.html", cors(templ.Handler(partials.GoodbyeWorld())))
	mux.HandleFunc("/hello_world", cors(http.HandlerFunc(helloWorld)))
	mux.HandleFunc("/hello_world_form", cors(http.HandlerFunc(helloWorldForm)))

	fmt.Printf("Starting API server on port 1314\n")
	if err := http.ListenAndServe("0.0.0.0:1314", mux); err != nil {
		log.Fatal(err)
	}
}

// the handler accepts GET requests to /hello_world
// It checks the URL params for the "name" param and populates the html/template variable with its value
// if no "name" url parameter is present, "name" is defaulted to "World"
//
// It responds with the the HTML partial `partials/helloworld.html`
func helloWorld(w http.ResponseWriter, r *http.Request) {
	var pokemonData Pokemon
	name := r.URL.Query().Get("name")
	if name == "null" || name == "" {
		name = "World"
	}

	// Fetch Pokémon data
	pokemonName := "ditto"
	pokemonData, err := getPokemonData(pokemonName)

	pokemonDataJSON, err := json.MarshalIndent(pokemonData, "", "  ")
	if err != nil {
		log.Printf("Failed to marshal Pokémon data: %v", err)
	} else {
		log.Printf("Pokémon data: %s", pokemonDataJSON)
	}

	tmpl := template.Must(template.ParseFiles("partials/helloworld.html"))
	buff := bytes.NewBufferString("")
	errs := tmpl.Execute(buff, map[string]interface{}{
		"Name":        "Linh",
		"PokemonData": pokemonData,
		"PokemonName": "ditto",
	})
	if errs != nil {
		ise(errs, w)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(buff.Bytes())
}

func getPokemonData(pokemonName string) (Pokemon, error) {
	url := fmt.Sprintf("https://pokeapi.co/api/v2/pokemon/%s", pokemonName)

	resp, err := http.Get(url)
	if err != nil {
		return Pokemon{}, fmt.Errorf("failed to fetch Pokémon data: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return Pokemon{}, fmt.Errorf("received non-200 response: %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return Pokemon{}, fmt.Errorf("failed to read response body: %v", err)
	}

	var pokemonData Pokemon
	err = json.Unmarshal(body, &pokemonData)
	if err != nil {
		return Pokemon{}, fmt.Errorf("failed to unmarshal Pokémon data: %v", err)
	}

	return pokemonData, nil
}

func getAllPokemonData(pokemonName string) (map[string]interface{}, error) {
	url := fmt.Sprintf("https://pokeapi.co/api/v2/pokemon/%s", pokemonName)

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Pokémon data: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("received non-200 response: %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	var pokemonData map[string]interface{}
	err = json.Unmarshal(body, &pokemonData)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal Pokémon data: %v", err)
	}

	return pokemonData, nil
}

// this handler accepts POST requests to /hello_world_form
// It checks the post request body for the form value "name" and populates the html/template
// variable with its value
//
// It responds with a simple greeting HTML partial
func helloWorldForm(w http.ResponseWriter, r *http.Request) {
	name := "World"
	if err := r.ParseForm(); err != nil {
		ise(err, w)
		return
	}

	name = r.FormValue("name")
	if err := partials.HelloWorldGreeting(name).Render(r.Context(), w); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func ise(err error, w http.ResponseWriter) {
	fmt.Fprintf(w, "error: %v", err)
	w.WriteHeader(http.StatusInternalServerError)
}
