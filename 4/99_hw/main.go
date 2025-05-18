package main

import (
	"net/http"
	"os"
	"encoding/xml"
	"fmt"
	"strings"
	"sort"
	"strconv"
)

const filePath string = "./dataset.xml"

type Person struct {
	Id int           `xml:"id"`
	Age int          `xml:"age"`
	FirstName string `xml:"first_name"`
	LastName string  `xml:"last_name"`
	Name string      // surrogate value: FirstName + LastName
	About string     `xml:"about"`
	Gender string    `xml:"gender"`
}

type Catalog struct {
	Persons  []Person `xml:"row"`
}

var cat = Catalog{}

func loadData() {
	data, err := os.ReadFile(filePath)
	if err != nil {
		panic(err)
	}

	err = xml.Unmarshal(data, &cat)
	if err != nil {
		panic(err)
	}

	for i := range cat.Persons {
		cat.Persons[i].Name = cat.Persons[i].FirstName + cat.Persons[i].LastName
	}
}

func makeUser(person *Person) User {
	u := User{}
	u.Id = person.Id
	u.Age = person.Age
	u.Name = person.Name
	u.Gender = person.Gender
	return u
}

func checkOrderField(order_field *string) error {
	if len(*order_field) > 0 {
		if !(strings.EqualFold(*order_field, "Id") || 
		     strings.EqualFold(*order_field, "Name") || 
			 strings.EqualFold(*order_field, "Age")) {
				return fmt.Errorf("%s", ErrorBadOrderField)
			 }
	}
	return nil
}

func cmpById(lhs, rhs User) bool {
	return lhs.Id < rhs.Id
}

func cmpByName(lhs, rhs User) bool {
	return lhs.Name < rhs.Name
}

func cmpByAge(lhs, rhs User) bool {
	return lhs.Age < rhs.Age
}

func getCmpFunction(order_field *string) func(lhs, rhs User) bool {
	switch *order_field {
	case "Id":
		return cmpById
	case "":
		fallthrough
	case "Name":
		return cmpByName
	case "Age":
		return cmpByAge
	}
	return nil
}

func SearchServer(query, order_field string, order_by, limit, offset int) ([]User, error) {
	if err := checkOrderField(&order_field); err != nil {
		return nil, err
	}

	result := make([]User, 0, 10)

	// searching
	for idx, p := range cat.Persons {
		if idx < offset {
			continue
		}
		
		found := len(query) == 0
		if !found {
			found = strings.Contains(p.Name, query)
			if !found {
				found = strings.Contains(p.About, query)
			}
		}

		if found {
			result = append(result, makeUser(&p))

			if limit > 0 && len(result) == limit {
				break
			}
		}
	}

	// sorting
	if len(result) > 1 {
		cmp := getCmpFunction(&order_field)
		switch order_by {
		case OrderByAsc:
			sort.SliceStable(result, func (i, j int) bool {
				return cmp(result[i], result[j])
			})
		case OrderByDesc:
			sort.SliceStable(result, func (i, j int) bool {
				return !cmp(result[i], result[j])
			})
		case OrderByAsIs:
			// don't sort
		}
	}

	return result, nil
}

func handler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("query")

	order_field := r.URL.Query().Get("order_field")

	order_by, err := strconv.Atoi(r.URL.Query().Get("order_by"))
	if (err != nil) || 
	   !(order_by == OrderByAsIs || order_by == OrderByAsc || order_by == OrderByDesc) {
		fmt.Fprintln(w, "`order_by` has wrong value")
		return
	}

	limit := -1
	if val, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil {
		limit = val
	}

	offset := 0
	offset_str := r.URL.Query().Get("offset")
	if offset_str != "" {
		if val, err := strconv.Atoi(offset_str); err == nil {
			offset = val
		}
	}
	
	result, err := SearchServer(query, order_field, order_by, limit, offset)
	if err != nil {
		fmt.Fprintln(w, err)
	}
	fmt.Fprintf(w, "result: %v", result)
}

func main() {
	loadData()

	http.HandleFunc("/", handler)

	fmt.Println("starting server at :8080")
	http.ListenAndServe(":8080", nil)
}