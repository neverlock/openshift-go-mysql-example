package main

import (
  "os"
  "log"
  "fmt"
  "runtime"
  "strings"
  "net/http"
  "database/sql"
  "encoding/json"
  "github.com/gorilla/mux"
  "github.com/PuerkitoBio/goquery"
  _ "github.com/ziutek/mymysql/godrv"
)

var change,changeP string

var db *sql.DB

func connect() {
        database:="set"
        user:=os.Getenv("OPENSHIFT_MYSQL_DB_USERNAME")
        password:=os.Getenv("OPENSHIFT_MYSQL_DB_PASSWORD")
        var err error
        db, err = sql.Open("mymysql", "tcp:"+os.Getenv("OPENSHIFT_MYSQL_DB_HOST")+":"+os.Getenv("OPENSHIFT_MYSQL_DB_PORT")+"*"+database+"/"+user+"/"+password)
        if err != nil {
                log.Fatal(err)
        }
}

func selectStock(stock string) int{
var id int
err := db.QueryRow("select id from lastupdate where name=?",stock).Scan(&id)
switch {
case err == sql.ErrNoRows:
        return 0
case err != nil:
    log.Fatal(err)
default:
        return id
}
return id
}

func insertStock(name string,price string,change string,changepercen string,update string){
_, err := db.Exec("insert into lastupdate (name,price,changeprice,changepercen,lastupdate) values (?,?,?,?,?)", name,price,change,changepercen,update)
if err != nil {
                log.Fatal(err)
        }
}

func updateStock(name string,price string,change string,changepercen string,update string){
_, err := db.Exec("update lastupdate set price=?,changeprice=?,changepercen=?,lastupdate=? where name=?",price,change,changepercen,update,name)
if err != nil {
                log.Fatal(err)
        }
}

func getCurrentPrice(ID string) []string{
	doc, err := goquery.NewDocument("http://marketdata.set.or.th/mkt/stockquotation.do?symbol=" + ID + "&language=en&country=US")
	if err != nil {
		log.Fatal(err)
	}
	price := strings.TrimSpace(doc.Find("td font strong").Text())
	update := strings.TrimSpace(doc.Find("td table[width|='200'] tr td[align|='right']").Text())

	var ch string
	doc.Find("tr td td font").Each(func(i int,s *goquery.Selection) {
	ch = strings.TrimSpace(s.Find("font").Text())
	if (i==1) {
		change = ch
	}
	if (i==3) {
		changeP= ch
	}
	})


        var time,status []string
        var PRICE,TIME,STATUS string
        if (price != ""){
                var update1 string
                var update2 []string
                update1 = strings.Replace(update,"              ","",-1)
                update2 = strings.Split(update1,"\n")
                time = strings.Split(update2[0],"Last Update ")
                status = strings.Split(update2[1],"Market Status : ")
                PRICE=price
                TIME=time[1]
                STATUS=status[1]
	}

	//data := []string{price,update}
	data := []string{PRICE,TIME,STATUS}
	return data
}

func main() {
	nCPU := runtime.NumCPU()
        runtime.GOMAXPROCS(nCPU)
        log.Println("Number of CPUs: ", nCPU)
if (len(os.Args) == 1) {
        rtr := mux.NewRouter()
        rtr.HandleFunc("/price/{id}", getPrice).Methods("GET")
        http.Handle("/", rtr)
        //port := ":8000"
	bind := fmt.Sprintf("%s:%s", os.Getenv("HOST"), os.Getenv("PORT"))
        log.Println("Listening:" + bind + "...")

        http.ListenAndServe(bind, nil)
	}

if (len(os.Args) == 2 && os.Args[1] == "--getdata") {
	log.Println("get data")
	connect()
	data := "intuch"
	price := getCurrentPrice(data)
	log.Println("Price : ",price[0])
	log.Println("Time : ",price[1])
	log.Println("Market Status : ",price[2])
	log.Println("Price change : ",change)
	log.Println("Price change % : ",changeP)
		PRICE:=price[0]
		TIME:=price[1]
		STATUS:=price[2]

                stockID := selectStock(data) // if stockID=0 mean don't have Stock name in DB must insert new Stock into DB
                if (stockID == 0) {
                        insertStock(data,PRICE,change,changeP,TIME)
                        log.Printf("Insert::[%s] Price[%s] Change[%s] ChangeP[%s] Update [%s] ServerStatus [%s] stockID=%d\n", data,PRICE,change,changeP,TIME,STATUS,stockID)
                } else { // if stockID != 0 mean have stock in DB must update Data
                        updateStock(data,PRICE,change,changeP,TIME)
                        log.Printf("Update::[%s] Price[%s] Change[%s] ChangeP[%s] Update [%s] ServerStatus [%s] stockID=%d\n", data,PRICE,change,changeP,TIME,STATUS,stockID)
                }
	}
	db.Close()

}

func getPrice(w http.ResponseWriter, r *http.Request){
	params := mux.Vars(r)
	ID := params["id"]

	var price []string
	price = getCurrentPrice(ID)

	mapD := map[string]string{"ID":ID,"Price":price[0],"LastUpdate":price[1],"MarketStatus":price[2],"Change":change,"ChangePercen":changeP}
	js,_ := json.Marshal(mapD)
	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
	log.Println(price)
	change = ""
	changeP = ""

}

