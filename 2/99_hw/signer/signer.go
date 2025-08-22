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
	for _, j := range jobs {
		output = make(chan any, 100)
		wg.Add(1)
		go func(j job, in, out chan any, wg *sync.WaitGroup) {
			defer wg.Done()
			defer close(out)
			j(in, out)
		}(j, input, output, wg)

		input = output
	}

	wg.Wait()
}

func SingleHash(in chan any, out chan any) {
	wg := &sync.WaitGroup{}
	mtx := &sync.Mutex{}
	for v := range in {
		wg.Add(1)
		go func(wg *sync.WaitGroup, out chan any) {
			defer wg.Done()

			data := fmt.Sprintf("%v", v)

			// To prevent from overheating guard call to
			// DataSignerMd5() from multithreading
			mtx.Lock()
			md5 := DataSignerMd5(data)
			mtx.Unlock()

			wg2 := &sync.WaitGroup{}
			var a, b string
			wg2.Add(2)
			go func() {
				defer wg2.Done()
				a = DataSignerCrc32(data)
			}()
			go func() {
				defer wg2.Done()
				b = DataSignerCrc32(md5)
			}()
			wg2.Wait()
			out <- fmt.Sprintf("%s~%s", a, b)
		}(wg, out)
	}
	wg.Wait()
}

func MultiHash(in, out chan any) {
	wg := &sync.WaitGroup{}
	for v := range in {
		wg.Add(1)
		go func() {
			defer wg.Done()
			data := fmt.Sprintf("%v", v)
			wg2 := &sync.WaitGroup{}
			var result [6]string
			for i := range 6 {
				wg2.Add(1)
				go func(i int, data string, result *[6]string) {
					defer wg2.Done()
					result[i] = DataSignerCrc32(fmt.Sprintf("%v%v", i, data))
				}(i, data, &result)
			}
			wg2.Wait()
			out <- strings.Join(result[:], "")
		}()
	}
	wg.Wait()
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
