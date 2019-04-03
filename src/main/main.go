/*
TODOs
0.) github workflow zu Ende !!!!

1.) Ausprobieren: Buttons mit Werten senden und schauen ob der Body ausgelesen werden kann. Ansonsten wie mit Login
machen.

*/
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"time"

	"github.com/SlyMarbo/gmail"
	"github.com/gomodule/redigo/redis"
	uuid "github.com/satori/go.uuid"
)

// === constants ====

// session token expiry time  in seconds
var tokenExpiryTime = 300

// === structs ===

type credentials struct {
	Password string
	Username string
}

type request struct {
	Name        string
	CompanyName string
	EmailAdress string
}

// === variables and slices ===

var cache redis.Conn

var users = map[string]string{
	"user1": "password1",
	"user2": "password2",
}

// === template pool ==

var templates = template.Must(template.ParseFiles(
	"templates/FrontPage.html",
	"templates/AboutMe.html",
	"templates/ContactMe.html",
	"templates/ProjectsPage.html",
	"templates/LoginPage.html",
	"templates/cvRequestsTable.html",
	"stylesheets/fp.css",
	"stylesheets/am.css",
	"stylesheets/pp.css",
	"stylesheets/cm.css"))

var validPath = regexp.MustCompile("^/(edit|save|view)/([a-zA-Z0-9]+)$")

// ===  functions ===
// ===  handler functions 1 ===
// === handler functions - additional 1.1===

func renderTemplate(w http.ResponseWriter, tmpl string) {
	err := templates.ExecuteTemplate(w, tmpl+".html", " ")
	if err != nil {

		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func renderDynamicTemplate(w http.ResponseWriter, tmpl string, data interface{}) {
	err := templates.ExecuteTemplate(w, tmpl+".html", data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func makeHandler(fn func(http.ResponseWriter, *http.Request)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		m := validPath.FindStringSubmatch(r.URL.Path)
		if m == nil {
			http.NotFound(w, r)
			return
		}
		fn(w, r) // the title is the second subexpression
	}
}

// ===  handler functions - template 1.2===

func rootHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, "FrontPage")
}

func frontPageHandler(w http.ResponseWriter, r *http.Request) {

	fmt.Println("open front page")
	renderTemplate(w, "FrontPage")
}

func loginPageHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("open login page")
	renderTemplate(w, "LoginPage")
}

func aboutmeHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("open about me page")
	renderTemplate(w, "AboutMe")
}

func projectPageHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("open projects page")
	renderTemplate(w, "ProjectsPage")
}

func contactmeHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("open contact me page")
	renderTemplate(w, "ContactMe")
}

func cvRequestsTableHandler(w http.ResponseWriter, r *http.Request) {
	/*var names = []string{"none", "none", "none", "none", "none"}
	var companyNames = []string{"none", "none", "none", "none", "none"}
	var emailAdresses = []string{"none", "none", "none", "none", "none"}
	*/
	fmt.Println("open cv requests table page")

	checkSession(w, r)

	resultSet, err := fetchData()

	if err != nil {
		http.Error(w, err.Error(), 500)
	}

	insertData := make(map[string]interface{})

	for i := 0; i < len(resultSet); i++ {
		insertData["name"+strconv.Itoa(i)] = resultSet[i].Name
		insertData["companyName"+strconv.Itoa(i)] = resultSet[i].CompanyName
		insertData["emailAdress"+strconv.Itoa(i)] = resultSet[i].EmailAdress
	}

	if len(resultSet) < 5 {
		for i := len(resultSet); i < len(resultSet); i++ {
			insertData["name"+strconv.Itoa(i)] = " "
			insertData["companyName"+strconv.Itoa(i)] = " "
			insertData["emailAdress"+strconv.Itoa(i)] = " "
		}
	}

	renderDynamicTemplate(w, "cvRequestsTable", insertData)
}

func cssHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("handle css ")
	http.ServeFile(w, r, "/stylesheets/fp.css")
}

// === handler functions - back processing 1.3 ===

func signin(w http.ResponseWriter, r *http.Request) {
	fmt.Println("signin handled")

	var creds credentials
	// Get the JSON body and decode into credentials
	err := json.NewDecoder(r.Body).Decode(&creds)

	fmt.Println("username entered: " + creds.Username + " password entered: " + creds.Password)
	if err != nil {
		// If the structure of the body is wrong, return an HTTP error
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Get the expected password from our in memory map
	expectedPassword, ok := users[creds.Username]

	// If a password exists for the given user
	// AND, if it is the same as the password we received, the we can move ahead
	// if NOT, then we return an "Unauthorized" status
	if !ok || expectedPassword != creds.Password {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Println("login failed")
		return
	}

	// Create a new random session token
	sessionToken := uuid.Must(uuid.NewV4()).String()
	// Set the token in the cache, along with the user whom it represents
	// The token has an expiry time of (tokenExpiryTime) seconds
	_, err = cache.Do("SETEX", sessionToken, strconv.Itoa(tokenExpiryTime), creds.Username)
	if err != nil {
		// If there is an error in setting the cache, return an internal server error
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Finally, we set the client cookie for "session_token" as the session token we just generated
	// we also set an expiry time of (tokenExpiryTime) seconds, the same as the cache
	http.SetCookie(w, &http.Cookie{
		Name:    "session_token",
		Value:   sessionToken,
		Expires: time.Now().Add(time.Duration(tokenExpiryTime) * time.Second),
	})

	fmt.Println("open cv requests table page")

	checkSession(w, r)

	resultSet, err := fetchData()

	if err != nil {
		http.Error(w, err.Error(), 500)
	}

	insertData := make(map[string]interface{})

	for i := 0; i < 5; i++ {
		insertData["name"+strconv.Itoa(i)] = resultSet[i].Name
		insertData["companyName"+strconv.Itoa(i)] = resultSet[i].CompanyName
		insertData["emailAdress"+strconv.Itoa(i)] = resultSet[i].EmailAdress
		fmt.Println(resultSet[i].Name)

	}

	renderDynamicTemplate(w, "cvRequestsTable", insertData)
}

func welcome(w http.ResponseWriter, r *http.Request) {
	fmt.Println("welcome handled")
	// We can obtain the session token from the requests cookies, which come with every request
	c, err := r.Cookie("session_token")
	if err != nil {
		if err == http.ErrNoCookie {
			// If the cookie is not set, return an unauthorized status
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		// For any other type of error, return a bad request status
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	sessionToken := c.Value

	// We then get the name of the user from our cache, where we set the session token
	response, err := cache.Do("GET", sessionToken)
	if err != nil {
		// If there is an error fetching from cache, return an internal server error status
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if response == nil {
		// If the session token is not present in cache, return an unauthorized error
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	// Finally, return the welcome message to the user
	w.Write([]byte(fmt.Sprintf("Welcome %s!", response)))
}

func refreshToken(w http.ResponseWriter, r *http.Request) {
	// (BEGIN) The code uptil this point is the same as the first part of the `Welcome` route
	c, err := r.Cookie("session_token")
	if err != nil {
		if err == http.ErrNoCookie {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	sessionToken := c.Value

	response, err := cache.Do("GET", sessionToken)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if response == nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	// (END) The code uptil this point is the same as the first part of the `Welcome` route

	// Now, create a new session token for the current user
	newSessionToken := uuid.Must(uuid.NewV4()).String()
	_, err = cache.Do("SETEX", newSessionToken, strconv.Itoa(tokenExpiryTime), fmt.Sprintf("%s", response))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Delete the older session token
	_, err = cache.Do("DEL", sessionToken)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Set the new token as the users `session_token` cookie
	http.SetCookie(w, &http.Cookie{
		Name:    "session_token",
		Value:   newSessionToken,
		Expires: time.Now().Add(time.Duration(tokenExpiryTime) * time.Second),
	})

	fmt.Printf("New session token value: %v", newSessionToken)
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	username := r.FormValue("username") // Data from the form
	pwd := r.FormValue("password")      // Data from the form

	creds := credentials{Password: pwd, Username: username}

	// Get the expected password from our in memory map
	expectedPassword, ok := users[creds.Username]

	// If a password exists for the given user
	// AND, if it is the same as the password we received, the we can move ahead
	// if NOT, then we return an "Unauthorized" status
	if !ok || expectedPassword != creds.Password {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Println("login failed")
		return
	}

	// Create a new random session token
	sessionToken := uuid.Must(uuid.NewV4()).String()
	// Set the token in the cache, along with the user whom it represents
	// The token has an expiry time of (tokenExpiryTime) seconds
	_, err := cache.Do("SETEX", sessionToken, strconv.Itoa(tokenExpiryTime), creds.Username)
	if err != nil {
		// If there is an error in setting the cache, return an internal server error
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Finally, we set the client cookie for "session_token" as the session token we just generated
	// we also set an expiry time of (tokenExpiryTime) seconds, the same as the cache
	http.SetCookie(w, &http.Cookie{
		Name:    "session_token",
		Value:   sessionToken,
		Expires: time.Now().Add(time.Duration(tokenExpiryTime) * time.Second),
	})

	fmt.Println("open cv requests table page")

	checkSession(w, r)

	http.Redirect(w, r, "/cvRequestsTable", http.StatusSeeOther)

}

func createNewCVRequest(w http.ResponseWriter, r *http.Request) {
	name := " "
	companyName := " "
	email := " "

	name = r.FormValue("name")
	companyName = r.FormValue("companyName")
	email = r.FormValue("email")

	if len(name) > 0 && len(companyName) > 0 && len(email) > 0 {

		var req = Request{0, name, companyName, email}

		error := insertData(&req)

		if error != nil {
			fmt.Println("main")
			http.Error(w, error.Error(), 500)
		}
	}
	fmt.Printf("name: %v. company name: %v. email: %v", name, companyName, email)
	renderTemplate(w, "ContactMe")

}

func fetchRequests(w http.ResponseWriter, r *http.Request) {
	resultSet, error := fetchData()
	if error != nil {
		http.Error(w, error.Error(), 500)
	}

	for i := 0; i < len(resultSet); i++ {
		fmt.Printf("request_id: " + string(resultSet[i].RequestId) +
			" name: " + resultSet[i].Name +
			" companyName: " + resultSet[i].CompanyName +
			" emailAdress: " + resultSet[i].EmailAdress + "\n")

	}

}

func deleteRequest(w http.ResponseWriter, r *http.Request) {
	fmt.Println("entered prehandler1")
	//check authorization

	name := r.PostFormValue("nameDelete") // Data from the form

	fmt.Println("name: " + name)

	err := deleteData(name)

	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	http.Redirect(w, r, "/cvRequestsTable", http.StatusSeeOther)

}

func replyRequest(w http.ResponseWriter, r *http.Request) {
	fmt.Println("entered prehandler1")
	//check authorization

	name := r.PostFormValue("nameReply") // Data from the form

	fmt.Println("name: " + name)

	requestorData, err := searchData(name)

	if err != nil {
		fmt.Println("user not found in DB")
		http.Error(w, err.Error(), 500)
	}

	fmt.Printf(requestorData[0].Name)
	if err := sendCVEmail(requestorData[0].Name, requestorData[0].EmailAdress); err != nil {
		fmt.Println("Could not send mail with cv.")
		http.Error(w, err.Error(), 500)
	}

	if err := deleteData(name); err != nil {
		fmt.Printf("Deleting record in DB not sucessful")
		http.Error(w, err.Error(), 500)
	}

	http.Redirect(w, r, "/cvRequestsTable", http.StatusSeeOther)

}

func outputHTML(w http.ResponseWriter, filename string, data interface{}) {
	t, err := template.ParseFiles(filename)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	if err := t.Execute(w, data); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
}

func checkSession(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie("session_token")
	if err != nil {
		if err == http.ErrNoCookie {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	sessionToken := c.Value

	response, err := cache.Do("GET", sessionToken)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if response == nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
}

// === email functionts ===

func sendCVEmail(recipientName string, recipientAdress string) error {
	email := gmail.Compose("Curriculum Vitae", "Hi "+recipientName+"! Attached you find my CV. ")
	email.From = "jkim17081@gmail.com"
	email.Password = "Neuespasswort1"

	// Defaults to "text/plain; charset=utf-8" if unset.
	email.ContentType = "text/html; charset=utf-8"

	// add attachments
	if err := email.Attach("/Users/josefkim/go-workspace/personal-website/Lebenslauf.pdf"); err != nil {
		return err
	}

	// Normally you'll only need one of these, but I thought I'd show both.
	email.AddRecipient(recipientAdress)

	err := email.Send()
	if err != nil {
		return err
	}
	return nil
}

// === cache functions ===

func initCache() {
	// Initialize the redis connection to a redis instance running on your local machine
	conn, err := redis.DialURL("redis://localhost")
	if err != nil {
		panic(err)
	}
	// Assign the connection to the package level `cache` variable
	cache = conn

	fmt.Println("cache initilized")
}

// === main function ===

func main() {

	initCache()

	stop := make(chan os.Signal, 1)

	signal.Notify(stop, os.Interrupt)

	addr := ":" + os.Getenv("PORT")
	if addr == ":" {
		addr = ":2017"
	}

	m := http.NewServeMux()

	s := http.Server{Addr: ":8000", Handler: m}

	// creating a new file server
	fileServer := http.FileServer(http.Dir("/Users/josefkim/go-workspace/personal-website/src/main/stylesheets"))

	// Use the m.Handle() function to register the file server as the handler for
	// all URL paths that start with "/stylesheets/". For matching paths, we strip the
	// "/stylesheets" prefix before the request reaches the file server.
	m.Handle("/stylesheets/", http.StripPrefix("/stylesheets", fileServer))

	//looks for css files and returns if cannot find them
	if _, err := os.Stat("/Users/josefkim/go-workspace/personal-website/src/main/stylesheets/fp.css"); err != nil {
		fmt.Println("couldnt find css files")
	}

	m.HandleFunc("/", frontPageHandler)
	m.HandleFunc("/AboutMe", aboutmeHandler)
	m.HandleFunc("/ProjectsPage", projectPageHandler)
	m.HandleFunc("/ContactMe", contactmeHandler)
	m.HandleFunc("/loginPage", loginPageHandler)
	m.HandleFunc("/cvRequestsTable", cvRequestsTableHandler)

	//login handler without json body
	m.HandleFunc("/login", loginHandler)
	m.HandleFunc("/CreateNewCVRequest", createNewCVRequest)

	//login handler with json body
	//m.HandleFunc("/signin", signin)
	m.HandleFunc("/welcome", welcome)
	m.HandleFunc("/refreshToken", refreshToken)

	m.HandleFunc("/fetchRequests", fetchRequests)

	m.HandleFunc("/deleteRequest", deleteRequest)

	m.HandleFunc("/replyRequest", replyRequest)

	m.HandleFunc("/shutdown", func(w http.ResponseWriter, r *http.Request) {
		s.Shutdown(context.Background())
	})
	go func() {
		if erro := s.ListenAndServe(); erro != nil {
			log.Fatal(erro)

		}
	}()
	<-stop

	log.Printf("Finsihed")
	s.Shutdown(context.Background())

}
