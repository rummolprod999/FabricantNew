package main

type FileProtocols struct {
	TradeList []Trade `xml:"tender"`
}

type Trade struct {
	TradeId         string `xml:"contract_number"`
	Id              string `xml:"id"`
	PublicationDate string `xml:"purchase_start"`
	ChangeDate      string `xml:"change_date"`
	//AppEndDate      string `xml:"dates>application_end_date"`
	//EndDate         string `xml:"dates>end_date"`
	//AuctStartDate   string `xml:"dates>auction_start_date"`
	//FinishDate       string `xml:"FinishDate"`
	TradeUri   string `xml:"link"`
	TradeType  string `xml:"contract_type"`
	Title      string `xml:"title"`
	CommonName string `xml:"name"`
	//DocumentationUrl string `xml:"DocumentationUrl"`
	//Lots             []Lot  `xml:"Lot"`
	//Currency         string `xml:"Currency>ISO"`
	CustomerId   string          `xml:"customers>id"`
	CustomerName string          `xml:"customers>name"`
	IdOrganizer  string          `xml:"organizer>id"`
	Documents    []Documentation `xml:"documentation"`
	Dates        []Dates         `xml:"dates"`
	FirstName    string          `xml:"additional_data>contact_name>first_name"`
	LastName     string          `xml:"additional_data>contact_name>last_name"`
	MiddleName   string          `xml:"additional_data>contact_name>middle_name"`
	Phone        string          `xml:"additional_data>contact_name>phone"`
	Email        string          `xml:"additional_data>contact_name>email"`
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

type Documentation struct {
	Name        string `xml:"name"`
	Description string `xml:"description"`
	Link        string `xml:"link"`
}

type Lot struct {
	MaxPrice            string `xml:"PriceInfo>Price"`
	ContractSubject     string `xml:"ContractSubject"`
	ContractSubjectText string `xml:"ContractSubjectText"`
	Description         string `xml:"Description"`
	Quantity            string `xml:"Quantity"`
	MeasureUnit         string `xml:"MeasureUnit"`
}
