package main

// код писать тут

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
)

func writeSearchError(w http.ResponseWriter, err error) {
	errResponce := &SearchErrorResponse{}
	errResponce.Error = err.Error()
	json, err := json.Marshal(errResponce)
	if err == nil {
		fmt.Fprintf(w, "%s", json)
	}
}

func searchHandler(w http.ResponseWriter, r *http.Request) {
	result, err := SearchServer(r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		writeSearchError(w, err)
		return
	}

	jsonResult, err := json.Marshal(result)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, err)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "%s", jsonResult)
}

type TestCase struct {
	ErrorStr string
	SearchRequest
	SearchResponse
}

func compareUser(lhs, rhs User) bool {
	return lhs.Id == rhs.Id && lhs.Name == rhs.Name && lhs.Age == rhs.Age && lhs.Gender == rhs.Gender && lhs.About == rhs.About
}

func TestServerClientPositive(t *testing.T) {
	LoadTestData()
	ts := httptest.NewServer(http.HandlerFunc(searchHandler))
	defer ts.Close()

	client := &SearchClient{
		AccessToken: "testToken",
		URL:         ts.URL,
	}

	cases := []TestCase{
		{
			ErrorStr: "",
			SearchRequest: SearchRequest{
				Limit:      1,
				Offset:     0,
				Query:      "Annie",
				OrderField: "Id",
				OrderBy:    OrderByAsIs,
			},
			SearchResponse: SearchResponse{
				Users: []User{
					{
						Id:     16,
						Name:   "AnnieOsborn",
						Age:    35,
						About:  "Consequat fugiat veniam commodo nisi nostrud culpa pariatur. Aliquip velit adipisicing dolor et nostrud. Eu nostrud officia velit eiusmod ullamco duis eiusmod ad non do quis.",
						Gender: "female",
					}},
				NextPage: false,
			},
		},
		{
			ErrorStr: "",
			SearchRequest: SearchRequest{
				Limit:      2,
				Offset:     7,
				Query:      "magna",
				OrderField: "Id",
				OrderBy:    OrderByAsIs,
			},
			SearchResponse: SearchResponse{
				Users: []User{
					{
						Id:     7,
						Name:   "LeannTravis",
						Age:    34,
						About:  "Lorem magna dolore et velit ut officia. Cupidatat deserunt elit mollit amet nulla voluptate sit. Quis aute aliquip officia deserunt sint sint nisi. Laboris sit et ea dolore consequat laboris non. Consequat do enim excepteur qui mollit consectetur eiusmod laborum ut duis mollit dolor est. Excepteur amet duis enim laborum aliqua nulla ea minim.",
						Gender: "female",
					},
					{
						Id:     8,
						Name:   "GlennJordan",
						Age:    29,
						About:  "Duis reprehenderit sit velit exercitation non aliqua magna quis ad excepteur anim. Eu cillum cupidatat sit magna cillum irure occaecat sunt officia officia deserunt irure. Cupidatat dolor cupidatat ipsum minim consequat Lorem adipisicing. Labore fugiat cupidatat nostrud voluptate ea eu pariatur non. Ipsum quis occaecat irure amet esse eu fugiat deserunt incididunt Lorem esse duis occaecat mollit.",
						Gender: "male",
					}},
				NextPage: true,
			},
		},
		{
			ErrorStr: "",
			SearchRequest: SearchRequest{
				Limit:      2,
				Offset:     7,
				Query:      "magna",
				OrderField: "Age",
				OrderBy:    OrderByAsc,
			},
			SearchResponse: SearchResponse{
				Users: []User{
					{
						Id:     8,
						Name:   "GlennJordan",
						Age:    29,
						About:  "Duis reprehenderit sit velit exercitation non aliqua magna quis ad excepteur anim. Eu cillum cupidatat sit magna cillum irure occaecat sunt officia officia deserunt irure. Cupidatat dolor cupidatat ipsum minim consequat Lorem adipisicing. Labore fugiat cupidatat nostrud voluptate ea eu pariatur non. Ipsum quis occaecat irure amet esse eu fugiat deserunt incididunt Lorem esse duis occaecat mollit.",
						Gender: "male",
					},
					{
						Id:     7,
						Name:   "LeannTravis",
						Age:    34,
						About:  "Lorem magna dolore et velit ut officia. Cupidatat deserunt elit mollit amet nulla voluptate sit. Quis aute aliquip officia deserunt sint sint nisi. Laboris sit et ea dolore consequat laboris non. Consequat do enim excepteur qui mollit consectetur eiusmod laborum ut duis mollit dolor est. Excepteur amet duis enim laborum aliqua nulla ea minim.",
						Gender: "female",
					}},
				NextPage: true,
			},
		},
	}

	for caseNum, item := range cases {
		response, err := client.FindUsers(item.SearchRequest)
		if err != nil {
			t.Errorf("[%d] got `%s`, expected no errors",
				caseNum, err.Error())
		}

		if response == nil {
			t.Errorf("[%d] wrong Response: got <nil>, expected `%v`",
				caseNum, &item.SearchResponse)
			continue
		}

		if !slices.EqualFunc(response.Users, item.SearchResponse.Users, compareUser) {
			t.Errorf("[%d] wrong Response: got %+v, expected %+v",
				caseNum, response, &item.SearchResponse)
		}
	}

}

func TestServerClientErrors(t *testing.T) {
	LoadTestData()
	ts := httptest.NewServer(http.HandlerFunc(searchHandler))
	defer ts.Close()

	client := &SearchClient{
		AccessToken: "testToken",
		URL:         ts.URL,
	}

	cases := []TestCase{
		{
			ErrorStr: ErrorBadOrderField,
			SearchRequest: SearchRequest{
				Limit:      1,
				Offset:     0,
				Query:      "za",
				OrderField: "About",
				OrderBy:    OrderByAsc,
			},
		},
		{
			ErrorStr: "limit must be > 0",
			SearchRequest: SearchRequest{
				Limit:      -1,
				Offset:     0,
				Query:      "za",
				OrderField: "About",
				OrderBy:    OrderByAsc,
			},
		},
		{
			ErrorStr: "offset must be > 0",
			SearchRequest: SearchRequest{
				Limit:      1,
				Offset:     -1,
				Query:      "za",
				OrderField: "About",
				OrderBy:    OrderByAsc,
			},
		},
		{
			ErrorStr: "",
			SearchRequest: SearchRequest{
				Limit:      1,
				Offset:     1,
				Query:      "a",
				OrderField: "Id",
				OrderBy:    -100,
			},
		},
	}

	for caseNum, item := range cases {
		_, err := client.FindUsers(item.SearchRequest)
		if err == nil {
			t.Errorf("[%d] got error <nil>, expected `%s`",
				caseNum, item.ErrorStr)
			continue
		}

		if err.Error() != item.ErrorStr &&
			!strings.Contains(err.Error(), item.ErrorStr) {
			t.Errorf("[%d] wrong error: got `%s`, expected `%s`",
				caseNum, err.Error(), item.ErrorStr)
		}
	}
}
