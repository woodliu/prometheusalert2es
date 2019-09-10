package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/google/uuid"
	"github.com/prometheus/prometheus/pkg/labels"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

var (
	logger = log.New(os.Stdout,"[PrmetheusAlert2Es]",log.LstdFlags)
)

const(
	ok = 0
	ng = 1

	template = "prometheus_alert_template"
)

type AlertHandler struct {
}

func (th *AlertHandler)ServeHTTP(w http.ResponseWriter, r *http.Request){
	if r.Body == nil {
		logger.Println("[Warn] Got empty request body")
		return
	}
	defer r.Body.Close()

	reqbody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		logger.Println("[Warn] Read request body err,",err)
		return
	}

	var alerts Alerts
	json.Unmarshal(reqbody,&alerts)

	t := time.Unix(time.Now().Unix(), 0)
	nowDate := fmt.Sprintf("%d_%d_%d",t.Year(), t.Month(), t.Day())
	indice := "prometheus_alert_" + nowDate
	//Check indice and template
	if ok != DoRequest(http.MethodGet,esurl+"/"+indice,nil){
		logger.Println("[Info] Not found indice:",indice,",begin to create...")
		ret := DoRequest(http.MethodPut,esurl+"/"+indice,nil)
		if ok != ret{
			logger.Println("[Error] Create indice failed")
			return
		}
	}

	if ok != DoRequest(http.MethodGet,esurl+"/_template/" + template,nil){
		logger.Println("[Info] Not found template:",template,",begin to create...")
		reqbody := []byte(`
{
  "index_patterns": ["prometheus_alert*"],
  "settings": {
    "number_of_shards": 2,
    "number_of_replicas": 0,
    "index.lifecycle.name": "prometheus_alert*",
    "index.refresh_interval": "10s",
    "index.query.default_field": "groupLabels.alertname"
  },
  "mappings": {
    "_doc": {
      "properties": {
        "@timestamp": {
          "type": "date",
          "doc_values": true
        }
      },
      "dynamic_templates": [
        {
          "string_fields": {
            "match": "*",
            "match_mapping_type": "string",
            "mapping": {
              "type": "text",
              "fields": {
                "raw": {
                  "type":  "keyword",
                  "ignore_above": 256
                }
              }
            }
          }
        }
      ]
    }
  }
}`)
		newBody := bytes.NewBuffer(reqbody)
		ret := DoRequest(http.MethodPost,esurl+"/_template/" + template,newBody)
		if ok != ret{
			logger.Println("[Error] Create template failed")
			return
		}
	}

	//Send alerts
	for _,alert := range alerts{
		logger.Println("[Info] Begin to Send alert to elasticsearch:",alert)
		jsonalert, err := json.Marshal(alert)
		if nil != err {
			logger.Println("[Error] Transfor alert to json error,",err)
		}
		ret := DoRequest(http.MethodPut,esurl+"/prometheus_alert/_doc/"+uuid.New().String(),bytes.NewBuffer(jsonalert))
		if ok != ret{
			logger.Println("[Error] Put alter failed,alert:",alerts)
			return
		}
	}
}


//Do request base on elasticsearch of version 6.8
func DoRequest(method,url string, body io.Reader) int8{
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	client := &http.Client{Transport:tr}
	req, err := http.NewRequest(method, url, body)
	if nil != err{
		logger.Println("[Error] New request error",err)
		return ng
	}
	req.SetBasicAuth(esusername, espasswd)
	//We just use GET/PUT/POST
	req.Header.Set("Content-Type", "application/json")
	resp,err := client.Do(req)
	if nil != err{
		logger.Println("[Error] Do request error",err)
		return ng
	}
	defer resp.Body.Close()

	if 200 <= resp.StatusCode && resp.StatusCode < 300 {
		return ok
	}

	logger.Println("[Error] Request error,resp.StatusCode =",resp.StatusCode)
	return ng
}


var (
	esusername, espasswd, esurl, port string
	h bool
	defaultPort = "8888"
)

func usage() {
	fmt.Fprintf(os.Stderr, `
Usage: prometheusalert2es [options...]

prometheusalert2es --esurl=${url} --esusername=${username} --espasswd=${passwd} --port=${port}

Options:
`)
	flag.PrintDefaults()
}

func main(){
	flag.BoolVar(&h, "h", false, "Help info")
	flag.StringVar(&esusername, "esusername", "", "Elasticsearch username")
	flag.StringVar(&espasswd, "espasswd", "", "Elasticsearch password")
	flag.StringVar(&esurl, "esurl", "", "Elasticsearch url")
	flag.StringVar(&port, "port", defaultPort, "Prometheusalert2es listen port")
	flag.Parse()

	if h{
		usage()
		return
	}

	if "" == esurl{
		esurl = os.Getenv("ESURL")
	}

	if "" == esusername{
		esusername = os.Getenv("ESUSERNAME")
	}

	if "" == espasswd{
		espasswd = os.Getenv("ESPASSWD")
	}

	if defaultPort == port{
		if port = os.Getenv("PORT"); "" == port {
			port = defaultPort
		}
	}

	if "" == esurl || "" == esusername || "" == espasswd {
		logger.Println("[Error] Must specific esusername, espasswd and esurl")
		usage()
		return
	}

	esurl = strings.TrimRight(esurl, "/")
	serverHandler := http.NewServeMux()
	serverHandler.Handle("/", &AlertHandler{})

	logger.Println("[Info] Start listen on:"+port)
	http.ListenAndServe(":"+port,serverHandler)

}


//Copy from github.com/prometheus/prometheus/notifier/notifier.go
//Different version of prometheus may hava different Alert struct, be careful
type Alerts []Alert
// Alert is a generic representation of an alert in the Prometheus eco-system.
type Alert struct {
	// Label value pairs for purpose of aggregation, matching, and disposition
	// dispatching. This must minimally include an "alertname" label.
	Labels labels.Labels `json:"labels"`

	// Extra key/value information which does not define alert identity.
	Annotations labels.Labels `json:"annotations"`

	// The known time range for this alert. Both ends are optional.
	StartsAt     time.Time `json:"startsAt,omitempty"`
	EndsAt       time.Time `json:"endsAt,omitempty"`
	GeneratorURL string    `json:"generatorURL,omitempty"`
}
