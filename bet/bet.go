// This package handles the bet generation logic, from random numbers to rule checking

package bet;

import ( 	"net/http"
			"io/ioutil"
			"html/template"
			str "strings"
			"encoding/hex"
			"sort"
			"log"
			conv "strconv"
			"github.com/hoisie/redis"
		)

type Bet struct {
	Numbers []byte
	Bet []int
	Stars []int
	Index int
	Message template.HTML
	Evens int
	Primes int
	Fibo int
	Sum int
	High int
	Counter string
	Results string
	ResultsStars string
	ResultsDate string
	Duration string  
} 

func handleError(err error) {
	if err != nil {
		log.Printf("%v", err)
		panic(err)
	}
}

// Simple web page fetch and content extraction. No Parsing since only ONE <td> is used.
// It can/WILL rotten with time but keeps things simple for now.
// At the and, numbers will contain a 1024 array of quantum random bytes 
// Cached by Redis

func load(bet *Bet) {
	var client redis.Client
	
	resp, err := http.Get("http://150.203.48.55/RawHex.php"); 
	handleError(err)
	
	defer resp.Body.Close()
	
	body, err := ioutil.ReadAll(resp.Body);
	handleError(err)
		
	bet.Index = 0;
	
	s  	:= string(body);
	x1 	:= str.Index(s, "<td>")
	x2 	:= str.Index(s, "</td>") 
	
	bet.Numbers, 
		err = hex.DecodeString(s[x1 + 5:x2])
	handleError(err)
	
	// This works in a non atomic way. For this application it's ok! Each client is "seeing" a diferent set of numbers
	client.Set("numbers", bet.Numbers)
	client.Set("index", []byte("0"))
}

// Iterate the numbers array and build the bets and stars array without duplicates.

func Build(bet *Bet, evenLimit int, primeLimit int, fiboLimit, highLimit, sumLimit int) (bool) {
	var n, s, tries int
	var client redis.Client
	var index []byte
	var ok bool
	
	options := (evenLimit + primeLimit + fiboLimit + highLimit + sumLimit) != -5

	index, err := client.Get("index")
	if err != nil {
		load(bet)
	} else {
		bet.Numbers, _ = client.Get("numbers")
		bet.Index, _ = conv.Atoi(string(index))
	}
again:
	n = 0
	s = 0
	bet.Bet = []int{}
	bet.Stars = []int{}
	
	for (s < 2 || n < 5) && bet.Index < 1024 {
		switch {
		case n < 5:
			j := int(bet.Numbers[bet.Index] % 50)
			if j == 0 {
				j++
			}
			if !exists(bet.Bet, j) {
				bet.Bet = append(bet.Bet, j)
				n++
			}
		case s < 2:	
			j := int(bet.Numbers[bet.Index] % 11)
			if j == 0 {
				j++
			}
			if !exists(bet.Stars, j) {
				bet.Stars = append(bet.Stars, j)
				s++
			}
		}
		bet.Index++
	}
	
	client.Set("index", []byte(conv.Itoa(bet.Index)))
	//log.Printf("%d", index)
	
	if len(bet.Bet) == 5 && len(bet.Stars) == 2 {
		sort.Ints(bet.Bet)
		sort.Ints(bet.Stars)
	
		ok, bet.Evens, bet.Primes, bet.Fibo, bet.High, bet.Sum = Validate(bet.Bet, evenLimit, primeLimit, fiboLimit, highLimit, sumLimit) 
	
		if bet.Index < 1024 && tries < 3 {
			if options {			
				if ok == true {
					return true
				} else {
					goto again
				}
			} else if !options {
				return true
			} else {
				goto again
			}
		}
	} else {
		bet.Index = 1024
	}
	
	if bet.Index == 1024 && tries < 3 {
		load(bet)
		tries++;
		goto again
	}
	
	return false
}

// Simple aid function to check the existance of a value inside an array.

func exists(a []int, b int) (bool) {
	for _, v := range a {
		if v == b {
			return true
		}
	} 
	return false
}

// Simple aid function to intercect two arrays - returns the number of equal elements 

func Overlap(a []int, b []int) (int) {
	r := 0
	for _, v := range b {
		if exists(a, v) {
			r++
		}
	} 
	return r
}

// Validates the bets array with a fixed set of rules
// Clearly a bit more work since we don't enjoy list comprehensions as in other languages. 

func Evaluate(bet []int) (e, p, f, h, s int) {
	evens := []int {} 
	primes := []int {2, 3, 5, 7, 11, 13, 17, 19, 23, 29, 31, 37, 41, 43, 47}
	fibo := []int {1, 2, 3, 5, 8, 13, 21, 34}
	
	for i := 2; i < 50; i += 2 {
		evens = append(evens, i)
	}
	
	e = Overlap(bet, evens)
	p = Overlap(bet, primes)
	f = Overlap(bet, fibo)
	h, _ = Highlow(bet)
	s = Sum(bet)
	
	return
}

// Returns the sum of the first 5 numbers
func Sum(bet []int) (int) {
	var s int
	
	for i := 0; i < 5; i++ {
		s += bet[i];
	}
	return s
}


// Returns the number of high/low occurences
func Highlow(bet []int) (h, l int) {
	for i := 0; i < 5; i ++ {
		if bet[i] > 25 {
			h++;
		} else {
			l++;
		}
	}
	return
}

// I think the number of named return values is obcene but code is like mushrooms.
func Validate(bet []int, evenLimit int, primeLimit int, fiboLimit int, highLimit int, sumLimit int) (ok bool, e, p, f, h, s int) {
	e, p, f, h, s = Evaluate(bet)

	ok = true	
	if evenLimit >= 0 && e != evenLimit {
		ok = false
	}
	
	if primeLimit >= 0 && p != primeLimit {
		ok = false
	}
	
	if fiboLimit >= 0 && f != fiboLimit {
		ok = false
	}
	
	if highLimit >= 0 && h != highLimit {
		ok = false
	}
	
	if sumLimit >= 0 && (s < (sumLimit - 30) || s > (sumLimit + 30)) {
		ok = false
	}
	
	return
}