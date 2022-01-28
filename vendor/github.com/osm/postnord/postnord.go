package postnord

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

type postNord struct {
	apiKey string
	locale string
}

// New returns a new instance of the PostNord API client.
func New(apiKey, locale string) *postNord {
	return &postNord{apiKey, locale}
}

const traceURL = "https://api2.postnord.com/rest/shipment/v5/trackandtrace/findByIdentifier.json?apikey=%s&locale=%s&id=%s"

// FindByIdentifier returns tracking information for the given id number.
func (pn *postNord) FindByIdentifierV5(id string) (*ShipmentV5TrackAndTrace, error) {
	res, err := http.Get(fmt.Sprintf(traceURL, pn.apiKey, pn.locale, id))
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()
	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	var ret ShipmentV5TrackAndTrace
	err = json.Unmarshal(data, &ret)
	if err != nil {
		return nil, err
	}

	return &ret, nil
}
