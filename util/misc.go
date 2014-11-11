package util

var SequentialInts chan int = make(chan int, 20)

func init() {
	i := 0
	go func() {
		for {
			SequentialInts <- i
			i++
		}
	}()
}
