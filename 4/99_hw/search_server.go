package main

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
)

type Person struct {
	Id        int    `xml:"id"`
	Age       int    `xml:"age"`
	FirstName string `xml:"first_name"`
	LastName  string `xml:"last_name"`
	Name      string // surrogate value: FirstName + LastName
	About     string `xml:"about"`
	Gender    string `xml:"gender"`
}

type Catalog struct {
	Persons []Person `xml:"row"`
}

const filePath string = "./dataset.xml"

var cat = Catalog{}

func makeUser(person *Person) User {
	u := User{}
	u.Id = person.Id
	u.Age = person.Age
	u.Name = person.Name
	u.Gender = person.Gender
	u.About = strings.TrimSpace(person.About)
	return u
}

func checkOrderArg(order_field string) error {
	if !(strings.EqualFold(order_field, "Id") ||
		strings.EqualFold(order_field, "Name") ||
		strings.EqualFold(order_field, "Age")) {
		return fmt.Errorf("%s", ErrorBadOrderField)
	}
	return nil
}

func prepareOrderByArg(order_by string) (int, error) {
	if val, err := strconv.Atoi(order_by); err == nil {
		switch val {
			case OrderByAsc:
				fallthrough
			case OrderByAsIs:
				fallthrough
			case OrderByDesc:
				return val, nil
			default:
				return OrderByAsIs, fmt.Errorf("invalid argument")
		}
	} else {
		return OrderByAsIs, fmt.Errorf("invalid argument")
	}
}

func prepareLimitArg(limit string) (int, error) {
	if val, err := strconv.Atoi(limit); err == nil {
		return val, nil
	} else {
		return OrderByAsIs, fmt.Errorf("invalid argument")
	}
}

func prepareOffsetArg(offset string) (int, error) {
	if val, err := strconv.Atoi(offset); err == nil {
		return val, nil
	} else {
		return 0, fmt.Errorf("invalid argument")
	}
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

func SearchServer(r *http.Request) ([]User, error) {
	query := r.URL.Query().Get("query")
	order_field := r.URL.Query().Get("order_field")
	if err := checkOrderArg(order_field); err != nil {
		return nil, err
	}

	order_by, err := prepareOrderByArg(r.URL.Query().Get("order_by"))
	if err != nil {
		return nil, err
	}

	limit, err := prepareLimitArg(r.URL.Query().Get("limit"))
	if err != nil {
		return nil, err
	}

	offset, err := prepareOffsetArg(r.URL.Query().Get("offset"))
	if err != nil {
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
			sort.Slice(result, func(i, j int) bool {
				return cmp(result[i], result[j])
			})
		case OrderByDesc:
			sort.Slice(result, func(i, j int) bool {
				return !cmp(result[i], result[j])
			})
		case OrderByAsIs:
			// don't sort
		}
	}

	return result, nil
}

func LoadTestData() {
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
