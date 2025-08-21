package main

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

// сюда писать код

func ExecutePipeline(jobs ...job) {
	var input chan any
	var output chan any
	wg := &sync.WaitGroup{}
	for i, j := range jobs {
		output = make(chan any, 100)
		wg.Add(1)
		go func(j job, num int, in, out chan any, wg *sync.WaitGroup) {
			defer wg.Done()
			defer close(out)
			j(in, out)
		}(j, i, input, output, wg)

		input = output
	}

	wg.Wait()
}

func SingleHash(in, out chan any) {
	wg := &sync.WaitGroup{}
	for v := range in {
		data := fmt.Sprintf("%v", v)
		md5 := DataSignerMd5(data)
		wg.Add(1)
		go func(wg *sync.WaitGroup, data, md5 string, out chan any) {
			defer wg.Done()
			a := DataSignerCrc32(data)
			b := DataSignerCrc32(md5)
			out <- fmt.Sprintf("%s~%s", a, b)
		}(wg, data, md5, out)
	}
	wg.Wait()
}

func MultiHash(in, out chan any) {
	for v := range in {
		data := fmt.Sprintf("%v", v)
		wg := &sync.WaitGroup{}
		var result [6]string
		for i := range 6 {
			wg.Add(1)
			go func(i int, data string, result *[6]string) {
				defer wg.Done()
				result[i] = DataSignerCrc32(fmt.Sprintf("%v%v", i, data))
			}(i, data, &result)
		}
		wg.Wait()
		out <- strings.Join(result[:], "")
	}
}

func CombineResults(in, out chan any) {
	data := make([]string, 0, 100)
	for v := range in {
		data = append(data, fmt.Sprintf("%v", v))
	}

	sort.Slice(data, func(i, j int) bool {
		return data[i] < data[j]
	})

	out <- strings.Join(data, "_")
}
