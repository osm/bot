package postnord

// Data models as defined at
// https://developer.postnord.com/api/details?systemName=shipment-v5-trackandtrace

type ShipmentV5TrackAndTrace struct {
	TrackingInformationResponse TrackingInformationResponse `json:"TrackingInformationResponse"`
}

type TrackingInformationResponse struct {
	CompositeFault CompositeFault `json:"compositeFault"`
	Shipments      []ShipmentDto  `json:"shipments"`
}

type CompositeFault struct {
	Faults []Fault `json:"faults"`
}

type Fault struct {
	FaultCode       string       `json:"faultCode"`
	ExplanationText string       `json:"explanationText"`
	ParamValues     []ParamValue `json:"paramValues"`
}

type ParamValue struct {
	Param string `json:"param"`
	Value string `json:"value"`
}

type ShipmentDto struct {
	ShipmentId               string                 `json:"shipmentId"`
	URI                      string                 `json:"uri"`
	AssessedNumberOfItems    int                    `json:"assessedNumberOfItems"`
	CashOnDeliveryText       string                 `json:"cashOnDeliveryText"`
	DeliveryDate             string                 `json:"deliveryDate"`
	ReturnDate               string                 `json:"returnDate"`
	EstimatedTimeOfArrival   string                 `json:"estimatedTimeOfArrival"`
	NumberOfPallets          string                 `json:"numberOfPallets"`
	FlexChangePossible       bool                   `json:"flexChangePossible"`
	Service                  ServiceDto             `json:"service"`
	Consignor                ConsignorDto           `json:"consignor"`
	Consignee                ConsigneeDto           `json:"consignee"`
	ReturnParty              ReturnPartyDto         `json:"returnParty"`
	PickupParty              PickupPartyDto         `json:"pickupParty"`
	CollectionParty          CollectionPartyDto     `json:"collectionParty"`
	StatusText               ShipmentStatusTextDto  `json:"statusText"`
	Status                   string                 `json:"status"` //, optional = ['CREATED', 'AVAILABLE_FOR_DELIVERY', 'DELAYED', 'DELIVERED', 'DELIVERY_IMPOSSIBLE', 'DELIVERY_REFUSED', 'EXPECTED_DELAY', 'INFORMED', 'EN_ROUTE', 'OTHER', 'RETURNED', 'STOPPED', 'SPLIT'],
	DeliveryPoint            DeliveryPointDto       `json:"deliveryPoint"`
	DestinationDeliveryPoint DeliveryPointDto       `json:"destinationDeliveryPoint"`
	TotalWeight              WeightDto              `json:"totalWeight"`
	TotalVolume              VolumeDto              `json:"totalVolume"`
	AssessedWeight           WeightDto              `json:"assessedWeight"`
	AssessedVolume           VolumeDto              `json:"assessedVolume"`
	SplitStatuses            []SplitStatusDto       `json:"splitStatuses"`
	ShipmentReference        []ReferenceDto         `json:"shipmentReference"`
	AdditionalService        []AdditionalServiceDto `json:"additionalService"`
	HarmonizedVersion        int                    `json:"harmonizedVersion"`
	Items                    []ItemDto              `json:"items"`
}

type ServiceDto struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

type ConsignorDto struct {
	Name       string     `json:"name"`
	Issuercode string     `json:"issuercode"`
	Address    AddressDto `json:"address"`
}

type AddressDto struct {
	Street1     string `json:"street1"`
	Street2     string `json:"street2"`
	City        string `json:"city"`
	CountryCode string `json:"countryCode"`
	Country     string `json:"country"`
	PostCode    string `json:"postCode"`
}

type ConsigneeDto struct {
	Name    string     `json:"name"`
	Address AddressDto `json:"address"`
}

type ReturnPartyDto struct {
	Name    string     `json:"name"`
	Address AddressDto `json:"address"`
	Contact ContactDto `json:"contact"`
}

type ContactDto struct {
	ContactName string `json:"contactName"`
	Phone       string `json:"phone"`
	MobilePhone string `json:"mobilePhone"`
	Email       string `json:"email"`
}

type PickupPartyDto struct {
	Name    string     `json:"name"`
	Address AddressDto `json:"address"`
	Contact ContactDto `json:"contact"`
}

type CollectionPartyDto struct {
	Name    string     `json:"name"`
	Address AddressDto `json:"address"`
	Contact ContactDto `json:"contact"`
}

type ShipmentStatusTextDto struct {
	Header                 string `json:"header"`
	Body                   string `json:"body"`
	EstimatedTimeOfArrival string `json:"estimatedTimeOfArrival"`
}

type DeliveryPointDto struct {
	Name             string            `json:"name"`
	LocationDetail   string            `json:"locationDetail"`
	Address          AddressDto        `json:"address"`
	Contact          ContactDto        `json:"contact"`
	Coordinate       []CoordinateDto   `json:"coordinate"`
	OpeningHour      []OpeningHoursDto `json:"openingHour"`
	DisplayName      string            `json:"displayName"`
	LocationId       string            `json:"locationId"`
	ServicePointType string            `json:"servicePointType"`
}

type CoordinateDto struct {
	SrId     string `json:"srId"`
	Northing string `json:"northing"`
	Easting  string `json:"easting"`
}

type OpeningHoursDto struct {
	OpenFrom  string `json:"openFrom"`
	OpenTo    string `json:"openTo"`
	OpenFrom2 string `json:"openFrom2"`
	OpenTo2   string `json:"openTo2"`
	Monday    bool   `json:"monday"`
	Tuesday   bool   `json:"tuesday"`
	Wednesday bool   `json:"wednesday"`
	Thursday  bool   `json:"thursday"`
	Friday    bool   `json:"friday"`
	Saturday  bool   `json:"saturday"`
	Sunday    bool   `json:"sunday"`
}

