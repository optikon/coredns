package central

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

// Start listening for table updates on port 9090.
func (oc *OptikonCentral) startListeningForTableUpdates() {
	oc.server = &http.Server{Addr: ":9090"}
	http.HandleFunc("/", oc.parseTableUpdate)
	go func() {
		if err := oc.server.ListenAndServe(); err != nil {
			fmt.Printf("ListenAndServe() error: %s\n", err)
		}
	}()
}

// Parse incoming requests from edge sites.
func (oc *OptikonCentral) parseTableUpdate(w http.ResponseWriter, r *http.Request) {
	jsn, err := ioutil.ReadAll(r.Body)
	if err != nil {
		fmt.Println("ERROR while reading body:", err)
	}
	update := TableUpdate{}
	if err = json.Unmarshal(jsn, &update); err != nil {
		fmt.Println("ERROR while unmarshalling JSON:", err)
	}
	oc.table.Update(update.Meta.IP, update.Meta.Lon, update.Meta.Lat, update.Services)
}

// Stop listening for updates.
func (oc *OptikonCentral) stopListeningForTableUpdates() {
	oc.server.Shutdown(nil)
}
