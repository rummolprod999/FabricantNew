package main

import (
	"database/sql"
	"encoding/xml"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"os/exec"
	"strings"
	"time"
)

var DataTrades = time.Time{}

func Parser() {
	ParserPage()
	//time.Sleep(time.Second * 10)

}

func ParserPage() {
	defer func() {
		if p := recover(); p != nil {
			Logging(p)
		}
	}()
	UrlXml = ""
	if Count == 0 {
		UrlXml = fmt.Sprintf("https://www.fabrikant.ru/trade-feed/?action=xml_export_auctions")
	} else {
		Lastdate := time.Now().AddDate(0, 0, -1*Count)
		TStr := Lastdate.Format("02.01.2006")
		UrlXml = fmt.Sprintf("https://www.fabrikant.ru/trade-feed/?action=xml_export_auctions&date=%s&time=00:00", TStr)
	}
	Logging("Запрошенная страница ", UrlXml)
	r := DownloadPage(UrlXml)
	/*_ = ioutil.WriteFile(string(FileTmp), []byte(r), 0644)*/
	if r != "" {
		ParsingString(r)
	} else {
		Logging("Получили пустую строку", UrlXml)
	}
	for (DataTrades != time.Time{}) {
		Logging("Самая последняя дата в файле", DataTrades)
		tm := fmt.Sprintf("%02d:%02d", DataTrades.Hour(), DataTrades.Minute())
		dat := fmt.Sprintf("%02d.%02d.%d", DataTrades.Day(), DataTrades.Month(), DataTrades.Year())
		UrlXml = fmt.Sprintf("https://www.fabrikant.ru/trade-feed/?action=xml_export_auctions&date=%s&time=%s", dat, tm)
		r := DownloadPage(UrlXml)
		if r != "" {
			ParsingString(r)
		} else {
			Logging("Получили пустую строку", UrlXml)
			DataTrades = time.Time{}
		}
	}
}
func ParsingString(s string) {
	var FileProt FileProtocols
	if err := xml.Unmarshal([]byte(s), &FileProt); err != nil {
		Logging("Ошибка при парсинге строки", err)
		DataTrades = time.Time{}
		return
	}
	var Dsn = fmt.Sprintf("%s:%s@/%s?charset=utf8&parseTime=true&readTimeout=60m&maxAllowedPacket=0&timeout=60m&writeTimeout=60m&autocommit=true&loc=Local", UserDb, PassDb, DbName)
	db, err := sql.Open("mysql", Dsn)
	defer db.Close()
	//db.SetMaxOpenConns(2)
	db.SetConnMaxLifetime(time.Second * 3600)
	if err != nil {
		Logging("Ошибка подключения к БД", err)
		DataTrades = time.Time{}
	}
	if len(FileProt.TradeList) == 0 {
		Logging("Нет процедур в файле", UrlXml, s)
		DataTrades = time.Time{}
	}
	if len(FileProt.TradeList) >= 500 {
		Logging("Получено больше 500 тендеров ", len(FileProt.TradeList))
		DataTrades = getTimeMoscow(FileProt.TradeList[0].ChangeDate)
	} else {
		DataTrades = time.Time{}
	}
	for _, t := range FileProt.TradeList {
		e := ParsingTrade(t, db)
		if e != nil {
			Logging("Ошибка парсера в протоколе", e)
			continue
		}
	}
}

