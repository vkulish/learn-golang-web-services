# Отчет об оптимизации функции FastSearch

## Попытка 1
После переноса кода из `SlowSearch()` as is был снят профиль CPU:
```
(pprof) tree FastSearch -cum
Active filters:
   focus=FastSearch
Showing nodes accounting for 30ms, 60.00% of 50ms total
----------------------------------------------------------+-------------
      flat  flat%   sum%        cum   cum%   calls calls% + context              
----------------------------------------------------------+-------------
                                              30ms   100% |   hw3.TestSearch
         0     0%     0%       30ms 60.00%                | hw3.FastSearch
                                              20ms 66.67% |   regexp.MatchString
                                              10ms 33.33% |   io/ioutil.ReadAll (inline)
----------------------------------------------------------+-------------
                                              30ms   100% |   testing.tRunner
         0     0%     0%       30ms 60.00%                | hw3.TestSearch
                                              30ms   100% |   hw3.FastSearch
----------------------------------------------------------+-------------
                                              10ms 33.33% |   io.ReadAll
                                              10ms 33.33% |   regexp/syntax.(*compiler).inst
                                              10ms 33.33% |   regexp/syntax.(*parser).maybeConcat
         0     0%     0%       30ms 60.00%                | runtime.growslice
                                              20ms 66.67% |   runtime.memclrNoHeapPointers
                                              10ms 33.33% |   runtime.mallocgc
----------------------------------------------------------+-------------
         0     0%     0%       30ms 60.00%                | testing.tRunner
                                              30ms   100% |   hw3.TestSearch
----------------------------------------------------------+-------------
                                              20ms   100% |   regexp.MatchString (inline)
         0     0%     0%       20ms 40.00%                | regexp.Compile
                                              20ms   100% |   regexp.compile
----------------------------------------------------------+-------------
                                              20ms   100% |   hw3.FastSearch
         0     0%     0%       20ms 40.00%                | regexp.MatchString
                                              20ms   100% |   regexp.Compile (inline)
----------------------------------------------------------+-------------
                                              20ms   100% |   regexp.Compile
         0     0%     0%       20ms 40.00%                | regexp.compile
                                              10ms 50.00% |   regexp/syntax.Compile
                                              10ms 50.00% |   regexp/syntax.Parse (inline)
----------------------------------------------------------+-------------
                                              20ms   100% |   runtime.growslice
      20ms 40.00% 40.00%       20ms 40.00%                | runtime.memclrNoHeapPointers
----------------------------------------------------------+-------------
                                              10ms   100% |   io/ioutil.ReadAll
         0     0% 40.00%       10ms 20.00%                | io.ReadAll
                                              10ms   100% |   runtime.growslice
----------------------------------------------------------+-------------
                                              10ms   100% |   hw3.FastSearch (inline)
         0     0% 40.00%       10ms 20.00%                | io/ioutil.ReadAll
                                              10ms   100% |   io.ReadAll
----------------------------------------------------------+-------------
                                              10ms   100% |   regexp/syntax.Compile (inline)
         0     0% 40.00%       10ms 20.00%                | regexp/syntax.(*compiler).inst
                                              10ms   100% |   runtime.growslice
----------------------------------------------------------+-------------
                                              10ms   100% |   regexp/syntax.parse
         0     0% 40.00%       10ms 20.00%                | regexp/syntax.(*parser).literal
                                              10ms   100% |   regexp/syntax.(*parser).push
----------------------------------------------------------+-------------
                                              10ms   100% |   regexp/syntax.(*parser).push
         0     0% 40.00%       10ms 20.00%                | regexp/syntax.(*parser).maybeConcat
                                              10ms   100% |   runtime.growslice
----------------------------------------------------------+-------------
                                              10ms   100% |   regexp/syntax.(*parser).literal
         0     0% 40.00%       10ms 20.00%                | regexp/syntax.(*parser).push
                                              10ms   100% |   regexp/syntax.(*parser).maybeConcat
----------------------------------------------------------+-------------
                                              10ms   100% |   regexp.compile
         0     0% 40.00%       10ms 20.00%                | regexp/syntax.Compile
                                              10ms   100% |   regexp/syntax.(*compiler).inst (inline)
----------------------------------------------------------+-------------
                                              10ms   100% |   regexp.compile (inline)
         0     0% 40.00%       10ms 20.00%                | regexp/syntax.Parse
                                              10ms   100% |   regexp/syntax.parse
----------------------------------------------------------+-------------
                                              10ms   100% |   regexp/syntax.Parse
         0     0% 40.00%       10ms 20.00%                | regexp/syntax.parse
                                              10ms   100% |   regexp/syntax.(*parser).literal
----------------------------------------------------------+-------------
                                              10ms   100% |   runtime.mallocgc
      10ms 20.00% 60.00%       10ms 20.00%                | runtime.heapSetType
----------------------------------------------------------+-------------
                                              10ms   100% |   runtime.growslice
         0     0% 60.00%       10ms 20.00%                | runtime.mallocgc
                                              10ms   100% |   runtime.heapSetType
----------------------------------------------------------+-------------
```

Можно видеть, что вклад функции `regexp/syntax.Compile` весьма значительный. Ну и в коде видно, что паттерны компилируются прям в цикле, хотя можно и нужно это делать 1 раз до циклов.

После такой простой оптимизации, перформанс следующий:
```
$ go test -bench . -benchmem
goos: linux
goarch: arm64
pkg: hw3
BenchmarkSlow-8               42          27124940 ns/op        20238852 B/op     182838 allocs/op
BenchmarkFast-8              100          10443073 ns/op         6209823 B/op      46782 allocs/op
PASS
ok      hw3     3.263s
```
Как видно, ускорение составило 2.5 раза. Осталось ускорить еще в 5 раз.