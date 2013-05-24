package main

// Notice the changing of some imports for brevity

import ( 	"fmt"
			"net/http"
			"html/template"
			str "strings"
			conv "strconv"
			"github.com/hoisie/redis"
			"log"
			"code.google.com/p/gosqlite/sqlite"
			"time"
			"os/exec"
			"os"
			b "./bet"
			u "./utils"
		)

type Stat struct {
	Slider int
	Limit int
	Rows []template.HTML
} 

// Way of handling a fatal error. Throws a panic upstream and a defer function will catch it!

func handleError(err error) {
	if err != nil {
		log.Printf("%v", err)
		panic(err)
	}
}

func getResults(bet *b.Bet) {
	var client redis.Client
	var value []byte
	
	value, _ = client.Get("results")
	bet.Results = string(value)
	value, _ = client.Get("resultsstars")
	bet.ResultsStars = string(value)
	value, _ = client.Get("resultsdate")
	bet.ResultsDate = string(value)
} 

func locale(r *http.Request, file string) (string) {
	var page string
	l := r.Header.Get("Accept-Language")
	
	if str.Contains(l, "pt") {
		page = str.Replace(file,  ".",  "_pt.", 1)
	} else {
		page = file
	}
	return page
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	c := make(chan string)
	bet := b.Bet {nil, nil, nil, 0, "", 0, 0, 0, 0, 0, "", "", "", "", ""}
		
	go hitCounter(false, c)
	getResults(&bet)
	bet.Counter = <- c 
	err := templates.ExecuteTemplate(w, locale(r, "index.html"), &bet)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func staticHandler(w http.ResponseWriter, r *http.Request) {
	wd, err := os.Getwd()
	if err == nil {
		http.ServeFile(w, r, wd + r.RequestURI)		
	}
}

func setVal(s string, min, max int) (int) {
	p, err := conv.Atoi(s)
	if err != nil ||  p < min || p > max {
		p = -1
	}
	return p;
}

// Hit counter from the Redis keyvalue store and report to channel

func hitCounter(update bool, c chan string) {
	var client redis.Client
	var key = "hitcounter" 
	var counter string
	
	if update {
		client.Incr(key)
	}
	val, _ := client.Get(key)
	counter = string(val)
	c <- counter
}

// Main logic

func betHandler(w http.ResponseWriter, r *http.Request) {
	t0 := time.Now()
	bet := b.Bet {nil, nil, nil, 0, "", 0, 0, 0, 0, 0, "", "", "", "", ""}
	even := setVal(r.FormValue("even"), 0, 5)
	prime := setVal(r.FormValue("prime"), 0, 5)
	fibo := setVal(r.FormValue("fibonacci"), 0, 5)
	high := setVal(r.FormValue("high"), 0, 5)
	sum := setVal(r.FormValue("sum"), 50, 240)
	c := make(chan string)
	
	// Execute this before exiting, in order to handle any misshap
	defer func () {
			if x := recover(); x != nil {
				var s string
				
				s = fmt.Sprintf("%v", x)
				http.Error(w, s, http.StatusInternalServerError)
		}
	} ()
	
	// Next we fire up a thread to handle the hit counter while we crunch numbers
	go hitCounter(true, c)
	getResults(&bet)
	
	if !b.Build(&bet, even, prime, fibo, high, sum) {
		bet.Message = "timeout!"
	}
	
	// We should have the hitcounter by now, in a non-blocking way
	
	bet.Counter, _ = <- c 
	t1 := time.Now()
	bet.Duration = fmt.Sprintf("%v", t1.Sub(t0))
	err := templates.ExecuteTemplate(w, locale(r, "index.html"), &bet)
	if err != nil {
		panic(err)
	}
}

// Fetch and update the results database
func fetchHandler(w http.ResponseWriter, r *http.Request) {
	var numbers [7]int
	var client redis.Client
	
	con, err := sqlite.Open("results")
	
	// Execute this before exiting, in order to handle any misshap
	defer func () {
			con.Close()
			if x := recover(); x != nil {
				var s string
				
				s = fmt.Sprintf("%v", x)
				http.Error(w, s, http.StatusInternalServerError)
			}
			checkBets(numbers)
	} ()
	
	if err != nil {
		panic(err)
	}
	
	output, err := exec.Command("./fetchandstrip.sh").Output()
	if err != nil {
		panic(err)
	}
	
	s := string(output)
	
	n, err := fmt.Sscanf(s, "%d\n%d\n%d\n%d\n%d\n%d\n%d", 
						&numbers[0],
						&numbers[1],
						&numbers[2],
						&numbers[3],
						&numbers[4],
						&numbers[5],
						&numbers[6])	
	if n != 7 || err != nil {
		panic(err)
	}
	
	results := fmt.Sprintf("%d, %d, %d, %d, %d", 
		numbers[0],
		numbers[1],
		numbers[2],
		numbers[3],
		numbers[4])
	
	stars := fmt.Sprintf("%d, %d", numbers[5], numbers[6])
	now := time.Now()
	date := fmt.Sprintf("%d-%d-%d", now.Day(), now.Month(), now.Year())
	
	client.Set("results", []byte(results))
	client.Set("resultsstars", []byte(stars))
	client.Set("resultsdate", []byte(date))	
	
	// Warning - Ugly code ahead.
	s = fmt.Sprintf("insert into results (n1, n2, n3, n4, n5, s1, s2) values (%d, %d, %d, %d, %d, %d, %d)", numbers[0], numbers[1], numbers[2], numbers[3], numbers[4], numbers[5], numbers[6])
	
    err = con.Exec(s)
    if err!=nil {
		panic(err)
	}
}

func testHandler(w http.ResponseWriter, r *http.Request) {
	checkBets([7]int {7, 8, 19, 28, 29, 5, 9})
}

func checkBets(draw [7]int) {
	var id int
	var date string
	var checked int
	var lang string
	var numbers = make([]int, 5)
	var stars = make([]int, 2)
	var mail string
	
	con, err := sqlite.Open("results")
	if err != nil {
		panic(err)
	}
	defer con.Close()
	
	sel, err := con.Prepare("select * from bets where checked=0 order by date desc")
	if err != nil {
		panic(err)
	}
	err = sel.Exec()
	if err != nil {
		panic(err)
	}
	
	for sel.Next() {
		err = sel.Scan(&id, &date, &numbers[0], &numbers[1], &numbers[2], &numbers[3], &numbers[4], &stars[0], &stars[1], &mail, &checked, &lang)
			
		if err != nil {
			panic(err)
		}
		
		mh := b.Overlap(numbers, draw[0:5])
		sh := b.Overlap(stars, draw[6:])
		u.Notify(mail, mh, sh, lang)
	}
	
	err = con.Exec("update bets set checked=1 where checked=0")
	if err != nil {
		panic(err)
	}
}

func statsHandler(w http.ResponseWriter, r *http.Request) {
	var id int
	var date string
	var numbers = make([]int, 5)
	var stars = make([]int, 2)
	var stat = Stat {2, 5, nil}
	var row template.HTML
	var rows, sumE, sumP, sumF, sumS, sumH int
	
	con, err := sqlite.Open("results")
	
	defer func () {
			con.Close()
			if x := recover(); x != nil {
				var s string
				
				s = fmt.Sprintf("%v", x)
				http.Error(w, s, http.StatusInternalServerError)
			}
	} ()
	
	if err != nil {
		panic(err)
	}
	
	v := setVal(r.FormValue("slidervalue"), 0, 5)
	if v == -1 {
		v = 2
	}
	stat.Slider = v
	v--
	if v == 0 {
		stat.Limit = 1
	} else {
		stat.Limit = 5 * v;
	}
	s := fmt.Sprintf("select * from results order by date desc limit %d", stat.Limit)	
	sel, err := con.Prepare(s)
	
	if err != nil {
		panic(err)
	}
	err = sel.Exec()
	if err != nil {
		panic(err)
	}
	
	for (sel.Next()) {
		err = sel.Scan(&id, &date, &numbers[0], &numbers[1], &numbers[2], &numbers[3], &numbers[4], &stars[0], &stars[1])
			
		if err != nil {
			panic(err)
		}
		
		t, _ := time.Parse("2006-01-02 15:04:05", date)
		
		e, p, f, high, sum := b.Evaluate(numbers)
		
		sumE += e
		sumP += p
		sumF += f
		sumS += sum
		sumH += high
		
		s = fmt.Sprintf("<tr><td>%d-%d-%d</td><td>%d, %d, %d, %d, %d</td><td>%d, %d</td><td>%d</td><td>%d</td><td>%d</td><td>%d</td><td>%d</td><tr>", 
			t.Day(), t.Month(), t.Year(),
			numbers[0],
			numbers[1],
			numbers[2],
			numbers[3],
			numbers[4],
			stars[0], 
			stars[1],
			e,
			p,
			f,
			sum,
			high)
		
		row = template.HTML(s)
		stat.Rows = append(stat.Rows, row)
		rows++
	}
	
	row = template.HTML(
		fmt.Sprintf("<tr><td></td><td></td><td><></td><td>%2.1f</td><td>%2.1f</td><td>%2.1f</td><td>%3.1f</td><td>%2.1f</td><tr>", 
			float32(sumE) / float32(rows),
			float32(sumP) / float32(rows),
			float32(sumF) / float32(rows),
			float32(sumS) / float32(rows),
			float32(sumH) / float32(rows)))
	stat.Rows = append(stat.Rows, row)
	
	err = templates.ExecuteTemplate(w, locale(r, "stats.html"), &stat)
	if err != nil {
		panic(err)
	}
}

func saveHandler(w http.ResponseWriter, r *http.Request) {
	var numbers = make([]int, 5)
	var stars = make([]int, 2)
	var s string
	var err error
	var lang string
	
	mail := r.FormValue("mail")
	for i := 0; i < 5; i++ {
		s = fmt.Sprintf("n%d", i)
		numbers[i], err = conv.Atoi(r.FormValue(s))
		if i < 2 {
			s = fmt.Sprintf("s%d", i)
			stars[i], err = conv.Atoi(r.FormValue(s))
		}
	}
	
	con, err := sqlite.Open("results")
	
	defer func () {
			con.Close()
			if x := recover(); x != nil {
				var s string
				
				s = fmt.Sprintf("%v", x)
				http.Error(w, s, http.StatusInternalServerError)
			}
	} ()
	
	if err != nil {
		panic(err)
	}

	l := r.Header.Get("Accept-Language")	
	if str.Contains(l, "pt") {
		lang = "pt"
	} else {
		lang = "*"
	}
	
	s = fmt.Sprintf("insert into bets (n1, n2, n3, n4, n5, s1, s2, mail, checked, lang) values (%d, %d, %d, %d, %d, %d, %d, \"%s\", 0, \"%s\")", numbers[0], numbers[1], numbers[2], numbers[3], numbers[4], stars[0], stars[1], mail, lang)
	
    err = con.Exec(s)
    if err!=nil {
		panic(err)
	}
	
	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}

func readResults(bet *b.Bet) {
	var id int
	var date string
	var numbers = make([]int, 5)
	var stars = make([]int, 2)
	
	con, err := sqlite.Open("results")
	if err != nil {
		panic(err)
	}
	
	defer con.Close()
	
	sel, err := con.Prepare("select * from results order by date desc limit 1")
	
	if err != nil {
		panic(err)
	}
	err = sel.Exec()
	if err != nil {
		panic(err)
	}
	
	if sel.Next() {
		err = sel.Scan(&id, &date, &numbers[0], &numbers[1], &numbers[2], &numbers[3], &numbers[4], &stars[0], &stars[1])
			
		if err != nil {
			panic(err)
		}
		
		t, _ := time.Parse("2006-01-02 15:04:05", date)
		
		bet.ResultsDate = fmt.Sprintf("%d-%d-%d", t.Day(), t.Month(), t.Year())
		bet.Results = fmt.Sprintf("%d, %d, %d, %d, %d", 
			numbers[0],
			numbers[1],
			numbers[2],
			numbers[3],
			numbers[4])
		bet.ResultsStars = fmt.Sprintf("%d, %d", stars[0], stars[1])
	}
}

func main() {
	http.HandleFunc("/", rootHandler)
	http.HandleFunc("/bet", betHandler)
	http.HandleFunc("/fetchresults", fetchHandler)
	http.HandleFunc("/stats", statsHandler)
	http.HandleFunc("/save", saveHandler)
	http.HandleFunc("/test", testHandler)
	// Only useful in dev
	http.HandleFunc("/static/", staticHandler)
	http.ListenAndServe(":8080", nil)
}

// Global template caching
var templates = template.Must(template.ParseFiles("index.html", "index_pt.html", "stats.html", "stats_pt.html")) 