func ParsingTrade(t Trade, db *sql.DB) error {
	TradeId := t.TradeId
	if TradeId == "" {
		TradeId = t.Id
	}
	if TradeId == "" {
		Logging("Пустой идентификатор закупки")
		return nil
	}
	PublicationDate := getTimeMoscow(t.PublicationDate)
	DateUpdated := getTimeMoscow(t.ChangeDate)
	if (DateUpdated == time.Time{}) {
		DateUpdated = PublicationDate
	}
	IdXml := t.Id
	Version := 0
	stmt, _ := db.Prepare(fmt.Sprintf("SELECT id_tender FROM %stender WHERE purchase_number = ? AND date_version = ? AND type_fz = ?", Prefix))
	res, err := stmt.Query(TradeId, DateUpdated, typeFz)
	stmt.Close()
	if err != nil {
		Logging("Ошибка выполения запроса", err)
		return err
	}
	if res.Next() {
		//Logging("Такой тендер уже есть", TradeId)
		res.Close()
		return nil
	}
	res.Close()
	var cancelStatus = 0
	if TradeId != "" {
		stmt, err := db.Prepare(fmt.Sprintf("SELECT id_tender, date_version FROM %stender WHERE purchase_number = ? AND cancel=0 AND type_fz = ?", Prefix))
		rows, err := stmt.Query(TradeId, typeFz)
		stmt.Close()
		if err != nil {
			Logging("Ошибка выполения запроса", err)
			return err
		}
		for rows.Next() {
			var idTender int
			var dateVersion time.Time
			err = rows.Scan(&idTender, &dateVersion)
			if err != nil {
				Logging("Ошибка чтения результата запроса", err)
				return err
			}
			//fmt.Println(DateUpdated.Sub(dateVersion))
			if dateVersion.Sub(DateUpdated) <= 0 {
				stmtupd, _ := db.Prepare(fmt.Sprintf("UPDATE %stender SET cancel=1 WHERE id_tender = ?", Prefix))
				_, err = stmtupd.Exec(idTender)
				stmtupd.Close()

			} else {
				cancelStatus = 1
			}

		}
		rows.Close()

	}
	Href := t.TradeUri
	Title := t.Title
	CommonName := t.CommonName
	PurchaseObjectInfo := strings.TrimSpace(fmt.Sprintf("%s %s", Title, CommonName))
	//if len(t.Lots) > 0 {
	//	if len(t.Lots[0].LotDataSubj) > 0 {
	//		PurchaseObjectInfo = strings.TrimSpace(fmt.Sprintf("%s %s", PurchaseObjectInfo, t.Lots[0].LotDataSubj[0]))
	//	}
	//}

	NoticeVersion := ""
	PrintForm := Href
	IdOrganizer := 0
	if t.IdOrganizer != "" {
		UrlOrg := fmt.Sprintf("https://www.fabrikant.ru/trade-feed/?action=xml_export_firm&id=%s", t.IdOrganizer)
		org := DownloadPage(UrlOrg)
		//fmt.Println(org)
		if org != "" {
			var Org Organizer
			if err := xml.Unmarshal([]byte(org), &Org); err != nil {
				Logging("Ошибка при парсинге строки", err)
				IdOrganizer = 0
			} else {
				if Org.OrganizerINN != "" {
					stmt, _ := db.Prepare(fmt.Sprintf("SELECT id_organizer FROM %sorganizer WHERE inn = ? AND kpp = ?", Prefix))
					rows, err := stmt.Query(Org.OrganizerINN, Org.OrganizerKPP)
					stmt.Close()
					if err != nil {
						Logging("Ошибка выполения запроса", err)
						return err
					}
					if rows.Next() {
						err = rows.Scan(&IdOrganizer)
						if err != nil {
							Logging("Ошибка чтения результата запроса", err)
							return err
						}
						rows.Close()
					} else {
						rows.Close()
						ContactPerson := ""
						Email := ""
						Phone := ""
						if len(t.AdditData) > 0 {
							ContactPerson = strings.TrimSpace(fmt.Sprintf("%s %s %s", t.AdditData[0].LastName, t.AdditData[0].FirstName, t.AdditData[0].MiddleName))
							Email = t.AdditData[0].Email
							Phone = t.AdditData[0].Phone
						}
						stmt, _ := db.Prepare(fmt.Sprintf("INSERT INTO %sorganizer SET full_name = ?, inn = ?, kpp = ?, post_address = ?, fact_address = ?, contact_email = ?, contact_phone = ?, contact_person = ?", Prefix))
						res, err := stmt.Exec(Org.OrganizerName, Org.OrganizerINN, Org.OrganizerKPP, Org.OrganizerPostAddress, Org.OrganizerLegalAddress, Email, Phone, ContactPerson)
						stmt.Close()
						if err != nil {
							Logging("Ошибка вставки организатора", err)
							return err
						}
						id, err := res.LastInsertId()
						IdOrganizer = int(id)
					}
				}
			}

		} else {
			Logging("Получили пустую строку", UrlOrg)
		}

	}
	IdPlacingWay := 0
	if t.TradeType != "" {
		stmt, _ := db.Prepare(fmt.Sprintf("SELECT id_placing_way FROM %splacing_way WHERE name = ? LIMIT 1", Prefix))
		rows, err := stmt.Query(t.TradeType)
		stmt.Close()
		if err != nil {
			Logging("Ошибка выполения запроса", err)
			return err
		}
		if rows.Next() {
			err = rows.Scan(&IdPlacingWay)
			if err != nil {
				Logging("Ошибка чтения результата запроса", err)
				return err
			}
			rows.Close()
		} else {
			rows.Close()
			conf := GetConformity(t.TradeType)
			stmt, _ := db.Prepare(fmt.Sprintf("INSERT INTO %splacing_way SET name = ?, conformity = ?", Prefix))
			res, err := stmt.Exec(t.TradeType, conf)
			stmt.Close()
			if err != nil {
				Logging("Ошибка вставки placing way", err)
				return err
			}
			id, err := res.LastInsertId()
			IdPlacingWay = int(id)

		}
	}
	IdEtp := 0
	etpName := "ЭТП «Фабрикант»"
	etpUrl := "https://www.fabrikant.ru"
	if true {
		stmt, _ := db.Prepare(fmt.Sprintf("SELECT id_etp FROM %setp WHERE name = ? AND url = ? LIMIT 1", Prefix))
		rows, err := stmt.Query(etpName, etpUrl)
		stmt.Close()
		if err != nil {
			Logging("Ошибка выполения запроса", err)
			return err
		}
		if rows.Next() {
			err = rows.Scan(&IdEtp)
			if err != nil {
				Logging("Ошибка чтения результата запроса", err)
				return err
			}
			rows.Close()
		} else {
			rows.Close()
			stmt, _ := db.Prepare(fmt.Sprintf("INSERT INTO %setp SET name = ?, url = ?, conf=0", Prefix))
			res, err := stmt.Exec(etpName, etpUrl)
			stmt.Close()
			if err != nil {
				Logging("Ошибка вставки etp", err)
				return err
			}
			id, err := res.LastInsertId()
			IdEtp = int(id)
		}
	}
	var EndDate = time.Time{}
	var BiddingDate = time.Time{}
	if len(t.Dates) > 0 {
		EndDate = getTimeMoscow(t.Dates[0].AppEndDate)
		if (EndDate == time.Time{}) {
			EndDate = getTimeMoscow(t.Dates[0].EndDate)
		}
		BiddingDate = getTimeMoscow(t.Dates[0].AuctStartDate)
	}
	idTender := 0
	stmtt, _ := db.Prepare(fmt.Sprintf("INSERT INTO %stender SET id_region = 0, id_xml = ?, purchase_number = ?, doc_publish_date = ?, href = ?, purchase_object_info = ?, type_fz = ?, id_organizer = ?, id_placing_way = ?, id_etp = ?, end_date = ?, cancel = ?, date_version = ?, num_version = ?, notice_version = ?, xml = ?, print_form = ?, bidding_date = ?", Prefix))
	rest, err := stmtt.Exec(IdXml, TradeId, PublicationDate, Href, PurchaseObjectInfo, typeFz, IdOrganizer, IdPlacingWay, IdEtp, EndDate, cancelStatus, DateUpdated, Version, NoticeVersion, UrlXml, PrintForm, BiddingDate)
	stmtt.Close()
	if err != nil {
		Logging("Ошибка вставки tender", err)
		return err
	}
	idt, err := rest.LastInsertId()
	idTender = int(idt)
	Addtender++
	for _, doc := range t.Documents {
		if doc.Name != "" {
			stmt, _ := db.Prepare(fmt.Sprintf("INSERT INTO %sattachment SET id_tender = ?, file_name = ?, url = ?, description = ?", Prefix))
			_, err := stmt.Exec(idTender, doc.Name, doc.Link, doc.Description)
			stmt.Close()
			if err != nil {
				Logging("Ошибка вставки attachment", err)
				return err
			}
		}
	}
	idCustomer := 0
	if len(t.Customers) > 0 {
		if t.Customers[0].CustomerId != "" {
			UrlOrg := fmt.Sprintf("https://www.fabrikant.ru/trade-feed/?action=xml_export_firm&id=%s", t.IdOrganizer)
			org := DownloadPage(UrlOrg)
			//fmt.Println(org)
			if org != "" {
				var Org Organizer
				if err := xml.Unmarshal([]byte(org), &Org); err != nil {
					Logging("Ошибка при парсинге строки", err)
					IdOrganizer = 0
				} else {
					if Org.OrganizerINN != "" {
						stmt, _ := db.Prepare(fmt.Sprintf("SELECT id_customer FROM %scustomer WHERE inn = ?", Prefix))
						rows, err := stmt.Query(Org.OrganizerINN)
						stmt.Close()
						if err != nil {
							Logging("Ошибка выполения запроса", err)
							return err
						}
						if rows.Next() {
							err = rows.Scan(&idCustomer)
							if err != nil {
								Logging("Ошибка чтения результата запроса", err)
								return err
							}
							rows.Close()
						} else {
							rows.Close()
							out, err := exec.Command("uuidgen").Output()
							if err != nil {
								Logging("Ошибка генерации UUID", err)
								return err
							}
							stmt, _ := db.Prepare(fmt.Sprintf("INSERT INTO %scustomer SET full_name = ?, inn = ?, reg_num = ?, is223=1", Prefix))
							res, err := stmt.Exec(Org.OrganizerName, Org.OrganizerINN, out)
							stmt.Close()
							if err != nil {
								Logging("Ошибка вставки заказчика", err)
								return err
							}
							id, err := res.LastInsertId()
							idCustomer = int(id)
						}
					}
				}

			} else {
				Logging("Получили пустую строку", UrlOrg)
			}

		} else if t.Customers[0].CustomerName != "" {
			stmt, _ := db.Prepare(fmt.Sprintf("SELECT id_customer FROM %scustomer WHERE full_name LIKE ? LIMIT 1", Prefix))
			rows, err := stmt.Query(t.Customers[0].CustomerName)
			stmt.Close()
			if err != nil {
				Logging("Ошибка выполения запроса", err)
				return err
			}
			if rows.Next() {
				err = rows.Scan(&idCustomer)
				if err != nil {
					Logging("Ошибка чтения результата запроса", err)
					return err
				}
				rows.Close()
			} else {
				rows.Close()
				out, err := exec.Command("uuidgen").Output()
				if err != nil {
					Logging("Ошибка генерации UUID", err)
					return err
				}
				stmt, _ := db.Prepare(fmt.Sprintf("INSERT INTO %scustomer SET full_name = ?, is223=1, reg_num = ?", Prefix))
				res, err := stmt.Exec(t.Customers[0].CustomerName, out)
				stmt.Close()
				if err != nil {
					Logging("Ошибка вставки заказчика", err)
					return err
				}
				id, err := res.LastInsertId()
				idCustomer = int(id)
			}
		}
	}
	var LotNumber = 1
	for _, lot := range t.Lots {
		var MaxPrice float64 = 0
		if len(lot.Prices) > 0 {
			for _, v := range lot.Prices {
				MaxPrice += v
			}
		}
		idLot := 0
		Currency := ""
		if len(t.AdditData) > 0 {
			Currency = t.AdditData[0].Currency
		}
		stmtl, _ := db.Prepare(fmt.Sprintf("INSERT INTO %slot SET id_tender = ?, lot_number = ?, max_price = ?, currency = ?", Prefix))
		resl, err := stmtl.Exec(idTender, LotNumber, MaxPrice, Currency)
		stmtl.Close()
		if err != nil {
			Logging("Ошибка вставки lot", err)
			return err
		}
		id, _ := resl.LastInsertId()
		idLot = int(id)
		for _, cf := range lot.Classifiers {

			if strings.Index(cf.Type, "ОКПД2") != -1 {
				okpd2Code := cf.CatCode
				okpdName := cf.CatName
				okpd2GroupCode, okpd2GroupLevel1Code := GetOkpd(okpd2Code)
				Name := ""
				if len(lot.LotDataSubj) > 0 {
					Name = lot.LotDataSubj[0]
				}
				if Name == "" && len(lot.LotDataSubj) > 0 {
					Name = lot.LotDataSubj[0]
				}
				if Name == "" {
					Name = okpdName
				}
				stmtr, _ := db.Prepare(fmt.Sprintf("INSERT INTO %spurchase_object SET id_lot = ?, id_customer = ?, okpd2_code = ?, okpd2_group_code = ?, okpd2_group_level1_code = ?, okpd_name = ?, name = ?, quantity_value = ?, customer_quantity_value = ?, okei = ?, price = ?", Prefix))
				_, errr := stmtr.Exec(idLot, idCustomer, okpd2Code, okpd2GroupCode, okpd2GroupLevel1Code, okpdName, Name, lot.Quantity, lot.Quantity, lot.Okei, MaxPrice)
				stmtr.Close()
				if errr != nil {
					Logging("Ошибка вставки purchase_object", errr)
					return err
				}
			}

		}
		for _, pos := range lot.Positions {
			stmtr, _ := db.Prepare(fmt.Sprintf("INSERT INTO %spurchase_object SET id_lot = ?, id_customer = ?, name = ?, quantity_value = ?, customer_quantity_value = ?, price = ?, okei = ?", Prefix))
			_, errr := stmtr.Exec(idLot, idCustomer, pos.Name, pos.Quantity, pos.Quantity, pos.PriceUnit, pos.Unit)
			stmtr.Close()
			if errr != nil {
				Logging("Ошибка вставки purchase_object", errr)
				return err
			}
		}
		for _, appd := range t.AdditData {
			DelivTerm := strings.TrimSpace(fmt.Sprintf("%s %s %s", appd.PaymentCond, appd.DelivCond, appd.Comments))
			stmtr, _ := db.Prepare(fmt.Sprintf("INSERT INTO %scustomer_requirement SET id_lot = ?, id_customer = ?, delivery_term = ?, delivery_place = ?", Prefix))
			_, errr := stmtr.Exec(idLot, idCustomer, DelivTerm, lot.DeliveryPlace)
			stmtr.Close()
			if errr != nil {
				Logging("Ошибка вставки customer_requirement", errr)
				return err
			}
			if appd.Preference != "" {
				spr, _ := db.Prepare(fmt.Sprintf("INSERT INTO %spreferense SET id_lot = ?, name = ?", Prefix))
				_, errr := spr.Exec(idLot, appd.Preference)
				spr.Close()
				if errr != nil {
					Logging("Ошибка вставки preferense", errr)
					return err
				}
			}
			if appd.Requirement != "" {
				spr, _ := db.Prepare(fmt.Sprintf("INSERT INTO %srequirement SET id_lot = ?, content = ?", Prefix))
				_, errr := spr.Exec(idLot, appd.Requirement)
				spr.Close()
				if errr != nil {
					Logging("Ошибка вставки requirement", errr)
					return err
				}
			}
		}
		LotNumber++
	}
	e := TenderKwords(db, idTender)
	if e != nil {
		Logging("Ошибка обработки TenderKwords", e)
	}

	e1 := AddVerNumber(db, TradeId)
	if e1 != nil {
		Logging("Ошибка обработки AddVerNumber", e1)
	}
	return nil
}
