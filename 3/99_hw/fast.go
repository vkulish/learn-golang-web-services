package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"
	"sync"
)

type User struct {
	IsAndroid bool
	IsMSIE    bool
	Name      string
	Email     string
}

var dataPool = sync.Pool{
	New: func() interface{} {
		return bytes.NewBuffer(make([]byte, 0, 256))
	},
}

var androidBrowser = []byte("Android")
var msieBrowser = []byte("MSIE")

func scanBrowsers(browser []byte, seenBrowsers *[]string, uniqueBrowsers *int, user *User) {
	//fmt.Printf("Checking browser: %s\n", browser)

	isAndroid := bytes.Contains(browser, androidBrowser)
	isMSIE := bytes.Contains(browser, msieBrowser)
	if isAndroid || isMSIE {
		//fmt.Printf("Browser found: android=%v, MSIE=%v\n", isAndroid, isMSIE)
		user.IsAndroid = user.IsAndroid || isAndroid
		user.IsMSIE = user.IsMSIE || isMSIE

		browserStr := string(browser)
		if !slices.Contains(*seenBrowsers, browserStr) {
			//fmt.Printf("FAST New browser: %s\n", browser)
			*seenBrowsers = append(*seenBrowsers, browserStr)
			*uniqueBrowsers++
		}
	}
}

func scanUser(userStr *[]byte, seenBrowsers *[]string, uniqueBrowsers *int, user *User) error {
	user.IsAndroid = false
	user.IsMSIE = false

	var token *bytes.Buffer
	var tokenProcessing bool
	var arrayProcessing int
	var browserProcessing, nameProcessing, emailProcessing int

	for _, c := range *userStr {

		switch c {
		case '"':
			if !tokenProcessing {
				tokenProcessing = true
				token = dataPool.Get().(*bytes.Buffer)
			} else {
				//fmt.Printf("token: %s\n", token)
				if nameProcessing == 1 {
					result := token.String()
					nameProcessing++
					user.Name = result
					nameProcessing++
					//fmt.Printf("--> User name: %v\n", user.Name)
				} else if emailProcessing == 1 {
					result := token.String()
					emailProcessing++
					user.Email = strings.Replace(result, "@", " [at] ", 1)
					emailProcessing++
					//fmt.Printf("--> User email: %v\n", user.Email)
				} else if browserProcessing == 1 && arrayProcessing > 0 {
					scanBrowsers(token.Bytes(), seenBrowsers, uniqueBrowsers, user)
				} else {
					result := token.String()
					switch result {
					case "name":
						nameProcessing++
					case "email":
						emailProcessing++
					case "browsers":
						browserProcessing++
					}
				}

				tokenProcessing = false
				token.Reset()
				dataPool.Put(token)
			}
		case '[':
			if !tokenProcessing {
				arrayProcessing++
			}
		case ']':
			if !tokenProcessing {
				arrayProcessing = 0
			}
		default:
			if tokenProcessing {
				token.WriteByte(c)
			}
		}
	}

	if len(user.Name) == 0 || len(user.Email) == 0 {
		return fmt.Errorf("no data found")
	}

	return nil
}

// вам надо написать более быструю оптимальную этой функции
func FastSearch(out io.Writer) {
	/*
		!!! !!! !!!
		обратите внимание - в задании обязательно нужен отчет
		делать его лучше в самом начале, когда вы видите уже узкие места, но еще не оптимизировалм их
		так же обратите внимание на команду в параметром -http
		перечитайте еще раз задание
		!!! !!! !!!
	*/

	file, err := os.Open(filePath)
	if err != nil {
		panic(err)
	}

	var uniqueBrowsers int
	var user User
	seenBrowsers := make([]string, 0, 200)
	foundUsers := make([]string, 0, 200)

	scanner := bufio.NewScanner(file)

	i := -1
	for scanner.Scan() {
		line := scanner.Bytes()
		i++

		err := scanUser(&line, &seenBrowsers, &uniqueBrowsers, &user)
		if err != nil {
			//fmt.Printf("error: %s, user: %v\n", err, user)
			continue
		}

		if !(user.IsAndroid && user.IsMSIE) {
			continue
		}
		//fmt.Printf("FAST total unique browsers: %d, user: %s\n", uniqueBrowsers, user.Name)

		foundUsers = append(foundUsers, fmt.Sprintf("[%d] %s <%s>\n", i, user.Name, user.Email))
	}

	fmt.Fprintln(out, "found users:\n"+strings.Join(foundUsers, ""))
	fmt.Fprintln(out, "Total unique browsers", len(seenBrowsers))
}
