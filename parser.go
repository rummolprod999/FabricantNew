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
		UrlXml = fmt.Sprintf("https://www.fabrikant.ru/trade-feed/?action=xml_export_auctions&date=24.01.2018&time=13:00")
	} else {
		Lastdate := time.Now().AddDate(0, 0, -1*Count)
		TStr := Lastdate.Format("02.01.2006")
		UrlXml = fmt.Sprintf("https://www.fabrikant.ru/trade-feed/?action=xml_export_auctions&date=%s", TStr)
	}
	r := DownloadPage(UrlXml)
	if r != "" {
		ParsingString(r)
	} else {
		Logging("Получили пустую строку", UrlXml)
	}
}

func ParsingString(s string) {
	var FileProt FileProtocols
	if err := xml.Unmarshal([]byte(s), &FileProt); err != nil {
		Logging("Ошибка при парсинге строки", err)
		return
	}
	var Dsn = fmt.Sprintf("%s:%s@/%s?charset=utf8&parseTime=true&readTimeout=60m&maxAllowedPacket=0&timeout=60m&writeTimeout=60m&autocommit=true&loc=Local", UserDb, PassDb, DbName)
	db, err := sql.Open("mysql", Dsn)
	defer db.Close()
	//db.SetMaxOpenConns(2)
	db.SetConnMaxLifetime(time.Second * 3600)
	if err != nil {
		Logging("Ошибка подключения к БД", err)
	}
	if len(FileProt.TradeList) == 0 {
		Logging("Нет процедур в файле", UrlXml, s)
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
		Logging("Пустой идентификатор закупки")
		return nil
	}
	PublicationDate := getTimeMoscow(t.PublicationDate)
	DateUpdated := getTimeMoscow(t.ChangeDate)
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
						ContactPerson := strings.TrimSpace(fmt.Sprintf("%s %s %s", t.LastName, t.FirstName, t.MiddleName))
						stmt, _ := db.Prepare(fmt.Sprintf("INSERT INTO %sorganizer SET full_name = ?, inn = ?, kpp = ?, post_address = ?, fact_address = ?, contact_email = ?, contact_phone = ?, contact_person = ?", Prefix))
						res, err := stmt.Exec(Org.OrganizerName, Org.OrganizerINN, Org.OrganizerKPP, Org.OrganizerPostAddress, Org.OrganizerLegalAddress, t.Email, t.Phone, ContactPerson)
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
	EndDate = getTimeMoscow(t.Dates[0].AppEndDate)
	if (EndDate == time.Time{}) {
		EndDate = getTimeMoscow(t.Dates[0].EndDate)
	}
	var BiddingDate = time.Time{}
	BiddingDate = getTimeMoscow(t.Dates[0].AuctStartDate)
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
	if t.CustomerId != "" {
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
					stmt, _ := db.Prepare(fmt.Sprintf("SELECT id_customer FROM %sorganizer WHERE inn = ? AND full_name = ?", Prefix))
					rows, err := stmt.Query(Org.OrganizerINN, Org.OrganizerName)
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
						stmt, _ := db.Prepare(fmt.Sprintf("INSERT INTO %scustomer SET full_name = ?, inn = ?, reg_num = ?", Prefix))
						res, err := stmt.Exec(Org.OrganizerName, Org.OrganizerINN, out)
						stmt.Close()
						if err != nil {
							Logging("Ошибка вставки организатора", err)
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

	} else if t.CustomerName != "" {
		stmt, _ := db.Prepare(fmt.Sprintf("SELECT id_customer FROM %scustomer WHERE full_name LIKE ? LIMIT 1", Prefix))
		rows, err := stmt.Query(t.CustomerName)
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
			res, err := stmt.Exec(t.CustomerName, out)
			stmt.Close()
			if err != nil {
				Logging("Ошибка вставки организатора", err)
				return err
			}
			id, err := res.LastInsertId()
			idCustomer = int(id)
		}
	}
	fmt.Println(idCustomer)
	return nil
}
