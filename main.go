package main

import (
	"fmt"
	"sync"
)

type Cache struct {
	m sync.Mutex
	v map[string]uint
}

func (s *Cache) Exist(key string) bool {
	s.m.Lock()
	defer s.m.Unlock()
	if _, ok := s.v[key]; ok {
		s.v[key]++
		return true
	} else {
		s.v[key] = 1
		return false
	}
}

func (s *Cache) String() string {
	s.m.Lock()
	defer s.m.Unlock()
	var sb string
	for k, v := range s.v {
		sb += fmt.Sprintf("key: %s => value: %q", k, v)
	}
	return fmt.Sprintln(sb)
}

type Fetcher interface {
	Fetch(url string) (body string, urls []string, err error)
}

//
// Serial crawler
//
func SerialCrawl(url string, depth int, fetcher Fetcher, cache *Cache) {
	if cache.Exist(url) {
		return
	}
	//fmt.Printf("cached values %q\n", cache)

	fmt.Println("depth ", depth)
	if depth <= 0 {
		return
	}
	body, urls, err := fetcher.Fetch(url)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("Found: %s %q\n", url, body)
	//fmt.Printf("fetch children %q\n", urls)
	for _, u := range urls {
		//fmt.Printf("recursive crawl url %s\n", u)
		SerialCrawl(u, depth-1, fetcher, cache)
	}
}

//
// Concurrent crawler with shared state and Mutex
//

func ConcurrentMutex(url string, depth int, fetcher Fetcher, cache *Cache) {
	if cache.Exist(url) {
		return
	}
	//fmt.Printf("cached values %q\n", cache)

	if depth <= 0 {
		return
	}

	body, urls, err := fetcher.Fetch(url)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("Found: %s %q\n", url, body)
	//fmt.Printf("fetch children %q\n", urls)
	var done sync.WaitGroup
	for _, u := range urls {
		done.Add(1)
		//fmt.Printf("recursive crawl url %s\n", u)
		u2 := u
		go func() {
			defer done.Done()
			ConcurrentMutex(u2, depth-1, fetcher, cache)
		}()
		//go func(u string) {
		//	defer done.Done()
		//	ConcurrentMutex(u, depth -1,fetcher, cache)
		//}(u)
	}
	done.Wait()
}

//
// Concurrent crawler with channels
//

func worker(url string, ch chan []string, fetcher Fetcher, cache *Cache) {
	body, urls, err := fetcher.Fetch(url)
	if err != nil {
		ch <- []string{}
	} else {
		fmt.Printf("Found: %s %q\n", url, body)
		ch <- urls
	}
}

func master(url string, ch chan []string, depth *uint, fetcher Fetcher, cache *Cache) {
	n := 1
	for urls := range ch {
		fmt.Printf("dep :%s\n", fmt.Sprintf("%q", *depth))
		if *depth == 0 {
			break
		}
		for _, url := range urls {
			if !cache.Exist(url) {
				n += 1
				go worker(url, ch, fetcher, cache)
			}
		}

		n -= 1
		if n == 0 {
			break
		}
	}
}

func ConcurrentChan(url string, depth uint, fetcher Fetcher, cache *Cache) {
	ch := make(chan []string)
	go func() {
		ch <- []string{url}
	}()
	master(url, ch, &depth, fetcher, cache)
}

func main() {
	cache := Cache{v: make(map[string]uint)}
	//SerialCrawl("https://golang.org/", 4, fetcher, &cache)

	//ConcurrentMutex("https://golang.org/", 4, fetcher, &cache)

	ConcurrentChan("https://golang.org/", 2, fetcher, &cache)

	fmt.Printf("\nCached urls %q\n", cache.v)
}

type fakeFetcher map[string]*fakeResult // to avoid copy value

type fakeResult struct {
	body string
	urls []string
}

func (f fakeFetcher) Fetch(url string) (string, []string, error) {
	if res, ok := f[url]; ok {
		return res.body, res.urls, nil
	}
	return "", nil, fmt.Errorf("not found: %s", url)
}

var fetcher = fakeFetcher{
	"https://golang.org/": &fakeResult{
		"The Go Programming Language",
		[]string{
			"https://golang.org/pkg/",
			"https://golang.org/cmd/",
		},
	},
	"https://golang.org/pkg/": &fakeResult{
		"Packages",
		[]string{
			"https://golang.org/",
			"https://golang.org/cmd/",
			"https://golang.org/pkg/fmt/",
			"https://golang.org/pkg/os/",
		},
	},
	"https://golang.org/pkg/fmt/": &fakeResult{
		"Package fmt",
		[]string{
			"https://golang.org/",
			"https://golang.org/pkg/",
		},
	},
	"https://golang.org/pkg/os/": &fakeResult{
		"Package os",
		[]string{
			"https://golang.org/",
			"https://golang.org/pkg/",
		},
	},
}
