package middleware

import (
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"ss/models"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func enableCors(w *http.ResponseWriter) {
	(*w).Header().Set("Access-Control-Allow-Origin", "*")
	(*w).Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	(*w).Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
}

type product struct {
	refid, category, title, description string
	mrp                                 int
}

type auctionLogic struct {
	start, end                                time.Time
	duration, interval, itemsInSet, auctionId int
}

func insertIntoTable(db *sql.DB, item product, auction int, insertQuery string, start, update time.Time) {
	base := int(0.99 * float64(item.mrp))
	_, execErr := db.Exec(insertQuery, auction, start, update, 1, 1, item.refid, item.category, item.title, item.description, item.mrp, 1, base)
	if execErr != nil {
		log.Fatal("err in inserting into scheduled: ", execErr)
	}
}

func schedule(freqItems, nfItems []product, db *sql.DB, auctionId int) {
	var logic auctionLogic
	var update time.Time
	var totalAuctions int
	insertQuery := "insert into scheduled values($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)"
	fi := -1
	nfi := -1
	line := db.QueryRow("select * from logic where id=$1", auctionId)
	err := line.Scan(&logic.auctionId, &logic.itemsInSet, &logic.start, &logic.duration, &logic.interval, &logic.end)
	if err != nil {
		log.Fatal("err scanning logic ", err)
	}
	_, truncateErr := db.Exec("truncate table scheduled")
	if truncateErr != nil {
		log.Fatal("err truncating scheduled ", truncateErr)
	}
	if auctionId != 5 {
		totalAuctions = int(math.Floor(((logic.end.Sub(logic.start).Minutes() - float64(logic.duration)) / float64(logic.interval)) + 1))
	} else {
		totalAuctions = 1
		logic.duration = int(logic.end.Sub(logic.start).Minutes())
	}
	if totalAuctions == 0 { // for testing files
		totalAuctions = 1
	}
	numberOfFreqItems := calX(totalAuctions, len(freqItems), len(nfItems), logic.itemsInSet)
	fmt.Println("freq items:", len(freqItems), " non freq items:", len(nfItems))
	fmt.Println("number of frequent items in a set: ", numberOfFreqItems)
	for auction := 0; auction < totalAuctions; auction++ {
		update = logic.start.Add(time.Duration(logic.duration) * time.Minute)
		for i := 0; i < numberOfFreqItems; i++ {
			fi = (fi + 1) % len(freqItems)
			insertIntoTable(db, freqItems[fi], auction, insertQuery, logic.start, update)
		}
		for i := 0; i < (logic.itemsInSet - numberOfFreqItems); i++ {
			nfi = (nfi + 1) % len(nfItems)
			insertIntoTable(db, nfItems[nfi], auction, insertQuery, logic.start, update)
		}
		fmt.Println("start:", logic.start, " update:", update)
		logic.start = logic.start.Add(time.Duration(logic.interval) * time.Minute)
	}
}

func calX(totalAuctions, freqCount, nfCount, itemsInSet int) int { //function to return the number of frequently occuring items in a set of itemsInSet(currently 10) items
	if freqCount == 0 {
		return 0
	}
	x := 1
	var minFreq int
	var maxFreq int
	fmt.Println("total auctions:", totalAuctions)
	for x < itemsInSet {
		// fmt.Println("x:", x, "min:", minFreq, " max:", maxFreq)
		minFreq = int(math.Floor((float64(totalAuctions) * float64(x) / float64(freqCount))))         //min frequency of an item in frequent array
		maxFreq = int(math.Ceil((float64(totalAuctions) * float64(itemsInSet-x) / float64(nfCount)))) //max frequency of an item in non frequent array
		if minFreq > maxFreq {
			return x
		}
		x++
	}
	if x == itemsInSet { // for testing files
		return int(math.Ceil((float64(freqCount) * float64(itemsInSet))) / float64(nfCount))
	}
	return x
}

func getRowsInSlice(db *sql.DB, freq string) []product {
	products := []product{}
	var item product
	rows, err := db.Query("select refid,category,title,description,mrp from input where frequency=$1 order by random()", freq)
	if err != nil {
		log.Fatal("err getRowsInSlice ", err)
	}
	for rows.Next() {
		scanErr := rows.Scan(&item.refid, &item.category, &item.title, &item.description, &item.mrp)
		if scanErr != nil {
			log.Fatal("scanErr in getRowsInSlice", scanErr)
		}
		products = append(products, item)
	}
	return products
}

func createConnection() *sql.DB {
	// load .env file
	err := godotenv.Load(".env")

	if err != nil {
		log.Fatalf("Error loading .env file")
	}

	// Open the connection
	db, err := sql.Open("postgres", os.Getenv("POSTGRES_URL"))

	if err != nil {
		panic(err)
	}

	// check the connection
	err = db.Ping()

	if err != nil {
		panic(err)
	}

	fmt.Println("Successfully connected!")
	// return the connection
	return db
}

func DbUpdate(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)

	if (*r).Method == "OPTIONS" {
		return
	}

	var logic models.Logic

	// b, errte := io.ReadAll(r.Body)
	// // b, err := ioutil.ReadAll(resp.Body)  Go.1.15 and earlier
	// if errte != nil {
	// 	log.Fatalln(errte)
	// }

	// fmt.Println(string(b))

	// decode the json request to user
	err := json.NewDecoder(r.Body).Decode(&logic)

	if err != nil {
		// log.Fatalf("Unable to decode the request body.  %v", err)
		fmt.Printf("error in json.decoder %v \n", err)
	}

	db := createConnection()

	sqlStatement := `update LOGIC set count=$2,starttime=$3,duration=$4,interval=$5,endtime=$6 where id=$1`
	timeList := strings.Split(logic.STARTIME, " ")
	timeFormat := timeList[0] + "T" + timeList[1] + "+00:00"
	newStartTime, _ := time.Parse("2006-01-02T15:04:05Z07:00", timeFormat)

	timeList = strings.Split(logic.ENDTIME, " ")
	timeFormat = timeList[0] + "T" + timeList[1] + "+00:00"
	newEndTime, _ := time.Parse("2006-01-02T15:04:05Z07:00", timeFormat)

	_, err1 := db.Exec(sqlStatement, logic.ID, logic.COUNT, newStartTime, logic.DURATION, logic.INTERVAL, newEndTime)

	if err1 != nil {
		log.Fatalf("Unable to execute the query. %v", err1)
	}
	log.Println("Updated")

}

