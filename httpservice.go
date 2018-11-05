package snogo

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	flagPort = flag.String("port", "8080", "Listening Port")
	snClient = DefaultServiceNowClient()
)

func init() {
	log.SetFlags(log.Lmicroseconds | log.Lshortfile)
	flag.Parse()
}

type prometheusAlertPayload struct {
	Version     string `json:"version"`
	GroupKey    string `json:"groupKey"`
	Status      string `json:"status"`
	Receiver    string `json:"receiver"`
	ExternalURL string `json:"externalURL"`
	Alerts      []struct {
		Status string `json:"status"`
		Labels struct {
			SnowGroup        string `json:"snow_group"`
			CmdbCI           string `json:"cmdb_ci"`
			OpenShiftCluster string `json:"openshift_cluster"`
		} `json:"labels"`
		Annotations struct {
			Description      string `json:"description"`
			ShortDescription string `json:"summary"`
			RunBook          string `json:"runbook"`
		} `json:"annotations"`
	} `json:"alerts"`
}

func StartServer() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", postHandler)
	mux.Handle("/metrics", promhttp.Handler())
	log.Printf("listening on port %s", *flagPort)
	log.Fatal(http.ListenAndServe("0.0.0.0:"+*flagPort, mux))
}

func postHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		bytesReturned, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Error processing request body",
				http.StatusBadRequest)
		}

		// It would be better to check if the body is valid JSON
		body := string(bytesReturned)
		if body == "" {
			http.Error(w, "Request must contain a body",
				http.StatusBadRequest)
		}

		prometheusPayload, err := serializeJSON(body)
		if err != nil {
			http.Error(w, "Unable to deserialize JSON", http.StatusInternalServerError)
		} else {
			fmt.Printf("%+v\n", prometheusPayload)
			fmt.Fprint(w, "Deserialized JSON successfully!")
		}

		// TODO: Why are we ignoring errors here? I assume because the input is 'validated'
		// per the serializeJSON func? If so, let's just kill error checking here altogether.
		incidentsToCreate, _ := transform(&prometheusPayload)

		for _, incident := range incidentsToCreate {
			if len(incident.AssignmentGroup) > 0 {
				postBody, err := json.Marshal(incident)
				if err != nil {
					fmt.Println(err)
				} else {
					_, err := snClient.Create("incident", postBody)
					if err != nil {
						// Do a thing
					}
					fmt.Printf("Incident Posted")
				}
			} else {
				fmt.Printf("No assignment group for incident %s, not created\n", incident.Description)
			}
		}
	} else {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
	}
}

func serializeJSON(j string) (prometheusAlertPayload, error) {
	// Prepare JSON as bytecode
	b := []byte(j)

	// Unmarshal into struct
	var m prometheusAlertPayload
	err := json.Unmarshal(b, &m)

	return m, err
}

func transform(payload *prometheusAlertPayload) ([]IncidentCreationPayload, error) {
	var incidentList []IncidentCreationPayload
	for _, alert := range payload.Alerts {
		incident := IncidentCreationPayload{
			AssignmentGroup:  strings.Replace(alert.Labels.SnowGroup, "-", " ", -1),
			CmdbCI:           alert.Labels.CmdbCI,
			ServiceCI:        "Server Admin",
			ContactType:      "Event",
			Customer:         "Monitoring",
			Description:      alert.Annotations.Description,
			Impact:           "4",
			ShortDescription: alert.Annotations.ShortDescription,
			State:            json.Number("60"),
			Urgency:          "3",
			AltContactInfo:   "Prometheus Alerting",
			Category:         "Availability",
			SubCategory:      "Completely Unavailable",
			OutageStart:      time.Now(),
		}
		incidentList = append(incidentList, incident)
	}
	return incidentList, nil
}
