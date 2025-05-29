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
	ID        int    `xml:"id"`
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

func makeUser(rec *Record) User {
	return User{
		Id:     rec.ID,
		Age:    rec.Age,
		Name:   rec.Name,
		Gender: rec.Gender,
		About:  strings.TrimSpace(rec.About),
	}
}

func checkOrderArg(orderField string) error {
	if !(strings.EqualFold(orderField, "Id") ||
		strings.EqualFold(orderField, "Name") ||
		strings.EqualFold(orderField, "Age")) {
		return fmt.Errorf("%s", "ErrorBadOrderField")
	}
	return nil
}

func prepareOrderByArg(orderBy string) (int, error) {
	val, err := strconv.Atoi(orderBy)
	if err == nil {
		switch val {
		case OrderByAsc, OrderByAsIs, OrderByDesc:
			return val, nil
		default:
			return 0, fmt.Errorf("%s", "order_by must be in a range [-1, 1]")
		}
	}
	return 0, err
}

func prepareNumberArg(limit string) (int, error) {
	val, err := strconv.Atoi(limit)
	if err != nil {
		return 0, err
	}
	return val, nil
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

func getCmpFunction(orderField string) func(lhs, rhs User) bool {
	switch orderField {
	case "Id":
		return cmpById
	case "", "Name":
		return cmpByName
	case "Age":
		return cmpByAge
	}
	return nil
}

func SearchServer(catalog *Catalog, r *http.Request) (*SearchResponse, error) {
	// if there is no data in the "database"
	// then return an internal server error because
	// there is no ability to handle any request at all.
	if catalog == nil || len(catalog.Records) == 0 {
		return nil, fmt.Errorf("InternalServerError")
	}

	query := r.URL.Query().Get("query")
	orderField := r.URL.Query().Get("order_field")
	if err := checkOrderArg(orderField); err != nil {
		return nil, err
	}

	orderBy, err := prepareOrderByArg(r.URL.Query().Get("order_by"))
	if err != nil {
		return nil, fmt.Errorf("prepare orderBy arg: [%w]", err)
	}

	limit, err := prepareNumberArg(r.URL.Query().Get("limit"))
	if err != nil {
		return nil, fmt.Errorf("prepare limit arg: [%w]", err)
	}

	offset, err := prepareNumberArg(r.URL.Query().Get("offset"))
	if err != nil {
		return nil, fmt.Errorf("prepare offset arg: [%w]", err)
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
		cmp := getCmpFunction(orderField)
		switch orderBy {
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

func LoadTestData() (*Catalog, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("could not load test data: [%w]", err)
	}

	var catalog Catalog
	err = xml.Unmarshal(data, &catalog)
	if err != nil {
		return nil, fmt.Errorf("could not unmarshal test data: [%w]", err)
	}

	for i := range catalog.Records {
		// optimization: make surrogate value to speed up furner search
		catalog.Records[i].Name = catalog.Records[i].FirstName + catalog.Records[i].LastName
	}

	return &catalog, nil
}
