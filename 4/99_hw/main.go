package main

import (
	//"net/http"
	"os"
	"encoding/xml"
	"fmt"
	"strings"
	"sort"
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
		if !(strings.Contains(*order_field, "Id") || 
		     strings.Contains(*order_field, "Name") || 
			 strings.Contains(*order_field, "Age")) {
				return fmt.Errorf(ErrorBadOrderField)
			 }
	}
	return nil
}

func cmpById(lhs, rhs *User) bool {
	return lhs.Id < rhs.Id
}

func cmpByName(lhs, rhs *User) bool {
	return lhs.Name < rhs.Name
}

func cmpByAge(lhs, rhs *User) bool {
	return lhs.Age < rhs.Age
}

func getCmpFunction(order_field *string) func(lhs, rhs *User) bool {
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
	cmp := getCmpFunction(&order_field)
	switch order_by {
	case OrderByAsc:
		sort.SliceStable(result, func (i, j int) bool {
			return cmp(&result[i], &result[j])
		})
	case OrderByDesc:
		sort.SliceStable(result, func (i, j int) bool {
			return !cmp(&result[i], &result[j])
		})
	case OrderByAsIs:
		// don't sort
	}

	return result, nil
}

func main() {
	loadData()

	// perform search
	res1, err := SearchServer("", "Id", OrderByAsIs, -1, 0)
	if err != nil {
		panic(err)
	}
	fmt.Printf("search 1:\n===\n %v\n===\n\n", res1)
	
	res2, err := SearchServer("Guerr", "Id", OrderByDesc, 1, 0)
	if err != nil {
		panic(err)
	}
	fmt.Printf("search 2:\n===\n %v\n===\n\n", res2)
}