type WeightDto struct {
	Value string `json:"value"`
	Unit  string `json:"unit"` // ['g', 'kg']
}

type VolumeDto struct {
	Value string `json:"value"`
	Unit  string `json:"unit"` // ['cm3', 'dm3', 'm3']
}

type SplitStatusDto struct {
	NoItemsWithStatus int    `json:"noItemsWithStatus"`
	NoItems           int    `json:"noItems"`
	StatusDescription string `json:"statusDescription"`
	Status            string `json:"status"` // ['CREATED', 'AVAILABLE_FOR_DELIVERY', 'AVAILABLE_FOR_DELIVERY_PAR_LOC', 'DELAYED', 'DELIVERED', 'DELIVERY_IMPOSSIBLE', 'DELIVERY_REFUSED', 'EXPECTED_DELAY', 'INFORMED', 'EN_ROUTE', 'OTHER', 'RETURNED', 'STOPPED']
}

type ReferenceDto struct {
	Value string `json:"value"`
	Type  string `json:"type"`
	Name  string `json:"name"`
}

type AdditionalServiceDto struct {
	Code      string `json:"code"`
	GroupCode string `json:"groupCode"`
	Name      string `json:"name"`
}

type ItemDto struct {
	ItemId                 string             `json:"itemId"`
	EstimatedTimeOfArrival string             `json:"estimatedTimeOfArrival"`
	DropOffDate            string             `json:"dropOffDate"`
	DeliveryDate           string             `json:"deliveryDate"`
	ReturnDate             string             `json:"returnDate"`
	TypeOfItem             string             `json:"typeOfItem"`
	TypeOfItemName         string             `json:"typeOfItemName"`
	TypeOfItemActual       string             `json:"typeOfItemActual"`
	TypeOfItemActualName   string             `json:"typeOfItemActualName"`
	AdditionalInformation  string             `json:"additionalInformation"`
	NoItems                int                `json:"noItems"`
	NumberOfPallets        string             `json:"numberOfPallets"`
	Status                 string             `json:"status"` // ['CREATED', 'AVAILABLE_FOR_DELIVERY', 'AVAILABLE_FOR_DELIVERY_PAR_LOC', 'DELAYED', 'DELIVERED', 'DELIVERY_IMPOSSIBLE', 'DELIVERY_REFUSED', 'EXPECTED_DELAY', 'INFORMED', 'EN_ROUTE', 'OTHER', 'RETURNED', 'STOPPED'],
	StatusText             StatusTextDto      `json:"statusText"`
	Acceptor               AcceptorDto        `json:"acceptor"`
	StatedMeasurement      MeasurementDto     `json:"statedMeasurement"`
	AssessedMeasurement    MeasurementDto     `json:"assessedMeasurement"`
	Events                 []TrackingEventDto `json:"events"`
	References             []ReferenceDto     `json:"references"`
	PreviousItemStates     []ItemStatus       `json:"previousItemStates"`
	FreeText               []ItemFreeTextDto  `json:"freeText"`
}

type StatusTextDto struct {
	Header                 string `json:"header"`
	Body                   string `json:"body"`
	EstimatedTimeOfArrival string `json:"estimatedTimeOfArrival"`
}

type AcceptorDto struct {
	SignatureReference string `json:"signatureReference"`
	Name               string `json:"name"`
}

type MeasurementDto struct {
	Weight WeightDto   `json:"weight"`
	Length DistanceDto `json:"length"`
	Height DistanceDto `json:"height"`
	Width  DistanceDto `json:"width"`
	Volume VolumeDto   `json:"volume"`
}

type DistanceDto struct {
	Value string `json:"value"`
	Unit  string `json:"unit"` // ['mm', 'cm', 'dm', 'm']
}

type TrackingEventDto struct {
	EventTime          string         `json:"eventTime"`
	EventCode          string         `json:"eventCode"`
	Location           LocationDto    `json:"location"`
	GeoLocation        GeoLocationDto `json:"geoLocation"`
	Status             string         `json:"status"`
	EventDescription   string         `json:"eventDescription"`
	LocalDeviationDode string         `json:"localDeviationDode"`
}

type LocationDto struct {
	Name         string `json:"name"`
	CountryCode  string `json:"countryCode"`
	Country      string `json:"country"`
	LocationId   string `json:"locationId"`
	DisplayName  string `json:"displayName"`
	Postcode     string `json:"postcode"`
	City         string `json:"city"`
	LocationType string `json:"locationType"` // ['HUB', 'DEPOT', 'DPD_DEPOT', 'LOAD_POINT', 'SERVICE_POINT', 'UNDEF', 'CUSTOMER_LOCATION', 'SLINGA', 'DISTRIBUTION_PARTNER', 'IPS_LOCATION', 'POSTAL_SERVICE_TERMINAL', 'DELIVERY_POINT']
}

type GeoLocationDto struct {
	GeoNorthing        float64 `json:"geoNorthing"`
	GeoEasting         float64 `json:"geoEasting"`
	GeoReferenceSystem string  `json:"geoReferenceSystem"`
	GeoPostalCode      string  `json:"geoPostalCode"`
	GeoCity            string  `json:"geoCity"`
	GeoCountryCode     string  `json:"geoCountryCode"`
}

type ItemStatus string

type ItemFreeTextDto struct {
	Text string `json:"text"`
	Type string `json:"type"` // ['ICN', 'SIC', 'DEL', 'DIN', 'ADR', 'NEI', 'UNDEF']
}
