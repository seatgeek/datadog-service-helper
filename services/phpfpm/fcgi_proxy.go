package phpfpm

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/scukonick/go-fastcgi-client"
)

// Connect to the upstream php-fpm process and get its current status
func Proxy(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)

	// variables we require to have present in the URL
	// they always exist thanks to the router
	project := params["project"]
	ip := params["ip"]
	port := params["port"]
	endpoint := params["type"]

	// convert the string port to int
	realPort, err := strconv.Atoi(port)
	if err != nil {
		message := fmt.Sprintf("[php-fpm] Invalid port %s: %s", port, err)
		logger.Errorf(message)
		http.Error(w, message, 500)
		return
	}

	// construct the env we need for php-fpm to allow ac
	env := make(map[string]string)
	env["REQUEST_METHOD"] = "GET"
	env["SCRIPT_FILENAME"] = fmt.Sprintf("/%s/internal/%s", project, endpoint)
	env["SCRIPT_NAME"] = fmt.Sprintf("/%s/internal/%s", project, endpoint)
	env["SERVER_SOFTWARE"] = "go / fcgiclient "
	env["QUERY_STRING"] = "json=1"

	// create fastcgi client
	fcgi, err := fcgiclient.New(ip, realPort)
	if err != nil {
		message := fmt.Sprintf("[php-fpm] Could not create fastcgi client: %s (%s)", err, r.URL.Path)
		logger.Errorf(message)
		http.Error(w, message, 500)
		return
	}

	// do the fastcgi request
	response, err := fcgi.Request(env, "json=1")
	if err != nil {
		message := fmt.Sprintf("[php-fpm] Failed fastcgi request: %s (%s)", err, r.URL.Path)
		logger.Errorf(message)
		http.Error(w, message, 500)
		return
	}

	// parse the fastcgi response
	body, err := response.ParseStdouts()
	if err != nil {
		message := fmt.Sprintf("[php-fpm] Failed to parse fastcgi response: %s (%s)", err, r.URL.Path)
		logger.Errorf(message)
		http.Error(w, message, 500)
		return
	}

	// read the response into a []bytes
	resp, err := ioutil.ReadAll(body.Body)
	if err != nil {
		message := fmt.Sprintf("[php-fpm] Failed to read fastcgi response: %s (%s)", err, r.URL.Path)
		logger.Errorf(message)
		http.Error(w, message, 500)
		return
	}

	// write to client
	w.Write(resp)

	logger.Debugf("[php-fpm] Request complete. Sent %d bytes (%s)", len(resp), r.URL.Path)
}
