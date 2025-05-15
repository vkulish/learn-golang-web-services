package main

import (
	"io"
	"fmt"
	"os"
	"strings"
	"bufio"
	"slices"
)

type User struct {
	IsAndroid bool
	IsMSIE    bool
	Name      string
	Email     string
}

func scanBrowsers(browser *string, seenBrowsers *[]string, uniqueBrowsers *int, user *User) {
	//fmt.Printf("Checking browser: %s\n", browser)
	
	isAndroid := strings.Contains(*browser, "Android")
	isMSIE := strings.Contains(*browser, "MSIE")
	if isAndroid || isMSIE {
		//fmt.Printf("Browser found: android=%v, MSIE=%v\n", isAndroid, isMSIE)
		user.IsAndroid = user.IsAndroid || isAndroid
		user.IsMSIE = user.IsMSIE || isMSIE

		if !slices.Contains(*seenBrowsers, *browser) {
			//fmt.Printf("FAST New browser: %s\n", browser)
			*seenBrowsers = append(*seenBrowsers, *browser)
			*uniqueBrowsers++
		}
	}
}

func scanUser(userStr *[]byte, seenBrowsers *[]string, uniqueBrowsers *int, user *User) error {
	user.IsAndroid = false
	user.IsMSIE = false

	var token []byte
	var tokenProcessing int 
	var arrayProcessing int
	var browserProcessing, nameProcessing, emailProcessing int

	for _, c := range *userStr {

		switch c {			
			case '"':
				switch tokenProcessing {
				case 0:
					tokenProcessing++
					token = []byte{}
				case 1:
					//fmt.Printf("token: %s\n", token)
					if nameProcessing == 1 {
						result := string(token)
						nameProcessing++
						user.Name = result
						nameProcessing++
						//fmt.Printf("--> User name: %v\n", user.Name)
					} else if emailProcessing == 1 {
						result := string(token)
						emailProcessing++
						user.Email = strings.Replace(result, "@", " [at] ", 1)
						emailProcessing++
						//fmt.Printf("--> User email: %v\n", user.Email)
					} else if browserProcessing == 1 && arrayProcessing > 0 {
						result := string(token)
						scanBrowsers(&result, seenBrowsers, uniqueBrowsers, user)
					} else {
						result := string(token)
						switch result {
						case "name":
							nameProcessing++
						case "email":
							emailProcessing++
						case "browsers":
							browserProcessing++
						}
					}

					tokenProcessing = 0
				}
			case '[':
				if tokenProcessing == 0 {
					arrayProcessing++
				}
			case ']':
				if tokenProcessing == 0 {
					arrayProcessing = 0
				}
			default:
				if tokenProcessing == 1 {
					token = append(token, c)
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

	uniqueBrowsers := 0
	seenBrowsers := make([]string, 0, 200)
	foundUsers := make([]string, 0, 200)
	user := User{}

	scanner := bufio.NewScanner(file)

	i := -1
SCAN_USERS:
	for scanner.Scan() {
		line := scanner.Bytes()
		i++

		err := scanUser(&line, &seenBrowsers, &uniqueBrowsers, &user)
		if err != nil {
			//fmt.Printf("error: %s, user: %v\n", err, user)
			continue SCAN_USERS
		}

		if !(user.IsAndroid && user.IsMSIE) {
			continue SCAN_USERS
		}
		//fmt.Printf("FAST total unique browsers: %d, user: %s\n", uniqueBrowsers, user.Name)

		email := strings.Replace(user.Email, "@", " [at] ", 1)
		foundUsers = append(foundUsers,fmt.Sprintf("[%d] %s <%s>\n", i, user.Name, email))
	}

	fmt.Fprintln(out, "found users:\n"+strings.Join(foundUsers, ""))
	fmt.Fprintln(out, "Total unique browsers", len(seenBrowsers))
}
