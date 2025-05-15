# Отчет об оптимизации функции FastSearch

## Попытка 3
Дальнейшее профилирование по CPU показало следующую картину:
```
(pprof) top Fast
Active filters:
   focus=Fast
Showing nodes accounting for 1640ms, 50.62% of 3240ms total
Showing top 10 nodes out of 89
      flat  flat%   sum%        cum   cum%
     620ms 19.14% 19.14%      990ms 30.56%  encoding/json.checkValid
     280ms  8.64% 27.78%      280ms  8.64%  encoding/json.stateInString
     220ms  6.79% 34.57%      220ms  6.79%  encoding/json.unquoteBytes
     160ms  4.94% 39.51%      190ms  5.86%  encoding/json.(*decodeState).rescanLiteral
     100ms  3.09% 42.59%      100ms  3.09%  indexbytebody
      70ms  2.16% 44.75%       70ms  2.16%  internal/runtime/syscall.Syscall6
      60ms  1.85% 46.60%      130ms  4.01%  runtime.mallocgc
      50ms  1.54% 48.15%       60ms  1.85%  encoding/json.stateBeginValue
      40ms  1.23% 49.38%       40ms  1.23%  encoding/json.stateBeginString
      40ms  1.23% 50.62%     1970ms 60.80%  hw3.FastSearch
```
По-прежнему, конвертация из JSON в структуру занимает основное время по CPU.
А что с памятью? Картина примерно такая же:
```
(pprof) top Fast
Active filters:
   focus=Fast
Showing nodes accounting for 1276.66MB, 61.79% of 2066.25MB total
Dropped 27 nodes (cum <= 10.33MB)
Showing top 10 nodes out of 15
      flat  flat%   sum%        cum   cum%
  782.20MB 37.86% 37.86%   782.20MB 37.86%  io.ReadAll
  327.44MB 15.85% 53.70%  1284.23MB 62.15%  hw3.FastSearch
  110.51MB  5.35% 59.05%   110.51MB  5.35%  encoding/json.(*decodeState).literalStore
      35MB  1.69% 60.75%   167.02MB  8.08%  encoding/json.Unmarshal
      15MB  0.73% 61.47%   125.51MB  6.07%  encoding/json.(*decodeState).object
    6.50MB  0.31% 61.79%     6.50MB  0.31%  encoding/json.(*scanner).pushParseState
         0     0% 61.79%   101.51MB  4.91%  encoding/json.(*decodeState).array
         0     0% 61.79%   125.51MB  6.07%  encoding/json.(*decodeState).unmarshal
         0     0% 61.79%   125.51MB  6.07%  encoding/json.(*decodeState).value
         0     0% 61.79%     6.50MB  0.31%  encoding/json.checkValid
```
Вывод: дальнейшая оптимизация возможна только на счет отказа от использования библиотеки `encoding/json` и переписывания парсера. Возможные варианты:
1. Использовать готовое решение в духе easyjson. Хорошо для прода.
2. Реализовать парсер самостоятельно. Хорошо для тренировки.

Поскольку эту учебная задача, то лучше немного упороться и пойти по пути №2. Так интереснее.
...
После некоторого числа потраченного времени, удалось написать функцию `scanUser()` для парсинга строки JSON'a в структуру `User`. Бенчмарк показал следующие результаты:
```
$ GOGC=off go test -bench . -benchmem -cpuprofile cpu.out -memprofile mem.out
goos: linux
goarch: arm64
pkg: hw3
BenchmarkSlow-8               49          31974980 ns/op        19919985 B/op     182731 allocs/op
BenchmarkFast-8              218           5579692 ns/op         1877699 B/op      46595 allocs/op
PASS
ok      hw3     3.642s
```
 Получилось ускорить программу еще почти в 1.5 раза. Как видно, уменьшился, прежде всего, размер аллоцируемых на операцию данных, при этом само число аллокаций увеличилось, вернувшись к значению, полученному в попытке №1. Идем оптимизировать дальше.


## Попытка 2
Дальнейшее профилирование по CPU показало слещующую картину:
```
(pprof) tree Fast -cum
Active filters:
   focus=Fast
Showing nodes accounting for 1.88s, 35.40% of 5.31s total
Dropped 41 nodes (cum <= 0.03s)
Showing top 80 nodes out of 101
----------------------------------------------------------+-------------
      flat  flat%   sum%        cum   cum%   calls calls% + context              
----------------------------------------------------------+-------------
                                             2.07s 99.52% |   testing.(*B).launch
         0     0%     0%      2.08s 39.17%                | testing.(*B).runN
                                             2.02s 97.12% |   hw3.BenchmarkFast
                                             0.06s  2.88% |   hw3.BenchmarkSlow
----------------------------------------------------------+-------------
         0     0%     0%      2.07s 38.98%                | testing.(*B).launch
                                             2.07s   100% |   testing.(*B).runN
----------------------------------------------------------+-------------
                                             2.02s 99.51% |   hw3.BenchmarkFast
         0     0%     0%      2.03s 38.23%                | hw3.FastSearch
                                             1.74s 85.71% |   encoding/json.Unmarshal
                                             0.08s  3.94% |   io/ioutil.ReadAll (inline)
                                             0.08s  3.94% |   strings.Contains (inline)
                                             0.03s  1.48% |   strings.Split (inline)
                                             0.02s  0.99% |   fmt.Sprintf
                                             0.02s  0.99% |   memeqbody
                                             0.02s  0.99% |   os.Open (inline)
                                             0.02s  0.99% |   runtime.memmove
                                             0.01s  0.49% |   runtime.growslice
----------------------------------------------------------+-------------
                                             2.02s   100% |   testing.(*B).runN
         0     0%     0%      2.02s 38.04%                | hw3.BenchmarkFast
                                             2.02s   100% |   hw3.FastSearch
----------------------------------------------------------+-------------
                                             1.74s   100% |   hw3.FastSearch
         0     0%     0%      1.74s 32.77%                | encoding/json.Unmarshal
                                             1.04s 59.77% |   encoding/json.(*decodeState).unmarshal
                                             0.69s 39.66% |   encoding/json.checkValid
                                             0.01s  0.57% |   runtime.newobject
----------------------------------------------------------+-------------
```
Как видно, основное время ушло на демаршалинг в JSON (ну  немного на чтение файла). Попробуем  помочь маршаллеру, определив структуру `User` и разметив его поля.
В итоге, пришли к следующему результату:
```
$ GOGC=off go test -bench . -benchmem
goos: linux
goarch: arm64
pkg: hw3
BenchmarkSlow-8               48          28102877 ns/op        19918322 B/op     182730 allocs/op
BenchmarkFast-8              147           8003122 ns/op         5335640 B/op      12806 allocs/op
PASS
ok      hw3     3.506s
```
Как видно, количество аллокаций стало почти в 4 раза меньше, но суммарно скорость повысилась только 1.4 раза. Драматического прироста производительности не случилось, оптимизируем дальше.


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