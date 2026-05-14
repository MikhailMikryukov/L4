package or

import (
	"fmt"
	"time"
)

// Базовый пример
func ExampleOr_basic() {
	sig := func(after time.Duration) <-chan interface{} {
		c := make(chan interface{})
		go func() {
			defer close(c)
			time.Sleep(after)
		}()
		return c
	}

	start := time.Now()
	<-Or(
		sig(1*time.Second),
		sig(2*time.Second),
		sig(3*time.Second),
	)

	fmt.Printf("Done after %v\n", time.Since(start))
}

// Пример с таймаутом
func ExampleOr_timeout() {
	work := func() <-chan interface{} {
		c := make(chan interface{})
		go func() {
			defer close(c)
			time.Sleep(5 * time.Second) // долгая работа
			c <- "result"
		}()
		return c
	}

	// Таймаут
	t := func(d time.Duration) <-chan interface{} {
		ch := make(chan interface{})
		go func() {
			time.Sleep(d)
			close(ch)
		}()
		return ch
	}(1 * time.Second)

	select {
	case result := <-Or(work(), t):
		fmt.Printf("Got: %v\n", result)
	case <-t:
		fmt.Println("Timeout!")
	}
}

// Пример с отменой через канал
func ExampleOr_cancellation() {
	cancel := make(chan interface{})
	data := make(chan interface{})

	// Горутина, которая генерирует данные
	go func() {
		for i := 0; i < 10; i++ {
			select {
			case data <- i:
				time.Sleep(100 * time.Millisecond)
			case <-cancel:
				close(data)
				return
			}
		}
	}()

	// Отменяем через 250ms
	go func() {
		time.Sleep(250 * time.Millisecond)
		close(cancel)
	}()

	// Читаем данные, пока не будет отмена
	for val := range Or(data, cancel) {
		fmt.Printf("Received: %v\n", val)
		if val == nil {
			break
		}
	}
}

// Пример ожидание первого успешного результата
func ExampleOr_firstResult() {
	fetchAPI := func(url string, delay time.Duration) <-chan interface{} {
		c := make(chan interface{})
		go func() {
			defer close(c)
			time.Sleep(delay)
			c <- fmt.Sprintf("Result from %s", url)
		}()
		return c
	}

	// Отправляем запросы на несколько серверов
	result := Or(
		fetchAPI("server1.com", 100*time.Millisecond),
		fetchAPI("server2.com", 200*time.Millisecond),
		fetchAPI("server3.com", 50*time.Millisecond),
	)

	res := <-result
	fmt.Println(res)
}