func Generate(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)

	now := time.Now()
	defer func() {
		fmt.Println("Time taken: ", time.Now().Sub(now).Seconds())
	}()

	params := mux.Vars(r)

	// convert the id in string to int
	auctionId, convErr := strconv.Atoi(params["id"])

	if convErr != nil {
		log.Fatal("Enter the number denoting the auction type as argument ", convErr)
	}
	envErr := godotenv.Load(".env")
	if envErr != nil {
		log.Fatal("env load err", envErr)
	}
	db, dbErr := sql.Open("postgres", os.Getenv("POSTGRES_URL"))
	if dbErr != nil {
		log.Fatal("err connecting db", dbErr)
	}
	fmt.Println("Database Bzinga opened and ready")
	schedule(getRowsInSlice(db, "Show More Frequently"), getRowsInSlice(db, ""), db, auctionId)

}

func Input(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)

	err := r.ParseMultipartForm(32 << 20) // maxMemory 32MB
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	file, h, err := r.FormFile("photo")
	if err != nil {
		fmt.Println("err in Formfile: ", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	tmpfile, err := os.Create(h.Filename)
	defer tmpfile.Close()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	_, err = io.Copy(tmpfile, file)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	csvFile, err := os.Open(h.Filename)
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println("Successfully Opened CSV file")
	defer csvFile.Close()

	csvLines, err := csv.NewReader(csvFile).ReadAll()
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println(csvLines[1][0])

	err1 := godotenv.Load(".env")

	if err1 != nil {
		log.Fatalf("Error loading .env file")
	}

	db, err2 := sql.Open("postgres", os.Getenv("POSTGRES_URL"))
	if err2 != nil {
		log.Fatalf("error connecting to the database: ", err2)
	}
	fmt.Println("Database bzinga opened and ready.")
	defer db.Close()
	_, err4 := db.Exec("truncate table input")
	if err4 != nil {
		log.Fatal("connection problem ", err4)
	}
	sqlStatement := "insert into input values($1,$2,$3,$4,$5,$6)"

	for i := 1; i < len(csvLines); i++ {
		id := csvLines[i][0]
		cate := csvLines[i][1]
		prod := csvLines[i][2]
		desc := csvLines[i][3]
		mr := csvLines[i][4]
		freq := csvLines[i][5]
		mr = strings.ReplaceAll(mr, ".00", "")
		mr = strings.ReplaceAll(mr, ",", "")
		mr = strings.Trim(mr, " ")
		mrp, er := strconv.Atoi(mr)
		if er != nil {
			log.Fatal(er)
		}
		_, err3 := db.Exec(sqlStatement, id, cate, prod, desc, mrp, freq)
		if err3 != nil {
			log.Fatal(err3, ": error in db.exec of sql statement")
		}
	}
	fmt.Println("Finished getting input")

	w.WriteHeader(200)
	return

}

func Receive(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)

	var set int
	start := time.Now()
	endtime := time.Now()
	var startbidprice int
	var initialbidprice int
	var ref string
	var category string
	var prod_name string
	var desc string
	var mrp int
	var base int
	var statusCheck int

	params := mux.Vars(r)
	outFile := params["name"]
	header := []string{
		"Batch Number", "Start Listing Date and Time (IST)", "Stop Listing at Date and Time (IST)", "Start bid price (Rs.)", "Initial bid cost (Tickets)", "Initial status check cost (Tickets)", "Reference ID", "Category", "Product Title", "Description", "MSRP", "Base Price",
	}
	csvFile, csverr := os.Create(outFile)
	if csverr != nil {
		log.Fatalf("failed creating file: %s", csverr)
	}
	csvwriter := csv.NewWriter(csvFile)
	_ = csvwriter.Write(header)

	err := godotenv.Load(".env")
	if err != nil {
		log.Fatal("err: ", err)
	}
	db, dberr := sql.Open("postgres", os.Getenv("POSTGRES_URL"))
	if dberr != nil {
		log.Fatal("dberr: ", dberr)
	}
	fmt.Println("opened the db connection")
	rows, rowserr := db.Query("select * from scheduled")
	if rowserr != nil {
		log.Fatal("rowserr: ", rowserr)
	}
	fmt.Println("Rows are read")
	for rows.Next() {
		err := rows.Scan(&set, &start, &endtime, &startbidprice, &initialbidprice, &ref, &category, &prod_name, &desc, &mrp, &statusCheck, &base)
		if err != nil {
			log.Fatal("looping: ", err)
		}
		startTime := strings.Replace(start.String(), " +0000 +0000", "", 1)
		endTime := strings.Replace(endtime.String(), " +0000 +0000", "", 1)
		row := []string{
			strconv.Itoa(set),
			startTime,
			endTime,
			strconv.Itoa(startbidprice),
			strconv.Itoa(initialbidprice),
			strconv.Itoa(statusCheck),
			ref,
			category,
			prod_name,
			desc,
			strconv.Itoa(mrp),
			strconv.Itoa(base),
		}
		_ = csvwriter.Write(row)

	}
	csvwriter.Flush()
	csvFile.Close()

	// fileBytes, err := ioutil.ReadFile(outFile)
	// if err != nil {
	// 	panic(err)
	// }
	// w.WriteHeader(http.StatusOK)
	// w.Header().Set("Content-Type", "text/csv")
	// w.Header().Add("Content-Disposition", "attachment;filename=tests.csv")
	// w.Write(fileBytes)

	http.ServeFile(w, r, outFile)

}
