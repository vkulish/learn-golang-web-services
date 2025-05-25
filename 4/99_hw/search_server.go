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

type Record struct {
	Id        int    `xml:"id"`
	Age       int    `xml:"age"`
	FirstName string `xml:"first_name"`
	LastName  string `xml:"last_name"`
	Name      string // surrogate value for search optimization: FirstName + LastName
	About     string `xml:"about"`
	Gender    string `xml:"gender"`
}

type Catalog struct {
	Records []Record `xml:"row"`
}

const filePath string = "./dataset.xml"

var catalog = &Catalog{}

func makeUser(rec *Record) User {
	u := User{}
	u.Id = rec.Id
	u.Age = rec.Age
	u.Name = rec.Name
	u.Gender = rec.Gender
	u.About = strings.TrimSpace(rec.About)
	return u
}

func checkOrderArg(order_field string) error {
	if !(strings.EqualFold(order_field, "Id") ||
		strings.EqualFold(order_field, "Name") ||
		strings.EqualFold(order_field, "Age")) {
		return fmt.Errorf("%s", "ErrorBadOrderField")
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
		}
	}
	return OrderByAsIs, fmt.Errorf("invalid argument")
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

func SearchServer(r *http.Request) (*SearchResponse, error) {
	// if there is no data in the "database"
	// then return an internal server error because
	// there is no ability to handle any request at all.
	if catalog == nil || len(catalog.Records) == 0 {
		return nil, fmt.Errorf("InternalServerError")
	}

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

	result := &SearchResponse{}
	result.Users = make([]User, 0, 10)

	// searching
	for idx, p := range catalog.Records {
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
			if limit > 0 && len(result.Users) == limit {
				result.NextPage = true
				break
			}

			result.Users = append(result.Users, makeUser(&p))
		}
	}

	// sorting
	if len(result.Users) > 1 {
		cmp := getCmpFunction(&order_field)
		switch order_by {
		case OrderByAsc:
			sort.Slice(result.Users, func(i, j int) bool {
				return cmp(result.Users[i], result.Users[j])
			})
		case OrderByDesc:
			sort.Slice(result.Users, func(i, j int) bool {
				return !cmp(result.Users[i], result.Users[j])
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

	catalog = &Catalog{}
	err = xml.Unmarshal(data, catalog)
	if err != nil {
		panic(err)
	}

	for i := range catalog.Records {
		// optimization: make surrogate value to speed up furner search
		catalog.Records[i].Name = catalog.Records[i].FirstName + catalog.Records[i].LastName
	}
}

func ClearTestData() {
	catalog = nil
}
