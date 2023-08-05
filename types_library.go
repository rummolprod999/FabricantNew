package main

type FileProtocols struct {
	TradeList []Trade `xml:"tender"`
}

type Trade struct {
	TradeId         string           `xml:"contract_number"`
	Id              string           `xml:"id"`
	PublicationDate string           `xml:"purchase_start"`
	ChangeDate      string           `xml:"change_date"`
	TradeUri        string           `xml:"link"`
	TradeType       string           `xml:"contract_type"`
	Title           string           `xml:"title"`
	CommonName      string           `xml:"name"`
	IdOrganizer     string           `xml:"organizer>id"`
	Documents       []Documentation  `xml:"documentation"`
	Dates           []Dates          `xml:"dates"`
	Customers       []Customer       `xml:"customers"`
	Lots            []Lot            `xml:"lot_data"`
	AdditData       []additionalData `xml:"additional_data"`
}
type PositionLot struct {
	Name      string `xml:"name"`
	Quantity  string `xml:"quantity"`
	PriceUnit string `xml:"price_unit"`
	Unit      string `xml:"unit"`
}
type additionalData struct {
	FirstName   string `xml:"contact_name>first_name"`
	LastName    string `xml:"contact_name>last_name"`
	MiddleName  string `xml:"contact_name>middle_name"`
	Phone       string `xml:"contact_name>phone"`
	Email       string `xml:"contact_name>email"`
	Currency    string `xml:"currency"`
	DelivCond   string `xml:"delivery_condition"`
	PaymentCond string `xml:"payment_condition"`
	Comments    string `xml:"comments"`
	Preference  string `xml:"participant_preference"`
	Requirement string `xml:"participant_requirement"`
}
type Classifier struct {
	CatCode string `xml:"category_code"`
	CatName string `xml:"category_name"`
	Type    string `xml:"type"`
}
type Dates struct {
	AppEndDate    string `xml:"application_end_date"`
	EndDate       string `xml:"end_date"`
	AuctStartDate string `xml:"auction_start_date"`
}
type Organizer struct {
	OrganizerName         string `xml:"firm>Name"`
	OrganizerINN          string `xml:"firm>Inn"`
	OrganizerKPP          string `xml:"firm>Kpp"`
	OrganizerOGRN         string `xml:"firm>Ogrn"`
	OrganizerPostAddress  string `xml:"firm>PostAddress"`
	OrganizerLegalAddress string `xml:"firm>LegalAddress"`
}
type Customer struct {
	CustomerId   string `xml:"id"`
	CustomerName string `xml:"name"`
}
type Documentation struct {
	Name        string `xml:"name"`
	Description string `xml:"description"`
	Link        string `xml:"link"`
}

type Lot struct {
	LotId         int           `xml:"lot_id"`
	LotDataSubj   []string      `xml:"subject"`
	Prices        []string      `xml:"price"`
	Classifiers   []Classifier  `xml:"classifier"`
	Quantity      string        `xml:"quantity"`
	Okei          string        `xml:"unit"`
	Positions     []PositionLot `xml:"positions"`
	DeliveryPlace string        `xml:"delivery_place"`
}
