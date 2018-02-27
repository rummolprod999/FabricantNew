package main

import (
	"database/sql"
	"fmt"
	"os"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

var layout = "2006-01-02T15:04:05"
var typeFz = 2

func getTimeMoscow(st string) time.Time {
	var p = time.Time{}
	location, _ := time.LoadLocation("Europe/Moscow")
	tz, e := time.Parse(time.RFC3339, st)
	if e != nil {
		tmp, err := strconv.ParseInt(st, 10, 64)
		if err != nil {
			return time.Time{}
		}
		tz = time.Unix(tmp, 0)
	}

	p = tz.In(location)
	return p
}

func SaveStack() {
	if p := recover(); p != nil {
		var buf [4096]byte
		n := runtime.Stack(buf[:], false)
		file, err := os.OpenFile(string(FileLog), os.O_RDWR|os.O_APPEND|os.O_CREATE, 0666)
		defer file.Close()
		if err != nil {
			fmt.Println("Ошибка записи stack log", err)
			return
		}
		fmt.Fprintln(file, fmt.Sprintf("Fatal Error %v", p))
		fmt.Fprintf(file, "%v  ", string(buf[:n]))
	}

}

func GetConformity(conf string) int {
	s := strings.ToLower(conf)
	switch {
	case strings.Index(s, "открыт") != -1:
		return 5
	case strings.Index(s, "аукцион") != -1:
		return 1
	case strings.Index(s, "котиров") != -1:
		return 2
	case strings.Index(s, "предложен") != -1:
		return 3
	case strings.Index(s, "единств") != -1:
		return 4
	default:
		return 6
	}

}

func GetOkpd(s string) (int, string) {
	okpd2GroupCode := 0
	okpd2GroupLevel1Code := ""
	if len(s) > 1 {
		if strings.Index(s, ".") != -1 {
			okpd2GroupCode, _ = strconv.Atoi(s[:2])
		} else {
			okpd2GroupCode, _ = strconv.Atoi(s[:2])
		}
	}
	if len(s) > 3 {
		if strings.Index(s, ".") != -1 {
			okpd2GroupLevel1Code = s[3:4]
		}
	}
	return okpd2GroupCode, okpd2GroupLevel1Code
}

func TenderKwords(db *sql.DB, idTender int) error {
	resString := ""
	stmt, _ := db.Prepare(fmt.Sprintf("SELECT DISTINCT po.name, po.okpd_name FROM %spurchase_object AS po LEFT JOIN %slot AS l ON l.id_lot = po.id_lot WHERE l.id_tender = ?", Prefix, Prefix))
	rows, err := stmt.Query(idTender)
	stmt.Close()
	if err != nil {
		Logging("Ошибка выполения запроса", err)
		return err
	}
	for rows.Next() {
		var name sql.NullString
		var okpdName sql.NullString
		err = rows.Scan(&name, &okpdName)
		if err != nil {
			Logging("Ошибка чтения результата запроса", err)
			return err
		}
		if name.Valid {
			resString = fmt.Sprintf("%s %s ", resString, name.String)
		}
		if okpdName.Valid {
			resString = fmt.Sprintf("%s %s ", resString, okpdName.String)
		}
	}
	rows.Close()
	stmt1, _ := db.Prepare(fmt.Sprintf("SELECT DISTINCT file_name FROM %sattachment WHERE id_tender = ?", Prefix))
	rows1, err := stmt1.Query(idTender)
	stmt1.Close()
	if err != nil {
		Logging("Ошибка выполения запроса", err)
		return err
	}
	for rows1.Next() {
		var attName sql.NullString
		err = rows1.Scan(&attName)
		if err != nil {
			Logging("Ошибка чтения результата запроса", err)
			return err
		}
		if attName.Valid {
			resString = fmt.Sprintf("%s %s ", resString, attName.String)
		}
	}
	rows1.Close()
	idOrg := 0
	stmt2, _ := db.Prepare(fmt.Sprintf("SELECT purchase_object_info, id_organizer FROM %stender WHERE id_tender = ?", Prefix))
	rows2, err := stmt2.Query(idTender)
	stmt2.Close()
	if err != nil {
		Logging("Ошибка выполения запроса", err)
		return err
	}
	for rows2.Next() {
		var idOrgNull sql.NullInt64
		var purOb sql.NullString
		err = rows2.Scan(&purOb, &idOrgNull)
		if err != nil {
			Logging("Ошибка чтения результата запроса", err)
			return err
		}
		if idOrgNull.Valid {
			idOrg = int(idOrgNull.Int64)
		}
		if purOb.Valid {
			resString = fmt.Sprintf("%s %s ", resString, purOb.String)
		}

	}
	rows2.Close()
	if idOrg != 0 {
		stmt3, _ := db.Prepare(fmt.Sprintf("SELECT full_name, inn FROM %sorganizer WHERE id_organizer = ?", Prefix))
		rows3, err := stmt3.Query(idOrg)
		stmt3.Close()
		if err != nil {
			Logging("Ошибка выполения запроса", err)
			return err
		}
		for rows3.Next() {
			var innOrg sql.NullString
			var nameOrg sql.NullString
			err = rows3.Scan(&nameOrg, &innOrg)
			if err != nil {
				Logging("Ошибка чтения результата запроса", err)
				return err
			}
			if innOrg.Valid {

				resString = fmt.Sprintf("%s %s ", resString, innOrg.String)
			}
			if nameOrg.Valid {
				resString = fmt.Sprintf("%s %s ", resString, nameOrg.String)
			}

		}
		rows3.Close()
	}
	stmt4, _ := db.Prepare(fmt.Sprintf("SELECT DISTINCT cus.inn, cus.full_name FROM %scustomer AS cus LEFT JOIN %spurchase_object AS po ON cus.id_customer = po.id_customer LEFT JOIN %slot AS l ON l.id_lot = po.id_lot WHERE l.id_tender = ?", Prefix, Prefix, Prefix))
	rows4, err := stmt4.Query(idTender)
	stmt4.Close()
	if err != nil {
		Logging("Ошибка выполения запроса", err)
		return err
	}
	for rows4.Next() {
		var innC sql.NullString
		var fullNameC sql.NullString
		err = rows4.Scan(&innC, &fullNameC)
		if err != nil {
			Logging("Ошибка чтения результата запроса", err)
			return err
		}
		if innC.Valid {

			resString = fmt.Sprintf("%s %s ", resString, innC.String)
		}
		if fullNameC.Valid {
			resString = fmt.Sprintf("%s %s ", resString, fullNameC.String)
		}
	}
	rows4.Close()
	re := regexp.MustCompile(`\s+`)
	resString = re.ReplaceAllString(resString, " ")
	stmtr, _ := db.Prepare(fmt.Sprintf("UPDATE %stender SET tender_kwords = ? WHERE id_tender = ?", Prefix))
	_, errr := stmtr.Exec(resString, idTender)
	stmtr.Close()
	if errr != nil {
		Logging("Ошибка вставки TenderKwords", errr)
		return err
	}
	return nil
}

func AddVerNumber(db *sql.DB, RegistryNumber string) error {
	verNum := 1
	mapTenders := make(map[int]int)
	stmt, _ := db.Prepare(fmt.Sprintf("SELECT id_tender FROM %stender WHERE purchase_number = ? AND type_fz = ? ORDER BY UNIX_TIMESTAMP(date_version) ASC", Prefix))
	rows, err := stmt.Query(RegistryNumber, typeFz)
	stmt.Close()
	if err != nil {
		Logging("Ошибка выполения запроса", err)
		return err
	}
	for rows.Next() {
		var rNum int
		err = rows.Scan(&rNum)
		if err != nil {
			Logging("Ошибка чтения результата запроса", err)
			return err
		}
		mapTenders[verNum] = rNum
		verNum++
	}
	rows.Close()
	for vn, idt := range mapTenders {
		stmtr, _ := db.Prepare(fmt.Sprintf("UPDATE %stender SET num_version = ? WHERE id_tender = ?", Prefix))
		_, errr := stmtr.Exec(vn, idt)
		stmtr.Close()
		if errr != nil {
			Logging("Ошибка вставки NumVersion", errr)
			return err
		}
	}

	return nil
}

func getRegion(sp string) string {
	sp = strings.ToLower(sp)
	s := ""
	switch {
	case strings.Contains(sp, "белгор"):
		s = "белгор"
	case strings.Contains(sp, "брянск"):
		s = "брянск"
	case strings.Contains(sp, "владимир"):
		s = "владимир"
	case strings.Contains(sp, "воронеж"):
		s = "воронеж"
	case strings.Contains(sp, "иванов"):
		s = "иванов"
	case strings.Contains(sp, "калужск"):
		s = "калужск"
	case strings.Contains(sp, "костром"):
		s = "костром"
	case strings.Contains(sp, "курск"):
		s = "курск"
	case strings.Contains(sp, "липецк"):
		s = "липецк"
	case strings.Contains(sp, "москва"):
		s = "москва"
	case strings.Contains(sp, "московск"):
		s = "московск"
	case strings.Contains(sp, "орлов"):
		s = "орлов"
	case strings.Contains(sp, "рязан"):
		s = "рязан"
	case strings.Contains(sp, "смолен"):
		s = "смолен"
	case strings.Contains(sp, "тамбов"):
		s = "тамбов"
	case strings.Contains(sp, "твер"):
		s = "твер"
	case strings.Contains(sp, "тульс"):
		s = "тульс"
	case strings.Contains(sp, "яросл"):
		s = "яросл"
	case strings.Contains(sp, "архан"):
		s = "архан"
	case strings.Contains(sp, "вологод"):
		s = "вологод"
	case strings.Contains(sp, "калинин"):
		s = "калинин"
	case strings.Contains(sp, "карел"):
		s = "карел"
	case strings.Contains(sp, "коми"):
		s = "коми"
	case strings.Contains(sp, "ленинг"):
		s = "ленинг"
	case strings.Contains(sp, "мурм"):
		s = "мурм"
	case strings.Contains(sp, "ненец"):
		s = "ненец"
	case strings.Contains(sp, "новгор"):
		s = "новгор"
	case strings.Contains(sp, "псков"):
		s = "псков"
	case strings.Contains(sp, "санкт"):
		s = "санкт"
	case strings.Contains(sp, "адыг"):
		s = "адыг"
	case strings.Contains(sp, "астрахан"):
		s = "астрахан"
	case strings.Contains(sp, "волгог"):
		s = "волгог"
	case strings.Contains(sp, "калмык"):
		s = "калмык"
	case strings.Contains(sp, "краснод"):
		s = "краснод"
	case strings.Contains(sp, "ростов"):
		s = "ростов"
	case strings.Contains(sp, "дагест"):
		s = "дагест"
	case strings.Contains(sp, "ингуш"):
		s = "ингуш"
	case strings.Contains(sp, "кабардин"):
		s = "кабардин"
	case strings.Contains(sp, "карача"):
		s = "карача"
	case strings.Contains(sp, "осети"):
		s = "осети"
	case strings.Contains(sp, "ставроп"):
		s = "ставроп"
	case strings.Contains(sp, "чечен"):
		s = "чечен"
	case strings.Contains(sp, "башкор"):
		s = "башкор"
	case strings.Contains(sp, "киров"):
		s = "киров"
	case strings.Contains(sp, "марий"):
		s = "марий"
	case strings.Contains(sp, "мордов"):
		s = "мордов"
	case strings.Contains(sp, "нижерог"):
		s = "нижерог"
	case strings.Contains(sp, "оренбур"):
		s = "оренбур"
	case strings.Contains(sp, "пензен"):
		s = "пензен"
	case strings.Contains(sp, "пермс"):
		s = "пермс"
	case strings.Contains(sp, "самар"):
		s = "самар"
	case strings.Contains(sp, "сарат"):
		s = "сарат"
	case strings.Contains(sp, "татарс"):
		s = "татарс"
	case strings.Contains(sp, "удмурт"):
		s = "удмурт"
	case strings.Contains(sp, "ульян"):
		s = "ульян"
	case strings.Contains(sp, "чуваш"):
		s = "чуваш"
	case strings.Contains(sp, "курган"):
		s = "курган"
	case strings.Contains(sp, "свердлов"):
		s = "свердлов"
	case strings.Contains(sp, "тюмен"):
		s = "тюмен"
	case strings.Contains(sp, "ханты"):
		s = "ханты"
	case strings.Contains(sp, "челяб"):
		s = "челяб"
	case strings.Contains(sp, "ямало"):
		s = "ямало"
	case strings.Contains(sp, "алтайск"):
		s = "алтайск"
	case strings.Contains(sp, "алтай"):
		s = "алтай"
	case strings.Contains(sp, "бурят"):
		s = "бурят"
	case strings.Contains(sp, "забайк"):
		s = "забайк"
	case strings.Contains(sp, "иркут"):
		s = "иркут"
	case strings.Contains(sp, "кемеров"):
		s = "кемеров"
	case strings.Contains(sp, "краснояр"):
		s = "краснояр"
	case strings.Contains(sp, "новосиб"):
		s = "новосиб"
	case strings.Contains(sp, "томск"):
		s = "томск"
	case strings.Contains(sp, "омск"):
		s = "омск"
	case strings.Contains(sp, "тыва"):
		s = "тыва"
	case strings.Contains(sp, "хакас"):
		s = "хакас"
	case strings.Contains(sp, "амурск"):
		s = "амурск"
	case strings.Contains(sp, "еврей"):
		s = "еврей"
	case strings.Contains(sp, "камчат"):
		s = "камчат"
	case strings.Contains(sp, "магад"):
		s = "магад"
	case strings.Contains(sp, "примор"):
		s = "примор"
	case strings.Contains(sp, "сахалин"):
		s = "сахалин"
	case strings.Contains(sp, "якут"):
		s = "якут"
	case strings.Contains(sp, "саха"):
		s = "саха"
	case strings.Contains(sp, "хабар"):
		s = "хабар"
	case strings.Contains(sp, "чукот"):
		s = "чукот"
	case strings.Contains(sp, "крым"):
		s = "крым"
	case strings.Contains(sp, "севастоп"):
		s = "севастоп"
	case strings.Contains(sp, "байкон"):
		s = "байкон"
	default:
		s = ""
	}
	return s
}
